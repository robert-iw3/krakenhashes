package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
)

// FileSync handles synchronization of files between the agent and backend
type FileSync struct {
	client     *http.Client
	urlConfig  *config.URLConfig
	dataDirs   *config.DataDirs
	sem        chan struct{} // Semaphore for limiting concurrent downloads
	maxRetries int           // Maximum number of retries for downloads
}

// Config holds configuration for file synchronization
type Config struct {
	MaxConcurrentDownloads int
	DownloadTimeout        time.Duration
	MaxRetries             int
}

// FileInfo represents information about a file for synchronization
type FileInfo struct {
	Name     string `json:"name"`
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	FileType string `json:"file_type"` // "wordlist", "rule", "binary", "hashlist"
}

// NewFileSync creates a new file synchronization handler
func NewFileSync(urlConfig *config.URLConfig, dataDirs *config.DataDirs) (*FileSync, error) {
	maxDownloads, _ := strconv.Atoi(getEnvOrDefault("KH_MAX_CONCURRENT_DOWNLOADS", "3"))
	timeout, _ := time.ParseDuration(getEnvOrDefault("KH_DOWNLOAD_TIMEOUT", "1h"))
	maxRetries, _ := strconv.Atoi(getEnvOrDefault("KH_MAX_DOWNLOAD_RETRIES", "3"))

	debug.Info("Initializing file sync with max downloads: %d, timeout: %s, max retries: %d",
		maxDownloads, timeout, maxRetries)

	client := &http.Client{
		Timeout: timeout,
	}

	return &FileSync{
		client:     client,
		urlConfig:  urlConfig,
		dataDirs:   dataDirs,
		sem:        make(chan struct{}, maxDownloads),
		maxRetries: maxRetries,
	}, nil
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// DownloadFile downloads a file from the backend and saves it to the appropriate directory
func (fs *FileSync) DownloadFile(ctx context.Context, fileType, filename, expectedHash string) error {
	return fs.DownloadFileWithRetry(ctx, fileType, filename, expectedHash, 0)
}

// DownloadFileWithRetry downloads a file with retry logic
func (fs *FileSync) DownloadFileWithRetry(ctx context.Context, fileType, filename, expectedHash string, retryCount int) error {
	if expectedHash == "" {
		debug.Error("Hash verification required but no hash provided for %s", filename)
		return fmt.Errorf("hash verification required but no hash provided")
	}

	// Get target directory based on file type
	var targetDir string
	switch fileType {
	case "wordlist":
		targetDir = fs.dataDirs.Wordlists
	case "rule":
		targetDir = fs.dataDirs.Rules
	case "hashlist":
		targetDir = fs.dataDirs.Hashlists
	case "binary":
		targetDir = fs.dataDirs.Binaries
	default:
		debug.Error("Unsupported file type: %s", fileType)
		return fmt.Errorf("unsupported file type: %s", fileType)
	}

	// Acquire semaphore slot
	select {
	case fs.sem <- struct{}{}:
		defer func() { <-fs.sem }()
	case <-ctx.Done():
		debug.Error("Context cancelled while waiting for download slot: %v", ctx.Err())
		return ctx.Err()
	}

	// Create target file path
	targetPath := filepath.Join(targetDir, filename)
	tempPath := targetPath + ".tmp"

	debug.Info("Starting download of %s to %s (attempt %d/%d)",
		filename, targetPath, retryCount+1, fs.maxRetries+1)

	// Create temporary file
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		debug.Error("Failed to create temporary file %s: %v", tempPath, err)
		return fs.retryOrFail(ctx, fileType, filename, expectedHash, retryCount,
			fmt.Errorf("failed to create temporary file: %w", err))
	}
	defer os.Remove(tempPath) // Clean up temp file on error

	// Create download URL
	url := fmt.Sprintf("%s/api/files/%s/%s", fs.urlConfig.BaseURL, fileType, filename)
	debug.Info("Downloading file from %s", url)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		debug.Error("Failed to create request for %s: %v", url, err)
		return fs.retryOrFail(ctx, fileType, filename, expectedHash, retryCount,
			fmt.Errorf("failed to create request: %w", err))
	}

	// Send request
	resp, err := fs.client.Do(req)
	if err != nil {
		debug.Error("Failed to download %s: %v", filename, err)
		return fs.retryOrFail(ctx, fileType, filename, expectedHash, retryCount,
			fmt.Errorf("failed to download file: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		debug.Error("Server returned status %d for %s", resp.StatusCode, filename)
		return fs.retryOrFail(ctx, fileType, filename, expectedHash, retryCount,
			fmt.Errorf("server returned status %d", resp.StatusCode))
	}

	// Calculate hash while downloading
	hash := sha256.New()
	writer := io.MultiWriter(tempFile, hash)

	// Copy data with progress reporting
	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		debug.Error("Failed to save %s: %v", filename, err)
		return fs.retryOrFail(ctx, fileType, filename, expectedHash, retryCount,
			fmt.Errorf("failed to save file: %w", err))
	}

	// Close temp file before moving
	tempFile.Close()

	// Always verify hash
	actualHash := hex.EncodeToString(hash.Sum(nil))
	if actualHash != expectedHash {
		debug.Error("Hash mismatch for %s: expected %s, got %s", filename, expectedHash, actualHash)
		return fs.retryOrFail(ctx, fileType, filename, expectedHash, retryCount,
			fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash))
	}
	debug.Info("File hash verified successfully for %s", filename)

	// Move temp file to final location
	if err := os.Rename(tempPath, targetPath); err != nil {
		debug.Error("Failed to move %s to final location: %v", filename, err)
		return fs.retryOrFail(ctx, fileType, filename, expectedHash, retryCount,
			fmt.Errorf("failed to move file to final location: %w", err))
	}

	debug.Info("Successfully downloaded %s (%d bytes)", filename, written)
	return nil
}

