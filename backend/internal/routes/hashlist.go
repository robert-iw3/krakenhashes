package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/config"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	adminclient "github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/admin/client"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/processor"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	clientsvc "github.com/ZerkerEOD/krakenhashes/backend/internal/services/client"
	retentionsvc "github.com/ZerkerEOD/krakenhashes/backend/internal/services/retention"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// SetupHashlistRoutes configures minimal hashlist routes
// The comprehensive hashlist routes are configured in registerHashlistRoutes
func SetupHashlistRoutes(jwtRouter *mux.Router) {
	debug.Info("Skipping SetupHashlistRoutes - using registerHashlistRoutes instead")
}

// hashlistHandler handles HTTP requests for hashlist-related operations
type hashlistHandler struct {
	db                 *db.DB
	hashlistRepo       *repository.HashListRepository
	hashTypeRepo       *repository.HashTypeRepository
	clientRepo         *repository.ClientRepository
	hashRepo           *repository.HashRepository
	fileRepo           *repository.FileRepository
	clientSettingsRepo *repository.ClientSettingsRepository
	systemSettingsRepo *repository.SystemSettingsRepository
	dataDir            string // Base directory for storing hashlist files
	cfg                *config.Config
	agentService       *services.AgentService
	processor          *processor.HashlistDBProcessor
	// Job-related dependencies
	jobsHandler interface {
		GetAvailablePresetJobs(w http.ResponseWriter, r *http.Request)
		CreateJobFromHashlist(w http.ResponseWriter, r *http.Request)
	}
}

// registerHashlistRoutes configures all hashlist, hash type, client, and hash search routes
func registerHashlistRoutes(r *mux.Router, sqlDB *sql.DB, cfg *config.Config, agentService *services.AgentService, jobsHandler interface {
	GetAvailablePresetJobs(w http.ResponseWriter, r *http.Request)
	CreateJobFromHashlist(w http.ResponseWriter, r *http.Request)
}) {
	debug.Info("Registering hashlist, hash type, client, and hash search routes")

	// Create DB wrapper for repositories
	database := &db.DB{DB: sqlDB}

	// Create repositories
	hashlistRepo := repository.NewHashListRepository(database)
	hashTypeRepo := repository.NewHashTypeRepository(database)
	clientRepo := repository.NewClientRepository(database)
	hashRepo := repository.NewHashRepository(database)
	fileRepo := repository.NewFileRepository(database, cfg.HashUploadDir)
	clientSettingsRepo := repository.NewClientSettingsRepository(database)
	systemSettingsRepo := repository.NewSystemSettingsRepository(database)

	// Define the storage directory for hashlists
	hashlistDataDir := filepath.Join(cfg.DataDir, "hashlists")

	// Ensure storage directory exists
	if err := os.MkdirAll(hashlistDataDir, 0755); err != nil {
		debug.Error("Failed to create hashlist storage directory %s: %v", hashlistDataDir, err)
		// Depending on requirements, might want to panic or handle differently
	}

	// Create processor
	proc := processor.NewHashlistDBProcessor(hashlistRepo, hashTypeRepo, hashRepo, cfg)

	// Create handler
	h := &hashlistHandler{
		db:                 database,
		hashlistRepo:       hashlistRepo,
		hashTypeRepo:       hashTypeRepo,
		clientRepo:         clientRepo,
		clientSettingsRepo: clientSettingsRepo,
		systemSettingsRepo: systemSettingsRepo,
		hashRepo:           hashRepo,
		fileRepo:           fileRepo,
		dataDir:            hashlistDataDir,
		cfg:                cfg,
		agentService:       agentService,
		processor:          proc,
		jobsHandler:        jobsHandler,
	}

	// === User Routes (Authenticated via JWT) ===
	// Use the provided router 'r' directly. It should already have the /api prefix
	// and the RequireAuth middleware applied from where it was called.
	// userRouter := r.PathPrefix("/api").Subrouter() // REMOVE: Don't create a new /api subrouter
	// userRouter.Use(middleware.RequireAuth(database)) // REMOVE: Middleware already applied

	// 2.1. Hashlist Management API
	hashlistRouter := r.PathPrefix("/hashlists").Subrouter() // Use 'r' directly
	hashlistRouter.HandleFunc("", h.handleUploadHashlist).Methods(http.MethodPost, http.MethodOptions)
	hashlistRouter.HandleFunc("", h.handleListHashlists).Methods(http.MethodGet, http.MethodOptions)
	hashlistRouter.HandleFunc("/{id}", h.handleGetHashlist).Methods(http.MethodGet, http.MethodOptions)
	hashlistRouter.HandleFunc("/{id}", h.handleDeleteHashlist).Methods(http.MethodDelete, http.MethodOptions)
	hashlistRouter.HandleFunc("/{id}/download", h.handleDownloadHashlist).Methods(http.MethodGet, http.MethodOptions)
	hashlistRouter.HandleFunc("/{id}/hashes", h.handleGetHashlistHashes).Methods(http.MethodGet, http.MethodOptions)
	hashlistRouter.HandleFunc("/{id}/available-jobs", h.handleGetAvailableJobs).Methods(http.MethodGet, http.MethodOptions)
	hashlistRouter.HandleFunc("/{id}/create-job", h.handleCreateJob).Methods(http.MethodPost, http.MethodOptions)
	hashlistRouter.HandleFunc("/{id}/client", h.handleUpdateHashlistClient).Methods(http.MethodPatch, http.MethodOptions)

	// 2.2. Hash Types API
	hashTypeRouter := r.PathPrefix("/hashtypes").Subrouter() // Use 'r' directly
	hashTypeRouter.HandleFunc("", h.handleListHashTypes).Methods(http.MethodGet, http.MethodOptions)
	hashTypeRouter.HandleFunc("", h.handleCreateHashType).Methods(http.MethodPost, http.MethodOptions)
	hashTypeRouter.HandleFunc("/{id}", h.handleUpdateHashType).Methods(http.MethodPut, http.MethodOptions)
	hashTypeRouter.HandleFunc("/{id}", h.handleDeleteHashType).Methods(http.MethodDelete, http.MethodOptions)

	// 2.3. Clients API - Using admin handler for all authenticated users
	// Create the admin client handler with full functionality (including cracked counts)
	clientRepoForHandler := repository.NewClientRepository(database)
	clientSettingsRepoForHandler := repository.NewClientSettingsRepository(database)
	analyticsRepoForHandler := repository.NewAnalyticsRepository(database)
	retentionService := retentionsvc.NewRetentionService(database, hashlistRepo, hashRepo, clientRepoForHandler, clientSettingsRepoForHandler, analyticsRepoForHandler)
	clientService := clientsvc.NewClientService(clientRepoForHandler, hashlistRepo, clientSettingsRepoForHandler, retentionService)
	clientHandler := adminclient.NewClientHandler(clientRepoForHandler, clientService)

	// Register client routes for all authenticated users
	clientRouter := r.PathPrefix("/clients").Subrouter() // Use 'r' directly
	clientRouter.HandleFunc("", clientHandler.ListClients).Methods(http.MethodGet)
	clientRouter.HandleFunc("/search", h.handleSearchClients).Methods(http.MethodGet) // Keep the search handler from hashlist
	clientRouter.HandleFunc("", clientHandler.CreateClient).Methods(http.MethodPost)
	clientRouter.HandleFunc("/{id:[0-9a-fA-F-]+}", clientHandler.GetClient).Methods(http.MethodGet)
	clientRouter.HandleFunc("/{id:[0-9a-fA-F-]+}", clientHandler.UpdateClient).Methods(http.MethodPut)
	clientRouter.HandleFunc("/{id:[0-9a-fA-F-]+}", clientHandler.DeleteClient).Methods(http.MethodDelete)

	// 2.4. Hash Search API
	hashSearchRouter := r.PathPrefix("/hashes").Subrouter() // Use 'r' directly
	hashSearchRouter.HandleFunc("/search", h.handleSearchHashes).Methods(http.MethodPost)

	// 2.5. User-specific routes
	userRouter := r.PathPrefix("/user").Subrouter() // Use 'r' directly
	userRouter.HandleFunc("/hashlists", h.handleListUserHashlists).Methods(http.MethodGet, http.MethodOptions)

	// Agent routes are handled in filesync.go with proper API key authentication

	debug.Info("Registered hashlist, hash type, client, and hash search routes under JWT router")
}

