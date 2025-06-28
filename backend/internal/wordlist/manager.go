package wordlist

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/fsutil"
	"github.com/google/uuid"
)

// Manager handles wordlist operations
type Manager interface {
	ListWordlists(ctx context.Context, filters map[string]interface{}) ([]*models.Wordlist, error)
	GetWordlist(ctx context.Context, id int) (*models.Wordlist, error)
	GetWordlistByFilename(ctx context.Context, filename string) (*models.Wordlist, error)
	GetWordlistByMD5Hash(ctx context.Context, md5Hash string) (*models.Wordlist, error)
	AddWordlist(ctx context.Context, req *models.WordlistAddRequest, userID uuid.UUID) (*models.Wordlist, error)
	UpdateWordlist(ctx context.Context, id int, req *models.WordlistUpdateRequest, userID uuid.UUID) (*models.Wordlist, error)
	DeleteWordlist(ctx context.Context, id int) error
	VerifyWordlist(ctx context.Context, id int, req *models.WordlistVerifyRequest) error
	UpdateWordlistFileInfo(ctx context.Context, id int, md5Hash string, fileSize int64) error
	AddWordlistTag(ctx context.Context, id int, tag string, userID uuid.UUID) error
	DeleteWordlistTag(ctx context.Context, id int, tag string) error
	GetWordlistPath(filename string, wordlistType string) string
	CountWordsInFile(filepath string) (int64, error)
	CalculateFileMD5(filepath string) (string, error)
}

// Store defines the interface for wordlist data storage operations
type WordlistStore interface {
	// Wordlist operations
	ListWordlists(ctx context.Context, filters map[string]interface{}) ([]*models.Wordlist, error)
	GetWordlist(ctx context.Context, id int) (*models.Wordlist, error)
	GetWordlistByFilename(ctx context.Context, filename string) (*models.Wordlist, error)
	GetWordlistByMD5Hash(ctx context.Context, md5Hash string) (*models.Wordlist, error)
	CreateWordlist(ctx context.Context, wordlist *models.Wordlist) error
	UpdateWordlist(ctx context.Context, wordlist *models.Wordlist) error
	DeleteWordlist(ctx context.Context, id int) error
	UpdateWordlistVerification(ctx context.Context, id int, status string, wordCount *int64) error
	UpdateWordlistFileInfo(ctx context.Context, id int, md5Hash string, fileSize int64) error

	// Tag operations
	GetWordlistTags(ctx context.Context, id int) ([]string, error)
	AddWordlistTag(ctx context.Context, id int, tag string, userID uuid.UUID) error
	DeleteWordlistTag(ctx context.Context, id int, tag string) error
}

type manager struct {
	store            WordlistStore
	wordlistsDir     string
	maxUploadSize    int64
	allowedFormats   []string
	allowedMimeTypes []string
}

// NewManager creates a new wordlist manager
func NewManager(store WordlistStore, wordlistsDir string, maxUploadSize int64, allowedFormats, allowedMimeTypes []string) Manager {
	// Ensure wordlists directory exists
	if err := os.MkdirAll(wordlistsDir, 0755); err != nil {
		debug.Error("Failed to create wordlists directory: %v", err)
		panic(err)
	}

	return &manager{
		store:            store,
		wordlistsDir:     wordlistsDir,
		maxUploadSize:    maxUploadSize,
		allowedFormats:   allowedFormats,
		allowedMimeTypes: allowedMimeTypes,
	}
}

// ListWordlists retrieves all wordlists with optional filtering
func (m *manager) ListWordlists(ctx context.Context, filters map[string]interface{}) ([]*models.Wordlist, error) {
	return m.store.ListWordlists(ctx, filters)
}

// GetWordlist retrieves a wordlist by ID
func (m *manager) GetWordlist(ctx context.Context, id int) (*models.Wordlist, error) {
	return m.store.GetWordlist(ctx, id)
}

// GetWordlistByFilename retrieves a wordlist by filename
func (m *manager) GetWordlistByFilename(ctx context.Context, filename string) (*models.Wordlist, error) {
	return m.store.GetWordlistByFilename(ctx, filename)
}

// GetWordlistByMD5Hash retrieves a wordlist by MD5 hash
func (m *manager) GetWordlistByMD5Hash(ctx context.Context, md5Hash string) (*models.Wordlist, error) {
	return m.store.GetWordlistByMD5Hash(ctx, md5Hash)
}