// retryOrFail handles retry logic for downloads
func (fs *FileSync) retryOrFail(ctx context.Context, fileType, filename, expectedHash string,
	retryCount int, err error) error {

	if retryCount >= fs.maxRetries {
		debug.Error("Max retries reached for %s, giving up: %v", filename, err)
		return fmt.Errorf("max retries reached: %w", err)
	}

	nextRetry := retryCount + 1
	backoff := time.Duration(nextRetry*nextRetry) * time.Second
	debug.Info("Retrying download of %s in %v (attempt %d/%d)",
		filename, backoff, nextRetry+1, fs.maxRetries+1)

	select {
	case <-time.After(backoff):
		return fs.DownloadFileWithRetry(ctx, fileType, filename, expectedHash, nextRetry)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SyncDirectory synchronizes all files of a given type with the backend
func (fs *FileSync) SyncDirectory(ctx context.Context, fileType string) error {
	url := fmt.Sprintf("%s/api/files/%s/list", fs.urlConfig.BaseURL, fileType)
	debug.Info("Fetching file list from %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		debug.Error("Failed to create request for file list: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := fs.client.Do(req)
	if err != nil {
		debug.Error("Failed to fetch file list from %s: %v", url, err)
		return fmt.Errorf("failed to fetch file list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		debug.Error("Server returned non-200 status %d when fetching file list from %s", resp.StatusCode, url)
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	type FileInfo struct {
		Name string `json:"name"`
		Hash string `json:"hash"`
	}

	var files []FileInfo
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		debug.Error("Failed to decode file list response: %v", err)
		return fmt.Errorf("failed to decode file list: %w", err)
	}

	debug.Info("Found %d files to sync for type %s", len(files), fileType)

	var wg sync.WaitGroup
	errors := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)
		go func(file FileInfo) {
			defer wg.Done()
			if err := fs.DownloadFile(ctx, fileType, file.Name, file.Hash); err != nil {
				debug.Error("Failed to download %s/%s: %v", fileType, file.Name, err)
				errors <- fmt.Errorf("failed to download %s: %w", file.Name, err)
			} else {
				debug.Info("Successfully synchronized %s/%s", fileType, file.Name)
			}
		}(file)
	}

	// Wait for all downloads to complete
	wg.Wait()
	close(errors)

	// Collect any errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		debug.Error("Encountered %d errors while syncing %s directory", len(errs), fileType)
		for _, err := range errs {
			debug.Error("Sync error: %v", err)
		}
		return fmt.Errorf("encountered %d errors during sync", len(errs))
	}

	debug.Info("Successfully synchronized %s directory", fileType)
	return nil
}

// GetFileTypeDir returns the directory path for a given file type
func (fs *FileSync) GetFileTypeDir(fileType string) (string, error) {
	switch fileType {
	case "wordlist":
		return fs.dataDirs.Wordlists, nil
	case "rule":
		return fs.dataDirs.Rules, nil
	case "hashlist":
		return fs.dataDirs.Hashlists, nil
	case "binary":
		return fs.dataDirs.Binaries, nil
	default:
		return "", fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// ScanDirectory scans a directory and returns information about all files
func (fs *FileSync) ScanDirectory(fileType string) ([]FileInfo, error) {
	dir, err := fs.GetFileTypeDir(fileType)
	if err != nil {
		return nil, err
	}

	debug.Info("Scanning directory %s for %s files", dir, fileType)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0750); err != nil {
		debug.Error("Failed to create directory %s: %v", dir, err)
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	var files []FileInfo

	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			debug.Error("Error accessing path %s: %v", path, err)
			return nil // Continue walking despite errors
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			debug.Error("Error getting file info for %s: %v", path, err)
			return nil // Continue walking despite errors
		}

		// Calculate hash
		hash, err := fs.CalculateFileHash(path)
		if err != nil {
			debug.Error("Error calculating hash for %s: %v", path, err)
			return nil // Continue walking despite errors
		}

		// Add file info to list
		files = append(files, FileInfo{
			Name:     d.Name(),
			Hash:     hash,
			Size:     info.Size(),
			FileType: fileType,
		})

		return nil
	})

	if err != nil {
		debug.Error("Error walking directory %s: %v", dir, err)
		return nil, fmt.Errorf("error scanning directory: %w", err)
	}

	debug.Info("Found %d files in %s directory", len(files), fileType)
	return files, nil
}

// CalculateFileHash calculates the SHA-256 hash of a file
func (fs *FileSync) CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ScanAllDirectories scans all data directories and returns information about all files
func (fs *FileSync) ScanAllDirectories(fileTypes []string) (map[string][]FileInfo, error) {
	result := make(map[string][]FileInfo)

	for _, fileType := range fileTypes {
		files, err := fs.ScanDirectory(fileType)
		if err != nil {
			debug.Error("Error scanning %s directory: %v", fileType, err)
			continue // Continue with other directories despite errors
		}
		result[fileType] = files
	}

	return result, nil
}
