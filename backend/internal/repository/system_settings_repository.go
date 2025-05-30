package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// SystemSettingsRepository handles database operations for system settings.
type SystemSettingsRepository struct {
	db *db.DB
}

// NewSystemSettingsRepository creates a new instance of SystemSettingsRepository.
func NewSystemSettingsRepository(database *db.DB) *SystemSettingsRepository {
	return &SystemSettingsRepository{db: database}
}

// GetSetting retrieves a specific setting by its key.
func (r *SystemSettingsRepository) GetSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	query := `
		SELECT key, value, description, data_type, updated_at
		FROM system_settings
		WHERE key = $1`

	row := r.db.QueryRowContext(ctx, query, key)
	var setting models.SystemSetting
	err := row.Scan(
		&setting.Key,
		&setting.Value,
		&setting.Description,
		&setting.DataType,
		&setting.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("system setting with key '%s' not found: %w", key, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get system setting by key '%s': %w", key, err)
	}
	return &setting, nil
}

// SetSetting updates a specific setting's value by its key.
func (r *SystemSettingsRepository) SetSetting(ctx context.Context, key string, value *string) error {
	now := time.Now()
	query := `
		UPDATE system_settings
		SET value = $1, updated_at = $2
		WHERE key = $3`

	result, err := r.db.ExecContext(ctx, query, value, now, key)
	if err != nil {
		return fmt.Errorf("failed to set system setting '%s': %w", key, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		debug.Warning("Could not get rows affected after updating system setting %s: %v", key, err)
	} else if rowsAffected == 0 {
		return fmt.Errorf("system setting with key '%s' not found for update: %w", key, ErrNotFound)
	}

	return nil
}

// GetAllSettings retrieves all system settings.
func (r *SystemSettingsRepository) GetAllSettings(ctx context.Context) ([]models.SystemSetting, error) {
	query := `
		SELECT key, value, description, data_type, updated_at
		FROM system_settings
		ORDER BY key ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list system settings: %w", err)
	}
	defer rows.Close()

	var settings []models.SystemSetting
	for rows.Next() {
		var setting models.SystemSetting
		if err := rows.Scan(
			&setting.Key,
			&setting.Value,
			&setting.Description,
			&setting.DataType,
			&setting.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan system setting row: %w", err)
		}
		settings = append(settings, setting)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating system setting rows: %w", err)
	}

	return settings, nil
}

// GetMaxJobPriority retrieves the maximum job priority setting as an integer.
func (r *SystemSettingsRepository) GetMaxJobPriority(ctx context.Context) (int, error) {
	setting, err := r.GetSetting(ctx, "max_job_priority")
	if err != nil {
		// Return default value if setting not found
		if err == ErrNotFound {
			return 1000, nil // Default max priority
		}
		return 0, err
	}

	if setting.Value == nil {
		return 1000, nil // Default if value is null
	}

	maxPriority, err := strconv.Atoi(*setting.Value)
	if err != nil {
		debug.Error("Invalid max_job_priority value in database: %s", *setting.Value)
		return 1000, nil // Return default on invalid value
	}

	return maxPriority, nil
}

// SetMaxJobPriority updates the maximum job priority setting.
func (r *SystemSettingsRepository) SetMaxJobPriority(ctx context.Context, maxPriority int) error {
	value := strconv.Itoa(maxPriority)
	return r.SetSetting(ctx, "max_job_priority", &value)
}
