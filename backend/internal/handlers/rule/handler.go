package rule

import (
	"bufio"
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
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/config"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/rule"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/fsutil"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/httputil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles rule management HTTP requests
type Handler struct {
	manager rule.Manager
	config  *config.Config
}

// NewHandler creates a new rule management handler
func NewHandler(manager rule.Manager, cfg *config.Config) *Handler {
	return &Handler{
		manager: manager,
		config:  cfg,
	}
}

// Request/Response types
type AddRuleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	RuleType    string   `json:"rule_type"`
	Tags        []string `json:"tags,omitempty"`
}

type RuleResponse struct {
	ID                 int      `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	RuleType           string   `json:"rule_type"`
	FileName           string   `json:"file_name"`
	MD5Hash            string   `json:"md5_hash"`
	FileSize           int64    `json:"file_size"`
	RuleCount          int64    `json:"rule_count"`
	CreatedAt          string   `json:"created_at"`
	CreatedBy          string   `json:"created_by"`
	UpdatedAt          string   `json:"updated_at"`
	UpdatedBy          string   `json:"updated_by,omitempty"`
	LastVerifiedAt     string   `json:"last_verified_at,omitempty"`
	VerificationStatus string   `json:"verification_status"`
	Tags               []string `json:"tags,omitempty"`
}

type AddTagRequest struct {
	Tag string `json:"tag"`
}

// HandleListRules handles listing all rules
func (h *Handler) HandleListRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters for filtering
	filters := make(map[string]interface{})
	if ruleType := r.URL.Query().Get("type"); ruleType != "" {
		filters["rule_type"] = ruleType
	}
	if tag := r.URL.Query().Get("tag"); tag != "" {
		filters["tag"] = tag
	}

	// Get rules from database
	rules, err := h.manager.ListRules(ctx, filters)
	if err != nil {
		debug.Error("Failed to list rules: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to list rules")
		return
	}

	// Convert rules to response format
	response := make([]RuleResponse, len(rules))
	for i, rule := range rules {
		response[i] = convertRuleToResponse(rule)
	}

	httputil.RespondWithJSON(w, http.StatusOK, response)
}

// HandleListRulesForAgent handles listing rules for agents
func (h *Handler) HandleListRulesForAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters for filtering
	filters := make(map[string]interface{})
	if ruleType := r.URL.Query().Get("type"); ruleType != "" {
		filters["rule_type"] = ruleType
	}
	if tag := r.URL.Query().Get("tag"); tag != "" {
		filters["tag"] = tag
	}

	// Only return verified rules for agents
	filters["verification_status"] = "verified"

	// Get rules from database
	rules, err := h.manager.ListRules(ctx, filters)
	if err != nil {
		debug.Error("Failed to list rules: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to list rules")
		return
	}

	// Convert rules to response format
	response := make([]RuleResponse, len(rules))
	for i, rule := range rules {
		response[i] = convertRuleToResponse(rule)
	}

	httputil.RespondWithJSON(w, http.StatusOK, response)
}

// HandleGetRule handles getting a rule by ID
func (h *Handler) HandleGetRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get rule ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Rule ID is required")
		return
	}

	// Convert ID to int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	rule, err := h.manager.GetRule(ctx, id)
	if err != nil {
		debug.Error("Failed to get rule %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get rule")
		return
	}

	if rule == nil {
		httputil.RespondWithError(w, http.StatusNotFound, "Rule not found")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, convertRuleToResponse(rule))
}

// HandleAddRule handles the request to add a new rule
func (h *Handler) HandleAddRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userIDStr, ok := ctx.Value("user_id").(string)
	if !ok {
		debug.Error("Failed to get user ID from context")
		httputil.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Convert string to UUID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		debug.Error("Failed to parse user ID as UUID: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Invalid user ID")
		return
	}

	// Parse the multipart form
	err = r.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		debug.Error("Failed to parse multipart form: %v", err)
		httputil.RespondWithError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("file")
	if err != nil {
		debug.Error("Failed to get file from form: %v", err)
		httputil.RespondWithError(w, http.StatusBadRequest, "Failed to get file")
		return
	}
	defer file.Close()

	// Get the rule name from the form
	ruleName := r.FormValue("name")
	if ruleName == "" {
		// Use the base filename without any extensions
		// Convert to lowercase to match what the monitor does
		ruleName = strings.ToLower(fsutil.ExtractBaseNameWithoutExt(header.Filename))
	}

	description := r.FormValue("description")

	// Get rule type
	ruleType := r.FormValue("rule_type")
	if ruleType == "" {
		debug.Error("Rule type is required")
		httputil.RespondWithError(w, http.StatusBadRequest, "Rule type is required")
		return
	}

	// Get tags
	tagsStr := r.FormValue("tags")
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	// Create a temporary file to store the uploaded file
	tempFile, err := os.CreateTemp("", "rule-*"+filepath.Ext(header.Filename))
	if err != nil {
		debug.Error("Failed to create temporary file: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy the uploaded file to the temporary file
	_, err = io.Copy(tempFile, file)
	if err != nil {
		debug.Error("Failed to copy file: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}

	// Calculate MD5 hash
	hash := md5.New()
	if _, err := io.Copy(hash, tempFile); err != nil {
		debug.Error("HandleAddRule: Failed to calculate MD5: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}
	md5Hash := fmt.Sprintf("%x", hash.Sum(nil))
	debug.Info("HandleAddRule: Checking for duplicate rule with MD5 hash: %s", md5Hash)

	// Check if a rule with the same MD5 hash already exists
	existingRule, err := h.manager.GetRuleByMD5Hash(ctx, md5Hash)
	if err != nil {
		debug.Error("Failed to check for duplicate rule: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to check for duplicate rule")
		return
	}

	if existingRule != nil {
		debug.Info("HandleAddRule: Duplicate rule detected with MD5 hash: %s", md5Hash)
		httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"id":        existingRule.ID,
			"name":      existingRule.Name,
			"message":   "Rule already exists",
			"duplicate": true,
			"success":   true,
		})
		return
	}

	debug.Info("HandleAddRule: No duplicate rule found, proceeding with upload")

	// Get file size
	fileInfo, err := tempFile.Stat()
	if err != nil {
		debug.Error("HandleAddRule: Failed to get file size: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}
	fileSize := fileInfo.Size()
	debug.Info("HandleAddRule: File size: %d bytes", fileSize)

	// Count the number of rules in the file
	tempFile.Seek(0, 0)
	scanner := bufio.NewScanner(tempFile)
	ruleCount := int64(0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			ruleCount++
		}
	}
	if err := scanner.Err(); err != nil {
		debug.Error("Failed to count rules in file: %v", err)
		// Continue with ruleCount = 0
	}

	// Use the original filename but sanitize it
	baseFileName := fsutil.SanitizeFilename(header.Filename)

	// Create the relative path with subdirectory (matching what the monitor would create)
	fileName := filepath.Join(ruleType, baseFileName)
	debug.Info("HandleAddRule: Using sanitized filename with subdirectory: %s", fileName)

	// Check if a file with the same name already exists
	existingRuleByName, err := h.manager.GetRuleByFilename(ctx, fileName)
	if err != nil {
		debug.Error("Failed to check for rule with same filename: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to check for existing rule")
		return
	}

	var ruleObj *models.Rule

	// If a rule with the same name exists
	if existingRuleByName != nil {
		debug.Info("HandleAddRule: Found existing rule with same filename: %s", fileName)

		// If the MD5 hash is the same, just return the existing rule
		if existingRuleByName.MD5Hash == md5Hash {
			debug.Info("HandleAddRule: Existing rule has same MD5 hash, returning existing rule")
			httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
				"id":        existingRuleByName.ID,
				"name":      existingRuleByName.Name,
				"message":   "Rule already exists with same content",
				"duplicate": true,
				"success":   true,
			})
			return
		}

		// If the MD5 hash is different, update the existing rule
		debug.Info("HandleAddRule: Existing rule has different MD5 hash, updating")

		// Update the existing rule with new file info
		updateReq := &models.RuleUpdateRequest{
			Name:        ruleName,
			Description: description,
			RuleType:    ruleType,
			Tags:        append(tags, "updated"),
		}

		if _, err := h.manager.UpdateRule(ctx, existingRuleByName.ID, updateReq, userID); err != nil {
			debug.Error("HandleAddRule: Failed to update existing rule: %v", err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to update existing rule")
			return
		}

		// We'll continue with the upload process but use the existing rule ID
		ruleObj = existingRuleByName
	} else {
		// Create rule in database
		req := &models.RuleAddRequest{
			Name:        ruleName,
			Description: description,
			RuleType:    ruleType,
			FileName:    fileName,
			MD5Hash:     md5Hash,
			FileSize:    fileSize,
			RuleCount:   ruleCount,
			Tags:        tags,
		}

		// Add rule to database
		var err error
		ruleObj, err = h.manager.AddRule(ctx, req, userID)
		if err != nil {
			debug.Error("Failed to add rule to database: %v", err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to add rule")
			return
		}
	}

	// Save the file to the rules directory
	destPath := h.manager.GetRulePath(fileName, ruleType)
	debug.Info("HandleAddRule: Saving rule file to: %s", destPath)

	// Create directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		debug.Error("Failed to create rules directory: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to save rule file")
		return
	}

	// Open the temp file for reading
	tempFile.Seek(0, 0)

	// Create the destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		debug.Error("Failed to create destination file: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to save rule file")
		return
	}
	defer destFile.Close()

	// Copy the file
	if _, err := io.Copy(destFile, tempFile); err != nil {
		debug.Error("Failed to copy rule file: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to save rule file")
		return
	}

	debug.Info("HandleAddRule: Successfully saved rule file to: %s", destPath)

	// For large files (over 10MB), perform verification asynchronously
	if fileSize > 10*1024*1024 {
		debug.Info("HandleAddRule: Large file detected (%d bytes), starting async verification", fileSize)

		// Start verification in a goroutine
		go func() {
			// Create a new context for the background task
			bgCtx := context.Background()

			debug.Info("HandleAddRule: Starting async rule count for rule ID %d", ruleObj.ID)

			// Count rules in the file
			newRuleCount, err := h.manager.CountRulesInFile(tempFile.Name())
			if err != nil {
				debug.Error("HandleAddRule: Async rule count failed for rule ID %d: %v", ruleObj.ID, err)

				// Update status to failed if rule count fails
				failedStatus := "failed"
				failReq := &models.RuleVerifyRequest{
					Status: failedStatus,
				}
				if verifyErr := h.manager.VerifyRule(bgCtx, ruleObj.ID, failReq); verifyErr != nil {
					debug.Error("HandleAddRule: Failed to update verification status to failed: %v", verifyErr)
				}
				return
			}

			debug.Info("HandleAddRule: Async rule count completed for rule ID %d: %d rules", ruleObj.ID, newRuleCount)

			// Create verification request
			verifyReq := &models.RuleVerifyRequest{
				Status:    "verified",
				RuleCount: &newRuleCount,
			}

			// Verify rule
			if err := h.manager.VerifyRule(bgCtx, ruleObj.ID, verifyReq); err != nil {
				debug.Error("HandleAddRule: Async verification failed for rule ID %d: %v", ruleObj.ID, err)
			} else {
				debug.Info("HandleAddRule: Successfully verified rule %d with %d rules", ruleObj.ID, newRuleCount)
			}
		}()

		// Return success response immediately with pending status
		httputil.RespondWithJSON(w, http.StatusCreated, ruleObj)
		return
	}

	// For smaller files, verify synchronously
	verifyReq := &models.RuleVerifyRequest{
		Status:    "verified",
		RuleCount: &ruleCount,
	}

	if err := h.manager.VerifyRule(ctx, ruleObj.ID, verifyReq); err != nil {
		debug.Warning("Failed to verify rule: %v", err)
		// Don't fail the upload if verification fails
	} else {
		ruleObj.VerificationStatus = "verified"
	}

	httputil.RespondWithJSON(w, http.StatusCreated, ruleObj)
}

// HandleUpdateRule handles updating a rule
func (h *Handler) HandleUpdateRule(w http.ResponseWriter, r *http.Request) {
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

	// Get rule ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Rule ID is required")
		return
	}

	// Convert ID to int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	// Parse request body
	var req models.RuleUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update rule
	rule, err := h.manager.UpdateRule(ctx, id, &req, userID)
	if err != nil {
		debug.Error("Failed to update rule %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to update rule")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, convertRuleToResponse(rule))
}

// HandleDeleteRule handles deleting a rule
func (h *Handler) HandleDeleteRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get rule ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Rule ID is required")
		return
	}

	// Convert ID to int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	// Delete rule
	if err := h.manager.DeleteRule(ctx, id); err != nil {
		debug.Error("Failed to delete rule %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to delete rule")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Rule deleted successfully"})
}

// HandleVerifyRule handles verifying a rule
func (h *Handler) HandleVerifyRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get rule ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Rule ID is required")
		return
	}

	// Convert ID to int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	// Parse request body
	var req models.RuleVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Verify rule
	if err := h.manager.VerifyRule(ctx, id, &req); err != nil {
		debug.Error("Failed to verify rule %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to verify rule")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Rule verified successfully"})
}

// HandleUploadRule handles uploading a rule file
func (h *Handler) HandleUploadRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse multipart form
	if err := r.ParseMultipartForm(1 << 30); err != nil { // 1GB limit
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get user ID from context
	userIDStr, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get form values
	ruleName := r.FormValue("name")
	if ruleName == "" {
		// Use the base filename without any extensions
		// Convert to lowercase to match what the monitor does
		ruleName = strings.ToLower(fsutil.ExtractBaseNameWithoutExt(header.Filename))
	}

	description := r.FormValue("description")

	// Get rule type
	ruleType := r.FormValue("rule_type")
	if ruleType == "" {
		debug.Error("Rule type is required")
		httputil.RespondWithError(w, http.StatusBadRequest, "Rule type is required")
		return
	}

	// Get tags
	tagsStr := r.FormValue("tags")
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	// Create temporary file to save the uploaded file
	tempFile, err := os.CreateTemp("", "rule-*"+filepath.Ext(header.Filename))
	if err != nil {
		debug.Error("Failed to create temp file: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy file to temp location
	if _, err := io.Copy(tempFile, file); err != nil {
		debug.Error("Failed to copy file to temp location: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}

	// Get file size
	fileInfo, err := tempFile.Stat()
	if err != nil {
		debug.Error("Failed to get file info: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}
	fileSize := fileInfo.Size()

	// Calculate MD5 hash
	hash := md5.New()
	if _, err := io.Copy(hash, tempFile); err != nil {
		debug.Error("HandleUploadRule: Failed to calculate MD5: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process file")
		return
	}
	md5Hash := fmt.Sprintf("%x", hash.Sum(nil))
	debug.Info("HandleUploadRule: Checking for duplicate rule with MD5 hash: %s", md5Hash)

	// Check if a rule with the same MD5 hash already exists
	existingRule, err := h.manager.GetRuleByMD5Hash(ctx, md5Hash)
	if err != nil {
		debug.Error("Failed to check for duplicate rule: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to check for duplicate rule")
		return
	}

	if existingRule != nil {
		debug.Info("HandleUploadRule: Duplicate rule detected with MD5 hash: %s", md5Hash)
		httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"id":        existingRule.ID,
			"name":      existingRule.Name,
			"message":   "Rule already exists",
			"duplicate": true,
			"success":   true,
		})
		return
	}

	debug.Info("HandleUploadRule: No duplicate rule found, proceeding with upload")

	// Count the number of rules in the file
	tempFile.Seek(0, 0)
	scanner := bufio.NewScanner(tempFile)
	ruleCount := int64(0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			ruleCount++
		}
	}
	if err := scanner.Err(); err != nil {
		debug.Error("Failed to count rules in file: %v", err)
		// Continue with ruleCount = 0
	}

	// Use the original filename but sanitize it
	baseFileName := fsutil.SanitizeFilename(header.Filename)

	// Create the relative path with subdirectory (matching what the monitor would create)
	fileName := filepath.Join(ruleType, baseFileName)
	debug.Info("HandleAddRule: Using sanitized filename with subdirectory: %s", fileName)

	// Check if a file with the same name already exists
	existingRuleByName, err := h.manager.GetRuleByFilename(ctx, fileName)
	if err != nil {
		debug.Error("Failed to check for rule with same filename: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to check for existing rule")
		return
	}

	var ruleObj *models.Rule

	// If a rule with the same name exists
	if existingRuleByName != nil {
		debug.Info("HandleAddRule: Found existing rule with same filename: %s", fileName)

		// If the MD5 hash is the same, just return the existing rule
		if existingRuleByName.MD5Hash == md5Hash {
			debug.Info("HandleAddRule: Existing rule has same MD5 hash, returning existing rule")
			httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
				"id":        existingRuleByName.ID,
				"name":      existingRuleByName.Name,
				"message":   "Rule already exists with same content",
				"duplicate": true,
				"success":   true,
			})
			return
		}

		// If the MD5 hash is different, update the existing rule
		debug.Info("HandleAddRule: Existing rule has different MD5 hash, updating")

		// Update the existing rule with new file info
		updateReq := &models.RuleUpdateRequest{
			Name:        ruleName,
			Description: description,
			RuleType:    ruleType,
			Tags:        append(tags, "updated"),
		}

		if _, err := h.manager.UpdateRule(ctx, existingRuleByName.ID, updateReq, userID); err != nil {
			debug.Error("HandleAddRule: Failed to update existing rule: %v", err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to update existing rule")
			return
		}

		// We'll continue with the upload process but use the existing rule ID
		ruleObj = existingRuleByName
	} else {
		// Create rule in database
		req := &models.RuleAddRequest{
			Name:        ruleName,
			Description: description,
			RuleType:    ruleType,
			FileName:    fileName,
			MD5Hash:     md5Hash,
			FileSize:    fileSize,
			RuleCount:   ruleCount,
			Tags:        tags,
		}

		// Add rule to database
		var err error
		ruleObj, err = h.manager.AddRule(ctx, req, userID)
		if err != nil {
			debug.Error("Failed to add rule to database: %v", err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to add rule")
			return
		}
	}

	// Save the file to the rules directory
	destPath := h.manager.GetRulePath(fileName, ruleType)
	debug.Info("HandleAddRule: Saving rule file to: %s", destPath)

	// Create directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		debug.Error("Failed to create rules directory: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to save rule file")
		return
	}

	// Open the temp file for reading
	tempFile.Seek(0, 0)

	// Create the destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		debug.Error("Failed to create destination file: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to save rule file")
		return
	}
	defer destFile.Close()

	// Copy the file
	if _, err := io.Copy(destFile, tempFile); err != nil {
		debug.Error("Failed to copy rule file: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to save rule file")
		return
	}

	debug.Info("HandleAddRule: Successfully saved rule file to: %s", destPath)

	// For large files (over 10MB), perform verification asynchronously
	if fileSize > 10*1024*1024 {
		debug.Info("HandleAddRule: Large file detected (%d bytes), starting async verification", fileSize)

		// Start verification in a goroutine
		go func() {
			// Create a new context for the background task
			bgCtx := context.Background()

			debug.Info("HandleAddRule: Starting async rule count for rule ID %d", ruleObj.ID)

			// Count rules in the file
			newRuleCount, err := h.manager.CountRulesInFile(tempFile.Name())
			if err != nil {
				debug.Error("HandleAddRule: Async rule count failed for rule ID %d: %v", ruleObj.ID, err)

				// Update status to failed if rule count fails
				failedStatus := "failed"
				failReq := &models.RuleVerifyRequest{
					Status: failedStatus,
				}
				if verifyErr := h.manager.VerifyRule(bgCtx, ruleObj.ID, failReq); verifyErr != nil {
					debug.Error("HandleAddRule: Failed to update verification status to failed: %v", verifyErr)
				}
				return
			}

			debug.Info("HandleAddRule: Async rule count completed for rule ID %d: %d rules", ruleObj.ID, newRuleCount)

			// Create verification request
			verifyReq := &models.RuleVerifyRequest{
				Status:    "verified",
				RuleCount: &newRuleCount,
			}

			// Verify rule
			if err := h.manager.VerifyRule(bgCtx, ruleObj.ID, verifyReq); err != nil {
				debug.Error("HandleAddRule: Async verification failed for rule ID %d: %v", ruleObj.ID, err)
			} else {
				debug.Info("HandleAddRule: Successfully verified rule %d with %d rules", ruleObj.ID, newRuleCount)
			}
		}()

		// Return success response immediately with pending status
		httputil.RespondWithJSON(w, http.StatusCreated, ruleObj)
		return
	}

	// For smaller files, verify synchronously
	verifyReq := &models.RuleVerifyRequest{
		Status:    "verified",
		RuleCount: &ruleCount,
	}

	if err := h.manager.VerifyRule(ctx, ruleObj.ID, verifyReq); err != nil {
		debug.Warning("Failed to verify rule: %v", err)
		// Don't fail the upload if verification fails
	} else {
		ruleObj.VerificationStatus = "verified"
	}

	httputil.RespondWithJSON(w, http.StatusCreated, ruleObj)
}

// HandleGetRuleTags handles getting tags for a rule
func (h *Handler) HandleGetRuleTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	// Get rule to access its tags
	rule, err := h.manager.GetRule(ctx, id)
	if err != nil {
		debug.Error("Failed to get rule %d: %v", id, err)
		http.Error(w, "Failed to get rule", http.StatusInternalServerError)
		return
	}

	if rule == nil {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule.Tags)
}

// HandleAddRuleTag handles adding a tag to a rule
func (h *Handler) HandleAddRuleTag(w http.ResponseWriter, r *http.Request) {
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

	// Get rule ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Rule ID is required")
		return
	}

	// Convert ID to int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	// Parse request body
	var req struct {
		Tag string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Add tag
	if err := h.manager.AddRuleTag(ctx, id, req.Tag, userID); err != nil {
		debug.Error("Failed to add tag to rule %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to add tag")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Tag added successfully"})
}

// HandleDeleteRuleTag handles deleting a tag from a rule
func (h *Handler) HandleDeleteRuleTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get rule ID and tag from URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Rule ID is required")
		return
	}
	tag := vars["tag"]
	if tag == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Tag is required")
		return
	}

	// Convert ID to int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	// Delete tag
	if err := h.manager.DeleteRuleTag(ctx, id, tag); err != nil {
		debug.Error("Failed to delete tag from rule %d: %v", id, err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to delete tag")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Tag deleted successfully"})
}

// HandleDownloadRule handles downloading a rule file
func (h *Handler) HandleDownloadRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get filename from URL
	vars := mux.Vars(r)
	filename := vars["filename"]
	if filename == "" {
		// Try to get rule ID from URL
		idStr := vars["id"]
		if idStr == "" {
			http.Error(w, "Missing rule ID or filename", http.StatusBadRequest)
			return
		}

		// Convert ID to int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid rule ID", http.StatusBadRequest)
			return
		}

		// Get rule by ID
		rule, err := h.manager.GetRule(ctx, id)
		if err != nil {
			debug.Error("Failed to get rule %d: %v", id, err)
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}

		filename = rule.FileName
	}

	// Get rule by filename
	rule, err := h.manager.GetRuleByFilename(ctx, filename)
	if err != nil {
		debug.Error("Failed to get rule by filename %s: %v", filename, err)
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	// Get file path
	filePath := h.manager.GetRulePath(rule.FileName, rule.RuleType)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		debug.Error("Rule file not found: %s", filePath)
		http.Error(w, "Rule file not found", http.StatusNotFound)
		return
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		debug.Error("Failed to open rule file %s: %v", filePath, err)
		http.Error(w, "Failed to open rule file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", rule.FileSize))

	// Stream file to response
	if _, err := io.Copy(w, file); err != nil {
		debug.Error("Failed to stream rule file %s: %v", filePath, err)
		// Can't send error response here as headers are already sent
	}
}

// Helper function to convert a rule to response format
func convertRuleToResponse(rule *models.Rule) RuleResponse {
	response := RuleResponse{
		ID:                 rule.ID,
		Name:               rule.Name,
		Description:        rule.Description,
		RuleType:           rule.RuleType,
		FileName:           rule.FileName,
		MD5Hash:            rule.MD5Hash,
		FileSize:           rule.FileSize,
		RuleCount:          rule.RuleCount,
		CreatedAt:          rule.CreatedAt.Format(time.RFC3339),
		CreatedBy:          rule.CreatedBy.String(),
		UpdatedAt:          rule.UpdatedAt.Format(time.RFC3339),
		VerificationStatus: rule.VerificationStatus,
		Tags:               rule.Tags,
	}

	// Add optional fields if present
	if rule.UpdatedBy != uuid.Nil {
		response.UpdatedBy = rule.UpdatedBy.String()
	}

	if !rule.LastVerifiedAt.IsZero() {
		response.LastVerifiedAt = rule.LastVerifiedAt.Format(time.RFC3339)
	}

	return response
}
