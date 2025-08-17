package wordlist

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/wordlist"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/fsutil"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/httputil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles wordlist HTTP requests
type Handler struct {
	manager wordlist.Manager
}

// NewHandler creates a new wordlist handler
func NewHandler(manager wordlist.Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

// HandleListWordlists handles requests to list wordlists
func (h *Handler) HandleListWordlists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters for filtering
	filters := make(map[string]interface{})
	if wordlistType := r.URL.Query().Get("type"); wordlistType != "" {
		filters["wordlist_type"] = wordlistType
	}
	if format := r.URL.Query().Get("format"); format != "" {
		filters["format"] = format
	}
	if tag := r.URL.Query().Get("tag"); tag != "" {
		filters["tag"] = tag
	}

	// Get wordlists
	wordlists, err := h.manager.ListWordlists(ctx, filters)
	if err != nil {
		debug.Error("Failed to list wordlists: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to list wordlists")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, wordlists)
}

// HandleGetWordlist handles requests to get a wordlist
func (h *Handler) HandleGetWordlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get wordlist ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid wordlist ID")
		return
	}

	// Get wordlist
	wordlist, err := h.manager.GetWordlist(ctx, id)
	if err != nil {
		debug.Error("Failed to get wordlist %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get wordlist")
		return
	}

	if wordlist == nil {
		httputil.RespondWithError(w, http.StatusNotFound, "Wordlist not found")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, wordlist)
}

