package sync

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileSync(t *testing.T) {
	// Create test directories
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	dataDir := filepath.Join(tempDir, "data")
	
	// Create a test CA certificate
	err := os.MkdirAll(configDir, 0700)
	require.NoError(t, err)
	
	// Create a dummy CA cert file (not valid, but enough for testing)
	certPath := filepath.Join(configDir, "ca.crt")
	testCert := `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHHGwEAACfPMA0GCSqGSIb3DQEBCwUAMBYxFDASBgNVBAMMC0Rl
dmVsb3BtZW50MB4XDTE2MDEwMTAwMDAwMFoXDTI2MDEwMTAwMDAwMFowFjEUMBIG
A1UEAwwLRGV2ZWxvcG1lbnQwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBALdm
2rLb1UUz7m8fHO/v3RLWV8RFf2NdPcFV2LiEPYu6H9n7VpPm0j0Gj4ifmR0p8PsL
kwDB+6q0WqRNkkkDrYcNwDy5LgDuOl9g2LR1yNpwGo6BpMqVYVCUcwDPAoGBAK2X
-----END CERTIFICATE-----`
	err = os.WriteFile(certPath, []byte(testCert), 0600)
	require.NoError(t, err)
	
	// Set environment to use test config dir
	t.Setenv("KH_CONFIG_DIR", configDir)
	
	urlConfig := &config.URLConfig{
		BaseURL:      "https://localhost:8080",
		WebSocketURL: "wss://localhost:8080",
		HTTPPort:     "8081",
	}
	
	dataDirs := &config.DataDirs{
		Binaries:  filepath.Join(dataDir, "binaries"),
		Wordlists: filepath.Join(dataDir, "wordlists"),
		Rules:     filepath.Join(dataDir, "rules"),
		Hashlists: filepath.Join(dataDir, "hashlists"),
	}
	
	_, err = NewFileSync(urlConfig, dataDirs, "test-api-key", "agent-123")
	// Will fail due to invalid cert, but that's expected
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse CA certificate")
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "environment variable set",
			key:          "TEST_ENV_VAR",
			envValue:     "custom_value",
			defaultValue: "default",
			expected:     "custom_value",
		},
		{
			name:         "environment variable not set",
			key:          "NONEXISTENT_VAR",
			envValue:     "",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "empty environment variable",
			key:          "EMPTY_VAR",
			envValue:     "",
			defaultValue: "fallback",
			expected:     "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			
			result := getEnvOrDefault(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileInfo_Structure(t *testing.T) {
	fileInfo := FileInfo{
		Name:      "rockyou.txt",
		MD5Hash:   "9076652d92ae99b7f064d351ff7b15cd",
		Size:      139921507,
		FileType:  "wordlist",
		Category:  "general",
		ID:        1,
		Timestamp: time.Now().Unix(),
	}

	assert.Equal(t, "rockyou.txt", fileInfo.Name)
	assert.Equal(t, "9076652d92ae99b7f064d351ff7b15cd", fileInfo.MD5Hash)
	assert.Equal(t, int64(139921507), fileInfo.Size)
	assert.Equal(t, "wordlist", fileInfo.FileType)
	assert.Equal(t, "general", fileInfo.Category)
	assert.Equal(t, 1, fileInfo.ID)
	assert.Greater(t, fileInfo.Timestamp, int64(0))
}

func TestCalculateFileHash(t *testing.T) {
	// Create a test file sync instance with minimal setup
	fs := &FileSync{}
	
	// Create a test file
	tempFile := filepath.Join(t.TempDir(), "test.txt")
	testContent := []byte("Hello, World!")
	err := os.WriteFile(tempFile, testContent, 0644)
	require.NoError(t, err)
	
	// Calculate hash
	hash, err := fs.CalculateFileHash(tempFile)
	assert.NoError(t, err)
	
	// Verify hash
	expectedHash := md5.Sum(testContent)
	expectedHashStr := hex.EncodeToString(expectedHash[:])
	assert.Equal(t, expectedHashStr, hash)
	
	// Test non-existent file
	_, err = fs.CalculateFileHash("/nonexistent/file")
	assert.Error(t, err)
}

func TestGetFileTypeDir(t *testing.T) {
	dataDir := t.TempDir()
	dataDirs := &config.DataDirs{
		Binaries:            filepath.Join(dataDir, "binaries"),
		Wordlists:           filepath.Join(dataDir, "wordlists"),
		Rules:               filepath.Join(dataDir, "rules"),
		Hashlists:           filepath.Join(dataDir, "hashlists"),
		WordlistGeneral:     filepath.Join(dataDir, "wordlists", "general"),
		WordlistSpecialized: filepath.Join(dataDir, "wordlists", "specialized"),
		WordlistTargeted:    filepath.Join(dataDir, "wordlists", "targeted"),
		WordlistCustom:      filepath.Join(dataDir, "wordlists", "custom"),
		RuleHashcat:         filepath.Join(dataDir, "rules", "hashcat"),
		RuleJohn:            filepath.Join(dataDir, "rules", "john"),
		RuleCustom:          filepath.Join(dataDir, "rules", "custom"),
	}
	
	fs := &FileSync{
		dataDirs: dataDirs,
	}
	
	tests := []struct {
		name     string
		fileType string
		expected string
		wantErr  bool
	}{
		{
			name:     "binary type",
			fileType: "binary",
			expected: dataDirs.Binaries,
			wantErr:  false,
		},
		{
			name:     "wordlist type",
			fileType: "wordlist",
			expected: dataDirs.Wordlists,
			wantErr:  false,
		},
		{
			name:     "rule type",
			fileType: "rule",
			expected: dataDirs.Rules,
			wantErr:  false,
		},
		{
			name:     "hashlist type",
			fileType: "hashlist",
			expected: dataDirs.Hashlists,
			wantErr:  false,
		},
		{
			name:     "invalid type",
			fileType: "invalid",
			expected: "",
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := fs.GetFileTypeDir(tt.fileType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, dir)
			}
		})
	}
}



