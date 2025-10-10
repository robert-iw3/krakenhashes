package hashlist

import (
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
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/processor"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// Handler handles HTTP requests for hashlist operations
// Changed to use direct repository/processor dependencies
type Handler struct {
	hashlistRepo       repository.HashListRepository
	hashRepo           repository.HashRepository
	hashTypeRepo       repository.HashTypeRepository
	clientRepo         repository.ClientRepository
	clientSettingsRepo repository.ClientSettingsRepository
	fileRepo           repository.FileRepository // Assuming file operations are needed
	processor          *processor.HashlistDBProcessor
	agentService       *services.AgentService // Keep if needed for specific actions
	cfg                *config.Config         // Keep config if needed
}

// NewHandler creates a new hashlist handler
// Changed to accept direct dependencies
func NewHandler(
	hashlistRepo repository.HashListRepository,
	hashRepo repository.HashRepository,
	hashTypeRepo repository.HashTypeRepository,
	clientRepo repository.ClientRepository,
	clientSettingsRepo repository.ClientSettingsRepository,
	fileRepo repository.FileRepository,
	processor *processor.HashlistDBProcessor,
	agentService *services.AgentService,
	cfg *config.Config,
) *Handler {
	return &Handler{
		hashlistRepo:       hashlistRepo,
		hashRepo:           hashRepo,
		hashTypeRepo:       hashTypeRepo,
		clientRepo:         clientRepo,
		clientSettingsRepo: clientSettingsRepo,
		fileRepo:           fileRepo,
		processor:          processor,
		agentService:       agentService,
		cfg:                cfg,
	}
}

// HandleListHashlists handles GET requests for listing hashlists
func (h *Handler) HandleListHashlists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, isUser := ctx.Value("user_id").(uuid.UUID)
	// role, _ := ctx.Value("user_role").(string)
	// isAdmin := role == "admin"

	// Extract pagination and filter parameters (using ListHashlistsParams)
	params := repository.ListHashlistsParams{
		Limit:  50, // Default limit
		Offset: 0,  // Default offset
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = offset
		}
	}
	if status := r.URL.Query().Get("status"); status != "" {
		params.Status = &status
	}
	if clientIDStr := r.URL.Query().Get("client_id"); clientIDStr != "" {
		if clientID, err := uuid.Parse(clientIDStr); err == nil {
			params.ClientID = &clientID
		}
	}

	// If it's a regular user, filter by their ID
	if isUser {
		params.UserID = &userID
	} // else (agent or potentially admin without user context), don't filter by user ID

	hashlists, totalCount, err := h.hashlistRepo.List(ctx, params)
	if err != nil {
		debug.Error("Error getting hashlists: %v", err)
		jsonError(w, "Error retrieving hashlists", http.StatusInternalServerError)
		return
	}

	// Prepare response with pagination
	response := map[string]interface{}{
		"hashlists":   hashlists,
		"total_count": totalCount,
		"limit":       params.Limit,
		"offset":      params.Offset,
	}

	jsonResponse(w, http.StatusOK, response)
}

// HandleGetHashlist handles GET requests for a specific hashlist
func (h *Handler) HandleGetHashlist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		debug.Error("Invalid hashlist ID format: %s", idStr)
		jsonError(w, "Invalid hashlist ID format", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	hashlist, err := h.hashlistRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			jsonError(w, "Hashlist not found", http.StatusNotFound)
		} else {
			debug.Error("Error getting hashlist %d: %v", id, err)
			jsonError(w, "Error retrieving hashlist", http.StatusInternalServerError)
		}
		return
	}

	// Check if the user has access to this hashlist
	userID, isUser := r.Context().Value("user_id").(uuid.UUID)
	role, _ := r.Context().Value("user_role").(string)
	isAdmin := role == "admin"

	if isUser && !isAdmin && hashlist.UserID != userID {
		jsonError(w, "Access denied", http.StatusForbidden)
		return
	}

	jsonResponse(w, http.StatusOK, hashlist)
}

