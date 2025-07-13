package user

import (
	"encoding/json"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// Handler handles user-related HTTP requests
type Handler struct {
	userRepo *repository.UserRepository
}

// NewHandler creates a new user handler
func NewHandler(db *db.DB) *Handler {
	return &Handler{
		userRepo: repository.NewUserRepository(db),
	}
}

// ListUsers handles GET /api/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	debug.Info("Listing all users")

	// Get role from query parameter if provided
	filters := make(map[string]interface{})
	if role := r.URL.Query().Get("role"); role != "" {
		filters["role"] = role
		debug.Debug("Filtering users by role: %s", role)
	}

	// List users from repository
	users, err := h.userRepo.List(r.Context(), filters)
	if err != nil {
		debug.Error("Failed to list users: %v", err)
		http.Error(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}

	// Create response without sensitive information
	type UserResponse struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}

	var response []UserResponse
	for _, user := range users {
		response = append(response, UserResponse{
			ID:       user.ID.String(),
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		debug.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