// AddWordlist adds a new wordlist
func (m *manager) AddWordlist(ctx context.Context, req *models.WordlistAddRequest, userID uuid.UUID) (*models.Wordlist, error) {
	// Create wordlist model
	wordlist := &models.Wordlist{
		Name:               req.Name,
		Description:        req.Description,
		WordlistType:       req.WordlistType,
		Format:             req.Format,
		FileName:           req.FileName,
		MD5Hash:            req.MD5Hash,
		FileSize:           req.FileSize,
		WordCount:          req.WordCount,
		CreatedBy:          userID,
		VerificationStatus: "pending",
		Tags:               req.Tags,
	}

	// Create wordlist in database
	if err := m.store.CreateWordlist(ctx, wordlist); err != nil {
		return nil, err
	}

	return wordlist, nil
}

// UpdateWordlist updates an existing wordlist
func (m *manager) UpdateWordlist(ctx context.Context, id int, req *models.WordlistUpdateRequest, userID uuid.UUID) (*models.Wordlist, error) {
	// Get existing wordlist
	wordlist, err := m.store.GetWordlist(ctx, id)
	if err != nil {
		return nil, err
	}
	if wordlist == nil {
		return nil, fmt.Errorf("wordlist not found")
	}

	// Update fields
	wordlist.Name = req.Name
	wordlist.Description = req.Description
	wordlist.WordlistType = req.WordlistType

	// Only update format if provided
	if req.Format != "" {
		wordlist.Format = req.Format
	}

	wordlist.UpdatedBy = userID

	// Update in database
	if err := m.store.UpdateWordlist(ctx, wordlist); err != nil {
		return nil, err
	}

	// Handle tags
	if req.Tags != nil {
		// Get current tags
		currentTags, err := m.store.GetWordlistTags(ctx, id)
		if err != nil {
			return nil, err
		}

		// Add new tags
		for _, tag := range req.Tags {
			found := false
			for _, currentTag := range currentTags {
				if tag == currentTag {
					found = true
					break
				}
			}
			if !found {
				if err := m.store.AddWordlistTag(ctx, id, tag, userID); err != nil {
					return nil, err
				}
			}
		}

		// Remove tags that are no longer present
		for _, currentTag := range currentTags {
			found := false
			for _, tag := range req.Tags {
				if currentTag == tag {
					found = true
					break
				}
			}
			if !found {
				if err := m.store.DeleteWordlistTag(ctx, id, currentTag); err != nil {
					return nil, err
				}
			}
		}

		// Update tags in wordlist object
		wordlist.Tags = req.Tags
	}

	return wordlist, nil
}

// DeleteWordlist deletes a wordlist
func (m *manager) DeleteWordlist(ctx context.Context, id int) error {
	// Get wordlist to find filename
	wordlist, err := m.store.GetWordlist(ctx, id)
	if err != nil {
		return err
	}
	if wordlist == nil {
		return fmt.Errorf("wordlist not found")
	}

	// Delete from database
	if err := m.store.DeleteWordlist(ctx, id); err != nil {
		return err
	}

	// Delete file
	filePath := filepath.Join(m.wordlistsDir, wordlist.FileName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		debug.Error("Failed to delete wordlist file %s: %v", filePath, err)
		// Don't return error, as the database entry is already deleted
	}

	return nil
}

// VerifyWordlist updates a wordlist's verification status
func (m *manager) VerifyWordlist(ctx context.Context, id int, req *models.WordlistVerifyRequest) error {
	// Get wordlist
	wordlist, err := m.store.GetWordlist(ctx, id)
	if err != nil {
		return err
	}
	if wordlist == nil {
		return fmt.Errorf("wordlist not found")
	}

	// If status is "verified" and word count is not provided, calculate it
	if req.Status == "verified" && req.WordCount == nil {
		filePath := filepath.Join(m.wordlistsDir, wordlist.FileName)
		wordCount, err := m.CountWordsInFile(filePath)
		if err != nil {
			debug.Error("Failed to count words in file %s: %v", filePath, err)
			return err
		}
		req.WordCount = &wordCount
	}

	// Update verification status
	return m.store.UpdateWordlistVerification(ctx, id, req.Status, req.WordCount)
}