// --- Handler Implementations ---

// 2.1. Hashlist Management Handlers

func (h *hashlistHandler) handleUploadHashlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Limit request body size (e.g., 1GB for the whole request)
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		if err == http.ErrNotMultipart {
			jsonError(w, "Request content type is not multipart/form-data", http.StatusBadRequest)
		} else if err.Error() == "http: request body too large" {
			jsonError(w, "Request body too large", http.StatusRequestEntityTooLarge)
		} else {
			debug.Error("Error parsing multipart form: %v", err)
			jsonError(w, "Error processing upload form", http.StatusBadRequest)
		}
		return
	}

	// Parse form values
	name := r.FormValue("name")
	hashTypeIDStr := r.FormValue("hash_type_id")
	clientName := r.FormValue("client_name") // Expect client_name
	excludeStr := r.FormValue("exclude_from_potfile")
	debug.Info("Received hashlist upload: name='%s', hashTypeID='%s', clientName='%s', excludeFromPotfile='%s'", name, hashTypeIDStr, clientName, excludeStr)

	// --- Parse and validate hash type ID ---
	hashTypeID, err := strconv.Atoi(hashTypeIDStr)
	if err != nil {
		jsonError(w, "Invalid hash_type_id format", http.StatusBadRequest)
		return
	}
	// Re-verify hash type exists and is enabled (important!)
	hashType, err := h.hashTypeRepo.GetByID(ctx, hashTypeID)
	if err != nil || hashType == nil || !hashType.IsEnabled {
		debug.Error("Invalid or disabled hash type ID %d provided during upload: %v", hashTypeID, err)
		jsonError(w, fmt.Sprintf("Invalid or disabled hash type ID: %d", hashTypeID), http.StatusBadRequest)
		return
	}

	// --- Check if client is required ---
	trimmedClientName := strings.TrimSpace(clientName)
	requireClientSetting, err := h.systemSettingsRepo.GetSetting(ctx, "require_client_for_hashlist")
	if err == nil && requireClientSetting != nil && requireClientSetting.Value != nil {
		requireClient := *requireClientSetting.Value == "true"
		if requireClient && trimmedClientName == "" {
			debug.Warning("Client is required but not provided for hashlist upload")
			jsonError(w, "Client is required when uploading hashlists", http.StatusBadRequest)
			return
		}
	}

	// --- Lookup or Create Client by Name ---
	var clientID uuid.UUID = uuid.Nil // Default to Nil (no client)

	if trimmedClientName != "" {
		debug.Info("Processing client name from upload: '%s'", trimmedClientName)
		client, err := h.clientRepo.GetByName(ctx, trimmedClientName)

		// Corrected Error Handling:
		if err != nil {
			debug.Error("Error during client lookup for '%s': %v", trimmedClientName, err)
			jsonError(w, "Failed to lookup client", http.StatusInternalServerError)
			return
		}

		// If err is nil, check if client was found or not
		if client == nil {
			// *** Client Not Found - Proceed with Creation ***
			debug.Info("Client '%s' not found, creating new client.", trimmedClientName)

			if len(trimmedClientName) > 255 {
				jsonError(w, "Client name exceeds 255 character limit", http.StatusBadRequest)
				return
			}

			// Fetch default retention setting
			debug.Info("Fetching default retention setting...")
			defaultRetentionSetting, settingErr := h.clientSettingsRepo.GetSetting(ctx, "default_data_retention_months") // Use settingErr
			var defaultRetentionMonths *int                                                                              // Use pointer for nullable int

			if settingErr != nil {
				debug.Error("Failed to get default retention setting during client creation: %v. Client will have NULL retention.", settingErr)
			} else if defaultRetentionSetting.Value != nil {
				debug.Info("Default retention setting value found: '%s'", *defaultRetentionSetting.Value)
				val, convErr := strconv.Atoi(*defaultRetentionSetting.Value)
				if convErr != nil {
					debug.Error("Failed to convert default retention setting '%s' to int: %v. Client will have NULL retention.", *defaultRetentionSetting.Value, convErr)
				} else {
					defaultRetentionMonths = &val
					debug.Info("Successfully parsed and applying default retention of %d months to new client '%s'", val, trimmedClientName)
				}
			} else {
				debug.Warning("Default retention setting found but its value is nil. Client will have NULL retention.")
			}

			// Construct the new client model
			newClient := &models.Client{
				ID:                  uuid.New(),
				Name:                trimmedClientName,
				DataRetentionMonths: defaultRetentionMonths, // Assign fetched default (or nil)
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			}

			// Log before calling Create
			if defaultRetentionMonths == nil {
				debug.Warning("[Pre-Create] Attempting to create client '%s' with NULL DataRetentionMonths.", newClient.Name)
			} else {
				debug.Info("[Pre-Create] Attempting to create client '%s' with DataRetentionMonths = %d.", newClient.Name, *defaultRetentionMonths)
			}

			// Create the client
			createErr := h.clientRepo.Create(ctx, newClient) // Use createErr
			if createErr != nil {
				if repoErr, ok := createErr.(*pq.Error); ok && repoErr.Code == "23505" { // Check createErr
					debug.Warning("Race condition during client '%s' creation, re-fetching...", trimmedClientName)
					// Re-fetch necessary if race condition possible
					client, err = h.clientRepo.GetByName(ctx, trimmedClientName) // Re-assign client and err
					if err != nil || client == nil {
						debug.Error("Failed to re-fetch client '%s' after creation conflict: %v", trimmedClientName, err)
						jsonError(w, "Failed to create or find client after conflict", http.StatusInternalServerError)
						return
					}
					clientID = client.ID
					debug.Info("Successfully re-fetched client '%s' after conflict, ID: %s", trimmedClientName, clientID)
				} else {
					debug.Error("Error creating new client '%s': %v", trimmedClientName, createErr) // Use createErr
					jsonError(w, "Failed to create client", http.StatusInternalServerError)
					return
				}
			} else {
				// Creation successful
				clientID = newClient.ID
				debug.Info("Successfully created new client '%s' with ID %s", trimmedClientName, clientID)
			}
		} else {
			// *** Client Found - Use Existing ID ***
			clientID = client.ID
			debug.Info("Found existing client '%s' with ID %s", trimmedClientName, clientID)
		}
	}

	// --- Get the file ---
	file, header, err := r.FormFile("hashlist_file")
	if err != nil {
		if err == http.ErrMissingFile {
			jsonError(w, "hashlist_file is required", http.StatusBadRequest)
		} else {
			debug.Error("Error getting file from request: %v", err)
			jsonError(w, "Error processing uploaded file", http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	// --- Parse exclude_from_potfile boolean ---
	excludeFromPotfile := false
	if excludeStr != "" {
		excludeFromPotfile, err = strconv.ParseBool(excludeStr)
		if err != nil {
			debug.Error("Failed to parse exclude_from_potfile '%s': %v, defaulting to false", excludeStr, err)
			excludeFromPotfile = false
		}
	}
	debug.Info("Parsed exclude_from_potfile as: %v", excludeFromPotfile)

	// --- Create database entry ---
	now := time.Now()
	hashlist := &models.HashList{
		Name:               name,
		UserID:             userID,
		ClientID:           clientID, // Will be zero UUID if not provided
		HashTypeID:         hashTypeID,
		Status:             models.HashListStatusUploading,
		ExcludeFromPotfile: excludeFromPotfile,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	err = h.hashlistRepo.Create(ctx, hashlist)
	if err != nil {
		debug.Error("Error creating hashlist DB entry: %v", err)
		jsonError(w, "Failed to create hashlist record", http.StatusInternalServerError)
		return
	}

	// Ensure hashlist.ID is populated before using it
	if hashlist.ID == 0 {
		debug.Error("Hashlist ID is 0 after creation for name %s", name)
		jsonError(w, "Failed to retrieve generated hashlist ID", http.StatusInternalServerError)
		return
	}

	// --- Save the file ---
	// Create a unique-ish filename
	filename := fmt.Sprintf("%d_%s%s", // Use %d for int64 hashlist.ID
		hashlist.ID,
		SanitizeFilenameSimple(strings.ReplaceAll(strings.ToLower(name), " ", "_")),
		filepath.Ext(header.Filename),
	)
	hashlistPath := filepath.Join(h.dataDir, filename)

	// Create the destination file
	dst, err := os.Create(hashlistPath)
	if err != nil {
		debug.Error("Failed to create hashlist file %s: %v", hashlistPath, err)
		h.updateHashlistStatus(ctx, hashlist.ID, models.HashListStatusError, "Failed to save uploaded file")
		jsonError(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file data
	_, err = io.Copy(dst, file)
	if err != nil {
		debug.Error("Failed to copy uploaded file to %s: %v", hashlistPath, err)
		h.updateHashlistStatus(ctx, hashlist.ID, models.HashListStatusError, "Failed to copy uploaded file data")
		jsonError(w, "Failed to copy uploaded file data", http.StatusInternalServerError)
		return
	}

	// --- Update database entry with file path and trigger processing ---
	hashlist.FilePath = hashlistPath                  // Set the path for the update
	hashlist.Status = models.HashListStatusProcessing // Ready for background processing
	hashlist.UpdatedAt = time.Now()

	err = h.hashlistRepo.UpdateFilePathAndStatus(ctx, hashlist.ID, hashlist.FilePath, hashlist.Status)
	if err != nil {
		debug.Error("Failed to update hashlist path/status for %d: %v", hashlist.ID, err)
		// Attempt cleanup of the saved file
		os.Remove(hashlistPath)
		// Maybe update status again to error?
		jsonError(w, "Failed to finalize hashlist upload", http.StatusInternalServerError)
		return
	}

	// --- Start background processing ---
	go h.processor.SubmitHashlistForProcessing(hashlist.ID)
	debug.Info("Hashlist %d uploaded successfully, path: %s. Background processing triggered.", hashlist.ID, hashlistPath)

	// Return the initial hashlist record (without file path for security)
	hashlist.FilePath = ""                         // Don't expose file path in response
	jsonResponse(w, http.StatusAccepted, hashlist) // Use 202 Accepted as processing is happening
}

func (h *hashlistHandler) handleListHashlists(w http.ResponseWriter, r *http.Request) {
	debug.Error("***** ATTENTION: handleListHashlists FUNCTION ENTERED *****") // Added prominent log
	ctx := r.Context()

	// Parse query parameters
	queryVals := r.URL.Query()

	// Pagination
	limitStr := queryVals.Get("limit")
	offsetStr := queryVals.Get("offset")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50 // Default limit
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0 // Default offset
	}

	// Filtering
	params := repository.ListHashlistsParams{
		Limit:  limit,
		Offset: offset,
	}

	if status := queryVals.Get("status"); status != "" {
		params.Status = &status
	}
	if name := queryVals.Get("name"); name != "" {
		params.NameLike = &name
	}
	if clientIDStr := queryVals.Get("client_id"); clientIDStr != "" {
		clientID, err := uuid.Parse(clientIDStr)
		if err == nil {
			params.ClientID = &clientID
		} else {
			// Optionally return bad request if client_id format is invalid
			// jsonError(w, "Invalid client_id format", http.StatusBadRequest)
			// return
			debug.Warning("Invalid client_id format in query param: %s", clientIDStr)
		}
	}

	// Fetch data from repository
	hashlists, totalCount, err := h.hashlistRepo.List(ctx, params)
	if err != nil {
		// Error already logged in repository
		jsonError(w, "Failed to retrieve hashlists", http.StatusInternalServerError)
		return
	}

	// Log the data before sending the response
	debug.Info("[handleListHashlists] Retrieved %d hashlists (TotalCount: %d)", len(hashlists), totalCount)
	if len(hashlists) > 0 {
		debug.Debug("[handleListHashlists] First hashlist data to be sent: %+v", hashlists[0])
	} else {
		debug.Info("[handleListHashlists] No hashlists retrieved.")
	}

	// Prepare response with pagination metadata
	response := struct {
		Data       []models.HashList `json:"data"` // Ensure this matches frontend expectation
		TotalCount int               `json:"total_count"`
		Limit      int               `json:"limit"`
		Offset     int               `json:"offset"`
	}{
		Data:       hashlists,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}

	// Log the data just before sending the response for debugging
	debug.Debug("[handleListHashlists] Final response structure to be sent: %+v", response)
	if len(response.Data) > 0 {
		debug.Debug("[handleListHashlists] First hashlist in response data: ClientID=%s, ClientName=%v", response.Data[0].ClientID, response.Data[0].ClientName)
	}

	jsonResponse(w, http.StatusOK, response)
}

// handleListUserHashlists returns hashlists created by the authenticated user
func (h *hashlistHandler) handleListUserHashlists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	queryVals := r.URL.Query()

	// Pagination
	limitStr := queryVals.Get("limit")
	offsetStr := queryVals.Get("offset")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10 // Default limit for dashboard
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0 // Default offset
	}

	// Filtering
	params := repository.ListHashlistsParams{
		Limit:  limit,
		Offset: offset,
		UserID: &userID, // Filter by authenticated user
	}

	// Support additional filters
	if status := queryVals.Get("status"); status != "" {
		params.Status = &status
	}
	if name := queryVals.Get("name"); name != "" {
		params.NameLike = &name
	}

	// Fetch data from repository
	hashlists, totalCount, err := h.hashlistRepo.List(ctx, params)
	if err != nil {
		debug.Error("Error fetching user hashlists for user %s: %v", userID, err)
		jsonError(w, "Failed to retrieve hashlists", http.StatusInternalServerError)
		return
	}

	// Prepare response with pagination metadata
	response := struct {
		Data       []models.HashList `json:"data"`
		TotalCount int               `json:"total_count"`
		Limit      int               `json:"limit"`
		Offset     int               `json:"offset"`
	}{
		Data:       hashlists,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}

	jsonResponse(w, http.StatusOK, response)
}

func (h *hashlistHandler) handleGetHashlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Note: Ownership check removed - all authenticated users can access all hashlists
	// This will change when teams are implemented

	id, err := getInt64FromPath(r, "id")
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	hashlist, err := h.hashlistRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "Hashlist not found", http.StatusNotFound)
		} else {
			debug.Error("Error getting hashlist %d: %v", id, err)
			jsonError(w, "Failed to retrieve hashlist", http.StatusInternalServerError)
		}
		return
	}

	// Fetch hash type to enrich response
	hashType, err := h.hashTypeRepo.GetByID(ctx, hashlist.HashTypeID)
	if err != nil {
		debug.Error("Error getting hash type %d for hashlist %d: %v", hashlist.HashTypeID, id, err)
		// Don't fail the request, just won't have enriched data
	}

	// Create enriched response
	response := map[string]interface{}{
		"id":                   hashlist.ID,
		"name":                 hashlist.Name,
		"user_id":              hashlist.UserID,
		"client_id":            hashlist.ClientID,
		"client_name":          hashlist.ClientName,
		"hash_type_id":         hashlist.HashTypeID,
		"total_hashes":         hashlist.TotalHashes,
		"cracked_hashes":       hashlist.CrackedHashes,
		"status":               hashlist.Status,
		"error_message":        hashlist.ErrorMessage,
		"exclude_from_potfile": hashlist.ExcludeFromPotfile,
		"createdAt":            hashlist.CreatedAt,
		"updatedAt":            hashlist.UpdatedAt,
	}

	// Add enriched hash type field if available
	if hashType != nil {
		response["hashTypeName"] = fmt.Sprintf("%s (%d)", hashType.Name, hashType.ID)
		response["hashTypeID"] = hashType.ID
	}

	jsonResponse(w, http.StatusOK, response)
}

