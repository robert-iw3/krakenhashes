package wordlist

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
)

// Store handles database operations for wordlists
type Store struct {
	db *sql.DB
}

// NewStore creates a new wordlist store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ListWordlists retrieves all wordlists with optional filtering
func (s *Store) ListWordlists(ctx context.Context, filters map[string]interface{}) ([]*models.Wordlist, error) {
	// Base query
	query := `
		SELECT w.id, w.name, w.description, w.wordlist_type, w.format, w.file_name, 
		       w.md5_hash, w.file_size, w.word_count, w.created_at, w.created_by, 
		       w.updated_at, w.updated_by, w.last_verified_at, w.verification_status
		FROM wordlists w
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if wordlistType, ok := filters["wordlist_type"]; ok {
		query += " AND w.wordlist_type = $" + strconv.Itoa(argPos)
		args = append(args, wordlistType)
		argPos++
	}

	if format, ok := filters["format"]; ok {
		query += " AND w.format = $" + strconv.Itoa(argPos)
		args = append(args, format)
		argPos++
	}

	if tag, ok := filters["tag"]; ok {
		query += ` AND w.id IN (
			SELECT wordlist_id FROM wordlist_tags WHERE tag = $` + strconv.Itoa(argPos) + `
		)`
		args = append(args, tag)
		argPos++
	}

	query += " ORDER BY w.name ASC"

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		debug.Error("Failed to list wordlists: %v", err)
		return nil, err
	}
	defer rows.Close()

	// Parse results
	wordlists := []*models.Wordlist{}
	for rows.Next() {
		w := &models.Wordlist{}
		var lastVerifiedAt sql.NullTime

		err := rows.Scan(
			&w.ID, &w.Name, &w.Description, &w.WordlistType, &w.Format, &w.FileName,
			&w.MD5Hash, &w.FileSize, &w.WordCount, &w.CreatedAt, &w.CreatedBy,
			&w.UpdatedAt, &w.UpdatedBy, &lastVerifiedAt, &w.VerificationStatus,
		)
		if err != nil {
			debug.Error("Failed to scan wordlist row: %v", err)
			return nil, err
		}

		// Set LastVerifiedAt if valid
		if lastVerifiedAt.Valid {
			w.LastVerifiedAt = lastVerifiedAt.Time
		}

		// Get tags for this wordlist
		tags, err := s.GetWordlistTags(ctx, w.ID)
		if err != nil {
			debug.Error("Failed to get tags for wordlist %d: %v", w.ID, err)
			return nil, err
		}
		w.Tags = tags

		wordlists = append(wordlists, w)
	}

	if err := rows.Err(); err != nil {
		debug.Error("Error iterating wordlist rows: %v", err)
		return nil, err
	}

	return wordlists, nil
}

// GetWordlist retrieves a wordlist by ID
func (s *Store) GetWordlist(ctx context.Context, id int) (*models.Wordlist, error) {
	query := `
		SELECT w.id, w.name, w.description, w.wordlist_type, w.format, w.file_name, 
		       w.md5_hash, w.file_size, w.word_count, w.created_at, w.created_by, 
		       w.updated_at, w.updated_by, w.last_verified_at, w.verification_status
		FROM wordlists w
		WHERE w.id = $1
	`

	w := &models.Wordlist{}
	var lastVerifiedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&w.ID, &w.Name, &w.Description, &w.WordlistType, &w.Format, &w.FileName,
		&w.MD5Hash, &w.FileSize, &w.WordCount, &w.CreatedAt, &w.CreatedBy,
		&w.UpdatedAt, &w.UpdatedBy, &lastVerifiedAt, &w.VerificationStatus,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		debug.Error("Failed to get wordlist %d: %v", id, err)
		return nil, err
	}

	// Set LastVerifiedAt if valid
	if lastVerifiedAt.Valid {
		w.LastVerifiedAt = lastVerifiedAt.Time
	}

	// Get tags for this wordlist
	tags, err := s.GetWordlistTags(ctx, w.ID)
	if err != nil {
		debug.Error("Failed to get tags for wordlist %d: %v", w.ID, err)
		return nil, err
	}
	w.Tags = tags

	return w, nil
}

// GetWordlistByFilename retrieves a wordlist by filename
func (s *Store) GetWordlistByFilename(ctx context.Context, filename string) (*models.Wordlist, error) {
	query := `
		SELECT w.id, w.name, w.description, w.wordlist_type, w.format, w.file_name, 
		       w.md5_hash, w.file_size, w.word_count, w.created_at, w.created_by, 
		       w.updated_at, w.updated_by, w.last_verified_at, w.verification_status
		FROM wordlists w
		WHERE w.file_name = $1
	`

	w := &models.Wordlist{}
	var lastVerifiedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, filename).Scan(
		&w.ID, &w.Name, &w.Description, &w.WordlistType, &w.Format, &w.FileName,
		&w.MD5Hash, &w.FileSize, &w.WordCount, &w.CreatedAt, &w.CreatedBy,
		&w.UpdatedAt, &w.UpdatedBy, &lastVerifiedAt, &w.VerificationStatus,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		debug.Error("Failed to get wordlist by filename %s: %v", filename, err)
		return nil, err
	}

	// Set LastVerifiedAt if valid
	if lastVerifiedAt.Valid {
		w.LastVerifiedAt = lastVerifiedAt.Time
	}

	// Get tags for this wordlist
	tags, err := s.GetWordlistTags(ctx, w.ID)
	if err != nil {
		debug.Error("Failed to get tags for wordlist %d: %v", w.ID, err)
		return nil, err
	}
	w.Tags = tags

	return w, nil
}

// GetWordlistByMD5Hash retrieves a wordlist by MD5 hash
func (s *Store) GetWordlistByMD5Hash(ctx context.Context, md5Hash string) (*models.Wordlist, error) {
	query := `
		SELECT w.id, w.name, w.description, w.wordlist_type, w.format, w.file_name, 
		       w.md5_hash, w.file_size, w.word_count, w.created_at, w.created_by, 
		       w.updated_at, w.updated_by, w.last_verified_at, w.verification_status
		FROM wordlists w
		WHERE w.md5_hash = $1
	`

	w := &models.Wordlist{}
	var lastVerifiedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, md5Hash).Scan(
		&w.ID, &w.Name, &w.Description, &w.WordlistType, &w.Format, &w.FileName,
		&w.MD5Hash, &w.FileSize, &w.WordCount, &w.CreatedAt, &w.CreatedBy,
		&w.UpdatedAt, &w.UpdatedBy, &lastVerifiedAt, &w.VerificationStatus,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		debug.Error("Failed to get wordlist by MD5 hash %s: %v", md5Hash, err)
		return nil, err
	}

	// Set LastVerifiedAt if valid
	if lastVerifiedAt.Valid {
		w.LastVerifiedAt = lastVerifiedAt.Time
	}

	// Get tags for this wordlist
	tags, err := s.GetWordlistTags(ctx, w.ID)
	if err != nil {
		debug.Error("Failed to get tags for wordlist %d: %v", w.ID, err)
		return nil, err
	}
	w.Tags = tags

	return w, nil
}

// CreateWordlist creates a new wordlist
func (s *Store) CreateWordlist(ctx context.Context, wordlist *models.Wordlist) error {
	query := `
		INSERT INTO wordlists (
			name, description, wordlist_type, format, file_name, 
			md5_hash, file_size, word_count, created_by, verification_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		wordlist.Name, wordlist.Description, wordlist.WordlistType, wordlist.Format, wordlist.FileName,
		wordlist.MD5Hash, wordlist.FileSize, wordlist.WordCount, wordlist.CreatedBy, wordlist.VerificationStatus,
	).Scan(&wordlist.ID, &wordlist.CreatedAt, &wordlist.UpdatedAt)
	if err != nil {
		debug.Error("Failed to create wordlist: %v", err)
		return err
	}

	// Add tags if provided
	if len(wordlist.Tags) > 0 {
		for _, tag := range wordlist.Tags {
			err := s.AddWordlistTag(ctx, wordlist.ID, tag, wordlist.CreatedBy)
			if err != nil {
				debug.Error("Failed to add tag %s to wordlist %d: %v", tag, wordlist.ID, err)
				return err
			}
		}
	}

	return nil
}

