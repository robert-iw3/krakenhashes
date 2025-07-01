package jobs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/binary"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/rule"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/wordlist"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// UserJobsHandler handles job-related requests from users
type UserJobsHandler struct {
	jobExecRepo         *repository.JobExecutionRepository
	jobTaskRepo         *repository.JobTaskRepository
	presetJobRepo       repository.PresetJobRepository
	hashlistRepo        *repository.HashListRepository
	clientRepo          *repository.ClientRepository
	workflowRepo        repository.JobWorkflowRepository
	wordlistStore       *wordlist.Store
	ruleStore           *rule.Store
	binaryStore         binary.Store
	jobExecutionService *services.JobExecutionService
}

// NewUserJobsHandler creates a new user jobs handler
func NewUserJobsHandler(
	jobExecRepo *repository.JobExecutionRepository,
	jobTaskRepo *repository.JobTaskRepository,
	presetJobRepo repository.PresetJobRepository,
	hashlistRepo *repository.HashListRepository,
	clientRepo *repository.ClientRepository,
	workflowRepo repository.JobWorkflowRepository,
	wordlistStore *wordlist.Store,
	ruleStore *rule.Store,
	binaryStore binary.Store,
	jobExecutionService *services.JobExecutionService,
) *UserJobsHandler {
	return &UserJobsHandler{
		jobExecRepo:         jobExecRepo,
		jobTaskRepo:         jobTaskRepo,
		presetJobRepo:       presetJobRepo,
		hashlistRepo:        hashlistRepo,
		clientRepo:          clientRepo,
		workflowRepo:        workflowRepo,
		wordlistStore:       wordlistStore,
		ruleStore:           ruleStore,
		binaryStore:         binaryStore,
		jobExecutionService: jobExecutionService,
	}
}