func (h *hashlistHandler) handleDeleteHashlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Note: Ownership check removed - all authenticated users can delete all hashlists
	// This will change when teams are implemented
	// Check if user is admin
	role, _ := getUserRoleFromContext(ctx) // Keeping for potential future use
	isAdmin := role == "admin"
	_ = isAdmin // Suppress unused variable warning

	id, err := getInt64FromPath(r, "id")
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	hashlist, err := h.hashlistRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "Hashlist not found", http.StatusNotFound) // Or just return 204? Debateable.
		} else {
			debug.Error("Error checking hashlist %d before delete: %v", id, err)
			jsonError(w, "Failed to retrieve hashlist before deletion", http.StatusInternalServerError)
		}
		return
	}

	// Delete from database (associations are handled by ON DELETE CASCADE)
	err = h.hashlistRepo.Delete(ctx, id)
	if err != nil {
		debug.Error("Error deleting hashlist %d from DB: %v", id, err)
		jsonError(w, "Failed to delete hashlist record", http.StatusInternalServerError)
		return
	}

	// Delete the physical file
	if hashlist.FilePath != "" {
		err := os.Remove(hashlist.FilePath)
		if err != nil && !os.IsNotExist(err) {
			// Log error but proceed, DB record is gone.
			debug.Warning("Failed to delete hashlist file %s after DB delete for ID %d: %v", hashlist.FilePath, id, err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *hashlistHandler) handleUpdateHashlistClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Note: All authenticated users can update hashlist clients
	// This will change when teams are implemented

	id, err := getInt64FromPath(r, "id")
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse request body
	var request struct {
		ClientID *string `json:"client_id"` // UUID as string, nullable
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert string UUID to uuid.UUID
	var clientID uuid.UUID
	if request.ClientID != nil && *request.ClientID != "" {
		clientID, err = uuid.Parse(*request.ClientID)
		if err != nil {
			jsonError(w, "Invalid client_id format", http.StatusBadRequest)
			return
		}

		// Verify client exists
		_, err = h.clientRepo.GetByID(ctx, clientID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				jsonError(w, "Client not found", http.StatusNotFound)
			} else {
				debug.Error("Error verifying client %s: %v", clientID, err)
				jsonError(w, "Failed to verify client", http.StatusInternalServerError)
			}
			return
		}
	} else {
		clientID = uuid.Nil // Set to NULL
	}

	// Verify hashlist exists
	hashlist, err := h.hashlistRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "Hashlist not found", http.StatusNotFound)
		} else {
			debug.Error("Error getting hashlist %d: %v", id, err)
			jsonError(w, "Failed to retrieve hashlist", http.StatusInternalServerError)
		}
		return
	}

	// Update the client
	err = h.hashlistRepo.UpdateClientID(ctx, id, clientID)
	if err != nil {
		debug.Error("Error updating client for hashlist %d: %v", id, err)
		jsonError(w, "Failed to update hashlist client", http.StatusInternalServerError)
		return
	}

	// Fetch updated hashlist to return
	hashlist, err = h.hashlistRepo.GetByID(ctx, id)
	if err != nil {
		debug.Error("Error getting updated hashlist %d: %v", id, err)
		jsonError(w, "Failed to retrieve updated hashlist", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, hashlist)
}