// HandleAddWordlist handles adding a new wordlist (metadata only)
func (h *Handler) HandleAddWordlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Debug log cookies and headers
	debug.Info("HandleAddWordlist: Processing request with cookies: %v", r.Cookies())
	debug.Info("HandleAddWordlist: Request headers: %v", r.Header)

	// Get user ID from context
	userIDStr, ok := ctx.Value("user_id").(string)
	if !ok {
		debug.Error("HandleAddWordlist: User not authenticated, missing user_id in context")
		httputil.RespondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Convert string to UUID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		debug.Error("HandleAddWordlist: Failed to parse user ID as UUID: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Invalid user ID")
		return
	}

	debug.Info("HandleAddWordlist: Authenticated as user: %s", userID.String())

	// Parse multipart form with a very large size limit (effectively unlimited)
	if err := r.ParseMultipartForm(1 << 40); err != nil { // 1TB limit, effectively unlimited for wordlists
		debug.Error("HandleAddWordlist: Failed to parse form: %v", err)
		httputil.RespondWithError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	// Get form values
	name := r.FormValue("name")
	description := r.FormValue("description")
	wordlistType := r.FormValue("wordlist_type")
	format := r.FormValue("format")
	tagsStr := r.FormValue("tags")

	debug.Info("HandleAddWordlist: Received form values - name: %s, type: %s, format: %s", name, wordlistType, format)

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		debug.Error("HandleAddWordlist: Failed to get file from form: %v", err)
		httputil.RespondWithError(w, http.StatusBadRequest, "Failed to get file from form")
		return
	}
	defer file.Close()

	debug.Info("HandleAddWordlist: Received file: %s, size: %d", header.Filename, header.Size)

	// Map file extension to database format enum
	dbFormat := "plaintext" // Default to plaintext
	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".gz", ".zip":
		dbFormat = "compressed"
	case ".txt", ".lst", ".dict", "":
		dbFormat = "plaintext"
	}
	debug.Info("HandleAddWordlist: Determined format from extension %s: %s", ext, dbFormat)

	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	// Create a unique filename to avoid collisions
	if ext == "" {
		ext = "." + format // Use format as extension if not present
	}
	filename := fmt.Sprintf("%s-%s%s", strings.TrimSuffix(filepath.Base(header.Filename), ext), uuid.New().String()[:8], ext)
	debug.Info("HandleAddWordlist: Generated filename: %s", filename)

	// Create temporary file to save the uploaded file
	tempFile, err := os.CreateTemp("", "wordlist-upload-*"+ext)
	if err != nil {
		debug.Error("HandleAddWordlist: Failed to create temp file: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy file to temp location
	_, err = io.Copy(tempFile, file)
	if err != nil {
		debug.Error("HandleAddWordlist: Failed to save file: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to save file")
		return
	}

	// Rewind temp file for reading
	tempFile.Seek(0, 0)

	// Get file size
	fileInfo, err := tempFile.Stat()
	if err != nil {
		debug.Error("HandleAddWordlist: Failed to get file info: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}
	fileSize := fileInfo.Size()
	debug.Info("HandleAddWordlist: File size: %d bytes", fileSize)

	// Calculate MD5 hash
	hash := md5.New()
	tempFile.Seek(0, 0)
	if _, err := io.Copy(hash, tempFile); err != nil {
		debug.Error("HandleAddWordlist: Failed to calculate MD5: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}
	md5Hash := fmt.Sprintf("%x", hash.Sum(nil))
	debug.Info("HandleAddWordlist: Checking for duplicate wordlist with MD5 hash: %s", md5Hash)

	// Check if a wordlist with the same MD5 hash already exists
	existingWordlist, err := h.manager.GetWordlistByMD5Hash(ctx, md5Hash)
	if err != nil {
		debug.Error("Failed to check for duplicate wordlist: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to check for duplicate wordlist")
		return
	}

	if existingWordlist != nil {
		debug.Info("HandleAddWordlist: Duplicate wordlist detected with MD5 hash: %s", md5Hash)
		httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"id":        existingWordlist.ID,
			"name":      existingWordlist.Name,
			"message":   "Wordlist already exists",
			"duplicate": true,
			"success":   true,
		})
		return
	}

	debug.Info("HandleAddWordlist: No duplicate wordlist found, proceeding with upload")

	// Use the original filename but sanitize it
	baseFileName := fsutil.SanitizeFilename(header.Filename)

	// If name is not provided, use the base filename without extension
	if name == "" {
		// Convert to lowercase to match what the monitor does
		name = strings.ToLower(fsutil.ExtractBaseNameWithoutExt(header.Filename))
	}

	// Create the relative path with subdirectory (matching what the monitor would create)
	fileName := filepath.Join(wordlistType, baseFileName)
	debug.Info("HandleAddWordlist: Using sanitized filename with subdirectory: %s", fileName)

	// Check if a file with the same name already exists
	existingWordlistByName, err := h.manager.GetWordlistByFilename(ctx, fileName)
	if err != nil {
		debug.Error("Failed to check for wordlist with same filename: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to check for existing wordlist")
		return
	}

	var wordlistObj *models.Wordlist

	// If a wordlist with the same name exists
	if existingWordlistByName != nil {
		debug.Info("HandleAddWordlist: Found existing wordlist with same filename: %s", fileName)

		// If the MD5 hash is the same, just return the existing wordlist
		if existingWordlistByName.MD5Hash == md5Hash {
			debug.Info("HandleAddWordlist: Existing wordlist has same MD5 hash, returning existing wordlist")
			httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
				"id":        existingWordlistByName.ID,
				"name":      existingWordlistByName.Name,
				"message":   "Wordlist already exists with same content",
				"duplicate": true,
				"success":   true,
			})
			return
		}

		// If the MD5 hash is different, update the existing wordlist
		debug.Info("HandleAddWordlist: Existing wordlist has different MD5 hash, updating")

		// Update the existing wordlist with new file info
		updateReq := &models.WordlistUpdateRequest{
			Name:         name,
			Description:  description,
			WordlistType: wordlistType,
			Format:       dbFormat,
			Tags:         append(tags, "updated"),
		}

		if _, err := h.manager.UpdateWordlist(ctx, existingWordlistByName.ID, updateReq, userID); err != nil {
			debug.Error("HandleAddWordlist: Failed to update existing wordlist: %v", err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to update existing wordlist")
			return
		}

		// We'll continue with the upload process but use the existing wordlist ID
		wordlistObj = existingWordlistByName
	} else {
		// Create wordlist in database
		req := &models.WordlistAddRequest{
			Name:         name,
			Description:  description,
			WordlistType: wordlistType,
			Format:       dbFormat,
			FileName:     fileName,
			MD5Hash:      md5Hash,
			FileSize:     fileSize,
			WordCount:    0, // Will be updated during verification
			Tags:         tags,
		}

		// Add wordlist to database
		var err error
		wordlistObj, err = h.manager.AddWordlist(ctx, req, userID)
		if err != nil {
			debug.Error("HandleAddWordlist: Failed to add wordlist: %v", err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to add wordlist")
			return
		}
	}

	// Rewind temp file again for final copy
	tempFile.Seek(0, 0)

	// Get the destination path for the wordlist file
	destPath := h.manager.GetWordlistPath(fileName, wordlistType)

	// Create the destination directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		debug.Error("HandleAddWordlist: Failed to create destination directory: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to save wordlist file")
		return
	}

	// Create the destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		debug.Error("HandleAddWordlist: Failed to create destination file: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to save wordlist file")
		return
	}
	defer destFile.Close()

	// Copy the file to the destination
	if _, err := io.Copy(destFile, tempFile); err != nil {
		debug.Error("HandleAddWordlist: Failed to copy file to destination: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to save wordlist file")
		return
	}

	// Automatically trigger verification process
	debug.Info("HandleAddWordlist: Automatically triggering verification for wordlist ID %d", wordlistObj.ID)

	// For large files (over 100MB), perform verification asynchronously
	if fileSize > 100*1024*1024 {
		debug.Info("HandleAddWordlist: Large file detected (%d bytes), starting async verification", fileSize)

		// Start verification in a goroutine
		go func() {
			// Create a new context for the background task
			bgCtx := context.Background()

			debug.Info("HandleAddWordlist: Starting async word count for wordlist ID %d", wordlistObj.ID)

			// Count words in the file
			wordCount, err := h.manager.CountWordsInFile(destPath)
			if err != nil {
				debug.Error("HandleAddWordlist: Async word count failed for wordlist ID %d: %v", wordlistObj.ID, err)

				// Update status to failed if word count fails
				failedStatus := "failed"
				failReq := &models.WordlistVerifyRequest{
					Status: failedStatus,
				}
				if verifyErr := h.manager.VerifyWordlist(bgCtx, wordlistObj.ID, failReq); verifyErr != nil {
					debug.Error("HandleAddWordlist: Failed to update verification status to failed: %v", verifyErr)
				}
				return
			}

			debug.Info("HandleAddWordlist: Async word count completed for wordlist ID %d: %d words", wordlistObj.ID, wordCount)

			// Create verification request
			verifyReq := &models.WordlistVerifyRequest{
				Status:    "verified",
				WordCount: &wordCount,
			}

			// Verify wordlist
			if err := h.manager.VerifyWordlist(bgCtx, wordlistObj.ID, verifyReq); err != nil {
				debug.Error("HandleAddWordlist: Async verification failed for wordlist ID %d: %v", wordlistObj.ID, err)
			} else {
				debug.Info("HandleAddWordlist: Successfully verified wordlist %d with %d words", wordlistObj.ID, wordCount)
			}
		}()

		// Return success response immediately with pending status
		wordlistObj.VerificationStatus = "pending"
		httputil.RespondWithJSON(w, http.StatusCreated, wordlistObj)
		return
	}

	// For smaller files, verify synchronously
	// Count words in the file
	wordCount, err := h.manager.CountWordsInFile(destPath)
	if err != nil {
		debug.Warning("HandleAddWordlist: Failed to count words in file %s: %v", destPath, err)
		// Continue with verification even if word count fails
	}

	// Create verification request
	verifyReq := &models.WordlistVerifyRequest{
		Status:    "verified",
		WordCount: &wordCount,
	}

	// Verify wordlist
	if err := h.manager.VerifyWordlist(ctx, wordlistObj.ID, verifyReq); err != nil {
		debug.Warning("HandleAddWordlist: Failed to automatically verify wordlist %d: %v", wordlistObj.ID, err)
		// Don't fail the upload if verification fails
	} else {
		debug.Info("HandleAddWordlist: Successfully verified wordlist %d with %d words", wordlistObj.ID, wordCount)
		// Update the wordlist object with the verified status and word count for the response
		wordlistObj.VerificationStatus = "verified"
		wordlistObj.WordCount = wordCount
	}

	// Return success response
	httputil.RespondWithJSON(w, http.StatusCreated, wordlistObj)
}

// HandleUpdateWordlist handles requests to update a wordlist
func (h *Handler) HandleUpdateWordlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userIDStr, ok := ctx.Value("user_id").(string)
	if !ok {
		debug.Error("Failed to get user ID from context")
		httputil.RespondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Convert string to UUID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		debug.Error("Failed to parse user ID as UUID: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Invalid user ID")
		return
	}

	// Get wordlist ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid wordlist ID")
		return
	}

	// Parse request body
	var req models.WordlistUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update wordlist
	wordlist, err := h.manager.UpdateWordlist(ctx, id, &req, userID)
	if err != nil {
		debug.Error("Failed to update wordlist %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to update wordlist")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, wordlist)
}