// JobSummary represents a job summary for the UI
type JobSummary struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	HashlistID        int64   `json:"hashlist_id"`
	HashlistName      string  `json:"hashlist_name"`
	Status            string  `json:"status"`
	Priority          int     `json:"priority"`
	MaxAgents         int     `json:"max_agents"`
	DispatchedPercent float64 `json:"dispatched_percent"`
	SearchedPercent   float64 `json:"searched_percent"`
	CrackedCount      int     `json:"cracked_count"`
	AgentCount        int     `json:"agent_count"`
	TotalSpeed        int64   `json:"total_speed"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	ErrorMessage      *string `json:"error_message,omitempty"`
}

// ListJobs handles GET /api/jobs with pagination and filtering
func (h *UserJobsHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}
	
	// Parse filters
	status := r.URL.Query().Get("status")
	priorityStr := r.URL.Query().Get("priority")
	search := r.URL.Query().Get("search")
	
	var priority *int
	if priorityStr != "" {
		p, err := strconv.Atoi(priorityStr)
		if err == nil && p >= 1 && p <= 10 {
			priority = &p
		}
	}
	
	// Create filter
	filter := repository.JobFilter{
		Status:   &status,
		Priority: priority,
		Search:   &search,
	}
	
	// Get jobs with filters
	jobs, err := h.jobExecRepo.ListWithFilters(ctx, pageSize, (page-1)*pageSize, filter)
	if err != nil {
		debug.Error("Failed to list jobs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Get total count with filters
	total, err := h.jobExecRepo.GetFilteredCount(ctx, filter)
	if err != nil {
		debug.Error("Failed to get job count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Get status counts
	statusCounts, err := h.jobExecRepo.GetStatusCounts(ctx)
	if err != nil {
		debug.Error("Failed to get status counts: %v", err)
		// Don't fail the request, just log the error
		statusCounts = make(map[string]int)
	}
	
	// Convert to job summaries
	summaries := make([]JobSummary, 0, len(jobs))
	for _, job := range jobs {
		// Get hashlist details including cracked count
		hashlist, err := h.hashlistRepo.GetByID(ctx, job.HashlistID)
		if err != nil {
			debug.Error("Failed to get hashlist %d: %v", job.HashlistID, err)
			continue
		}
		
		// Get task statistics
		tasks, err := h.jobTaskRepo.GetTasksByJobExecution(ctx, job.ID)
		if err != nil {
			debug.Error("Failed to get tasks for job %s: %v", job.ID, err)
			tasks = []models.JobTask{}
		}
		
		// Calculate metrics
		var agentCount int
		var totalSpeed int64
		var crackedCount int
		var keyspaceSearched int64
		var keyspaceDispatched int64
		
		for _, task := range tasks {
			if task.Status == models.JobTaskStatusRunning {
				agentCount++
				if task.BenchmarkSpeed != nil {
					totalSpeed += *task.BenchmarkSpeed
				}
			}
			crackedCount += task.CrackCount
			keyspaceSearched += task.KeyspaceProcessed
			
			// Calculate dispatched keyspace (assigned to tasks)
			if task.Status != models.JobTaskStatusPending {
				// Task has been dispatched if it's not pending
				taskKeyspace := task.KeyspaceEnd - task.KeyspaceStart
				keyspaceDispatched += taskKeyspace
			}
		}
		
		// Calculate percentages
		dispatchedPercent := 0.0
		searchedPercent := 0.0
		if job.TotalKeyspace != nil && *job.TotalKeyspace > 0 {
			dispatchedPercent = float64(keyspaceDispatched) / float64(*job.TotalKeyspace) * 100
			searchedPercent = float64(keyspaceSearched) / float64(*job.TotalKeyspace) * 100
			
			// Cap percentages at 100%
			if dispatchedPercent > 100 {
				dispatchedPercent = 100
			}
			if searchedPercent > 100 {
				searchedPercent = 100
			}
		}
		
		summary := JobSummary{
			ID:                job.ID.String(),
			Name:              getJobName(job, hashlist),
			HashlistID:        job.HashlistID,
			HashlistName:      hashlist.Name,
			Status:            string(job.Status),
			Priority:          job.Priority,
			MaxAgents:         job.MaxAgents,
			DispatchedPercent: dispatchedPercent,
			SearchedPercent:   searchedPercent,
			CrackedCount:      crackedCount,
			AgentCount:        agentCount,
			TotalSpeed:        totalSpeed,
			CreatedAt:         job.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         job.UpdatedAt.Format(time.RFC3339),
		}
		
		// Add error message if present
		if job.ErrorMessage != nil && *job.ErrorMessage != "" {
			summary.ErrorMessage = job.ErrorMessage
		}
		
		summaries = append(summaries, summary)
	}
	
	// Prepare response
	response := map[string]interface{}{
		"jobs": summaries,
		"pagination": map[string]interface{}{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
		"status_counts": statusCounts,
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		debug.Error("Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// getJobName generates a display name for a job
func getJobName(job models.JobExecution, hashlist *models.HashList) string {
	// Just use the hashlist name as the job name
	return hashlist.Name
}

// CreateJobFromHashlist handles POST /api/hashlists/{id}/create-job
func (h *UserJobsHandler) CreateJobFromHashlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	
	hashlistID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid hashlist ID", http.StatusBadRequest)
		return
	}
	
	// Parse the request body to determine job type
	var rawReq json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Determine the job type
	var jobType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rawReq, &jobType); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	
	// Verify the hashlist exists
	_, err = h.hashlistRepo.GetByID(ctx, hashlistID)
	if err != nil {
		debug.Error("Failed to get hashlist %d: %v", hashlistID, err)
		http.Error(w, "Hashlist not found", http.StatusNotFound)
		return
	}
	
	var createdJobs []string
	
	switch jobType.Type {
	case "preset":
		// Handle preset jobs
		var req struct {
			Type         string   `json:"type"`
			PresetJobIDs []string `json:"preset_job_ids"`
		}
		if err := json.Unmarshal(rawReq, &req); err != nil {
			http.Error(w, "Invalid preset job request", http.StatusBadRequest)
			return
		}
		
		// Create a job execution for each selected preset job
		for _, presetJobIDStr := range req.PresetJobIDs {
			presetJobID, err := uuid.Parse(presetJobIDStr)
			if err != nil {
				debug.Error("Invalid preset job ID: %s", presetJobIDStr)
				continue
			}
			
			// Get the preset job to verify it exists
			_, err = h.presetJobRepo.GetByID(ctx, presetJobID)
			if err != nil {
				debug.Error("Failed to get preset job %s: %v", presetJobID, err)
				continue
			}
			
			// Use CreateJobExecution to create job with keyspace calculation
			jobExecution, err := h.jobExecutionService.CreateJobExecution(ctx, presetJobID, hashlistID)
			if err != nil {
				debug.Error("Failed to create job execution for preset %s: %v", presetJobID, err)
				continue
			}
			
			createdJobs = append(createdJobs, jobExecution.ID.String())
		}
		
	case "workflow":
		// Handle workflows
		var req struct {
			Type        string   `json:"type"`
			WorkflowIDs []string `json:"workflow_ids"`
		}
		if err := json.Unmarshal(rawReq, &req); err != nil {
			http.Error(w, "Invalid workflow request", http.StatusBadRequest)
			return
		}
		
		// For each workflow, create jobs for all its steps
		for _, workflowIDStr := range req.WorkflowIDs {
			workflowID, err := uuid.Parse(workflowIDStr)
			if err != nil {
				debug.Error("Invalid workflow ID: %s", workflowIDStr)
				continue
			}
			
			// Get workflow steps
			steps, err := h.workflowRepo.GetWorkflowSteps(ctx, workflowID)
			if err != nil {
				debug.Error("Failed to get workflow steps for %s: %v", workflowID, err)
				continue
			}
			
			// Create a job for each step in order
			for _, step := range steps {
				// Verify the preset job exists
				_, err := h.presetJobRepo.GetByID(ctx, step.PresetJobID)
				if err != nil {
					debug.Error("Failed to get preset job %s for workflow step: %v", step.PresetJobID, err)
					continue
				}
				
				// Use CreateJobExecution to create job with keyspace calculation
				jobExecution, err := h.jobExecutionService.CreateJobExecution(ctx, step.PresetJobID, hashlistID)
				if err != nil {
					debug.Error("Failed to create job execution for workflow step: %v", err)
					continue
				}
				
				createdJobs = append(createdJobs, jobExecution.ID.String())
			}
		}
		
	case "custom":
		// Handle custom job
		var req struct {
			Type      string `json:"type"`
			CustomJob struct {
				Name             string   `json:"name"`
				AttackMode       int      `json:"attack_mode"`
				WordlistIDs      []string `json:"wordlist_ids"`
				RuleIDs          []string `json:"rule_ids"`
				Mask             string   `json:"mask"`
				Priority         int      `json:"priority"`
				MaxAgents        int      `json:"max_agents"`
				BinaryVersionID  int      `json:"binary_version_id"`
			} `json:"custom_job"`
		}
		if err := json.Unmarshal(rawReq, &req); err != nil {
			http.Error(w, "Invalid custom job request", http.StatusBadRequest)
			return
		}
		
		// Create a custom preset job first
		customPresetJob := models.PresetJob{
			ID:              uuid.New(),
			Name:            req.CustomJob.Name,
			AttackMode:      models.AttackMode(req.CustomJob.AttackMode),
			Priority:        req.CustomJob.Priority,
			Mask:            req.CustomJob.Mask,
			BinaryVersionID: req.CustomJob.BinaryVersionID,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		
		// WordlistIDs and RuleIDs are already strings, just assign them
		customPresetJob.WordlistIDs = models.IDArray(req.CustomJob.WordlistIDs)
		customPresetJob.RuleIDs = models.IDArray(req.CustomJob.RuleIDs)
		
		// Save the custom preset job
		_, err = h.presetJobRepo.Create(ctx, customPresetJob)
		if err != nil {
			debug.Error("Failed to create custom preset job: %v", err)
			http.Error(w, "Failed to create custom job", http.StatusInternalServerError)
			return
		}
		
		// Use CreateJobExecution to create job with keyspace calculation
		jobExecution, err := h.jobExecutionService.CreateJobExecution(ctx, customPresetJob.ID, hashlistID)
		if err != nil {
			debug.Error("Failed to create job execution: %v", err)
			http.Error(w, "Failed to create job", http.StatusInternalServerError)
			return
		}
		
		// Update max agents if specified (CreateJobExecution doesn't set this)
		if req.CustomJob.MaxAgents > 0 {
			if err := h.jobExecRepo.UpdateMaxAgents(ctx, jobExecution.ID, req.CustomJob.MaxAgents); err != nil {
				debug.Error("Failed to update max agents for job: %v", err)
				// Don't fail the request, just log the error
			}
		}
		
		createdJobs = append(createdJobs, jobExecution.ID.String())
		
	default:
		http.Error(w, "Invalid job type", http.StatusBadRequest)
		return
	}
	
	if len(createdJobs) == 0 {
		http.Error(w, "No jobs were created", http.StatusInternalServerError)
		return
	}
	
	// Return the created jobs
	response := map[string]interface{}{
		"ids":     createdJobs,
		"message": fmt.Sprintf("%d job(s) created successfully", len(createdJobs)),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetJobDetail handles GET /api/jobs/{id}
func (h *UserJobsHandler) GetJobDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	
	jobID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}
	
	// Get job execution
	job, err := h.jobExecRepo.GetByID(ctx, jobID)
	if err != nil {
		debug.Error("Failed to get job %s: %v", jobID, err)
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}
	
	// Get hashlist
	hashlist, err := h.hashlistRepo.GetByID(ctx, job.HashlistID)
	if err != nil {
		debug.Error("Failed to get hashlist %d: %v", job.HashlistID, err)
		http.Error(w, "Hashlist not found", http.StatusNotFound)
		return
	}
	
	// Get tasks with pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 50
	offset := (page - 1) * pageSize
	
	tasks, err := h.jobTaskRepo.GetTasksByJobExecutionWithPagination(ctx, jobID, pageSize, offset)
	if err != nil {
		debug.Error("Failed to get tasks for job %s: %v", jobID, err)
		tasks = []models.JobTask{}
	}
	
	// Get total task count
	totalTasks, err := h.jobTaskRepo.GetTaskCountByJobExecution(ctx, jobID)
	if err != nil {
		debug.Error("Failed to get task count for job %s: %v", jobID, err)
		totalTasks = 0
	}
	
	// Calculate metrics
	var agentCount int
	var totalSpeed int64
	var crackedCount int
	var keyspaceSearched int64
	activeAgents := make(map[int]bool)
	
	for _, task := range tasks {
		if task.Status == models.JobTaskStatusRunning {
			activeAgents[task.AgentID] = true
			if task.BenchmarkSpeed != nil {
				totalSpeed += *task.BenchmarkSpeed
			}
		}
		crackedCount += task.CrackCount
		keyspaceSearched += task.KeyspaceProcessed
	}
	agentCount = len(activeAgents)
	
	// Calculate percentages
	dispatchedPercent := 0.0
	searchedPercent := 0.0
	if job.TotalKeyspace != nil && *job.TotalKeyspace > 0 {
		dispatchedPercent = float64(job.ProcessedKeyspace) / float64(*job.TotalKeyspace) * 100
		searchedPercent = float64(keyspaceSearched) / float64(*job.TotalKeyspace) * 100
	}
	
	// Prepare task summaries
	taskSummaries := make([]map[string]interface{}, 0, len(tasks))
	for _, task := range tasks {
		taskSummary := map[string]interface{}{
			"id":                 task.ID.String(),
			"agent_id":           task.AgentID,
			"status":             string(task.Status),
			"keyspace_start":     task.KeyspaceStart,
			"keyspace_end":       task.KeyspaceEnd,
			"keyspace_processed": task.KeyspaceProcessed,
			"crack_count":        task.CrackCount,
			"detailed_status":    task.DetailedStatus,
			"error_message":      task.ErrorMessage,
			"created_at":         task.CreatedAt.Format(time.RFC3339),
		}
		
		if task.StartedAt != nil {
			taskSummary["started_at"] = task.StartedAt.Format(time.RFC3339)
		}
		if task.CompletedAt != nil {
			taskSummary["completed_at"] = task.CompletedAt.Format(time.RFC3339)
		}
		if task.BenchmarkSpeed != nil {
			taskSummary["benchmark_speed"] = *task.BenchmarkSpeed
		}
		
		taskSummaries = append(taskSummaries, taskSummary)
	}
	
	// Prepare response
	response := map[string]interface{}{
		"id":                jobID.String(),
		"name":              getJobName(*job, hashlist),
		"hashlist_id":       job.HashlistID,
		"hashlist_name":     hashlist.Name,
		"status":            string(job.Status),
		"priority":          job.Priority,
		"max_agents":        job.MaxAgents,
		"attack_mode":       job.AttackMode,
		"total_keyspace":    job.TotalKeyspace,
		"dispatched_percent": dispatchedPercent,
		"searched_percent":   searchedPercent,
		"cracked_count":     crackedCount,
		"agent_count":       agentCount,
		"total_speed":       totalSpeed,
		"created_at":        job.CreatedAt.Format(time.RFC3339),
		"updated_at":        job.UpdatedAt.Format(time.RFC3339),
		"tasks":             taskSummaries,
		"task_pagination": map[string]interface{}{
			"page":        page,
			"page_size":   pageSize,
			"total":       totalTasks,
			"total_pages": (totalTasks + pageSize - 1) / pageSize,
		},
	}
	
	if job.StartedAt != nil {
		response["started_at"] = job.StartedAt.Format(time.RFC3339)
	}
	if job.CompletedAt != nil {
		response["completed_at"] = job.CompletedAt.Format(time.RFC3339)
	}
	if job.ErrorMessage != nil {
		response["error_message"] = *job.ErrorMessage
	}
	
	// Add preset job details if available
	if job.PresetJobID != uuid.Nil {
		presetJob, err := h.presetJobRepo.GetByID(ctx, job.PresetJobID)
		if err == nil {
			response["preset_job"] = map[string]interface{}{
				"id":   presetJob.ID.String(),
				"name": presetJob.Name,
			}
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		debug.Error("Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// UpdateJob handles PATCH /api/jobs/{id}
func (h *UserJobsHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	
	jobID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}
	
	var update struct {
		Priority  *int `json:"priority"`
		MaxAgents *int `json:"max_agents"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Update job execution
	if update.Priority != nil {
		if err := h.jobExecRepo.UpdatePriority(ctx, jobID, *update.Priority); err != nil {
			debug.Error("Failed to update job priority: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	
	if update.MaxAgents != nil {
		if err := h.jobExecRepo.UpdateMaxAgents(ctx, jobID, *update.MaxAgents); err != nil {
			debug.Error("Failed to update job max agents: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Job updated successfully",
	})
}

// RetryJob handles POST /api/jobs/{id}/retry
func (h *UserJobsHandler) RetryJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	
	jobID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}
	
	// Get the job
	job, err := h.jobExecRepo.GetByID(ctx, jobID)
	if err != nil {
		debug.Error("Failed to get job %s: %v", jobID, err)
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}
	
	// Check if job can be retried
	if job.Status != models.JobExecutionStatusFailed && 
	   job.Status != models.JobExecutionStatusInterrupted && 
	   job.Status != models.JobExecutionStatusCancelled {
		http.Error(w, "Job can only be retried if it's failed, interrupted, or cancelled", http.StatusBadRequest)
		return
	}
	
	// Reset the job to pending status
	if err := h.jobExecRepo.UpdateStatus(ctx, jobID, models.JobExecutionStatusPending); err != nil {
		debug.Error("Failed to reset job status: %v", err)
		http.Error(w, "Failed to retry job", http.StatusInternalServerError)
		return
	}
	
	// Clear error message
	if err := h.jobExecRepo.ClearError(ctx, jobID); err != nil {
		debug.Error("Failed to clear job error: %v", err)
		// Don't fail the request, just log the error
	}
	
	// Mark failed/cancelled tasks as pending so they can be retried
	tasks, err := h.jobTaskRepo.GetTasksByJobExecution(ctx, jobID)
	if err == nil {
		for _, task := range tasks {
			if task.Status == models.JobTaskStatusFailed || task.Status == models.JobTaskStatusCancelled {
				if err := h.jobTaskRepo.UpdateStatus(ctx, task.ID, models.JobTaskStatusPending); err != nil {
					debug.Error("Failed to reset task %s status: %v", task.ID, err)
				}
			}
		}
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Job retry initiated successfully",
	})
}

