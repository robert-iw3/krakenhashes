package routes

import (
	"github.com/gorilla/mux"
)

// SetupAdminJobRoutes configures the routes for preset jobs and job workflows under the admin subrouter.
func SetupAdminJobRoutes(adminRouter *mux.Router, jobHandler *AdminJobsHandler) {
	// Ensure jobHandler is not nil
	if jobHandler == nil {
		// Handle error appropriately, maybe panic or log fatal
		// For now, just return to avoid nil pointer dereference
		return
	}

	// --- Preset Job Routes --- (/api/admin/preset-jobs)
	presetRouter := adminRouter.PathPrefix("/preset-jobs").Subrouter()
	presetRouter.HandleFunc("", jobHandler.CreatePresetJob).Methods("POST", "OPTIONS")
	presetRouter.HandleFunc("", jobHandler.ListPresetJobs).Methods("GET", "HEAD", "OPTIONS")
	presetRouter.HandleFunc("/form-data", jobHandler.GetPresetJobFormData).Methods("GET", "HEAD", "OPTIONS")
	presetRouter.HandleFunc("/{preset_job_id:[0-9a-fA-F-]+}", jobHandler.GetPresetJob).Methods("GET", "HEAD", "OPTIONS")
	presetRouter.HandleFunc("/{preset_job_id:[0-9a-fA-F-]+}", jobHandler.UpdatePresetJob).Methods("PUT", "OPTIONS")
	presetRouter.HandleFunc("/{preset_job_id:[0-9a-fA-F-]+}", jobHandler.DeletePresetJob).Methods("DELETE", "OPTIONS")
	presetRouter.HandleFunc("/{preset_job_id:[0-9a-fA-F-]+}/recalculate-keyspace", jobHandler.RecalculatePresetJobKeyspace).Methods("POST", "OPTIONS")
	presetRouter.HandleFunc("/recalculate-all-keyspaces", jobHandler.RecalculateAllMissingKeyspaces).Methods("POST", "OPTIONS")

	// --- Job Workflow Routes --- (/api/admin/job-workflows)
	workflowRouter := adminRouter.PathPrefix("/job-workflows").Subrouter()
	workflowRouter.HandleFunc("", jobHandler.CreateJobWorkflow).Methods("POST", "OPTIONS")
	workflowRouter.HandleFunc("", jobHandler.ListJobWorkflows).Methods("GET", "HEAD", "OPTIONS")
	workflowRouter.HandleFunc("/form-data", jobHandler.GetJobWorkflowFormData).Methods("GET", "HEAD", "OPTIONS")
	workflowRouter.HandleFunc("/{job_workflow_id:[0-9a-fA-F-]+}", jobHandler.GetJobWorkflow).Methods("GET", "HEAD", "OPTIONS")
	workflowRouter.HandleFunc("/{job_workflow_id:[0-9a-fA-F-]+}", jobHandler.UpdateJobWorkflow).Methods("PUT", "OPTIONS")
	workflowRouter.HandleFunc("/{job_workflow_id:[0-9a-fA-F-]+}", jobHandler.DeleteJobWorkflow).Methods("DELETE", "OPTIONS")
}