// UpdateWordlist updates an existing wordlist
func (s *Store) UpdateWordlist(ctx context.Context, wordlist *models.Wordlist) error {
	query := `
		UPDATE wordlists
		SET name = $1, description = $2, wordlist_type = $3, format = $4,
		    updated_at = NOW(), updated_by = $5
		WHERE id = $6
		RETURNING updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		wordlist.Name, wordlist.Description, wordlist.WordlistType, wordlist.Format,
		wordlist.UpdatedBy, wordlist.ID,
	).Scan(&wordlist.UpdatedAt)
	if err != nil {
		debug.Error("Failed to update wordlist %d: %v", wordlist.ID, err)
		return err
	}

	return nil
}

// DeleteWordlist deletes a wordlist
func (s *Store) DeleteWordlist(ctx context.Context, id int) error {
	// Delete tags first (foreign key constraint)
	_, err := s.db.ExecContext(ctx, "DELETE FROM wordlist_tags WHERE wordlist_id = $1", id)
	if err != nil {
		debug.Error("Failed to delete tags for wordlist %d: %v", id, err)
		return err
	}

	// Delete wordlist
	_, err = s.db.ExecContext(ctx, "DELETE FROM wordlists WHERE id = $1", id)
	if err != nil {
		debug.Error("Failed to delete wordlist %d: %v", id, err)
		return err
	}

	return nil
}

// UpdateWordlistVerification updates a wordlist's verification status
func (s *Store) UpdateWordlistVerification(ctx context.Context, id int, status string, wordCount *int64) error {
	query := `
		UPDATE wordlists
		SET verification_status = $1, last_verified_at = NOW()
	`
	args := []interface{}{status, id}
	argPos := 3

	if wordCount != nil {
		query += ", word_count = $" + strconv.Itoa(argPos)
		args = append(args, *wordCount)
		argPos++
	}

	query += " WHERE id = $2"

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		debug.Error("Failed to update verification status for wordlist %d: %v", id, err)
		return err
	}

	return nil
}

// GetWordlistTags gets tags for a wordlist
func (s *Store) GetWordlistTags(ctx context.Context, id int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT tag FROM wordlist_tags WHERE wordlist_id = $1", id)
	if err != nil {
		debug.Error("Failed to get tags for wordlist %d: %v", id, err)
		return nil, err
	}
	defer rows.Close()

	tags := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			debug.Error("Failed to scan tag: %v", err)
			return nil, err
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		debug.Error("Error iterating tag rows: %v", err)
		return nil, err
	}

	return tags, nil
}

// AddWordlistTag adds a tag to a wordlist
func (s *Store) AddWordlistTag(ctx context.Context, id int, tag string, userID uuid.UUID) error {
	// Check if tag already exists
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM wordlist_tags WHERE wordlist_id = $1 AND tag = $2)", id, tag).Scan(&exists)
	if err != nil {
		debug.Error("Failed to check if tag exists: %v", err)
		return err
	}

	if exists {
		return nil // Tag already exists, nothing to do
	}

	// Add tag
	_, err = s.db.ExecContext(ctx, "INSERT INTO wordlist_tags (wordlist_id, tag, created_by) VALUES ($1, $2, $3)", id, tag, userID)
	if err != nil {
		debug.Error("Failed to add tag %s to wordlist %d: %v", tag, id, err)
		return err
	}

	return nil
}

// DeleteWordlistTag deletes a tag from a wordlist
func (s *Store) DeleteWordlistTag(ctx context.Context, id int, tag string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM wordlist_tags WHERE wordlist_id = $1 AND tag = $2", id, tag)
	if err != nil {
		debug.Error("Failed to delete tag %s from wordlist %d: %v", tag, id, err)
		return err
	}

	return nil
}