// DeleteJob handles DELETE /api/jobs/{id}
func (h *UserJobsHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	
	jobID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}
	
	// TODO: Implement job deletion with proper cleanup
	// This should:
	// 1. Stop all running tasks
	// 2. Delete task records
	// 3. Delete job execution record
	// 4. Notify agents to stop work
	
	if err := h.jobExecRepo.Delete(ctx, jobID); err != nil {
		debug.Error("Failed to delete job %s: %v", jobID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Job deleted successfully",
	})
}

// DeleteFinishedJobs handles DELETE /api/jobs/finished
func (h *UserJobsHandler) DeleteFinishedJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Delete all completed jobs
	deletedCount, err := h.jobExecRepo.DeleteFinished(ctx)
	if err != nil {
		debug.Error("Failed to delete finished jobs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Finished jobs deleted successfully",
		"deleted_count": deletedCount,
	})
}

// GetAvailablePresetJobs handles GET /api/hashlists/{id}/available-jobs
func (h *UserJobsHandler) GetAvailablePresetJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	
	hashlistID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid hashlist ID", http.StatusBadRequest)
		return
	}
	
	// Verify the hashlist exists
	_, err = h.hashlistRepo.GetByID(ctx, hashlistID)
	if err != nil {
		debug.Error("Failed to get hashlist %d: %v", hashlistID, err)
		http.Error(w, "Hashlist not found", http.StatusNotFound)
		return
	}
	
	// Get all preset jobs
	presetJobs, err := h.presetJobRepo.List(ctx)
	if err != nil {
		debug.Error("Failed to list preset jobs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Get all workflows
	workflows, err := h.workflowRepo.ListWorkflows(ctx)
	if err != nil {
		debug.Error("Failed to list workflows: %v", err)
		// Don't fail, just log and continue with empty workflows
		workflows = []models.JobWorkflow{}
	}
	
	// Get wordlists, rules, and binary versions for form data
	wordlists, err := h.wordlistStore.ListWordlists(ctx, map[string]interface{}{})
	if err != nil {
		debug.Error("Failed to list wordlists: %v", err)
		wordlists = []*models.Wordlist{}
	}
	
	rules, err := h.ruleStore.ListRules(ctx, nil)
	if err != nil {
		debug.Error("Failed to list rules: %v", err)
		rules = []*models.Rule{}
	}
	
	binaries, err := h.binaryStore.ListVersions(ctx, map[string]interface{}{"is_active": true})
	if err != nil {
		debug.Error("Failed to list binaries: %v", err)
		binaries = []*binary.BinaryVersion{}
	}
	
	// Format preset jobs
	availableJobs := make([]map[string]interface{}, 0)
	for _, job := range presetJobs {
		availableJobs = append(availableJobs, map[string]interface{}{
			"id":          job.ID.String(),
			"name":        job.Name,
			"priority":    job.Priority,
			"attack_mode": job.AttackMode,
			"wordlist_ids": job.WordlistIDs,
			"rule_ids":    job.RuleIDs,
			"mask":        job.Mask,
		})
	}
	
	// Format workflows with their steps
	formattedWorkflows := make([]map[string]interface{}, 0)
	for _, workflow := range workflows {
		// Get workflow steps
		steps, err := h.workflowRepo.GetWorkflowSteps(ctx, workflow.ID)
		if err != nil {
			debug.Error("Failed to get workflow steps for %s: %v", workflow.ID, err)
			steps = []models.JobWorkflowStep{}
		}
		
		// Format steps with preset job names
		formattedSteps := make([]map[string]interface{}, 0)
		for _, step := range steps {
			// Get preset job name for this step
			presetJob, err := h.presetJobRepo.GetByID(ctx, step.PresetJobID)
			stepData := map[string]interface{}{
				"id":            step.ID,
				"preset_job_id": step.PresetJobID.String(),
				"step_order":    step.StepOrder,
			}
			if err == nil && presetJob != nil {
				stepData["preset_job_name"] = presetJob.Name
			}
			formattedSteps = append(formattedSteps, stepData)
		}
		
		formattedWorkflows = append(formattedWorkflows, map[string]interface{}{
			"id":          workflow.ID.String(),
			"name":        workflow.Name,
			"steps":       formattedSteps,
		})
	}
	
	// Format form data
	formData := map[string]interface{}{
		"wordlists": formatWordlists(wordlists),
		"rules":     formatRules(rules),
		"binary_versions": formatBinaries(binaries),
	}
	
	// Return the expected structure
	response := map[string]interface{}{
		"preset_jobs": availableJobs,
		"workflows":   formattedWorkflows,
		"form_data":   formData,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions to format data
func formatWordlists(wordlists []*models.Wordlist) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(wordlists))
	for _, w := range wordlists {
		result = append(result, map[string]interface{}{
			"id":        w.ID,
			"name":      w.Name,
			"file_size": w.FileSize,
		})
	}
	return result
}

func formatRules(rules []*models.Rule) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(rules))
	for _, r := range rules {
		result = append(result, map[string]interface{}{
			"id":         r.ID,
			"name":       r.Name,
			"rule_count": r.RuleCount,
		})
	}
	return result
}

