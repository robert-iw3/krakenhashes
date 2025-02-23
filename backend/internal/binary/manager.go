package binary

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

const (
	maxDownloadAttempts = 3
	retryDelay          = 5 * time.Second
)

// manager implements the Manager interface
type manager struct {
	store  Store
	config Config
}

// NewManager creates a new binary manager instance
func NewManager(store Store, config Config) (Manager, error) {
	// The data directory initialization is now handled by the config package
	return &manager{
		store:  store,
		config: config,
	}, nil
}

// AddVersion implements Manager.AddVersion
func (m *manager) AddVersion(ctx context.Context, version *BinaryVersion) error {
	// Set initial verification status
	version.VerificationStatus = VerificationStatusPending

	// Create the version record
	if err := m.store.CreateVersion(ctx, version); err != nil {
		return fmt.Errorf("failed to create version record: %w", err)
	}

	// Try downloading the binary with retries
	var lastErr error
	for attempt := 1; attempt <= maxDownloadAttempts; attempt++ {
		debug.Info("Attempting to download binary (attempt %d/%d)", attempt, maxDownloadAttempts)

		if err := m.DownloadBinary(ctx, version); err != nil {
			lastErr = err
			debug.Error("Download attempt %d failed: %v", attempt, err)

			// Check if context is cancelled before retrying
			if ctx.Err() != nil {
				return fmt.Errorf("operation cancelled: %w", ctx.Err())
			}

			if attempt < maxDownloadAttempts {
				debug.Info("Waiting %v before next attempt", retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			break
		}

		// Download successful
		lastErr = nil
		break
	}

	if lastErr != nil {
		// Update version status to failed
		version.VerificationStatus = VerificationStatusFailed
		if updateErr := m.store.UpdateVersion(ctx, version); updateErr != nil {
			debug.Error("failed to update version status for version %d: %v",
				version.ID, updateErr)
		}
		return fmt.Errorf("failed to download binary after %d attempts: %w", maxDownloadAttempts, lastErr)
	}

	// Calculate hash and verify the binary
	filePath := m.getBinaryPath(version)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open binary file: %w", err)
	}
	defer file.Close()

	// Calculate initial hash
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	version.MD5Hash = calculatedHash
	version.VerificationStatus = VerificationStatusVerified
	now := time.Now()
	version.LastVerifiedAt = &now

	// Update version with hash and verification status
	if err := m.store.UpdateVersion(ctx, version); err != nil {
		return fmt.Errorf("failed to update version with calculated hash: %w", err)
	}

	// Verify the binary to ensure integrity
	if err := m.VerifyVersion(ctx, version.ID); err != nil {
		return fmt.Errorf("failed to verify binary integrity: %w", err)
	}

	debug.Info("Successfully added and verified binary version %d with hash %s", version.ID, calculatedHash)
	return nil
}

// GetVersion implements Manager.GetVersion
func (m *manager) GetVersion(ctx context.Context, id int64) (*BinaryVersion, error) {
	return m.store.GetVersion(ctx, id)
}

// ListVersions implements Manager.ListVersions
func (m *manager) ListVersions(ctx context.Context, filters map[string]interface{}) ([]*BinaryVersion, error) {
	return m.store.ListVersions(ctx, filters)
}

// VerifyVersion implements Manager.VerifyVersion
func (m *manager) VerifyVersion(ctx context.Context, id int64) error {
	version, err := m.store.GetVersion(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	// Skip verification for deleted versions
	if version.VerificationStatus == VerificationStatusDeleted {
		return fmt.Errorf("cannot verify deleted version")
	}

	// Check if binary file exists
	filePath := m.getBinaryPath(version)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		debug.Error("Binary file does not exist: %s", filePath)
		version.VerificationStatus = VerificationStatusDeleted
		if updateErr := m.store.UpdateVersion(ctx, version); updateErr != nil {
			debug.Error("Failed to update version status: %v", updateErr)
		}
		return fmt.Errorf("binary file does not exist")
	}

	// Open the binary file
	file, err := os.Open(filePath)
	if err != nil {
		debug.Error("Failed to open binary file: %v", err)
		version.VerificationStatus = VerificationStatusFailed
		if updateErr := m.store.UpdateVersion(ctx, version); updateErr != nil {
			debug.Error("Failed to update version status: %v", updateErr)
		}
		return fmt.Errorf("failed to open binary file: %w", err)
	}
	defer file.Close()

	// Calculate MD5 hash
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		debug.Error("Failed to calculate hash: %v", err)
		version.VerificationStatus = VerificationStatusFailed
		if updateErr := m.store.UpdateVersion(ctx, version); updateErr != nil {
			debug.Error("Failed to update version status: %v", updateErr)
		}
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	calculatedHash := hex.EncodeToString(hash.Sum(nil))

	// Get file info for size verification
	fileInfo, err := file.Stat()
	if err != nil {
		debug.Error("Failed to get file info: %v", err)
		version.VerificationStatus = VerificationStatusFailed
		if updateErr := m.store.UpdateVersion(ctx, version); updateErr != nil {
			debug.Error("Failed to update version status: %v", updateErr)
		}
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Verify file size matches
	if fileInfo.Size() != version.FileSize {
		debug.Error("File size mismatch: expected %d, got %d", version.FileSize, fileInfo.Size())
		version.VerificationStatus = VerificationStatusFailed
		if updateErr := m.store.UpdateVersion(ctx, version); updateErr != nil {
			debug.Error("Failed to update version status: %v", updateErr)
		}
		return fmt.Errorf("file size mismatch: expected %d, got %d", version.FileSize, fileInfo.Size())
	}

	// Verify hash matches
	if calculatedHash != version.MD5Hash {
		debug.Error("Hash mismatch: expected %s, got %s", version.MD5Hash, calculatedHash)
		version.VerificationStatus = VerificationStatusFailed
		if updateErr := m.store.UpdateVersion(ctx, version); updateErr != nil {
			debug.Error("Failed to update version status: %v", updateErr)
		}
		return fmt.Errorf("hash mismatch: expected %s, got %s", version.MD5Hash, calculatedHash)
	}

	// Update verification status
	now := time.Now()
	version.LastVerifiedAt = &now
	version.VerificationStatus = VerificationStatusVerified
	if err := m.store.UpdateVersion(ctx, version); err != nil {
		return fmt.Errorf("failed to update version status: %w", err)
	}

	debug.Info("Successfully verified binary version %d", id)
	return nil
}

// DownloadBinary implements Manager.DownloadBinary
func (m *manager) DownloadBinary(ctx context.Context, version *BinaryVersion) error {
	// Check if file already exists
	filePath := m.getBinaryPath(version)
	if _, err := os.Stat(filePath); err == nil {
		debug.Info("Binary file already exists at %s, skipping download", filePath)
		return nil
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Minute, // Long timeout for large files
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, version.SourceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Ensure binary directory exists
	binDir := filepath.Dir(filePath)
	if err := os.MkdirAll(binDir, 0750); err != nil {
		return fmt.Errorf("failed to create binary directory: %w", err)
	}

	// Create binary file with restricted permissions
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0750)
	if err != nil {
		return fmt.Errorf("failed to create binary file: %w", err)
	}
	defer file.Close()

	// Calculate hash while downloading
	hash := md5.New()
	writer := io.MultiWriter(file, hash)

	// Copy data
	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		// Clean up file on error
		os.Remove(filePath)
		return fmt.Errorf("failed to save binary: %w", err)
	}

	// Update file size
	version.FileSize = written

	// Calculate and store hash
	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	version.MD5Hash = calculatedHash
	debug.Info("Downloaded binary and calculated hash: %s", calculatedHash)

	return nil
}

// DeleteVersion implements Manager.DeleteVersion
func (m *manager) DeleteVersion(ctx context.Context, id int64) error {
	// Get version first to get file path
	version, err := m.store.GetVersion(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	// Delete binary file
	filePath := m.getBinaryPath(version)
	if err := os.Remove(filePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete binary file: %w", err)
		}
		// If file doesn't exist, just log it
		debug.Warning("Binary file already deleted: %s", filePath)
	}

	// Clean up the version directory if it's empty
	versionDir := filepath.Dir(filePath)
	if err := os.Remove(versionDir); err != nil && !os.IsNotExist(err) {
		debug.Warning("Failed to remove empty version directory: %v", err)
	}

	// Update version status to deleted and inactive
	version.IsActive = false
	version.VerificationStatus = VerificationStatusDeleted
	version.LastVerifiedAt = nil // Clear last verification time
	if err := m.store.UpdateVersion(ctx, version); err != nil {
		return fmt.Errorf("failed to update version status: %w", err)
	}

	debug.Info("Successfully deleted binary version %d", id)
	return nil
}

// GetLatestActive implements Manager.GetLatestActive
func (m *manager) GetLatestActive(ctx context.Context, binaryType BinaryType) (*BinaryVersion, error) {
	return m.store.GetLatestActive(ctx, binaryType)
}

// getBinaryPath returns the full path for a binary version
func (m *manager) getBinaryPath(version *BinaryVersion) string {
	return filepath.Join(m.config.DataDir, "binaries", fmt.Sprintf("%d", version.ID), version.FileName)
}
