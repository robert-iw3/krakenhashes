package analytics

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles analytics-related requests
type Handler struct {
	db           *db.DB
	repo         *repository.AnalyticsRepository
	service      *services.AnalyticsService
	queueService *services.AnalyticsQueueService
}

// NewHandler creates a new analytics handler
func NewHandler(database *db.DB, queueService *services.AnalyticsQueueService) *Handler {
	repo := repository.NewAnalyticsRepository(database)
	service := services.NewAnalyticsService(repo)

	return &Handler{
		db:           database,
		repo:         repo,
		service:      service,
		queueService: queueService,
	}
}

// CreateReport creates a new analytics report and queues it for processing
// POST /api/analytics/reports
func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAnalyticsReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debug.Error("Failed to decode request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate date range
	if req.EndDate.Before(req.StartDate) {
		http.Error(w, "End date must be after start date", http.StatusBadRequest)
		return
	}

	// Get user ID from context (set by auth middleware)
	userIDStr, ok := r.Context().Value("user_id").(string)
	if !ok {
		debug.Error("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		debug.Error("Invalid user ID format: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get next queue position
	queuePos, err := h.repo.GetNextQueuePosition(r.Context())
	if err != nil {
		debug.Error("Failed to get next queue position: %v", err)
		http.Error(w, "Failed to queue report", http.StatusInternalServerError)
		return
	}

	// Create report
	report := &models.AnalyticsReport{
		ID:             uuid.New(),
		ClientID:       req.ClientID,
		UserID:         userID,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		Status:         "queued",
		CustomPatterns: req.CustomPatterns,
		QueuePosition:  &queuePos,
		CreatedAt:      time.Now(),
	}

	if err := h.repo.Create(r.Context(), report); err != nil {
		debug.Error("Failed to create analytics report: %v", err)
		http.Error(w, "Failed to create report", http.StatusInternalServerError)
		return
	}

	debug.Info("Created analytics report %s for client %s (queue position: %d)", report.ID, report.ClientID, queuePos)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(report)
}

// GetReport retrieves an analytics report by ID
// GET /api/analytics/reports/{id}
func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reportID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid report ID", http.StatusBadRequest)
		return
	}

	report, err := h.repo.GetByID(r.Context(), reportID)
	if err != nil {
		debug.Error("Failed to get report: %v", err)
		if err.Error() == "not found" {
			http.Error(w, "Report not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve report", http.StatusInternalServerError)
		return
	}

	// Return different responses based on status
	response := map[string]interface{}{
		"report": report,
	}

	switch report.Status {
	case "queued":
		response["status"] = "queued"
		response["message"] = "Pending report generation"
	case "processing":
		response["status"] = "processing"
		response["message"] = "Report is still generating"
	case "failed":
		response["status"] = "failed"
		response["message"] = "Report generation failed"
	case "completed":
		response["status"] = "completed"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetClientReports retrieves all reports for a specific client
// GET /api/analytics/reports/client/{clientId}
func (h *Handler) GetClientReports(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID, err := uuid.Parse(vars["clientId"])
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	reports, err := h.repo.GetByClient(r.Context(), clientID)
	if err != nil {
		debug.Error("Failed to get reports for client: %v", err)
		http.Error(w, "Failed to retrieve reports", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

// DeleteReport deletes an analytics report
// DELETE /api/analytics/reports/{id}
func (h *Handler) DeleteReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reportID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid report ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), reportID); err != nil {
		debug.Error("Failed to delete report: %v", err)
		if err.Error() == "not found" {
			http.Error(w, "Report not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete report", http.StatusInternalServerError)
		return
	}

	debug.Info("Deleted analytics report %s", reportID)

	w.WriteHeader(http.StatusNoContent)
}

// RetryReport retries a failed analytics report
// POST /api/analytics/reports/{id}/retry
func (h *Handler) RetryReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reportID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid report ID", http.StatusBadRequest)
		return
	}

	// Get the report
	report, err := h.repo.GetByID(r.Context(), reportID)
	if err != nil {
		debug.Error("Failed to get report: %v", err)
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}

	// Check if report is in failed state
	if report.Status != "failed" {
		http.Error(w, "Can only retry failed reports", http.StatusBadRequest)
		return
	}

	// Get next queue position
	queuePos, err := h.repo.GetNextQueuePosition(r.Context())
	if err != nil {
		debug.Error("Failed to get next queue position: %v", err)
		http.Error(w, "Failed to queue report", http.StatusInternalServerError)
		return
	}

	// Reset report to queued status
	report.Status = "queued"
	report.ErrorMessage = nil
	report.QueuePosition = &queuePos
	report.StartedAt = nil
	report.CompletedAt = nil

	// Update the report (need to add this method to repository)
	if err := h.repo.UpdateStatus(r.Context(), reportID, "queued"); err != nil {
		debug.Error("Failed to update report status: %v", err)
		http.Error(w, "Failed to retry report", http.StatusInternalServerError)
		return
	}

	// Also need to update queue position
	// For now, let's just update the status and the queue service will handle it

	debug.Info("Retrying analytics report %s (queue position: %d)", reportID, queuePos)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetQueueStatus retrieves the current queue status
// GET /api/analytics/queue-status
func (h *Handler) GetQueueStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.queueService.GetQueueStatus(r.Context())
	if err != nil {
		debug.Error("Failed to get queue status: %v", err)
		http.Error(w, "Failed to get queue status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetClients retrieves all clients for analytics dropdown
// GET /api/analytics/clients
func (h *Handler) GetClients(w http.ResponseWriter, r *http.Request) {
	clientRepo := repository.NewClientRepository(h.db)
	clients, err := clientRepo.List(r.Context())
	if err != nil {
		debug.Error("Failed to list clients: %v", err)
		http.Error(w, "Failed to retrieve clients", http.StatusInternalServerError)
		return
	}

	// Return simple array of clients with just id and name
	type ClientSummary struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	summaries := make([]ClientSummary, len(clients))
	for i, client := range clients {
		summaries[i] = ClientSummary{
			ID:   client.ID.String(),
			Name: client.Name,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}
