package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db/queries"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
)

// ClientSettingsRepository handles database operations for client settings.
type ClientSettingsRepository struct {
	db *db.DB
}

// NewClientSettingsRepository creates a new instance of ClientSettingsRepository.
func NewClientSettingsRepository(database *db.DB) *ClientSettingsRepository {
	return &ClientSettingsRepository{db: database}
}

// GetSetting retrieves a specific setting by its key.
func (r *ClientSettingsRepository) GetSetting(ctx context.Context, key string) (*models.ClientSetting, error) {
	row := r.db.QueryRowContext(ctx, queries.GetClientSettingQuery, key)
	var setting models.ClientSetting
	err := row.Scan(
		&setting.Key,
		&setting.Value,
		&setting.Description,
		&setting.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Use ErrNotFound from errors.go
			return nil, fmt.Errorf("client setting with key '%s' not found: %w", key, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get client setting by key '%s': %w", key, err)
	}
	return &setting, nil
}

// SetSetting updates a specific setting's value by its key.
func (r *ClientSettingsRepository) SetSetting(ctx context.Context, key string, value *string) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, queries.SetClientSettingQuery, value, now, key)
	if err != nil {
		return fmt.Errorf("failed to set client setting '%s': %w", key, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Log this error, but don't necessarily fail the operation
		// fmt.Printf("Warning: Could not get rows affected after updating client setting %s: %v", key, err)
	} else if rowsAffected == 0 {
		// Use ErrNotFound from errors.go
		return fmt.Errorf("client setting with key '%s' not found for update: %w", key, ErrNotFound)
	}

	return nil
}

// GetAllSettings retrieves all client settings.
func (r *ClientSettingsRepository) GetAllSettings(ctx context.Context) ([]models.ClientSetting, error) {
	rows, err := r.db.QueryContext(ctx, queries.GetAllClientSettingsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to list client settings: %w", err)
	}
	defer rows.Close()

	var settings []models.ClientSetting
	for rows.Next() {
		var setting models.ClientSetting
		if err := rows.Scan(
			&setting.Key,
			&setting.Value,
			&setting.Description,
			&setting.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan client setting row: %w", err)
		}
		settings = append(settings, setting)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client setting rows: %w", err)
	}

	return settings, nil
}
