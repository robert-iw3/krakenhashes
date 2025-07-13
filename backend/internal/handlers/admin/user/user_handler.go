package user

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/httputil"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/password"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler handles API requests for admin user management
type UserHandler struct {
	userRepo *repository.UserRepository
}

// NewUserHandler creates a new handler instance
func NewUserHandler(ur *repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: ur,
	}
}

// ListUsers godoc
// @Summary List all users
// @Description Retrieves a list of all users in the system for admin view
// @Tags Admin Users
// @Produce json
// @Success 200 {object} httputil.SuccessResponse{data=[]models.User}
// @Failure 500 {object} httputil.ErrorResponse
// @Router /admin/users [get]
// @Security ApiKeyAuth
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListAll(r.Context())
	if err != nil {
		debug.Error("Failed to list users: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}

	// Clear sensitive data before sending
	for i := range users {
		users[i].PasswordHash = ""
		users[i].MFASecret = ""
		users[i].BackupCodes = nil
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"data": users})
}

// GetUser godoc
// @Summary Get user details
// @Description Retrieves detailed information about a specific user
// @Tags Admin Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} httputil.SuccessResponse{data=models.User}
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /admin/users/{id} [get]
// @Security ApiKeyAuth
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.userRepo.GetDetails(r.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		debug.Error("Failed to get user details: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve user details")
		return
	}

	// Clear sensitive data before sending
	user.PasswordHash = ""
	user.MFASecret = ""
	user.BackupCodes = nil

	httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"data": user})
}