// HandleDeleteWordlist handles requests to delete a wordlist
func (h *Handler) HandleDeleteWordlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get wordlist ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid wordlist ID")
		return
	}

	// Delete wordlist
	if err := h.manager.DeleteWordlist(ctx, id); err != nil {
		debug.Error("Failed to delete wordlist %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to delete wordlist")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Wordlist deleted"})
}

// HandleVerifyWordlist handles requests to verify a wordlist
func (h *Handler) HandleVerifyWordlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get wordlist ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid wordlist ID")
		return
	}

	// Parse request body
	var req models.WordlistVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Verify wordlist
	if err := h.manager.VerifyWordlist(ctx, id, &req); err != nil {
		debug.Error("Failed to verify wordlist %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to verify wordlist")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Wordlist verified"})
}

// HandleAddWordlistTag handles requests to add a tag to a wordlist
func (h *Handler) HandleAddWordlistTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userIDStr, ok := ctx.Value("user_id").(string)
	if !ok {
		debug.Error("Failed to get user ID from context")
		httputil.RespondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Convert string to UUID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		debug.Error("Failed to parse user ID as UUID: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Invalid user ID")
		return
	}

	// Get wordlist ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid wordlist ID")
		return
	}

	// Parse request body
	var req models.WordlistTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Add tag
	if err := h.manager.AddWordlistTag(ctx, id, req.Tag, userID); err != nil {
		debug.Error("Failed to add tag to wordlist %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to add tag")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Tag added"})
}

