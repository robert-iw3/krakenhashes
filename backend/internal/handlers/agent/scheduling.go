package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/gorilla/mux"
)

// SchedulingHandler handles agent scheduling endpoints
type SchedulingHandler struct {
	scheduleRepo *repository.AgentScheduleRepository
	agentRepo    *repository.AgentRepository
}

// NewSchedulingHandler creates a new scheduling handler
func NewSchedulingHandler(scheduleRepo *repository.AgentScheduleRepository, agentRepo *repository.AgentRepository) *SchedulingHandler {
	return &SchedulingHandler{
		scheduleRepo: scheduleRepo,
		agentRepo:    agentRepo,
	}
}

// GetAgentSchedules handles GET /api/agents/{id}/schedules
func (h *SchedulingHandler) GetAgentSchedules(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	// Check if agent exists
	agent, err := h.agentRepo.GetByID(r.Context(), agentID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		debug.Error("Failed to get agent: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get schedules
	schedules, err := h.scheduleRepo.GetSchedulesByAgent(r.Context(), agentID)
	if err != nil {
		debug.Error("Failed to get schedules: %v", err)
		http.Error(w, "Failed to get schedules", http.StatusInternalServerError)
		return
	}

	// Times are automatically formatted to HH:MM by TimeOnly's JSON marshaling

	// Return response
	response := map[string]interface{}{
		"agentId":           agentID,
		"schedulingEnabled": agent.SchedulingEnabled,
		"scheduleTimezone":  agent.ScheduleTimezone,
		"schedules":         schedules,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateAgentSchedule handles POST /api/agents/{id}/schedules
func (h *SchedulingHandler) UpdateAgentSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var dto models.AgentScheduleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate day of week
	if dto.DayOfWeek < 0 || dto.DayOfWeek > 6 {
		http.Error(w, "Invalid day of week", http.StatusBadRequest)
		return
	}

	// Parse times from DTO
	startTime, err := models.ParseTimeOnly(dto.StartTimeUTC)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid start time: %v", err), http.StatusBadRequest)
		return
	}
	endTime, err := models.ParseTimeOnly(dto.EndTimeUTC)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid end time: %v", err), http.StatusBadRequest)
		return
	}

	// Create schedule model
	schedule := &models.AgentSchedule{
		AgentID:   agentID,
		DayOfWeek: dto.DayOfWeek,
		StartTime: startTime,
		EndTime:   endTime,
		Timezone:  dto.Timezone,
		IsActive:  dto.IsActive,
	}

	// Validate schedule
	if err := schedule.ValidateSchedule(); err != nil {
		http.Error(w, "Invalid schedule", http.StatusBadRequest)
		return
	}

	// Update or create schedule
	if err := h.scheduleRepo.UpdateSchedule(r.Context(), schedule); err != nil {
		debug.Error("Failed to update schedule: %v", err)
		http.Error(w, "Failed to update schedule", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

// DeleteAgentSchedule handles DELETE /api/agents/{id}/schedules/{day}
func (h *SchedulingHandler) DeleteAgentSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	dayOfWeek, err := strconv.Atoi(vars["day"])
	if err != nil || dayOfWeek < 0 || dayOfWeek > 6 {
		http.Error(w, "Invalid day of week", http.StatusBadRequest)
		return
	}

	// Delete schedule
	if err := h.scheduleRepo.DeleteSchedule(r.Context(), agentID, dayOfWeek); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Schedule not found", http.StatusNotFound)
			return
		}
		debug.Error("Failed to delete schedule: %v", err)
		http.Error(w, "Failed to delete schedule", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ToggleAgentScheduling handles PUT /api/agents/{id}/scheduling-enabled
func (h *SchedulingHandler) ToggleAgentScheduling(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		Enabled  bool   `json:"enabled"`
		Timezone string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default timezone if not provided
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}

	// Update agent scheduling
	if err := h.scheduleRepo.UpdateAgentScheduling(r.Context(), agentID, req.Enabled, req.Timezone); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		debug.Error("Failed to update agent scheduling: %v", err)
		http.Error(w, "Failed to update agent scheduling", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agentId":           agentID,
		"schedulingEnabled": req.Enabled,
		"scheduleTimezone":  req.Timezone,
	})
}

// BulkUpdateSchedules handles POST /api/agents/{id}/schedules/bulk
func (h *SchedulingHandler) BulkUpdateSchedules(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		Schedules []models.AgentScheduleDTO `json:"schedules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update each schedule
	var updatedSchedules []models.AgentSchedule
	for _, dto := range req.Schedules {
		// Parse times from DTO
		startTime, err := models.ParseTimeOnly(dto.StartTimeUTC)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid start time for day %d: %v", dto.DayOfWeek, err), http.StatusBadRequest)
			return
		}
		endTime, err := models.ParseTimeOnly(dto.EndTimeUTC)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid end time for day %d: %v", dto.DayOfWeek, err), http.StatusBadRequest)
			return
		}

		schedule := &models.AgentSchedule{
			AgentID:   agentID,
			DayOfWeek: dto.DayOfWeek,
			StartTime: startTime,
			EndTime:   endTime,
			Timezone:  dto.Timezone,
			IsActive:  dto.IsActive,
		}

		if err := schedule.ValidateSchedule(); err != nil {
			http.Error(w, fmt.Sprintf("Invalid schedule for day %d", dto.DayOfWeek), http.StatusBadRequest)
			return
		}

		if err := h.scheduleRepo.UpdateSchedule(r.Context(), schedule); err != nil {
			debug.Error("Failed to update schedule for day %d: %v", dto.DayOfWeek, err)
			http.Error(w, "Failed to update schedules", http.StatusInternalServerError)
			return
		}
		
		updatedSchedules = append(updatedSchedules, *schedule)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agentId":   agentID,
		"schedules": updatedSchedules,
	})
}