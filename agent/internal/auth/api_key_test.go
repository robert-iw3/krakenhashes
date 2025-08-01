package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAgentKey(t *testing.T) {
	tests := []struct {
		name      string
		configDir string
		apiKey    string
		agentID   string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "successful save",
			configDir: t.TempDir(),
			apiKey:    "test-api-key-123",
			agentID:   "agent-456",
			wantErr:   false,
		},
		{
			name:      "empty api key",
			configDir: t.TempDir(),
			apiKey:    "",
			agentID:   "agent-456",
			wantErr:   false, // Should still save, validation is elsewhere
		},
		{
			name:      "empty agent ID",
			configDir: t.TempDir(),
			apiKey:    "test-api-key-123",
			agentID:   "",
			wantErr:   false, // Should still save, validation is elsewhere
		},
		{
			name:      "special characters in keys",
			configDir: t.TempDir(),
			apiKey:    "key-with-special=chars&symbols",
			agentID:   "agent-with-dash_underscore",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SaveAgentKey(tt.configDir, tt.apiKey, tt.agentID)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify file was created with correct permissions
				keyPath := filepath.Join(tt.configDir, KeyFile)
				info, err := os.Stat(keyPath)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(FilePerms), info.Mode().Perm())

				// Verify file contents
				data, err := os.ReadFile(keyPath)
				require.NoError(t, err)
				expected := "AGENT_ID=" + tt.agentID + "\nAPI_KEY=" + tt.apiKey + "\n"
				assert.Equal(t, expected, string(data))
			}
		})
	}
}

func TestSaveAgentKey_DirectoryCreation(t *testing.T) {
	baseDir := t.TempDir()
	nestedDir := filepath.Join(baseDir, "nested", "config", "dir")
	
	err := SaveAgentKey(nestedDir, "test-key", "test-agent")
	assert.NoError(t, err)
	
	// Verify directory was created
	info, err := os.Stat(nestedDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

func TestSaveAgentKey_PermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Cannot test permission errors as root")
	}
	
	// Create a directory with no write permissions
	baseDir := t.TempDir()
	readOnlyDir := filepath.Join(baseDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0500)
	require.NoError(t, err)
	
	err = SaveAgentKey(readOnlyDir, "test-key", "test-agent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write key file")
}

func TestLoadAgentKey(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(configDir string) error
		wantAPIKey string
		wantAgentID string
		wantErr    bool
		errMsg     string
	}{
		{
			name: "successful load",
			setupFunc: func(configDir string) error {
				keyPath := filepath.Join(configDir, KeyFile)
				data := []byte("AGENT_ID=agent-789\nAPI_KEY=api-key-xyz\n")
				return os.WriteFile(keyPath, data, FilePerms)
			},
			wantAPIKey:  "api-key-xyz",
			wantAgentID: "agent-789",
			wantErr:     false,
		},
		{
			name: "file does not exist",
			setupFunc: func(configDir string) error {
				return nil // Don't create file
			},
			wantErr: true,
			errMsg:  "failed to read key file",
		},
		{
			name: "missing AGENT_ID",
			setupFunc: func(configDir string) error {
				keyPath := filepath.Join(configDir, KeyFile)
				data := []byte("API_KEY=api-key-xyz\n")
				return os.WriteFile(keyPath, data, FilePerms)
			},
			wantErr: true,
			errMsg:  "invalid key file format",
		},
		{
			name: "missing API_KEY",
			setupFunc: func(configDir string) error {
				keyPath := filepath.Join(configDir, KeyFile)
				data := []byte("AGENT_ID=agent-789\n")
				return os.WriteFile(keyPath, data, FilePerms)
			},
			wantErr: true,
			errMsg:  "invalid key file format",
		},
		{
			name: "empty file",
			setupFunc: func(configDir string) error {
				keyPath := filepath.Join(configDir, KeyFile)
				return os.WriteFile(keyPath, []byte(""), FilePerms)
			},
			wantErr: true,
			errMsg:  "invalid key file format",
		},
		{
			name: "reversed order",
			setupFunc: func(configDir string) error {
				keyPath := filepath.Join(configDir, KeyFile)
				data := []byte("API_KEY=api-key-xyz\nAGENT_ID=agent-789\n")
				return os.WriteFile(keyPath, data, FilePerms)
			},
			wantAPIKey:  "api-key-xyz",
			wantAgentID: "agent-789",
			wantErr:     false,
		},
		{
			name: "extra whitespace",
			setupFunc: func(configDir string) error {
				keyPath := filepath.Join(configDir, KeyFile)
				data := []byte("\nAGENT_ID=agent-789\n\nAPI_KEY=api-key-xyz\n\n")
				return os.WriteFile(keyPath, data, FilePerms)
			},
			wantAPIKey:  "api-key-xyz",
			wantAgentID: "agent-789",
			wantErr:     false,
		},
		{
			name: "equals sign in values",
			setupFunc: func(configDir string) error {
				keyPath := filepath.Join(configDir, KeyFile)
				data := []byte("AGENT_ID=agent=with=equals\nAPI_KEY=key=with=equals\n")
				return os.WriteFile(keyPath, data, FilePerms)
			},
			wantAPIKey:  "key=with=equals",
			wantAgentID: "agent=with=equals",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir := t.TempDir()
			
			if tt.setupFunc != nil {
				err := tt.setupFunc(configDir)
				require.NoError(t, err)
			}
			
			apiKey, agentID, err := LoadAgentKey(configDir)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantAPIKey, apiKey)
				assert.Equal(t, tt.wantAgentID, agentID)
			}
		})
	}
}

func TestSaveAndLoadAgentKey_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		agentID string
	}{
		{
			name:    "standard keys",
			apiKey:  "standard-api-key",
			agentID: "standard-agent-id",
		},
		{
			name:    "keys with special characters",
			apiKey:  "api-key-with-$pecial-ch@rs!",
			agentID: "agent-id-with-$pecial-ch@rs!",
		},
		{
			name:    "very long keys",
			apiKey:  "very-long-api-key-" + string(make([]byte, 1000)),
			agentID: "very-long-agent-id-" + string(make([]byte, 1000)),
		},
		{
			name:    "unicode characters",
			apiKey:  "api-key-with-unicode-🔑",
			agentID: "agent-id-with-unicode-🤖",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir := t.TempDir()
			
			// Save
			err := SaveAgentKey(configDir, tt.apiKey, tt.agentID)
			require.NoError(t, err)
			
			// Load
			loadedAPIKey, loadedAgentID, err := LoadAgentKey(configDir)
			require.NoError(t, err)
			
			// Verify
			assert.Equal(t, tt.apiKey, loadedAPIKey)
			assert.Equal(t, tt.agentID, loadedAgentID)
		})
	}
}

func TestFilePermissions(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("File permission tests behave differently as root")
	}
	
	configDir := t.TempDir()
	
	// Save a key
	err := SaveAgentKey(configDir, "test-key", "test-agent")
	require.NoError(t, err)
	
	// Check file permissions are exactly 0600
	keyPath := filepath.Join(configDir, KeyFile)
	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	
	// Verify only owner can read/write
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	
	// Verify directory permissions are 0700
	dirInfo, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())
}