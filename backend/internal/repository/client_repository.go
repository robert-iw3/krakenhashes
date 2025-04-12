package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db/queries" // Import queries package
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/lib/pq" // Import pq for error handling
)

// ClientRepository handles database operations for clients.
type ClientRepository struct {
	db *db.DB
}

// NewClientRepository creates a new instance of ClientRepository.
func NewClientRepository(database *db.DB) *ClientRepository {
	return &ClientRepository{db: database}
}

// Create inserts a new client record into the database.
func (r *ClientRepository) Create(ctx context.Context, client *models.Client) error {
	client.CreatedAt = time.Now()                              // Ensure CreatedAt is set
	client.UpdatedAt = time.Now()                              // Ensure UpdatedAt is set
	_, err := r.db.ExecContext(ctx, queries.CreateClientQuery, // Use constant
		client.ID,
		client.Name,
		client.Description,
		client.ContactInfo,
		client.DataRetentionMonths,
		client.CreatedAt,
		client.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return fmt.Errorf("client with name '%s' already exists: %w", client.Name, ErrDuplicateRecord)
		}
		return fmt.Errorf("failed to create client: %w", err)
	}
	return nil
}

// GetByID retrieves a client by its ID.
func (r *ClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Client, error) {
	row := r.db.QueryRowContext(ctx, queries.GetClientByIDQuery, id) // Use constant
	var client models.Client
	err := row.Scan(
		&client.ID,
		&client.Name,
		&client.Description,
		&client.ContactInfo,
		&client.DataRetentionMonths,
		&client.CreatedAt,
		&client.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("client with ID %s not found: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get client by ID %s: %w", id, err)
	}
	return &client, nil
}

// GetByName retrieves a single client by its name.
// Note: This query is not in client_queries.go yet. Needs to be added.
// const getClientByNameQuery = `
// SELECT id, name, description, contact_info, data_retention_months, created_at, updated_at
// FROM clients
// WHERE name = $1
// `

func (r *ClientRepository) GetByName(ctx context.Context, name string) (*models.Client, error) {
	row := r.db.QueryRowContext(ctx, queries.GetClientByNameQuery, name) // Use constant
	var client models.Client
	err := row.Scan(
		&client.ID,
		&client.Name,
		&client.Description,
		&client.ContactInfo,
		&client.DataRetentionMonths,
		&client.CreatedAt,
		&client.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil, nil when not found
		}
		return nil, fmt.Errorf("failed to get client by name %s: %w", name, err)
	}
	return &client, nil
}

// List retrieves all clients from the database.
func (r *ClientRepository) List(ctx context.Context) ([]models.Client, error) {
	rows, err := r.db.QueryContext(ctx, queries.ListClientsQuery) // Use constant
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}
	defer rows.Close()

	var clients []models.Client
	for rows.Next() {
		var client models.Client
		if err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Description,
			&client.ContactInfo,
			&client.DataRetentionMonths,
			&client.CreatedAt,
			&client.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan client row: %w", err)
		}
		clients = append(clients, client)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client rows: %w", err)
	}

	return clients, nil
}

// Search retrieves clients matching a search query (name, description).
// Note: This query is not in client_queries.go yet. Needs to be added.
// const searchClientsQuery = `
// SELECT id, name, description, contact_info, data_retention_months, created_at, updated_at
// FROM clients
// WHERE name ILIKE $1 OR description ILIKE $1
// ORDER BY name ASC
// LIMIT 50
// `

func (r *ClientRepository) Search(ctx context.Context, query string) ([]models.Client, error) {
	searchTerm := "%" + strings.ToLower(query) + "%"                            // Case-insensitive search
	rows, err := r.db.QueryContext(ctx, queries.SearchClientsQuery, searchTerm) // Use constant
	if err != nil {
		return nil, fmt.Errorf("failed to search clients with query '%s': %w", query, err)
	}
	defer rows.Close()

	var clients []models.Client
	for rows.Next() {
		var client models.Client
		if err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Description,
			&client.ContactInfo,
			&client.DataRetentionMonths,
			&client.CreatedAt,
			&client.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan client search result row: %w", err)
		}
		clients = append(clients, client)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client search results: %w", err)
	}

	return clients, nil
}

// Update modifies an existing client record in the database.
func (r *ClientRepository) Update(ctx context.Context, client *models.Client) error {
	client.UpdatedAt = time.Now()                                   // Ensure UpdatedAt is set
	result, err := r.db.ExecContext(ctx, queries.UpdateClientQuery, // Use constant
		client.Name,
		client.Description,
		client.ContactInfo,
		client.DataRetentionMonths,
		client.UpdatedAt,
		client.ID,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("client with name '%s' already exists: %w", client.Name, ErrDuplicateRecord)
		}
		return fmt.Errorf("failed to update client %s: %w", client.ID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		debug.Warning("Could not get rows affected after updating client %s: %v", client.ID, err)
	} else if rowsAffected == 0 {
		return fmt.Errorf("client with ID %s not found for update: %w", client.ID, ErrNotFound)
	}

	return nil
}

// Delete removes a client record from the database by its ID.
func (r *ClientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, queries.DeleteClientQuery, id) // Use constant
	if err != nil {
		return fmt.Errorf("failed to delete client %s: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		debug.Warning("Could not get rows affected after deleting client %s: %v", id, err)
	} else if rowsAffected == 0 {
		return fmt.Errorf("client with ID %s not found for deletion: %w", id, ErrNotFound)
	}

	return nil
}
