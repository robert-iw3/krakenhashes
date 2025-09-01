package user

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// NotificationPreferencesHandler handles user notification preference operations
type NotificationPreferencesHandler struct {
	notificationService *services.NotificationService
}

// NewNotificationPreferencesHandler creates a new notification preferences handler
func NewNotificationPreferencesHandler(db *sql.DB) *NotificationPreferencesHandler {
	return &NotificationPreferencesHandler{
		notificationService: services.NewNotificationService(db),
	}
}

// GetNotificationPreferences retrieves the current user's notification preferences
func (h *NotificationPreferencesHandler) GetNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		debug.Error("Failed to get user ID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse user ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		debug.Error("Invalid user ID format: %v", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get notification preferences
	prefs, err := h.notificationService.GetUserNotificationPreferences(r.Context(), uid)
	if err != nil {
		debug.Error("Failed to get notification preferences for user %s: %v", userID, err)
		http.Error(w, "Failed to get notification preferences", http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(prefs); err != nil {
		debug.Error("Failed to encode notification preferences response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// UpdateNotificationPreferences updates the current user's notification preferences
func (h *NotificationPreferencesHandler) UpdateNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		debug.Error("Failed to get user ID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse user ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		debug.Error("Invalid user ID format: %v", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var prefs models.NotificationPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		debug.Error("Failed to decode notification preferences request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update notification preferences
	if err := h.notificationService.UpdateUserNotificationPreferences(r.Context(), uid, &prefs); err != nil {
		// Check if it's an email configuration error
		if err.Error() == "email notifications require an email gateway to be configured" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Email notifications require an email gateway to be configured. Please contact your administrator.",
				"code":  "EMAIL_GATEWAY_REQUIRED",
			})
			return
		}
		
		debug.Error("Failed to update notification preferences for user %s: %v", userID, err)
		http.Error(w, "Failed to update notification preferences", http.StatusInternalServerError)
		return
	}

	// Get updated preferences to return
	updatedPrefs, err := h.notificationService.GetUserNotificationPreferences(r.Context(), uid)
	if err != nil {
		debug.Error("Failed to get updated notification preferences: %v", err)
		http.Error(w, "Failed to get updated preferences", http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedPrefs); err != nil {
		debug.Error("Failed to encode notification preferences response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// RegisterRoutes registers the notification preference routes
func (h *NotificationPreferencesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/user/notification-preferences", h.GetNotificationPreferences).Methods("GET")
	router.HandleFunc("/api/user/notification-preferences", h.UpdateNotificationPreferences).Methods("PUT")
}