// UpdateWordlistFileInfo updates a wordlist's file information (MD5 hash and file size)
func (m *manager) UpdateWordlistFileInfo(ctx context.Context, id int, md5Hash string, fileSize int64) error {
	return m.store.UpdateWordlistFileInfo(ctx, id, md5Hash, fileSize)
}

// AddWordlistTag adds a tag to a wordlist
func (m *manager) AddWordlistTag(ctx context.Context, id int, tag string, userID uuid.UUID) error {
	return m.store.AddWordlistTag(ctx, id, tag, userID)
}

// DeleteWordlistTag deletes a tag from a wordlist
func (m *manager) DeleteWordlistTag(ctx context.Context, id int, tag string) error {
	return m.store.DeleteWordlistTag(ctx, id, tag)
}

// GetWordlistPath returns the full path to a wordlist file
func (m *manager) GetWordlistPath(filename string, wordlistType string) string {
	// Check if the filename already contains a subdirectory
	if strings.Contains(filename, string(filepath.Separator)) {
		return filepath.Join(m.wordlistsDir, filename)
	}

	// If no wordlist type is provided, use a default
	if wordlistType == "" {
		wordlistType = "general" // Default type
	} else {
		// Normalize wordlist type
		wordlistType = strings.ToLower(wordlistType)
		// Ensure it's one of the valid types
		switch wordlistType {
		case "general", "specialized", "targeted", "custom":
			// Valid type, use as is
		default:
			// Invalid type, use default
			wordlistType = "general"
		}
	}

	// Place in appropriate subdirectory
	return filepath.Join(m.wordlistsDir, wordlistType, filename)
}

// CountWordsInFile counts the number of words in a file
func (m *manager) CountWordsInFile(filepath string) (int64, error) {
	debug.Info("CountWordsInFile: Starting word count for file: %s", filepath)

	// Get file info for size
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		debug.Error("CountWordsInFile: Failed to get file info: %v", err)
		return 0, err
	}

	// Check if the file is compressed
	ext := strings.ToLower(path.Ext(filepath))
	if ext == ".gz" || ext == ".zip" {
		debug.Info("CountWordsInFile: Detected compressed file (%s), using estimation method", ext)

		// For compressed files, we'll use an estimation based on file size
		// This is much faster than decompressing and counting lines

		// Estimate word count based on file size and compression ratio
		// For text files, compression typically achieves 3:1 to 4:1 ratio
		// Assuming average word length of 8 bytes plus newline character
		// and a compression ratio of approximately 3.5:1
		estimatedCount := int64(float64(fileInfo.Size()) * 3.5 / 9)
		debug.Info("CountWordsInFile: Estimated %d words in compressed file (size: %d bytes)",
			estimatedCount, fileInfo.Size())
		return estimatedCount, nil
	}

	// For large text files (over 1GB), use a more efficient counting method
	if fileInfo.Size() > 1024*1024*1024 {
		debug.Info("CountWordsInFile: Large text file detected (%d bytes), using optimized counting method",
			fileInfo.Size())

		// Use a buffered reader with a large buffer size for better performance
		file, err := os.Open(filepath)
		if err != nil {
			debug.Error("CountWordsInFile: Failed to open file: %v", err)
			return 0, err
		}
		defer file.Close()

		// Use a 16MB buffer for large files
		const bufferSize = 16 * 1024 * 1024
		reader := bufio.NewReaderSize(file, bufferSize)

		var count int64
		var buf [4096]byte

		for {
			c, err := reader.Read(buf[:])
			if err != nil {
				if err == io.EOF {
					break
				}
				debug.Error("CountWordsInFile: Error reading file: %v", err)
				return 0, err
			}

			// Count newlines in the buffer
			for i := 0; i < c; i++ {
				if buf[i] == '\n' {
					count++
				}
			}
		}

		// Add 1 if the file doesn't end with a newline
		if count > 0 {
			lastByte := make([]byte, 1)
			if _, err := file.ReadAt(lastByte, fileInfo.Size()-1); err == nil {
				if lastByte[0] != '\n' {
					count++
				}
			}
		}

		debug.Info("CountWordsInFile: Counted %d lines in large text file", count)
		return count, nil
	}

	// For regular text files, use the standard line counting method
	debug.Info("CountWordsInFile: Counting lines in text file")
	return fsutil.CountLinesInFile(filepath)
}

// CalculateFileMD5 calculates the MD5 hash of a file
func (m *manager) CalculateFileMD5(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