func (h *hashlistHandler) handleDownloadHashlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := getInt64FromPath(r, "id")
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if request is from an agent or user
	isAgentRequest := strings.HasPrefix(r.URL.Path, "/agent/")

	if !isAgentRequest {
		_, err = getUserIDFromContext(ctx)
		if err != nil {
			jsonError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Note: Ownership check removed - all authenticated users can download all hashlists
		// This will change when teams are implemented
	} // Agent requests are authenticated by AgentAPIKeyMiddleware

	hashlist, err := h.hashlistRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "Hashlist not found", http.StatusNotFound)
		} else {
			debug.Error("Error getting hashlist %d for download: %v", id, err)
			jsonError(w, "Failed to retrieve hashlist", http.StatusInternalServerError)
		}
		return
	}

	// Check file path exists
	if hashlist.FilePath == "" {
		debug.Warning("Attempt to download hashlist %d with no file path", id)
		jsonError(w, "Hashlist file path not found", http.StatusNotFound)
		return
	}

	// Security check: Ensure the path is within the data directory
	cleanPath := filepath.Clean(hashlist.FilePath)
	if !strings.HasPrefix(cleanPath, h.dataDir) {
		debug.Error("Attempt to download file outside data directory: %s (Hashlist ID: %d)", hashlist.FilePath, id)
		jsonError(w, "Invalid file path", http.StatusInternalServerError) // Internal error, path shouldn't be like this
		return
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			debug.Warning("Hashlist file %s not found on disk for hashlist %d", cleanPath, id)
			jsonError(w, "Hashlist file not found on disk", http.StatusNotFound)
		} else {
			debug.Error("Error opening hashlist file %s (Hashlist ID: %d): %v", cleanPath, id, err)
			jsonError(w, "Error opening hashlist file", http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	// Set headers for file download
	filename := filepath.Base(cleanPath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename)) // Quoted filename
	w.Header().Set("Content-Type", "application/octet-stream")                                  // General binary type

	// Stream the file
	_, err = io.Copy(w, file)
	if err != nil {
		// Client might have disconnected, log but don't change status code if already sent
		debug.Error("Error streaming hashlist file %s (Hashlist ID: %d): %v", filename, id, err)
	}
}