// HandleUploadHashlist handles POST requests for uploading a new hashlist
func (h *Handler) HandleUploadHashlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form
	// Use MaxMemory from config if available, otherwise default
	maxMemory := int64(32 << 20) // Default 32MB
	if h.cfg != nil && h.cfg.MaxUploadSize > 0 {
		maxMemory = h.cfg.MaxUploadSize // Assuming MaxUploadSize exists in config
	}
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		debug.Error("Error parsing multipart form: %v", err)
		jsonError(w, "Error processing form data: too large or invalid format", http.StatusBadRequest)
		return
	}

	// Extract form data
	name := r.FormValue("name")
	if name == "" {
		jsonError(w, "Hashlist name is required", http.StatusBadRequest)
		return
	}

	hashTypeIDStr := r.FormValue("hash_type_id")
	hashTypeID, err := strconv.Atoi(hashTypeIDStr)
	if err != nil {
		debug.Error("Invalid hash type ID: %s", hashTypeIDStr)
		jsonError(w, "Invalid hash type ID", http.StatusBadRequest)
		return
	}

	// -- Lookup or Create Client by Name ---
	var clientID uuid.UUID = uuid.Nil // Default to Nil (no client)
	clientName := strings.TrimSpace(r.FormValue("client_name"))

	if clientName != "" {
		debug.Info("Processing client name from upload: '%s'", clientName)
		client, err := h.clientRepo.GetByName(ctx, clientName)

		// Corrected Error Handling:
		if err != nil {
			// Handle actual database errors during lookup (excluding ErrNoRows which GetByName doesn't return)
			debug.Error("Error during client lookup for '%s': %v", clientName, err)
			jsonError(w, "Failed to lookup client", http.StatusInternalServerError)
			return
		}

		// If err is nil, check if client was found or not
		if client == nil {
			// *** Client Not Found - Proceed with Creation ***
			debug.Info("Client '%s' not found, creating new client.", clientName)

			// --- Start Client Creation Logic ---

			if len(clientName) > 255 {
				jsonError(w, "Client name exceeds 255 character limit", http.StatusBadRequest)
				return
			}

			// Fetch default retention setting
			debug.Info("Fetching default retention setting...")
			defaultRetentionSetting, err := h.clientSettingsRepo.GetSetting(ctx, "default_data_retention_months")
			var defaultRetentionMonths *int // Use pointer for nullable int

			if err != nil {
				debug.Error("Failed to get default retention setting during client creation: %v. Client will have NULL retention.", err)
			} else if defaultRetentionSetting.Value != nil {
				debug.Info("Default retention setting value found: '%s'", *defaultRetentionSetting.Value)
				val, convErr := strconv.Atoi(*defaultRetentionSetting.Value)
				if convErr != nil {
					debug.Error("Failed to convert default retention setting '%s' to int: %v. Client will have NULL retention.", *defaultRetentionSetting.Value, convErr)
				} else {
					defaultRetentionMonths = &val
					debug.Info("Successfully parsed and applying default retention of %d months to new client '%s'", val, clientName)
				}
			} else {
				debug.Warning("Default retention setting found but its value is nil. Client will have NULL retention.")
			}

			// Construct the new client model
			newClient := &models.Client{
				ID:                  uuid.New(),
				Name:                clientName,
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

			// --- End Client Creation Logic ---

			err = h.clientRepo.Create(ctx, newClient)
			if err != nil {
				if repoErr, ok := err.(*pq.Error); ok && repoErr.Code == "23505" { // Use pq driver error code
					debug.Warning("Race condition during client '%s' creation, re-fetching...", clientName)
					client, err = h.clientRepo.GetByName(ctx, clientName)
					if err != nil || client == nil {
						debug.Error("Failed to re-fetch client '%s' after creation conflict: %v", clientName, err)
						jsonError(w, "Failed to create or find client after conflict", http.StatusInternalServerError)
						return
					}
					clientID = client.ID
					debug.Info("Successfully re-fetched client '%s' after conflict, ID: %s", clientName, clientID)
				} else {
					debug.Error("Error creating new client '%s': %v", clientName, err)
					jsonError(w, "Failed to create client", http.StatusInternalServerError)
					return
				}
			} else {
				clientID = newClient.ID
				debug.Info("Successfully created new client '%s' with ID %s", clientName, clientID)
			}
		} else {
			// *** Client Found - Use Existing ID ***
			clientID = client.ID
			debug.Info("Found existing client '%s' with ID %s", clientName, clientID)
		}
	}

	// Get the file from the request
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

	// Parse the exclude_from_potfile field (defaults to false if not provided)
	excludeStr := r.FormValue("exclude_from_potfile")
	debug.Info("=== POTFILE EXCLUSION DEBUG ===")
	debug.Info("Received exclude_from_potfile: '%s'", excludeStr)
	debug.Info("String is empty: %v", excludeStr == "")

	excludeFromPotfile := false
	if excludeStr != "" {
		var err error
		excludeFromPotfile, err = strconv.ParseBool(excludeStr)
		if err != nil {
			debug.Error("Failed to parse exclude_from_potfile '%s': %v", excludeStr, err)
		} else {
			debug.Info("Successfully parsed exclude_from_potfile as: %v", excludeFromPotfile)
		}
	} else {
		debug.Info("exclude_from_potfile field not provided or empty, using default: false")
	}

	// Create the hashlist record first (without file path)
	debug.Info("Creating hashlist with ExcludeFromPotfile=%v", excludeFromPotfile)
	hashlist := &models.HashList{
		// ID:         uuid.New(), // Removed: ID is now generated by DB BIGSERIAL
		Name:               name,
		UserID:             userID,
		ClientID:           clientID,
		HashTypeID:         hashTypeID,
		Status:             models.HashListStatusUploading,
		ExcludeFromPotfile: excludeFromPotfile,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := h.hashlistRepo.Create(ctx, hashlist); err != nil {
		debug.Error("Error creating hashlist entry: %v", err)
		jsonError(w, "Error saving hashlist metadata", http.StatusInternalServerError)
		return
	}

	// Save the uploaded file using FileRepository
	// Construct path: baseUploadPath/userID/hashlistID/originalFilename
	userUploadDir := filepath.Join(h.cfg.HashUploadDir, userID.String(), strconv.FormatInt(hashlist.ID, 10)) // Use strconv for int64 ID
	savedFilename, err := h.fileRepo.Save(file, userUploadDir, header.Filename)
	if err != nil {
		debug.Error("Error saving uploaded file for hashlist %d: %v", hashlist.ID, err)
		// Attempt to update status to error, but proceed to response even if this fails
		_ = h.hashlistRepo.UpdateStatus(ctx, hashlist.ID, models.HashListStatusError, "Failed to save uploaded file")
		jsonError(w, "Failed to store uploaded file", http.StatusInternalServerError)
		return
	}
	fullPath := filepath.Join(userUploadDir, savedFilename)

	// Update hashlist record with the file path and set status to Processing
	err = h.hashlistRepo.UpdateFilePathAndStatus(ctx, hashlist.ID, fullPath, models.HashListStatusProcessing)
	if err != nil {
		debug.Error("Error updating hashlist file path/status for %d: %v", hashlist.ID, err)
		// Attempt cleanup of saved file
		_ = h.fileRepo.Delete(userUploadDir, savedFilename)
		jsonError(w, "Error updating hashlist details after file save", http.StatusInternalServerError)
		return
	}

	// Submit for background processing
	go h.processor.SubmitHashlistForProcessing(hashlist.ID)

	// Return the initial hashlist record (status is now 'processing')
	hashlist.FilePath = fullPath // Update for response
	hashlist.Status = models.HashListStatusProcessing
	jsonResponse(w, http.StatusAccepted, hashlist)
}

// HandleDeleteHashlist handles DELETE requests for removing a hashlist
func (h *Handler) HandleDeleteHashlist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64) // Parse as int64
	if err != nil {
		debug.Error("Invalid hashlist ID format: %s", idStr)
		jsonError(w, "Invalid hashlist ID format", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	userID, isUser := ctx.Value("user_id").(uuid.UUID)
	role, _ := ctx.Value("user_role").(string)
	isAdmin := role == "admin"

	// Get the hashlist to check ownership and get file path
	hashlist, err := h.hashlistRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			jsonError(w, "Hashlist not found", http.StatusNotFound)
		} else {
			debug.Error("Error getting hashlist %d for delete check: %v", id, err)
			jsonError(w, "Error retrieving hashlist details", http.StatusInternalServerError)
		}
		return
	}

	// Check ownership
	if isUser && !isAdmin && hashlist.UserID != userID {
		jsonError(w, "Access denied", http.StatusForbidden)
		return
	}

	// Delete the DB record first
	err = h.hashlistRepo.Delete(ctx, id) // Pass int64 ID
	if err != nil {
		// Log error, but proceed to attempt file deletion if it wasn't a Not Found error
		debug.Error("Error deleting hashlist %d from DB: %v", id, err)
		if !errors.Is(err, repository.ErrNotFound) { // Only fail request if DB delete failed for reasons other than not found
			jsonError(w, "Error deleting hashlist metadata", http.StatusInternalServerError)
			return
		}
		// If it was NotFound in DB, still try to delete file if path exists?
		// Or assume if DB entry is gone, file should be too / cleanup handled elsewhere?
		// Let's proceed to delete the file based on the path we got IF we got one.
	}

	// Attempt to delete the associated file (using the path stored in the DB record)
	if hashlist != nil && hashlist.FilePath != "" {
		dir := filepath.Dir(hashlist.FilePath)
		filename := filepath.Base(hashlist.FilePath)
		if err := h.fileRepo.Delete(dir, filename); err != nil {
			// Log error but don't fail the request if DB entry was successfully deleted
			debug.Error("Failed to delete hashlist file %s (dir: %s) for hashlist %d: %v", filename, dir, id, err)
		} else {
			debug.Info("Deleted hashlist file %s (dir: %s) for hashlist %d", filename, dir, id)
			// Optionally try to remove empty parent directories - be cautious with this
			_ = os.Remove(dir)               // Try removing hashlistID/filename dir
			_ = os.Remove(filepath.Dir(dir)) // Try removing userID dir
		}
	} else {
		debug.Warning("No file path found or hashlist record was missing when attempting file deletion for hashlist %d", id)
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content on successful deletion (or if not found)
}