// UpdateUser godoc
// @Summary Update user details
// @Description Updates username and/or email for a user
// @Tags Admin Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body object{username=string,email=string} true "User update data"
// @Success 200 {object} httputil.SuccessResponse{data=object{message=string}}
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 409 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /admin/users/{id} [put]
// @Security ApiKeyAuth
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var updateData struct {
		Username *string `json:"username,omitempty"`
		Email    *string `json:"email,omitempty"`
		Role     *string `json:"role,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate email format if provided
	if updateData.Email != nil && *updateData.Email != "" {
		if !strings.Contains(*updateData.Email, "@") {
			httputil.RespondWithError(w, http.StatusBadRequest, "Invalid email format")
			return
		}
	}

	// Validate role if provided
	if updateData.Role != nil {
		// Prevent setting role to system
		if *updateData.Role == "system" {
			httputil.RespondWithError(w, http.StatusBadRequest, "Cannot assign system role")
			return
		}
		// Validate role is valid
		if *updateData.Role != "admin" && *updateData.Role != "user" {
			httputil.RespondWithError(w, http.StatusBadRequest, "Invalid role. Must be 'admin' or 'user'")
			return
		}
	}

	// Check if target user is system user
	targetUser, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		debug.Error("Failed to get user: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Prevent modifying system users
	if targetUser.Role == "system" {
		httputil.RespondWithError(w, http.StatusForbidden, "Cannot modify system users")
		return
	}

	err = h.userRepo.UpdateDetails(r.Context(), userID, updateData.Username, updateData.Email, updateData.Role)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			httputil.RespondWithError(w, http.StatusConflict, err.Error())
			return
		}
		debug.Error("Failed to update user: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	debug.Info("Admin updated user details: %s", userID)
	httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{"message": "User updated successfully"},
	})
}

// DisableUser godoc
// @Summary Disable user account
// @Description Disables a user account, preventing login
// @Tags Admin Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param data body object{reason=string} true "Disable reason"
// @Success 200 {object} httputil.SuccessResponse{data=object{message=string}}
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /admin/users/{id}/disable [post]
// @Security ApiKeyAuth
func (h *UserHandler) DisableUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Check if target user is system user
	targetUser, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		debug.Error("Failed to get user: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Prevent modifying system users
	if targetUser.Role == "system" {
		httputil.RespondWithError(w, http.StatusForbidden, "Cannot disable system users")
		return
	}

	var disableData struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&disableData); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if disableData.Reason == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Reason is required")
		return
	}

	// Get admin ID from context
	adminID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get admin ID")
		return
	}

	err = h.userRepo.DisableAccount(r.Context(), userID, disableData.Reason, adminID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		debug.Error("Failed to disable user: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to disable user")
		return
	}

	debug.Info("Admin %s disabled user account: %s (reason: %s)", adminID, userID, disableData.Reason)
	httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{"message": "User account disabled successfully"},
	})
}

// EnableUser godoc
// @Summary Enable user account
// @Description Re-enables a previously disabled user account
// @Tags Admin Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} httputil.SuccessResponse{data=object{message=string}}
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /admin/users/{id}/enable [post]
// @Security ApiKeyAuth
func (h *UserHandler) EnableUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err = h.userRepo.EnableAccount(r.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		debug.Error("Failed to enable user: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to enable user")
		return
	}

	// Get admin ID from context
	adminID, _ := r.Context().Value("user_id").(uuid.UUID)
	debug.Info("Admin %s enabled user account: %s", adminID, userID)

	httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{"message": "User account enabled successfully"},
	})
}

// ResetUserPassword godoc
// @Summary Reset user password
// @Description Resets a user's password to a new value
// @Tags Admin Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param data body object{password=string,temporary=boolean} true "New password"
// @Success 200 {object} httputil.SuccessResponse{data=object{message=string,temporary_password=string}}
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /admin/users/{id}/reset-password [post]
// @Security ApiKeyAuth
func (h *UserHandler) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Check if target user is system user
	targetUser, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		debug.Error("Failed to get user: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Prevent modifying system users
	if targetUser.Role == "system" {
		httputil.RespondWithError(w, http.StatusForbidden, "Cannot reset password for system users")
		return
	}

	var resetData struct {
		Password  string `json:"password"`
		Temporary bool   `json:"temporary"`
	}

	if err := json.NewDecoder(r.Body).Decode(&resetData); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Generate temporary password if not provided
	if resetData.Password == "" && resetData.Temporary {
		resetData.Password = password.GenerateTemporaryPassword()
	}

	if resetData.Password == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Password is required")
		return
	}

	// Validate password strength if not temporary
	if !resetData.Temporary {
		// TODO: Use password validator from auth settings
		if len(resetData.Password) < 8 {
			httputil.RespondWithError(w, http.StatusBadRequest, "Password must be at least 8 characters long")
			return
		}
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(resetData.Password), bcrypt.DefaultCost)
	if err != nil {
		debug.Error("Failed to hash password: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	err = h.userRepo.ResetPassword(r.Context(), userID, string(hashedPassword))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		debug.Error("Failed to reset password: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to reset password")
		return
	}

	// Get admin ID from context
	adminID, _ := r.Context().Value("user_id").(uuid.UUID)
	debug.Info("Admin %s reset password for user: %s", adminID, userID)

	response := map[string]interface{}{
		"data": map[string]string{
			"message": "Password reset successfully",
		},
	}

	// Include temporary password in response if generated
	if resetData.Temporary {
		response["data"].(map[string]string)["temporary_password"] = resetData.Password
	}

	httputil.RespondWithJSON(w, http.StatusOK, response)
}

// DisableUserMFA godoc
// @Summary Disable user MFA
// @Description Disables multi-factor authentication for a user
// @Tags Admin Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} httputil.SuccessResponse{data=object{message=string}}
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /admin/users/{id}/disable-mfa [post]
// @Security ApiKeyAuth
func (h *UserHandler) DisableUserMFA(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err = h.userRepo.DisableMFA(r.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		debug.Error("Failed to disable MFA: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to disable MFA")
		return
	}

	// Get admin ID from context
	adminID, _ := r.Context().Value("user_id").(uuid.UUID)
	debug.Info("Admin %s disabled MFA for user: %s", adminID, userID)

	httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{"message": "MFA disabled successfully"},
	})
}

// UnlockUser godoc
// @Summary Unlock user account
// @Description Unlocks a user account that was locked due to failed login attempts
// @Tags Admin Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} httputil.SuccessResponse{data=object{message=string}}
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /admin/users/{id}/unlock [post]
// @Security ApiKeyAuth
func (h *UserHandler) UnlockUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err = h.userRepo.UnlockAccount(r.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		debug.Error("Failed to unlock user: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to unlock user")
		return
	}

	// Get admin ID from context
	adminID, _ := r.Context().Value("user_id").(uuid.UUID)
	debug.Info("Admin %s unlocked user account: %s", adminID, userID)

	httputil.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{"message": "User account unlocked successfully"},
	})
}
