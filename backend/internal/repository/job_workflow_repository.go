package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// JobWorkflowRepository defines the interface for interacting with job workflows and steps.
type JobWorkflowRepository interface {
	CreateWorkflow(ctx context.Context, name string) (*models.JobWorkflow, error)
	GetWorkflowByID(ctx context.Context, id uuid.UUID) (*models.JobWorkflow, error)
	GetWorkflowByName(ctx context.Context, name string) (*models.JobWorkflow, error)
	ListWorkflows(ctx context.Context) ([]models.JobWorkflow, error)
	UpdateWorkflow(ctx context.Context, id uuid.UUID, name string) (*models.JobWorkflow, error)
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error

	CreateWorkflowStep(ctx context.Context, workflowID, presetJobID uuid.UUID, stepOrder int) (*models.JobWorkflowStep, error)
	GetWorkflowSteps(ctx context.Context, workflowID uuid.UUID) ([]models.JobWorkflowStep, error)
	DeleteWorkflowSteps(ctx context.Context, workflowID uuid.UUID) error
	CheckPresetJobsExist(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error)

	// Transactional methods (optional, can be handled in service layer)
	// ReplaceWorkflowStepsTx(ctx context.Context, tx *sql.Tx, workflowID uuid.UUID, steps []models.JobWorkflowStep) error
}

// jobWorkflowRepository implements JobWorkflowRepository.
type jobWorkflowRepository struct {
	db *sql.DB
}

// NewJobWorkflowRepository creates a new repository for job workflows.
func NewJobWorkflowRepository(db *sql.DB) JobWorkflowRepository {
	return &jobWorkflowRepository{db: db}
}

// CreateWorkflow inserts a new job workflow.
func (r *jobWorkflowRepository) CreateWorkflow(ctx context.Context, name string) (*models.JobWorkflow, error) {
	query := `INSERT INTO job_workflows (name) VALUES ($1) RETURNING id, name, created_at, updated_at`
	row := r.db.QueryRowContext(ctx, query, name)

	var wf models.JobWorkflow
	err := row.Scan(&wf.ID, &wf.Name, &wf.CreatedAt, &wf.UpdatedAt)
	if err != nil {
		// TODO: Handle potential unique constraint violation error (e.g., convert pq error)
		debug.Error("Error creating job workflow: %v", err)
		return nil, fmt.Errorf("error creating job workflow: %w", err)
	}
	return &wf, nil
}

// GetWorkflowByID retrieves a job workflow by ID, including its steps.
func (r *jobWorkflowRepository) GetWorkflowByID(ctx context.Context, id uuid.UUID) (*models.JobWorkflow, error) {
	query := `SELECT id, name, created_at, updated_at FROM job_workflows WHERE id = $1 LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, id)

	var wf models.JobWorkflow
	err := row.Scan(&wf.ID, &wf.Name, &wf.CreatedAt, &wf.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job workflow not found: %w", ErrNotFound)
		}
		debug.Error("Error getting job workflow by ID %s: %v", id, err)
		return nil, fmt.Errorf("error getting job workflow by ID: %w", err)
	}

	// Fetch associated steps
	steps, err := r.GetWorkflowSteps(ctx, wf.ID)
	if err != nil {
		// Log error but return the workflow info we already have
		debug.Error("Error fetching steps for workflow %s: %v", id, err)
	} else {
		wf.Steps = steps
	}

	return &wf, nil
}

// GetWorkflowByName retrieves a job workflow by name.
func (r *jobWorkflowRepository) GetWorkflowByName(ctx context.Context, name string) (*models.JobWorkflow, error) {
	query := `SELECT id, name, created_at, updated_at FROM job_workflows WHERE name = $1 LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, name)

	var wf models.JobWorkflow
	err := row.Scan(&wf.ID, &wf.Name, &wf.CreatedAt, &wf.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job workflow not found: %w", ErrNotFound)
		}
		debug.Error("Error getting job workflow by name %s: %v", name, err)
		return nil, fmt.Errorf("error getting job workflow by name: %w", err)
	}
	// Note: Steps are not fetched by default in GetByName, only basic workflow info.
	return &wf, nil
}

// ListWorkflows retrieves all job workflows.
func (r *jobWorkflowRepository) ListWorkflows(ctx context.Context) ([]models.JobWorkflow, error) {
	query := `
		SELECT w.id, w.name, w.created_at, w.updated_at, COUNT(s.id) as step_count
		FROM job_workflows w
		LEFT JOIN job_workflow_steps s ON w.id = s.job_workflow_id
		GROUP BY w.id, w.name, w.created_at, w.updated_at
		ORDER BY w.name
	` // TODO: Pagination
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		debug.Error("Error listing job workflows: %v", err)
		return nil, fmt.Errorf("error listing job workflows: %w", err)
	}
	defer rows.Close()

	workflows := []models.JobWorkflow{}
	for rows.Next() {
		var wf models.JobWorkflow
		var stepCount int
		if err := rows.Scan(&wf.ID, &wf.Name, &wf.CreatedAt, &wf.UpdatedAt, &stepCount); err != nil {
			debug.Error("Error scanning job workflow row: %v", err)
			return nil, fmt.Errorf("error scanning job workflow row: %w", err)
		}

		// Create empty steps slice with the correct count
		wf.Steps = make([]models.JobWorkflowStep, stepCount)

		workflows = append(workflows, wf)
	}

	if err = rows.Err(); err != nil {
		debug.Error("Error iterating job workflow rows: %v", err)
		return nil, fmt.Errorf("error iterating job workflow rows: %w", err)
	}

	return workflows, nil
}