// HandleDeleteWordlistTag handles requests to delete a tag from a wordlist
func (h *Handler) HandleDeleteWordlistTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get wordlist ID and tag from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid wordlist ID")
		return
	}
	tag := vars["tag"]

	// Delete tag
	if err := h.manager.DeleteWordlistTag(ctx, id, tag); err != nil {
		debug.Error("Failed to delete tag from wordlist %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to delete tag")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Tag deleted"})
}

// HandleDownloadWordlist handles requests to download a wordlist
func (h *Handler) HandleDownloadWordlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get wordlist ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid wordlist ID")
		return
	}

	// Get wordlist
	wordlist, err := h.manager.GetWordlist(ctx, id)
	if err != nil {
		debug.Error("Failed to get wordlist %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get wordlist")
		return
	}

	if wordlist == nil {
		httputil.RespondWithError(w, http.StatusNotFound, "Wordlist not found")
		return
	}

	// Get file path
	filePath := h.manager.GetWordlistPath(wordlist.FileName, wordlist.WordlistType)
	debug.Info("Downloading wordlist ID %d, filename: %s, path: %s", id, wordlist.FileName, filePath)

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		debug.Error("Wordlist file not found at path: %s", filePath)
		httputil.RespondWithError(w, http.StatusNotFound, "Wordlist file not found")
		return
	}
	debug.Info("File exists at %s, size: %d bytes", filePath, fileInfo.Size())

	// Set headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", wordlist.FileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	// Prevent caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	// Don't set Content-Length - let http.ServeFile handle it based on actual file size

	// Log before serving
	debug.Info("About to serve file from path: %s", filePath)
	
	// Serve file
	http.ServeFile(w, r, filePath)
	
	debug.Info("File served successfully")
}
