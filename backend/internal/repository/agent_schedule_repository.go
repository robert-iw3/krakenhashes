package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
)

// AgentScheduleRepository handles database operations for agent schedules
type AgentScheduleRepository struct {
	db *db.DB
}

// NewAgentScheduleRepository creates a new agent schedule repository
func NewAgentScheduleRepository(db *db.DB) *AgentScheduleRepository {
	return &AgentScheduleRepository{db: db}
}

// CreateSchedule creates a new schedule for an agent
func (r *AgentScheduleRepository) CreateSchedule(ctx context.Context, schedule *models.AgentSchedule) error {
	query := `
		INSERT INTO agent_schedules (agent_id, day_of_week, start_time, end_time, timezone, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		schedule.AgentID,
		schedule.DayOfWeek,
		schedule.StartTime.String(), // Convert TimeOnly to string for storage
		schedule.EndTime.String(),   // Convert TimeOnly to string for storage
		schedule.Timezone,
		schedule.IsActive,
	).Scan(&schedule.ID, &schedule.CreatedAt, &schedule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	return nil
}

// UpdateSchedule updates an existing schedule
func (r *AgentScheduleRepository) UpdateSchedule(ctx context.Context, schedule *models.AgentSchedule) error {
	query := `
		UPDATE agent_schedules 
		SET start_time = $3, end_time = $4, timezone = $5, is_active = $6, updated_at = CURRENT_TIMESTAMP
		WHERE agent_id = $1 AND day_of_week = $2
		RETURNING updated_at`

	err := r.db.QueryRowContext(ctx, query,
		schedule.AgentID,
		schedule.DayOfWeek,
		schedule.StartTime.String(), // Convert TimeOnly to string for storage
		schedule.EndTime.String(),   // Convert TimeOnly to string for storage
		schedule.Timezone,
		schedule.IsActive,
	).Scan(&schedule.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			// If no existing schedule, create one
			return r.CreateSchedule(ctx, schedule)
		}
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	return nil
}

// DeleteSchedule deletes a schedule for a specific day
func (r *AgentScheduleRepository) DeleteSchedule(ctx context.Context, agentID int, dayOfWeek int) error {
	query := `DELETE FROM agent_schedules WHERE agent_id = $1 AND day_of_week = $2`

	result, err := r.db.ExecContext(ctx, query, agentID, dayOfWeek)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetSchedulesByAgent gets all schedules for an agent
func (r *AgentScheduleRepository) GetSchedulesByAgent(ctx context.Context, agentID int) ([]models.AgentSchedule, error) {
	query := `
		SELECT id, agent_id, day_of_week, start_time, end_time, timezone, is_active, created_at, updated_at
		FROM agent_schedules
		WHERE agent_id = $1
		ORDER BY day_of_week`

	rows, err := r.db.QueryContext(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}
	defer rows.Close()

	schedules, err := models.ScanSchedules(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan schedules: %w", err)
	}

	return schedules, nil
}

// GetScheduleByAgentAndDay gets a specific schedule for an agent on a day
func (r *AgentScheduleRepository) GetScheduleByAgentAndDay(ctx context.Context, agentID int, dayOfWeek int) (*models.AgentSchedule, error) {
	query := `
		SELECT id, agent_id, day_of_week, start_time, end_time, timezone, is_active, created_at, updated_at
		FROM agent_schedules
		WHERE agent_id = $1 AND day_of_week = $2`

	var schedule models.AgentSchedule
	err := r.db.QueryRowContext(ctx, query, agentID, dayOfWeek).Scan(
		&schedule.ID,
		&schedule.AgentID,
		&schedule.DayOfWeek,
		&schedule.StartTime,
		&schedule.EndTime,
		&schedule.Timezone,
		&schedule.IsActive,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return &schedule, nil
}

// IsAgentScheduledNow checks if an agent is scheduled to work at the current UTC time
func (r *AgentScheduleRepository) IsAgentScheduledNow(ctx context.Context, agentID int) (bool, error) {
	now := time.Now().UTC()
	currentDay := int(now.Weekday())
	currentTimeStr := now.Format("15:04:05") // Include seconds for proper comparison

	// First check if there's an active schedule for today
	query := `
		SELECT COUNT(*) > 0
		FROM agent_schedules
		WHERE agent_id = $1 
		AND day_of_week = $2
		AND is_active = true
		AND (
			-- Normal schedule (start < end)
			(start_time < end_time AND $3::time >= start_time AND $3::time < end_time)
			OR
			-- Overnight schedule (start > end, e.g., 22:00 - 02:00)
			(start_time > end_time AND ($3::time >= start_time OR $3::time < end_time))
		)`

	var isScheduled bool
	err := r.db.QueryRowContext(ctx, query, agentID, currentDay, currentTimeStr).Scan(&isScheduled)
	if err != nil {
		return false, fmt.Errorf("failed to check schedule: %w", err)
	}

	// If scheduled for today, check if we also need to check yesterday for overnight schedules
	if !isScheduled {
		// Check if there's an overnight schedule from yesterday that extends into today
		yesterday := (currentDay - 1 + 7) % 7
		query = `
			SELECT COUNT(*) > 0
			FROM agent_schedules
			WHERE agent_id = $1 
			AND day_of_week = $2
			AND is_active = true
			AND start_time > end_time
			AND $3::time < end_time`

		err = r.db.QueryRowContext(ctx, query, agentID, yesterday, currentTimeStr).Scan(&isScheduled)
		if err != nil {
			return false, fmt.Errorf("failed to check overnight schedule: %w", err)
		}
	}

	return isScheduled, nil
}

// DeleteAllSchedules deletes all schedules for an agent
func (r *AgentScheduleRepository) DeleteAllSchedules(ctx context.Context, agentID int) error {
	query := `DELETE FROM agent_schedules WHERE agent_id = $1`

	_, err := r.db.ExecContext(ctx, query, agentID)
	if err != nil {
		return fmt.Errorf("failed to delete all schedules: %w", err)
	}

	return nil
}

// UpdateAgentScheduling updates the scheduling enabled flag for an agent
func (r *AgentScheduleRepository) UpdateAgentScheduling(ctx context.Context, agentID int, enabled bool, timezone string) error {
	query := `
		UPDATE agents 
		SET scheduling_enabled = $2, schedule_timezone = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, agentID, enabled, timezone)
	if err != nil {
		return fmt.Errorf("failed to update agent scheduling: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}