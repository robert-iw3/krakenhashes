package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// FileInfo represents information about a file for synchronization
type FileInfo struct {
	Name      string `json:"name"`
	MD5Hash   string `json:"md5_hash"` // MD5 hash used for synchronization
	Size      int64  `json:"size"`
	FileType  string `json:"file_type"` // "wordlist", "rule", "binary", "hashlist"
	Category  string `json:"category,omitempty"`
	ID        int    `json:"id,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// FileRepository handles database operations for files (wordlists, rules, binaries)
type FileRepository struct {
	db *db.DB
}

// NewFileRepository creates a new file repository
func NewFileRepository(db *db.DB) *FileRepository {
	return &FileRepository{db: db}
}

// GetWordlists retrieves wordlists matching the specified category
func (r *FileRepository) GetWordlists(ctx context.Context, category string) ([]FileInfo, error) {
	// Check if category is valid for wordlist_type enum
	var query string
	var rows *sql.Rows
	var err error

	if category == "" {
		// If category is empty, return all verified wordlists
		query = `
			SELECT id, name, file_name, md5_hash, file_size, wordlist_type, updated_at 
			FROM wordlists 
			WHERE verification_status = 'verified'
		`
		rows, err = r.db.QueryContext(ctx, query)
	} else if category == "general" || category == "specialized" || category == "targeted" || category == "custom" {
		// Category is a valid enum value, use it for filtering
		query = `
			SELECT id, name, file_name, md5_hash, file_size, wordlist_type, updated_at 
			FROM wordlists 
			WHERE wordlist_type = $1::wordlist_type
			AND verification_status = 'verified'
		`
		rows, err = r.db.QueryContext(ctx, query, category)
	} else {
		// Category is not a valid enum value, return empty set
		debug.Info("Invalid wordlist_type category: %s, returning empty set", category)
		return []FileInfo{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("error querying wordlists: %w", err)
	}
	defer rows.Close()

	var files []FileInfo

	for rows.Next() {
		var id int
		var name, fileName, md5Hash, wordlistType string
		var size int64
		var updatedAt time.Time

		if err := rows.Scan(&id, &name, &fileName, &md5Hash, &size, &wordlistType, &updatedAt); err != nil {
			debug.Error("Error scanning wordlist row: %v", err)
			continue
		}

		files = append(files, FileInfo{
			Name:      fileName,
			MD5Hash:   md5Hash,
			Size:      size,
			FileType:  "wordlist",
			Category:  wordlistType,
			ID:        id,
			Timestamp: updatedAt.Unix(),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating wordlist rows: %w", err)
	}

	return files, nil
}

// GetRules retrieves rules matching the specified category
func (r *FileRepository) GetRules(ctx context.Context, category string) ([]FileInfo, error) {
	// Check if category is valid for rule_type enum
	var query string
	var rows *sql.Rows
	var err error

	if category == "" {
		// If category is empty, return all verified rules
		query = `
			SELECT id, name, file_name, md5_hash, file_size, rule_type, updated_at 
			FROM rules 
			WHERE verification_status = 'verified'
		`
		rows, err = r.db.QueryContext(ctx, query)
	} else if category == "hashcat" || category == "john" {
		// Category is a valid enum value, use it for filtering
		query = `
			SELECT id, name, file_name, md5_hash, file_size, rule_type, updated_at 
			FROM rules 
			WHERE rule_type = $1::rule_type
			AND verification_status = 'verified'
		`
		rows, err = r.db.QueryContext(ctx, query, category)
	} else {
		// Category is not a valid enum value, return empty set
		debug.Info("Invalid rule_type category: %s, returning empty set", category)
		return []FileInfo{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("error querying rules: %w", err)
	}
	defer rows.Close()

	var files []FileInfo

	for rows.Next() {
		var id int
		var name, fileName, md5Hash, ruleType string
		var size int64
		var updatedAt time.Time

		if err := rows.Scan(&id, &name, &fileName, &md5Hash, &size, &ruleType, &updatedAt); err != nil {
			debug.Error("Error scanning rule row: %v", err)
			continue
		}

		files = append(files, FileInfo{
			Name:      fileName,
			MD5Hash:   md5Hash,
			Size:      size,
			FileType:  "rule",
			Category:  ruleType,
			ID:        id,
			Timestamp: updatedAt.Unix(),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rule rows: %w", err)
	}

	return files, nil
}

// GetBinaries retrieves binary versions matching the specified category
func (r *FileRepository) GetBinaries(ctx context.Context, category string) ([]FileInfo, error) {
	// Check if category is valid for binary_type enum
	var query string
	var rows *sql.Rows
	var err error

	if category == "" {
		// If category is empty, return all verified active binaries
		query = `
			SELECT id, file_name, md5_hash, file_size, binary_type, created_at 
			FROM binary_versions 
			WHERE verification_status = 'verified'
			AND is_active = true
		`
		rows, err = r.db.QueryContext(ctx, query)
	} else if category == "hashcat" || category == "john" {
		// Category is a valid enum value, use it for filtering
		query = `
			SELECT id, file_name, md5_hash, file_size, binary_type, created_at 
			FROM binary_versions 
			WHERE binary_type = $1::binary_type
			AND verification_status = 'verified'
			AND is_active = true
		`
		rows, err = r.db.QueryContext(ctx, query, category)
	} else {
		// Category is not a valid enum value, return empty set
		debug.Info("Invalid binary_type category: %s, returning empty set", category)
		return []FileInfo{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("error querying binary versions: %w", err)
	}
	defer rows.Close()

	var files []FileInfo

	for rows.Next() {
		var id int
		var fileName, md5Hash, binaryType string
		var size int64
		var createdAt time.Time

		if err := rows.Scan(&id, &fileName, &md5Hash, &size, &binaryType, &createdAt); err != nil {
			debug.Error("Error scanning binary row: %v", err)
			continue
		}

		files = append(files, FileInfo{
			Name:      fileName,
			MD5Hash:   md5Hash,
			Size:      size,
			FileType:  "binary",
			Category:  binaryType,
			ID:        id,
			Timestamp: createdAt.Unix(),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating binary rows: %w", err)
	}

	return files, nil
}

// GetFiles retrieves all files of the specified types and category
func (r *FileRepository) GetFiles(ctx context.Context, fileTypes []string, category string) ([]FileInfo, error) {
	debug.Info("Retrieving files for types: %v, category: %s", fileTypes, category)

	var files []FileInfo
	var err error
	var typeFiles []FileInfo

	for _, fileType := range fileTypes {
		switch fileType {
		case "wordlist":
			typeFiles, err = r.GetWordlists(ctx, category)
			if err != nil {
				debug.Error("Error getting wordlists: %v", err)
				continue
			}
			files = append(files, typeFiles...)

		case "rule":
			typeFiles, err = r.GetRules(ctx, category)
			if err != nil {
				debug.Error("Error getting rules: %v", err)
				continue
			}
			files = append(files, typeFiles...)

		case "binary":
			typeFiles, err = r.GetBinaries(ctx, category)
			if err != nil {
				debug.Error("Error getting binaries: %v", err)
				continue
			}
			files = append(files, typeFiles...)
		}
	}

	debug.Info("Retrieved %d files from database", len(files))
	return files, nil
}