// handleGetHashlistHashes retrieves hashes for a specific hashlist with pagination
func (h *hashlistHandler) handleGetHashlistHashes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		debug.Error("Error extracting user from context: %v", err)
		jsonError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Parse hashlist ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "Invalid hashlist ID", http.StatusBadRequest)
		return
	}

	// Check if the hashlist exists
	_, err = h.hashlistRepo.GetByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonError(w, "Hashlist not found", http.StatusNotFound)
			return
		}
		debug.Error("Error retrieving hashlist (ID: %d) for user %s: %v", id, userID, err)
		jsonError(w, "Failed to retrieve hashlist", http.StatusInternalServerError)
		return
	}

	// Note: Ownership check removed - all authenticated users can access all hashlists
	// This will change when teams are implemented

	// Parse pagination parameters
	limit := 500 // Default limit increased to 500 for better UX
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		// Support -1 for "all results" (similar to pot table)
		if limitStr == "-1" {
			limit = 999999 // Effectively unlimited
		} else if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 2000 {
			limit = parsedLimit // Max limit increased from 100 to 2000
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get hashes for this hashlist
	hashes, total, err := h.hashRepo.GetHashesByHashlistID(ctx, id, limit, offset)
	if err != nil {
		debug.Error("Error retrieving hashes for hashlist %d: %v", id, err)
		jsonError(w, "Failed to retrieve hashes", http.StatusInternalServerError)
		return
	}

	// Return the hashes with pagination info
	response := map[string]interface{}{
		"hashes": hashes,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	jsonResponse(w, http.StatusOK, response)
}

// 2.2. Hash Types Handlers

func (h *hashlistHandler) handleListHashTypes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// No user context needed, this is public info (or maybe require auth?)
	// For now, assume auth required by middleware

	// Optionally allow filtering by `enabled_only` query param
	enabledOnly := true // Default to enabled only
	if r.URL.Query().Get("enabled_only") == "false" {
		enabledOnly = false
	}

	hashTypes, err := h.hashTypeRepo.List(ctx, enabledOnly)
	if err != nil {
		debug.Error("Error listing hash types: %v", err)
		jsonError(w, "Failed to retrieve hash types", http.StatusInternalServerError)
		return
	}

	// Ensure we return an empty array [] instead of null if no hash types are found
	if hashTypes == nil {
		hashTypes = []models.HashType{} // Or make([]models.HashType, 0)
	}

	jsonResponse(w, http.StatusOK, hashTypes)
}

