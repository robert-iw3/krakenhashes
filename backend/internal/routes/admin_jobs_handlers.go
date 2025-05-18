package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/httputil"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// AdminJobsHandler holds handlers for admin job-related operations.
type AdminJobsHandler struct {
	presetJobService services.AdminPresetJobService
	workflowService  services.AdminJobWorkflowService
}

// NewAdminJobsHandler creates a new handler for admin job routes.
func NewAdminJobsHandler(presetJobService services.AdminPresetJobService, workflowService services.AdminJobWorkflowService) *AdminJobsHandler {
	return &AdminJobsHandler{
		presetJobService: presetJobService,
		workflowService:  workflowService,
	}
}

// --- Preset Job Handlers ---

func (h *AdminJobsHandler) CreatePresetJob(w http.ResponseWriter, r *http.Request) {
	var job models.PresetJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	createdJob, err := h.presetJobService.CreatePresetJob(r.Context(), job)
	if err != nil {
		// Basic error handling, could check for specific validation errors
		debug.Error("Error creating preset job: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create preset job: %v", err))
		return
	}

	httputil.RespondWithJSON(w, http.StatusCreated, createdJob)
}

func (h *AdminJobsHandler) ListPresetJobs(w http.ResponseWriter, r *http.Request) {
	// TODO: Add pagination/sorting based on query params
	jobs, err := h.presetJobService.ListPresetJobs(r.Context())
	if err != nil {
		debug.Error("Error listing preset jobs: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to list preset jobs")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, jobs)
}

func (h *AdminJobsHandler) GetPresetJobFormData(w http.ResponseWriter, r *http.Request) {
	formData, err := h.presetJobService.GetPresetJobFormData(r.Context())
	if err != nil {
		debug.Error("Error getting preset job form data: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get form data")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, formData)
}

func (h *AdminJobsHandler) GetPresetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["preset_job_id"]
	if !ok {
		httputil.RespondWithError(w, http.StatusBadRequest, "Missing preset job ID")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid preset job ID format")
		return
	}

	job, err := h.presetJobService.GetPresetJobByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			httputil.RespondWithError(w, http.StatusNotFound, "Preset job not found")
		} else {
			debug.Error("Error getting preset job %s: %v", id, err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get preset job")
		}
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, job)
}

func (h *AdminJobsHandler) UpdatePresetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["preset_job_id"]
	if !ok {
		httputil.RespondWithError(w, http.StatusBadRequest, "Missing preset job ID")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid preset job ID format")
		return
	}

	var job models.PresetJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	updatedJob, err := h.presetJobService.UpdatePresetJob(r.Context(), id, job)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			httputil.RespondWithError(w, http.StatusNotFound, "Preset job not found")
		} else {
			debug.Error("Error updating preset job %s: %v", id, err)
			httputil.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update preset job: %v", err))
		}
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, updatedJob)
}

func (h *AdminJobsHandler) DeletePresetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["preset_job_id"]
	if !ok {
		httputil.RespondWithError(w, http.StatusBadRequest, "Missing preset job ID")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid preset job ID format")
		return
	}

	err = h.presetJobService.DeletePresetJob(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			httputil.RespondWithError(w, http.StatusNotFound, "Preset job not found")
		} else {
			debug.Error("Error deleting preset job %s: %v", id, err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to delete preset job")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Job Workflow Handlers ---

// Define request/response structs specific to handlers if needed
type CreateWorkflowRequest struct {
	Name         string      `json:"name"`
	PresetJobIDs []uuid.UUID `json:"preset_job_ids"`
}

type UpdateWorkflowRequest struct {
	Name         string      `json:"name"`
	PresetJobIDs []uuid.UUID `json:"preset_job_ids"`
}

func (h *AdminJobsHandler) CreateJobWorkflow(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	createdWorkflow, err := h.workflowService.CreateJobWorkflow(r.Context(), req.Name, req.PresetJobIDs)
	if err != nil {
		debug.Error("Error creating job workflow: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create job workflow: %v", err))
		return
	}

	httputil.RespondWithJSON(w, http.StatusCreated, createdWorkflow)
}

func (h *AdminJobsHandler) ListJobWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows, err := h.workflowService.ListJobWorkflows(r.Context())
	if err != nil {
		debug.Error("Error listing job workflows: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to list job workflows")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, workflows)
}

func (h *AdminJobsHandler) GetJobWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["job_workflow_id"]
	if !ok {
		httputil.RespondWithError(w, http.StatusBadRequest, "Missing job workflow ID")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid job workflow ID format")
		return
	}

	workflow, err := h.workflowService.GetJobWorkflowByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			httputil.RespondWithError(w, http.StatusNotFound, "Job workflow not found")
		} else {
			debug.Error("Error getting job workflow %s: %v", id, err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get job workflow")
		}
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, workflow)
}

func (h *AdminJobsHandler) UpdateJobWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["job_workflow_id"]
	if !ok {
		httputil.RespondWithError(w, http.StatusBadRequest, "Missing job workflow ID")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid job workflow ID format")
		return
	}

	var req UpdateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	updatedWorkflow, err := h.workflowService.UpdateJobWorkflow(r.Context(), id, req.Name, req.PresetJobIDs)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			httputil.RespondWithError(w, http.StatusNotFound, "Job workflow not found")
		} else {
			debug.Error("Error updating job workflow %s: %v", id, err)
			httputil.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update job workflow: %v", err))
		}
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, updatedWorkflow)
}

func (h *AdminJobsHandler) DeleteJobWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["job_workflow_id"]
	if !ok {
		httputil.RespondWithError(w, http.StatusBadRequest, "Missing job workflow ID")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid job workflow ID format")
		return
	}

	err = h.workflowService.DeleteJobWorkflow(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			httputil.RespondWithError(w, http.StatusNotFound, "Job workflow not found")
		} else {
			debug.Error("Error deleting job workflow %s: %v", id, err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to delete job workflow")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminJobsHandler) GetJobWorkflowFormData(w http.ResponseWriter, r *http.Request) {
	presetJobs, err := h.workflowService.GetJobWorkflowFormData(r.Context())
	if err != nil {
		debug.Error("Error getting job workflow form data: %v", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to get workflow form data")
		return
	}

	// Create a response structure with the preset jobs for selection
	response := struct {
		PresetJobs []models.PresetJobBasic `json:"preset_jobs"`
	}{
		PresetJobs: presetJobs,
	}

	httputil.RespondWithJSON(w, http.StatusOK, response)
}