func TestScanDirectory(t *testing.T) {
	dataDir := t.TempDir()
	dataDirs := &config.DataDirs{
		Wordlists: filepath.Join(dataDir, "wordlists"),
		Rules:     filepath.Join(dataDir, "rules"),
		Binaries:  filepath.Join(dataDir, "binaries"),
		Hashlists: filepath.Join(dataDir, "hashlists"),
	}
	
	fs := &FileSync{
		dataDirs: dataDirs,
	}
	
	// Create test files
	wordlistDir := filepath.Join(dataDir, "wordlists")
	err := os.MkdirAll(wordlistDir, 0750)
	require.NoError(t, err)
	
	// Create test wordlist files
	testFiles := []struct {
		name    string
		content string
	}{
		{"rockyou.txt", "password\n123456\nadmin\n"},
		{"common.txt", "test\nuser\n"},
	}
	
	for _, tf := range testFiles {
		path := filepath.Join(wordlistDir, tf.name)
		err := os.WriteFile(path, []byte(tf.content), 0644)
		require.NoError(t, err)
	}
	
	// Scan directory
	files, err := fs.ScanDirectory("wordlist")
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	
	// Verify file info
	for _, file := range files {
		assert.NotEmpty(t, file.Name)
		assert.NotEmpty(t, file.MD5Hash)
		assert.Greater(t, file.Size, int64(0))
		assert.Equal(t, "wordlist", file.FileType)
	}
	
	// Test scanning empty directory
	emptyDir := filepath.Join(dataDir, "rules")
	err = os.MkdirAll(emptyDir, 0750)
	require.NoError(t, err)
	
	files, err = fs.ScanDirectory("rule")
	assert.NoError(t, err)
	assert.Empty(t, files)
	
	// Test invalid file type
	_, err = fs.ScanDirectory("invalid")
	assert.Error(t, err)
}

func TestFileSyncSemaphore(t *testing.T) {
	// Test that the semaphore properly limits concurrent operations
	fs := &FileSync{
		sem: make(chan struct{}, 2), // Allow 2 concurrent operations
	}
	
	// Track concurrent operations
	var activeOps int32
	var maxConcurrent int32
	
	// Simulate multiple concurrent operations
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Acquire semaphore
			fs.sem <- struct{}{}
			defer func() { <-fs.sem }()
			
			// Track concurrent operations
			current := atomic.AddInt32(&activeOps, 1)
			for {
				max := atomic.LoadInt32(&maxConcurrent)
				if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
					break
				}
			}
			
			// Simulate work
			time.Sleep(50 * time.Millisecond)
			
			atomic.AddInt32(&activeOps, -1)
		}()
	}
	
	wg.Wait()
	
	// Verify concurrency was limited
	assert.LessOrEqual(t, maxConcurrent, int32(2), "Concurrent operations should be limited by semaphore")
}