// requireAdmin checks if the user role in the context is 'admin'.
func requireAdmin(ctx context.Context) (bool, error) {
	role, err := getUserRoleFromContext(ctx)
	if err != nil {
		// Role not found or error parsing context
		debug.Warning("Could not determine user role from context: %v", err)
		return false, err
	}
	isAdmin := role == "admin"
	if !isAdmin {
		debug.Warning("Admin privileges required, but user role is '%s'", role)
	}
	return isAdmin, nil
}

func (h *hashlistHandler) handleCreateHashType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	isAdmin, err := requireAdmin(ctx)
	if err != nil {
		// Error already logged in requireAdmin
		jsonError(w, "Error checking admin privileges", http.StatusInternalServerError)
		return
	}
	if !isAdmin {
		jsonError(w, "Forbidden: Admin privileges required", http.StatusForbidden)
		return
	}

	var hashType models.HashType
	if err := json.NewDecoder(r.Body).Decode(&hashType); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Basic validation (add more as needed)
	if hashType.ID <= 0 || hashType.Name == "" {
		jsonError(w, "Hash Type ID must be positive and Name is required", http.StatusBadRequest)
		return
	}

	err = h.hashTypeRepo.Create(ctx, &hashType)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") { // Check based on repository error
			jsonError(w, err.Error(), http.StatusConflict)
		} else {
			debug.Error("Error creating hash type %d: %v", hashType.ID, err)
			jsonError(w, "Failed to create hash type", http.StatusInternalServerError)
		}
		return
	}

	jsonResponse(w, http.StatusCreated, hashType)
}

func (h *hashlistHandler) handleUpdateHashType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	isAdmin, err := requireAdmin(ctx)
	if err != nil {
		jsonError(w, "Error checking admin privileges", http.StatusInternalServerError)
		return
	}
	if !isAdmin {
		jsonError(w, "Forbidden: Admin privileges required", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, "Invalid hash type ID format", http.StatusBadRequest)
		return
	}

	var hashType models.HashType
	if err := json.NewDecoder(r.Body).Decode(&hashType); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure the ID in the path matches the ID in the body (if provided and not 0)
	if hashType.ID != 0 && hashType.ID != id {
		jsonError(w, "Hash type ID in path does not match ID in body", http.StatusBadRequest)
		return
	}
	hashType.ID = id // Use ID from path

	if hashType.Name == "" { // Basic validation
		jsonError(w, "Hash Type Name is required", http.StatusBadRequest)
		return
	}

	err = h.hashTypeRepo.Update(ctx, &hashType)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			jsonError(w, "Hash type not found", http.StatusNotFound)
		} else {
			debug.Error("Error updating hash type %d: %v", id, err)
			jsonError(w, "Failed to update hash type", http.StatusInternalServerError)
		}
		return
	}

	jsonResponse(w, http.StatusOK, hashType)
}

