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
	client    *http.Client
	urlConfig *config.URLConfig
	dataDirs  *config.DataDirs
	sem       chan struct{} // Semaphore for limiting concurrent downloads
}

// Config holds configuration for file synchronization
type Config struct {
	MaxConcurrentDownloads int
	DownloadTimeout        time.Duration
}

// NewFileSync creates a new file synchronization handler
func NewFileSync(urlConfig *config.URLConfig, dataDirs *config.DataDirs) (*FileSync, error) {
	maxDownloads, _ := strconv.Atoi(getEnvOrDefault("KH_MAX_CONCURRENT_DOWNLOADS", "3"))
	timeout, _ := time.ParseDuration(getEnvOrDefault("KH_DOWNLOAD_TIMEOUT", "1h"))

	debug.Info("Initializing file sync with max downloads: %d, timeout: %s", maxDownloads, timeout)

	client := &http.Client{
		Timeout: timeout,
	}

	return &FileSync{
		client:    client,
		urlConfig: urlConfig,
		dataDirs:  dataDirs,
		sem:       make(chan struct{}, maxDownloads),
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

	debug.Info("Starting download of %s to %s", filename, targetPath)

	// Create temporary file
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		debug.Error("Failed to create temporary file %s: %v", tempPath, err)
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempPath) // Clean up temp file on error

	// Create download URL
	url := fmt.Sprintf("%s/api/files/%s/%s", fs.urlConfig.BaseURL, fileType, filename)
	debug.Info("Downloading file from %s", url)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		debug.Error("Failed to create request for %s: %v", url, err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	resp, err := fs.client.Do(req)
	if err != nil {
		debug.Error("Failed to download %s: %v", filename, err)
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		debug.Error("Server returned status %d for %s", resp.StatusCode, filename)
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Calculate hash while downloading
	hash := sha256.New()
	writer := io.MultiWriter(tempFile, hash)

	// Copy data with progress reporting
	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		debug.Error("Failed to save %s: %v", filename, err)
		return fmt.Errorf("failed to save file: %w", err)
	}

	// Close temp file before moving
	tempFile.Close()

	// Always verify hash
	actualHash := hex.EncodeToString(hash.Sum(nil))
	if actualHash != expectedHash {
		debug.Error("Hash mismatch for %s: expected %s, got %s", filename, expectedHash, actualHash)
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	debug.Info("File hash verified successfully for %s", filename)

	// Move temp file to final location
	if err := os.Rename(tempPath, targetPath); err != nil {
		debug.Error("Failed to move %s to final location: %v", filename, err)
		return fmt.Errorf("failed to move file to final location: %w", err)
	}

	debug.Info("Successfully downloaded %s (%d bytes)", filename, written)
	return nil
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