func formatBinaries(binaries []*binary.BinaryVersion) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(binaries))
	for _, b := range binaries {
		result = append(result, map[string]interface{}{
			"id":      b.ID,
			"version": b.FileName,
			"type":    string(b.BinaryType),
		})
	}
	return result
}

// ListUserJobs handles GET /api/user/jobs with pagination and filtering (filtered by authenticated user)
func (h *UserJobsHandler) ListUserJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 25
	}
	
	// Parse filters
	status := r.URL.Query().Get("status")
	priorityStr := r.URL.Query().Get("priority")
	search := r.URL.Query().Get("search")
	
	var priority *int
	if priorityStr != "" {
		p, err := strconv.Atoi(priorityStr)
		if err == nil && p >= 1 && p <= 10 {
			priority = &p
		}
	}
	
	// Create filter with user ID
	filter := repository.JobFilter{
		Status:   &status,
		Priority: priority,
		Search:   &search,
		UserID:   &userID,
	}
	
	// Get jobs with filters
	jobs, err := h.jobExecRepo.ListWithFilters(ctx, pageSize, (page-1)*pageSize, filter)
	if err != nil {
		debug.Error("Failed to list user jobs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Get total count with filters
	total, err := h.jobExecRepo.GetFilteredCount(ctx, filter)
	if err != nil {
		debug.Error("Failed to get user job count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Get status counts for this user
	statusCounts, err := h.jobExecRepo.GetStatusCountsForUser(ctx, userID)
	if err != nil {
		debug.Error("Failed to get user status counts: %v", err)
		// Don't fail the request, just log the error
		statusCounts = make(map[string]int)
	}
	
	// Convert to job summaries (reuse the same logic as ListJobs)
	summaries := make([]JobSummary, 0, len(jobs))
	for _, job := range jobs {
		// Get hashlist details including cracked count
		hashlist, err := h.hashlistRepo.GetByID(ctx, job.HashlistID)
		if err != nil {
			debug.Error("Failed to get hashlist %d: %v", job.HashlistID, err)
			continue
		}
		
		// Get task statistics
		tasks, err := h.jobTaskRepo.GetTasksByJobExecution(ctx, job.ID)
		if err != nil {
			debug.Error("Failed to get tasks for job %s: %v", job.ID, err)
			tasks = []models.JobTask{}
		}
		
		// Calculate metrics
		var agentCount int
		var totalSpeed int64
		var crackedCount int
		var keyspaceSearched int64
		var keyspaceDispatched int64
		
		for _, task := range tasks {
			if task.Status == models.JobTaskStatusRunning {
				agentCount++
				if task.BenchmarkSpeed != nil {
					totalSpeed += *task.BenchmarkSpeed
				}
			}
			crackedCount += task.CrackCount
			keyspaceSearched += task.KeyspaceProcessed
			
			// Calculate dispatched keyspace (assigned to tasks)
			if task.Status != models.JobTaskStatusPending {
				// Task has been dispatched if it's not pending
				taskKeyspace := task.KeyspaceEnd - task.KeyspaceStart
				keyspaceDispatched += taskKeyspace
			}
		}
		
		// Calculate percentages
		dispatchedPercent := 0.0
		searchedPercent := 0.0
		if job.TotalKeyspace != nil && *job.TotalKeyspace > 0 {
			dispatchedPercent = float64(keyspaceDispatched) / float64(*job.TotalKeyspace) * 100
			searchedPercent = float64(keyspaceSearched) / float64(*job.TotalKeyspace) * 100
			
			// Cap percentages at 100%
			if dispatchedPercent > 100 {
				dispatchedPercent = 100
			}
			if searchedPercent > 100 {
				searchedPercent = 100
			}
		}
		
		// Get preset job name
		presetJobName := "Custom Job"
		if job.PresetJobID != uuid.Nil {
			presetJob, err := h.presetJobRepo.GetByID(ctx, job.PresetJobID)
			if err == nil && presetJob != nil {
				presetJobName = presetJob.Name
			}
		}
		
		summary := JobSummary{
			ID:                job.ID.String(),
			Name:              presetJobName,
			HashlistID:        job.HashlistID,
			HashlistName:      hashlist.Name,
			Status:            string(job.Status),
			Priority:          job.Priority,
			MaxAgents:         job.MaxAgents,
			DispatchedPercent: dispatchedPercent,
			SearchedPercent:   searchedPercent,
			CrackedCount:      crackedCount,
			AgentCount:        agentCount,
			TotalSpeed:        totalSpeed,
			CreatedAt:         job.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         job.UpdatedAt.Format(time.RFC3339),
			ErrorMessage:      job.ErrorMessage,
		}
		
		summaries = append(summaries, summary)
	}
	
	// Create response
	response := map[string]interface{}{
		"jobs":          summaries,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"total_pages":   (total + pageSize - 1) / pageSize,
		"status_counts": statusCounts,
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		debug.Error("Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}