func (h *hashlistHandler) handleDeleteHashType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	isAdmin, err := requireAdmin(ctx)
	if err != nil {
		jsonError(w, "Error checking admin privileges", http.StatusInternalServerError)
		return
	}
	if !isAdmin {
		jsonError(w, "Forbidden: Admin privileges required", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, "Invalid hash type ID format", http.StatusBadRequest)
		return
	}

	err = h.hashTypeRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			jsonError(w, "Hash type not found", http.StatusNotFound)
		} else if strings.Contains(err.Error(), "still referenced") { // Check repo error string
			jsonError(w, err.Error(), http.StatusConflict)
		} else {
			debug.Error("Error deleting hash type %d: %v", id, err)
			jsonError(w, "Failed to delete hash type", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// 2.3. Clients Handlers

func (h *hashlistHandler) handleListClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Auth middleware ensures user is logged in

	// TODO: Add pagination?
	clients, err := h.clientRepo.List(ctx)
	if err != nil {
		debug.Error("Error listing clients: %v", err)
		jsonError(w, "Failed to retrieve clients", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, clients)
}

func (h *hashlistHandler) handleSearchClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query().Get("q")
	if len(query) < 1 { // Minimum query length?
		jsonError(w, "Search query 'q' is required and must be at least 1 character", http.StatusBadRequest)
		return
	}

	clients, err := h.clientRepo.Search(ctx, query)
	if err != nil {
		debug.Error("Error searching clients with query '%s': %v", query, err)
		jsonError(w, "Failed to search clients", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, clients)
}

func (h *hashlistHandler) handleCreateClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var client models.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if client.Name == "" {
		jsonError(w, "Client name is required", http.StatusBadRequest)
		return
	}

	// Check if client name already exists
	existing, _ := h.clientRepo.GetByName(ctx, client.Name)
	if existing != nil {
		jsonError(w, fmt.Sprintf("Client with name '%s' already exists", client.Name), http.StatusConflict)
		return
	}

	client.ID = uuid.New()
	now := time.Now()
	client.CreatedAt = now
	client.UpdatedAt = now

	err := h.clientRepo.Create(ctx, &client)
	if err != nil {
		debug.Error("Error creating client '%s': %v", client.Name, err)
		jsonError(w, "Failed to create client", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusCreated, client)
}

func (h *hashlistHandler) handleGetClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := getUUIDFromPath(r, "id")
	if err != nil {
		jsonError(w, "Invalid client ID format", http.StatusBadRequest)
		return
	}

	client, err := h.clientRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "Client not found", http.StatusNotFound)
		} else {
			debug.Error("Error getting client %s: %v", id, err)
			jsonError(w, "Failed to retrieve client", http.StatusInternalServerError)
		}
		return
	}

	jsonResponse(w, http.StatusOK, client)
}

func (h *hashlistHandler) handleUpdateClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := getUUIDFromPath(r, "id")
	if err != nil {
		jsonError(w, "Invalid client ID format", http.StatusBadRequest)
		return
	}

	var updatedClient models.Client
	if err := json.NewDecoder(r.Body).Decode(&updatedClient); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if updatedClient.Name == "" {
		jsonError(w, "Client name is required", http.StatusBadRequest)
		return
	}

	// Check if client exists
	existingClient, err := h.clientRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "Client not found", http.StatusNotFound)
		} else {
			debug.Error("Error checking client %s before update: %v", id, err)
			jsonError(w, "Failed to retrieve client before update", http.StatusInternalServerError)
		}
		return
	}

	// Check if the new name conflicts with another client
	if updatedClient.Name != existingClient.Name {
		conflictClient, _ := h.clientRepo.GetByName(ctx, updatedClient.Name)
		if conflictClient != nil {
			jsonError(w, fmt.Sprintf("Another client with name '%s' already exists", updatedClient.Name), http.StatusConflict)
			return
		}
	}

	// Update fields
	existingClient.Name = updatedClient.Name
	existingClient.Description = updatedClient.Description
	existingClient.ContactInfo = updatedClient.ContactInfo
	existingClient.UpdatedAt = time.Now()

	err = h.clientRepo.Update(ctx, existingClient)
	if err != nil {
		debug.Error("Error updating client %s: %v", id, err)
		jsonError(w, "Failed to update client", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, existingClient)
}

