package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent_Structure(t *testing.T) {
	// Test that Agent struct has all expected fields
	agent := &Agent{
		ID:         123,
		Name:       "test-agent",
		Status:     "active",
		LastSeenAt: time.Now(),
		CreatedAt:  time.Now().Add(-24 * time.Hour),
		UpdatedAt:  time.Now(),
		Metadata: map[string]string{
			"version": "1.0.0",
			"os":      "linux",
		},
		DownloadToken: "token-123",
	}

	assert.Equal(t, 123, agent.ID)
	assert.Equal(t, "test-agent", agent.Name)
	assert.Equal(t, "active", agent.Status)
	assert.NotZero(t, agent.LastSeenAt)
	assert.NotZero(t, agent.CreatedAt)
	assert.NotZero(t, agent.UpdatedAt)
	assert.Equal(t, "1.0.0", agent.Metadata["version"])
	assert.Equal(t, "linux", agent.Metadata["os"])
	assert.Equal(t, "token-123", agent.DownloadToken)
}

func TestGetAgentID(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(configDir string) error
		expectedID  int
		wantErr     bool
		errContains string
	}{
		{
			name: "successful ID retrieval",
			setupFunc: func(configDir string) error {
				// Save a valid agent key file
				return auth.SaveAgentKey(configDir, "test-api-key", "456")
			},
			expectedID: 456,
			wantErr:    false,
		},
		{
			name: "agent key file not found",
			setupFunc: func(configDir string) error {
				// Don't create any file
				return nil
			},
			wantErr:     true,
			errContains: "failed to load agent ID from agent.key",
		},
		{
			name: "invalid agent ID format",
			setupFunc: func(configDir string) error {
				// Save agent key with non-numeric ID
				return auth.SaveAgentKey(configDir, "test-api-key", "not-a-number")
			},
			wantErr:     true,
			errContains: "failed to parse agent ID",
		},
		{
			name: "empty agent ID",
			setupFunc: func(configDir string) error {
				// Create an invalid key file with empty agent ID
				keyPath := filepath.Join(configDir, auth.KeyFile)
				data := []byte("API_KEY=test-key\nAGENT_ID=\n")
				return os.WriteFile(keyPath, data, auth.FilePerms)
			},
			wantErr:     true,
			errContains: "failed to load agent ID from agent.key",
		},
		{
			name: "malformed key file",
			setupFunc: func(configDir string) error {
				// Create a malformed key file
				keyPath := filepath.Join(configDir, auth.KeyFile)
				data := []byte("This is not a valid key file format")
				return os.WriteFile(keyPath, data, auth.FilePerms)
			},
			wantErr:     true,
			errContains: "failed to load agent ID from agent.key",
		},
		{
			name: "zero agent ID",
			setupFunc: func(configDir string) error {
				return auth.SaveAgentKey(configDir, "test-api-key", "0")
			},
			expectedID: 0,
			wantErr:    false,
		},
		{
			name: "large agent ID",
			setupFunc: func(configDir string) error {
				return auth.SaveAgentKey(configDir, "test-api-key", "999999999")
			},
			expectedID: 999999999,
			wantErr:    false,
		},
		{
			name: "negative agent ID",
			setupFunc: func(configDir string) error {
				return auth.SaveAgentKey(configDir, "test-api-key", "-123")
			},
			expectedID: -123,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for config
			tempDir := t.TempDir()
			t.Setenv("KH_CONFIG_DIR", tempDir)

			// Setup test data
			if tt.setupFunc != nil {
				err := tt.setupFunc(tempDir)
				require.NoError(t, err)
			}

			// Call GetAgentID
			id, err := GetAgentID()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestGetAgentID_FilePermissions(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Cannot test permission errors as root")
	}

	// Create a config directory with no read permissions
	baseDir := t.TempDir()
	configDir := filepath.Join(baseDir, "config")
	err := os.Mkdir(configDir, 0700)
	require.NoError(t, err)

	// Create agent.key file
	err = auth.SaveAgentKey(configDir, "test-key", "123")
	require.NoError(t, err)

	// Remove read permissions from the key file
	keyPath := filepath.Join(configDir, auth.KeyFile)
	err = os.Chmod(keyPath, 0200) // Write only
	require.NoError(t, err)

	// Restore permissions after test
	defer os.Chmod(keyPath, 0600)

	t.Setenv("KH_CONFIG_DIR", configDir)

	// Should fail to read the file
	_, err = GetAgentID()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load agent ID from agent.key")
}

func TestGetAgentID_ConcurrentAccess(t *testing.T) {
	// Test that concurrent calls to GetAgentID work correctly
	tempDir := t.TempDir()
	t.Setenv("KH_CONFIG_DIR", tempDir)

	// Create agent key file
	err := auth.SaveAgentKey(tempDir, "test-api-key", "789")
	require.NoError(t, err)

	const numGoroutines = 10
	done := make(chan bool)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			id, err := GetAgentID()
			if err != nil {
				errors <- err
			} else if id != 789 {
				errors <- assert.AnError
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestAgentMetadata(t *testing.T) {
	// Test various metadata scenarios
	tests := []struct {
		name     string
		metadata map[string]string
	}{
		{
			name:     "empty metadata",
			metadata: map[string]string{},
		},
		{
			name:     "nil metadata",
			metadata: nil,
		},
		{
			name: "standard metadata",
			metadata: map[string]string{
				"version":  "1.0.0",
				"os":       "linux",
				"arch":     "amd64",
				"hostname": "agent-host",
			},
		},
		{
			name: "metadata with special characters",
			metadata: map[string]string{
				"path":    "/usr/local/bin/agent",
				"config":  "key=value;other=data",
				"unicode": "测试数据",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				ID:       1,
				Name:     "test-agent",
				Metadata: tt.metadata,
			}

			// Verify metadata is stored correctly
			if tt.metadata == nil {
				assert.Nil(t, agent.Metadata)
			} else {
				assert.Equal(t, len(tt.metadata), len(agent.Metadata))
				for k, v := range tt.metadata {
					assert.Equal(t, v, agent.Metadata[k])
				}
			}
		})
	}
}

func TestAgentTimestamps(t *testing.T) {
	now := time.Now()
	created := now.Add(-48 * time.Hour)
	updated := now.Add(-1 * time.Hour)
	lastSeen := now.Add(-5 * time.Minute)

	agent := &Agent{
		ID:         1,
		Name:       "test-agent",
		CreatedAt:  created,
		UpdatedAt:  updated,
		LastSeenAt: lastSeen,
	}

	// Verify timestamps
	assert.Equal(t, created.Unix(), agent.CreatedAt.Unix())
	assert.Equal(t, updated.Unix(), agent.UpdatedAt.Unix())
	assert.Equal(t, lastSeen.Unix(), agent.LastSeenAt.Unix())

	// Verify timestamp relationships
	assert.True(t, agent.CreatedAt.Before(agent.UpdatedAt))
	assert.True(t, agent.UpdatedAt.Before(agent.LastSeenAt))
	assert.True(t, agent.LastSeenAt.Before(now))
}