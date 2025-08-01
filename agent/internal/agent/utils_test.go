package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigDir(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		validatePath func(t *testing.T, path string)
	}{
		{
			name: "custom cert directory from environment",
			envVars: map[string]string{
				"KH_CERT_DIR": t.TempDir(),
			},
			validatePath: func(t *testing.T, path string) {
				assert.DirExists(t, path)
				assert.True(t, filepath.IsAbs(path))
			},
		},
		{
			name: "relative path cert directory",
			envVars: map[string]string{
				"KH_CERT_DIR": "relative/cert/path",
			},
			validatePath: func(t *testing.T, path string) {
				// Should be converted to absolute path
				assert.True(t, filepath.IsAbs(path))
				assert.Contains(t, path, "relative/cert/path")
			},
		},
		{
			name: "absolute path cert directory",
			envVars: map[string]string{
				"KH_CERT_DIR": filepath.Join(t.TempDir(), "absolute", "certs"),
			},
			validatePath: func(t *testing.T, path string) {
				assert.True(t, filepath.IsAbs(path))
				assert.DirExists(t, path)
			},
		},
		{
			name:    "default to home directory",
			envVars: map[string]string{},
			validatePath: func(t *testing.T, path string) {
				home, err := os.UserHomeDir()
				if err == nil {
					expectedPath := filepath.Join(home, ".krakenhashes")
					assert.Equal(t, expectedPath, path)
				} else {
					// Fallback to current directory
					assert.Contains(t, path, ".krakenhashes")
				}
			},
		},
		{
			name: "special characters in path",
			envVars: map[string]string{
				"KH_CERT_DIR": filepath.Join(t.TempDir(), "path with spaces", "cert-dir"),
			},
			validatePath: func(t *testing.T, path string) {
				assert.DirExists(t, path)
				assert.Contains(t, path, "path with spaces")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			// Call getConfigDir
			configDir := getConfigDir()
			assert.NotEmpty(t, configDir)

			if tt.validatePath != nil {
				tt.validatePath(t, configDir)
			}

			// Verify directory exists and has correct permissions
			info, err := os.Stat(configDir)
			if err == nil {
				assert.True(t, info.IsDir())
				// Don't check exact permissions as they can vary based on umask
			}
		})
	}
}

func TestGetConfigDir_DirectoryCreation(t *testing.T) {
	// Test that getConfigDir creates the directory if it doesn't exist
	baseDir := t.TempDir()
	nonExistentDir := filepath.Join(baseDir, "new", "cert", "directory")
	
	t.Setenv("KH_CERT_DIR", nonExistentDir)

	// Directory shouldn't exist yet
	_, err := os.Stat(nonExistentDir)
	assert.True(t, os.IsNotExist(err))

	// Call getConfigDir
	configDir := getConfigDir()
	assert.Equal(t, nonExistentDir, configDir)

	// Directory should now exist
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

func TestGetConfigDir_PermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Cannot test permission errors as root")
	}

	// Create a directory with no write permissions
	baseDir := t.TempDir()
	readOnlyDir := filepath.Join(baseDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0500)
	require.NoError(t, err)

	certPath := filepath.Join(readOnlyDir, "certs")
	t.Setenv("KH_CERT_DIR", certPath)

	// Call getConfigDir - it should try to create the directory but fail
	configDir := getConfigDir()
	
	// Should return the intended path even if creation failed
	assert.Equal(t, certPath, configDir)
	
	// Directory should not exist due to permission error
	_, err = os.Stat(certPath)
	assert.True(t, os.IsNotExist(err))
}

func TestGetConfigDir_HomeDirFallback(t *testing.T) {
	// Test fallback when home directory is not available
	// This is tricky to test without mocking, so we'll test the logic indirectly
	
	// Clear KH_CERT_DIR to trigger default behavior
	t.Setenv("KH_CERT_DIR", "")
	
	configDir := getConfigDir()
	assert.NotEmpty(t, configDir)
	
	// Should contain .krakenhashes
	assert.Contains(t, configDir, ".krakenhashes")
	
	// Should be an absolute path
	assert.True(t, filepath.IsAbs(configDir))
}

func TestGetConfigDir_CurrentWorkingDirectory(t *testing.T) {
	// Test relative path resolution
	t.Setenv("KH_CERT_DIR", "relative/path")
	
	// Get current working directory
	cwd, err := os.Getwd()
	require.NoError(t, err)
	
	configDir := getConfigDir()
	
	// Should be absolute path
	assert.True(t, filepath.IsAbs(configDir))
	
	// Should be based on current working directory
	expectedPath := filepath.Join(cwd, "relative/path")
	assert.Equal(t, expectedPath, configDir)
}

func TestGetConfigDir_ConcurrentCalls(t *testing.T) {
	// Test that concurrent calls to getConfigDir don't cause issues
	certDir := filepath.Join(t.TempDir(), "concurrent", "certs")
	t.Setenv("KH_CERT_DIR", certDir)

	const numGoroutines = 10
	done := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			dir := getConfigDir()
			done <- dir
		}()
	}

	// Collect all results
	results := make([]string, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		results[i] = <-done
	}

	// All goroutines should return the same path
	for i := 1; i < numGoroutines; i++ {
		assert.Equal(t, results[0], results[i])
	}

	// Directory should exist
	assert.DirExists(t, certDir)
}

func TestGetConfigDir_NestedPaths(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected func(string) string
	}{
		{
			name: "single level",
			path: "certs",
			expected: func(cwd string) string {
				return filepath.Join(cwd, "certs")
			},
		},
		{
			name: "multiple levels",
			path: "config/agent/certs",
			expected: func(cwd string) string {
				return filepath.Join(cwd, "config", "agent", "certs")
			},
		},
		{
			name: "with dots",
			path: "./config/../certs",
			expected: func(cwd string) string {
				return filepath.Join(cwd, "certs")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KH_CERT_DIR", tt.path)
			
			cwd, err := os.Getwd()
			require.NoError(t, err)
			
			configDir := getConfigDir()
			expectedPath := tt.expected(cwd)
			
			// Clean both paths for comparison
			assert.Equal(t, filepath.Clean(expectedPath), filepath.Clean(configDir))
		})
	}
}

func TestGetConfigDir_SymbolicLinks(t *testing.T) {
	// Test behavior with symbolic links
	baseDir := t.TempDir()
	actualDir := filepath.Join(baseDir, "actual")
	linkDir := filepath.Join(baseDir, "link")
	
	// Create actual directory
	err := os.MkdirAll(actualDir, 0700)
	require.NoError(t, err)
	
	// Create symbolic link
	err = os.Symlink(actualDir, linkDir)
	if err != nil {
		t.Skip("Cannot create symbolic links on this system")
	}
	
	t.Setenv("KH_CERT_DIR", linkDir)
	
	configDir := getConfigDir()
	assert.Equal(t, linkDir, configDir)
	
	// Verify we can write to the directory
	testFile := filepath.Join(configDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0600)
	assert.NoError(t, err)
	
	// File should exist in the actual directory
	actualFile := filepath.Join(actualDir, "test.txt")
	assert.FileExists(t, actualFile)
}

// Benchmark getConfigDir performance
func BenchmarkGetConfigDir(b *testing.B) {
	tempDir := b.TempDir()
	os.Setenv("KH_CERT_DIR", tempDir)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getConfigDir()
	}
}

func BenchmarkGetConfigDir_NoEnv(b *testing.B) {
	// Benchmark without environment variable (uses home directory)
	os.Unsetenv("KH_CERT_DIR")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getConfigDir()
	}
}