// UpdateWorkflow updates a job workflow's name.
func (r *jobWorkflowRepository) UpdateWorkflow(ctx context.Context, id uuid.UUID, name string) (*models.JobWorkflow, error) {
	query := `UPDATE job_workflows SET name = $2, updated_at = NOW() WHERE id = $1 RETURNING id, name, created_at, updated_at`
	row := r.db.QueryRowContext(ctx, query, id, name)

	var wf models.JobWorkflow
	err := row.Scan(&wf.ID, &wf.Name, &wf.CreatedAt, &wf.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job workflow not found for update: %w", ErrNotFound)
		}
		// TODO: Handle potential unique constraint violation error
		debug.Error("Error updating job workflow %s: %v", id, err)
		return nil, fmt.Errorf("error updating job workflow: %w", err)
	}
	return &wf, nil
}

// DeleteWorkflow removes a job workflow and its steps (due to ON DELETE CASCADE).
func (r *jobWorkflowRepository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM job_workflows WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		debug.Error("Error deleting job workflow %s: %v", id, err)
		return fmt.Errorf("error deleting job workflow: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		debug.Warning("Could not get rows affected after deleting job workflow %s: %v", id, err)
	} else if rowsAffected == 0 {
		return fmt.Errorf("job workflow not found for deletion: %w", ErrNotFound)
	}
	return nil
}

// --- Workflow Step Methods ---

// CreateWorkflowStep inserts a single step for a workflow.
func (r *jobWorkflowRepository) CreateWorkflowStep(ctx context.Context, workflowID, presetJobID uuid.UUID, stepOrder int) (*models.JobWorkflowStep, error) {
	query := `
		INSERT INTO job_workflow_steps (job_workflow_id, preset_job_id, step_order)
		VALUES ($1, $2, $3)
		RETURNING id, job_workflow_id, preset_job_id, step_order`
	row := r.db.QueryRowContext(ctx, query, workflowID, presetJobID, stepOrder)

	var step models.JobWorkflowStep
	err := row.Scan(&step.ID, &step.JobWorkflowID, &step.PresetJobID, &step.StepOrder)
	if err != nil {
		// TODO: Handle potential unique constraint (workflowID, stepOrder) violation
		// TODO: Handle potential foreign key constraint violation (presetJobID)
		debug.Error("Error creating job workflow step: %v", err)
		return nil, fmt.Errorf("error creating job workflow step: %w", err)
	}
	return &step, nil
}

// GetWorkflowSteps retrieves all steps for a given workflow, ordered by step_order, including preset job names.
func (r *jobWorkflowRepository) GetWorkflowSteps(ctx context.Context, workflowID uuid.UUID) ([]models.JobWorkflowStep, error) {
	query := `
		SELECT 
			jws.id, jws.job_workflow_id, jws.preset_job_id, jws.step_order,
			pj.name as preset_job_name
		FROM job_workflow_steps jws
		JOIN preset_jobs pj ON jws.preset_job_id = pj.id
		WHERE jws.job_workflow_id = $1
		ORDER BY jws.step_order`

	rows, err := r.db.QueryContext(ctx, query, workflowID)
	if err != nil {
		debug.Error("Error getting steps for workflow %s: %v", workflowID, err)
		return nil, fmt.Errorf("error getting workflow steps: %w", err)
	}
	defer rows.Close()

	steps := []models.JobWorkflowStep{}
	for rows.Next() {
		var step models.JobWorkflowStep
		if err := rows.Scan(
			&step.ID, &step.JobWorkflowID, &step.PresetJobID, &step.StepOrder,
			&step.PresetJobName,
		); err != nil {
			debug.Error("Error scanning job workflow step row: %v", err)
			return nil, fmt.Errorf("error scanning job workflow step row: %w", err)
		}
		steps = append(steps, step)
	}

	if err = rows.Err(); err != nil {
		debug.Error("Error iterating job workflow step rows: %v", err)
		return nil, fmt.Errorf("error iterating job workflow step rows: %w", err)
	}

	return steps, nil
}

// DeleteWorkflowSteps removes all steps associated with a workflow ID.
// Often used within a transaction before inserting new steps.
func (r *jobWorkflowRepository) DeleteWorkflowSteps(ctx context.Context, workflowID uuid.UUID) error {
	query := `DELETE FROM job_workflow_steps WHERE job_workflow_id = $1`
	_, err := r.db.ExecContext(ctx, query, workflowID)
	if err != nil {
		debug.Error("Error deleting steps for workflow %s: %v", workflowID, err)
		return fmt.Errorf("error deleting workflow steps: %w", err)
	}
	// We don't check RowsAffected here, as deleting 0 rows isn't necessarily an error.
	return nil
}

// CheckPresetJobsExist checks if a list of preset job IDs exist in the database.
// Returns a list of IDs that *do* exist.
func (r *jobWorkflowRepository) CheckPresetJobsExist(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return []uuid.UUID{}, nil
	}
	query := `SELECT id FROM preset_jobs WHERE id = ANY($1::uuid[])`
	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		debug.Error("Error checking preset job existence: %v", err)
		return nil, fmt.Errorf("error checking preset job existence: %w", err)
	}
	defer rows.Close()

	existingIDs := make([]uuid.UUID, 0, len(ids))
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			debug.Error("Error scanning existing preset job ID: %v", err)
			return nil, fmt.Errorf("error scanning existing preset job ID: %w", err)
		}
		existingIDs = append(existingIDs, id)
	}

	if err = rows.Err(); err != nil {
		debug.Error("Error iterating existing preset job IDs: %v", err)
		return nil, fmt.Errorf("error iterating existing preset job IDs: %w", err)
	}

	return existingIDs, nil
}
