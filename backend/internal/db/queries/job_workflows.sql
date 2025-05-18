-- name: CreateJobWorkflow :one
INSERT INTO job_workflows (name)
VALUES ($1)
RETURNING *;

-- name: GetJobWorkflowByID :one
SELECT * FROM job_workflows
WHERE id = $1 LIMIT 1;

-- name: GetJobWorkflowByName :one
SELECT * FROM job_workflows
WHERE name = $1 LIMIT 1;

-- name: ListJobWorkflows :many
-- TODO: Add pagination and sorting
SELECT * FROM job_workflows
ORDER BY name;

-- name: UpdateJobWorkflow :one
UPDATE job_workflows
SET 
    name = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteJobWorkflow :exec
-- Note: Steps are deleted via ON DELETE CASCADE
DELETE FROM job_workflows
WHERE id = $1;

-- name: CreateJobWorkflowStep :one
INSERT INTO job_workflow_steps (job_workflow_id, preset_job_id, step_order)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetJobWorkflowStepsByWorkflowID :many
SELECT 
    jws.*,
    pj.name as preset_job_name
FROM job_workflow_steps jws
JOIN preset_jobs pj ON jws.preset_job_id = pj.id
WHERE jws.job_workflow_id = $1
ORDER BY jws.step_order;

-- name: DeleteJobWorkflowStepsByWorkflowID :exec
DELETE FROM job_workflow_steps
WHERE job_workflow_id = $1;

-- name: CheckPresetJobsExist :many
-- Check if a list of preset job IDs exist
SELECT id FROM preset_jobs WHERE id = ANY($1::uuid[]); 