func (h *hashlistHandler) handleDeleteClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := getUUIDFromPath(r, "id")
	if err != nil {
		jsonError(w, "Invalid client ID format", http.StatusBadRequest)
		return
	}

	// Check if client exists before deleting
	_, err = h.clientRepo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "Client not found", http.StatusNotFound)
		} else {
			debug.Error("Error checking client %s before delete: %v", id, err)
			jsonError(w, "Failed to retrieve client before deletion", http.StatusInternalServerError)
		}
		return
	}

	// Delete the client (hashlists referencing it will have client_id set to NULL)
	err = h.clientRepo.Delete(ctx, id)
	if err != nil {
		debug.Error("Error deleting client %s: %v", id, err)
		jsonError(w, "Failed to delete client", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// 2.4. Hash Search Handlers

func (h *hashlistHandler) handleSearchHashes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request struct {
		Hashes []string `json:"hashes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(request.Hashes) == 0 {
		jsonError(w, "'hashes' array cannot be empty", http.StatusBadRequest)
		return
	}
	if len(request.Hashes) > 1000 { // Limit bulk search size
		jsonError(w, "Too many hashes requested (max 1000)", http.StatusBadRequest)
		return
	}

	// Perform search
	results, err := h.hashRepo.SearchHashes(ctx, request.Hashes, userID)
	if err != nil {
		debug.Error("Error searching hashes for user %s: %v", userID, err)
		jsonError(w, "Failed to search hashes", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, results)
}

// --- Helper Functions ---

// AgentAPIKeyMiddleware validates agent API keys and IDs
func AgentAPIKeyMiddleware(agentService *services.AgentService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			agentIDStr := r.Header.Get("X-Agent-ID") // Agent ID is expected as an integer string

			if apiKey == "" || agentIDStr == "" {
				debug.Warning("Agent request missing X-API-Key or X-Agent-ID header")
				jsonError(w, "API Key and Agent ID required", http.StatusUnauthorized)
				return
			}

			// Convert Agent ID header to integer
			agentIDHeader, err := strconv.Atoi(agentIDStr)
			if err != nil {
				debug.Warning("Agent request with invalid X-Agent-ID format: %s", agentIDStr)
				jsonError(w, "Invalid Agent ID format", http.StatusUnauthorized)
				return
			}

			// Validate API Key using AgentService and get the agent associated with the key
			agent, err := agentService.GetByAPIKey(r.Context(), apiKey)
			if err != nil {
				debug.Error("Error validating agent API key: %v", err)
				// Check if the error indicates the key wasn't found vs. a DB error
				if strings.Contains(err.Error(), "not found") { // Adjust based on actual error message
					jsonError(w, "Invalid API Key", http.StatusUnauthorized)
				} else {
					jsonError(w, "Error validating agent credentials", http.StatusInternalServerError)
				}
				return
			}

			if agent == nil {
				debug.Warning("Invalid agent API key provided (key exists but no agent found?)") // Should ideally not happen if GetByAPIKey works correctly
				jsonError(w, "Invalid API Key", http.StatusUnauthorized)
				return
			}

			// Verify that the agent ID from the header matches the agent associated with the API key
			if agent.ID != agentIDHeader {
				debug.Warning("Agent ID mismatch: Header (%d) != API Key Owner (%d)", agentIDHeader, agent.ID)
				jsonError(w, "API Key does not match Agent ID", http.StatusUnauthorized)
				return
			}

			// Add agent info to context
			ctx := context.WithValue(r.Context(), "agent_id", agent.ID) // Store int agent ID
			debug.Debug("Agent authentication successful for agent ID: %d", agent.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// getUserIDFromContext extracts user ID from context set by RequireAuth middleware
func getUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userIDStr, ok := ctx.Value("user_id").(string) // Use string literal key
	if !ok || userIDStr == "" {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID format in context: %w", err)
	}
	return userID, nil
}

// getUserRoleFromContext extracts user role from context
func getUserRoleFromContext(ctx context.Context) (string, error) {
	role, ok := ctx.Value("user_role").(string) // Use the key set in RequireAuth
	if !ok || role == "" {
		return "", fmt.Errorf("role not found in context")
	}
	return role, nil
}

// getUUIDFromPath extracts a UUID from gorilla/mux path variables
func getUUIDFromPath(r *http.Request, key string) (uuid.UUID, error) {
	vars := mux.Vars(r)
	idStr := vars[key]
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID format for '%s' parameter: %s", key, idStr)
	}
	return id, nil
}

// getInt64FromPath extracts an int64 from gorilla/mux path variables
func getInt64FromPath(r *http.Request, key string) (int64, error) {
	vars := mux.Vars(r)
	idStr := vars[key]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid int64 format for '%s' parameter: %s", key, idStr)
	}
	return id, nil
}

// jsonError sends a JSON error response
func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// jsonResponse sends a JSON success response
func jsonResponse(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		json.NewEncoder(w).Encode(payload)
	}
}

// Helper to update hashlist status and error message
func (h *hashlistHandler) updateHashlistStatus(ctx context.Context, id int64, status string, errMsg string) {
	err := h.hashlistRepo.UpdateStatus(ctx, id, status, errMsg)
	if err != nil {
		debug.Error("Failed to update hashlist %d status to %s: %v", id, status, err)
	}
}

// processHashValue handles special processing for certain hash types (like NTLM)
func processHashValue(line string, hashType *models.HashType) (string, bool) {
	if !hashType.NeedsProcessing {
		return line, false
	}

	// NTLM special processing (Mode 1000)
	if hashType.ID == 1000 {
		// NTLM hashes are typically in the format: username:sid:LM hash:NT hash:::
		parts := strings.Split(line, ":")
		if len(parts) >= 4 {
			// Extract the NT hash (4th part)
			return parts[3], true
		}
	}

	// TODO: Implement other processing logic based on hashType.ProcessingLogic field

	// No special processing applied or recognized
	return line, false
}

// SanitizeFilenameSimple provides basic filename sanitization.
// Replace with a more robust library if needed.
func SanitizeFilenameSimple(filename string) string {
	// Replace potentially problematic characters with underscores
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
	)
	return replacer.Replace(filename)
}

// handleGetAvailableJobs delegates to the jobs handler
func (h *hashlistHandler) handleGetAvailableJobs(w http.ResponseWriter, r *http.Request) {
	if h.jobsHandler == nil {
		jsonError(w, "Job functionality not available", http.StatusNotImplemented)
		return
	}
	h.jobsHandler.GetAvailablePresetJobs(w, r)
}

// handleCreateJob delegates to the jobs handler
func (h *hashlistHandler) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	if h.jobsHandler == nil {
		jsonError(w, "Job functionality not available", http.StatusNotImplemented)
		return
	}
	h.jobsHandler.CreateJobFromHashlist(w, r)
}