// HandleDownloadHashlist handles GET requests for downloading a hashlist
func (h *Handler) HandleDownloadHashlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.ParseInt(idStr, 10, 64) // Parse as int64 instead of uuid.Parse
	if err != nil {
		debug.Error("Invalid hashlist ID format: %s", idStr)
		jsonError(w, "Invalid hashlist ID format", http.StatusBadRequest)
		return
	}

	// Get the hashlist
	hashlist, err := h.hashlistRepo.GetByID(ctx, id) // Pass int64 ID
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			jsonError(w, "Hashlist not found", http.StatusNotFound)
		} else {
			debug.Error("Error getting hashlist %d for download: %v", id, err) // Log with %d
			jsonError(w, "Error retrieving hashlist", http.StatusInternalServerError)
		}
		return
	}

	// Check if user has access
	userID, isUser := r.Context().Value("user_id").(uuid.UUID)
	role, _ := r.Context().Value("user_role").(string)
	isAdmin := role == "admin"
	if isUser && !isAdmin && hashlist.UserID != userID {
		jsonError(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if file exists
	if hashlist.FilePath == "" {
		jsonError(w, "Hashlist file not available or processing not complete", http.StatusNotFound)
		return
	}

	// Open the file using FileRepository
	file, err := h.fileRepo.Open(hashlist.FilePath)
	if err != nil {
		debug.Error("Error opening hashlist file %s: %v", hashlist.FilePath, err)
		jsonError(w, "Error accessing hashlist file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers for file download
	// Use a sanitized name for the download filename
	downloadFilename := SanitizeFilenameSimple(hashlist.Name) + ".txt"
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", downloadFilename))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Stream the file to the response
	_, err = io.Copy(w, file)
	if err != nil {
		debug.Error("Error sending hashlist file %s: %v", hashlist.FilePath, err)
		// Hard to send an error to client at this point
		return
	}
}

// HandleSearchHashes handles POST requests for searching hashes
func (h *Handler) HandleSearchHashes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Parse the request
	type hashSearchRequest struct {
		Hashes []string `json:"hashes"`
	}
	var req hashSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debug.Error("Error decoding search request: %v", err)
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Hashes) == 0 {
		jsonError(w, "At least one hash is required", http.StatusBadRequest)
		return
	}
	// TODO: Add limits on number of hashes allowed per search?

	// Get the user ID from context (needed to filter results)
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Search for the hashes using the hash repository
	results, err := h.hashRepo.SearchHashes(ctx, req.Hashes, userID)
	if err != nil {
		debug.Error("Error searching hashes for user %s: %v", userID, err)
		jsonError(w, "Error searching hashes", http.StatusInternalServerError)
		return
	}

	// Return the results
	jsonResponse(w, http.StatusOK, results)
}

// HandleGetHashlistHashes retrieves hashes associated with a specific hashlist.
func (h *Handler) HandleGetHashlistHashes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64) // Parse as int64
	if err != nil {
		debug.Error("Invalid hashlist ID format: %s", idStr)
		jsonError(w, "Invalid hashlist ID format", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	// Optional: Check if hashlist exists and user has access
	// hashlist, err := h.hashlistRepo.GetByID(ctx, id) ...

	// Get pagination params
	limit, offset := getPaginationParams(r)

	hashes, totalCount, err := h.hashRepo.GetHashesByHashlistID(ctx, id, limit, offset) // Pass int64 ID
	if err != nil {
		debug.Error("Error getting hashes for hashlist %d: %v", id, err) // Log with %d
		jsonError(w, "Error retrieving hashes", http.StatusInternalServerError)
		return
	}

	// Prepare response with pagination
	response := map[string]interface{}{
		"hashes":      hashes,
		"total_count": totalCount,
		"limit":       limit,
		"offset":      offset,
	}

	jsonResponse(w, http.StatusOK, response)
}

// Placeholder for pagination helper function
func getPaginationParams(r *http.Request) (limit int, offset int) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit = 100 // Default limit
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	offset = 0 // Default offset
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}
	return limit, offset
}

// --- Local Helper Functions for JSON responses ---

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

// --- TODO: Add handlers for Clients and Hash Types (CRUD) ---
