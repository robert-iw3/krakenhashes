package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name                  string
		envVars               map[string]string
		expectedExtraParams   string
		expectedDataDirExists bool
	}{
		{
			name:                  "default config",
			envVars:               map[string]string{},
			expectedExtraParams:   "",
			expectedDataDirExists: true,
		},
		{
			name: "with hashcat extra params",
			envVars: map[string]string{
				"HASHCAT_EXTRA_PARAMS": "-O -w 3",
			},
			expectedExtraParams:   "-O -w 3",
			expectedDataDirExists: true,
		},
		{
			name: "with custom data dir",
			envVars: map[string]string{
				"KH_DATA_DIR": t.TempDir(),
			},
			expectedDataDirExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg := NewConfig()
			assert.NotNil(t, cfg)
			assert.Equal(t, tt.expectedExtraParams, cfg.HashcatExtraParams)
			
			if tt.expectedDataDirExists {
				assert.NotEmpty(t, cfg.DataDirectory)
			}
		})
	}
}

func TestGetDataDirs(t *testing.T) {
	tests := []struct {
		name            string
		envVars         map[string]string
		validateDirs    func(t *testing.T, dirs *DataDirs)
		wantErr         bool
	}{
		{
			name:    "default data directories",
			envVars: map[string]string{},
			validateDirs: func(t *testing.T, dirs *DataDirs) {
				assert.NotEmpty(t, dirs.Binaries)
				assert.NotEmpty(t, dirs.Wordlists)
				assert.NotEmpty(t, dirs.Rules)
				assert.NotEmpty(t, dirs.Hashlists)
				assert.Contains(t, dirs.Binaries, "binaries")
				assert.Contains(t, dirs.Wordlists, "wordlists")
				assert.Contains(t, dirs.Rules, "rules")
				assert.Contains(t, dirs.Hashlists, "hashlists")
			},
			wantErr: false,
		},
		{
			name: "custom data directory",
			envVars: map[string]string{
				"KH_DATA_DIR": t.TempDir(),
			},
			validateDirs: func(t *testing.T, dirs *DataDirs) {
				// Verify all directories exist
				assert.DirExists(t, dirs.Binaries)
				assert.DirExists(t, dirs.Wordlists)
				assert.DirExists(t, dirs.Rules)
				assert.DirExists(t, dirs.Hashlists)
				
				// Verify subdirectories
				assert.DirExists(t, dirs.WordlistGeneral)
				assert.DirExists(t, dirs.WordlistSpecialized)
				assert.DirExists(t, dirs.WordlistTargeted)
				assert.DirExists(t, dirs.WordlistCustom)
				assert.DirExists(t, dirs.RuleHashcat)
				assert.DirExists(t, dirs.RuleJohn)
				assert.DirExists(t, dirs.RuleCustom)
			},
			wantErr: false,
		},
		{
			name: "relative path data directory",
			envVars: map[string]string{
				"KH_DATA_DIR": "relative/data/dir",
			},
			validateDirs: func(t *testing.T, dirs *DataDirs) {
				// Should be resolved to absolute path
				assert.True(t, filepath.IsAbs(filepath.Dir(dirs.Binaries)))
			},
			wantErr: false,
		},
		{
			name: "absolute path data directory",
			envVars: map[string]string{
				"KH_DATA_DIR": filepath.Join(t.TempDir(), "absolute", "data"),
			},
			validateDirs: func(t *testing.T, dirs *DataDirs) {
				assert.True(t, filepath.IsAbs(dirs.Binaries))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			dirs, err := GetDataDirs()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, dirs)
				if tt.validateDirs != nil {
					tt.validateDirs(t, dirs)
				}
			}
		})
	}
}

func TestGetDataDirs_DirectoryStructure(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("KH_DATA_DIR", baseDir)

	dirs, err := GetDataDirs()
	require.NoError(t, err)

	// Check base directories
	expectedBaseDirs := map[string]string{
		"binaries":  dirs.Binaries,
		"wordlists": dirs.Wordlists,
		"rules":     dirs.Rules,
		"hashlists": dirs.Hashlists,
	}

	for name, path := range expectedBaseDirs {
		t.Run("base_"+name, func(t *testing.T) {
			assert.DirExists(t, path)
			assert.Contains(t, path, name)
			
			// Check permissions
			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0750), info.Mode().Perm())
		})
	}

	// Check wordlist subdirectories
	wordlistSubdirs := map[string]string{
		"general":     dirs.WordlistGeneral,
		"specialized": dirs.WordlistSpecialized,
		"targeted":    dirs.WordlistTargeted,
		"custom":      dirs.WordlistCustom,
	}

	for name, path := range wordlistSubdirs {
		t.Run("wordlist_"+name, func(t *testing.T) {
			assert.DirExists(t, path)
			assert.Contains(t, path, filepath.Join("wordlists", name))
		})
	}

	// Check rule subdirectories
	ruleSubdirs := map[string]string{
		"hashcat": dirs.RuleHashcat,
		"john":    dirs.RuleJohn,
		"custom":  dirs.RuleCustom,
	}

	for name, path := range ruleSubdirs {
		t.Run("rule_"+name, func(t *testing.T) {
			assert.DirExists(t, path)
			assert.Contains(t, path, filepath.Join("rules", name))
		})
	}
}

func TestGetConfigDir(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		validatePath func(t *testing.T, path string)
	}{
		{
			name:    "default config directory",
			envVars: map[string]string{},
			validatePath: func(t *testing.T, path string) {
				assert.NotEmpty(t, path)
				assert.Contains(t, path, DefaultConfigDir)
			},
		},
		{
			name: "custom config directory",
			envVars: map[string]string{
				"KH_CONFIG_DIR": t.TempDir(),
			},
			validatePath: func(t *testing.T, path string) {
				assert.DirExists(t, path)
			},
		},
		{
			name: "relative path config directory",
			envVars: map[string]string{
				"KH_CONFIG_DIR": "relative/config/dir",
			},
			validatePath: func(t *testing.T, path string) {
				// Should be resolved to absolute path
				assert.True(t, filepath.IsAbs(path))
			},
		},
		{
			name: "absolute path config directory",
			envVars: map[string]string{
				"KH_CONFIG_DIR": filepath.Join(t.TempDir(), "absolute", "config"),
			},
			validatePath: func(t *testing.T, path string) {
				assert.True(t, filepath.IsAbs(path))
				assert.DirExists(t, path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			configDir := GetConfigDir()
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

func TestGetConfigDir_PermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Cannot test permission errors as root")
	}

	// Create a directory with no write permissions
	baseDir := t.TempDir()
	readOnlyDir := filepath.Join(baseDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0500)
	require.NoError(t, err)

	configPath := filepath.Join(readOnlyDir, "config")
	t.Setenv("KH_CONFIG_DIR", configPath)

	// Should fall back to default directory
	configDir := GetConfigDir()
	assert.NotEqual(t, configPath, configDir)
	assert.Contains(t, configDir, DefaultConfigDir)
}

func TestDirectoryCreationConcurrency(t *testing.T) {
	// Test that concurrent calls to GetDataDirs don't cause issues
	baseDir := t.TempDir()
	t.Setenv("KH_DATA_DIR", baseDir)

	const numGoroutines = 10
	done := make(chan bool)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			dirs, err := GetDataDirs()
			assert.NoError(t, err)
			assert.NotNil(t, dirs)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify directories were created correctly
	dirs, err := GetDataDirs()
	require.NoError(t, err)
	assert.DirExists(t, dirs.Binaries)
	assert.DirExists(t, dirs.Wordlists)
}

func TestDataDirs_PathConsistency(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("KH_DATA_DIR", baseDir)

	dirs, err := GetDataDirs()
	require.NoError(t, err)

	// Verify path relationships
	assert.Equal(t, filepath.Join(baseDir, "binaries"), dirs.Binaries)
	assert.Equal(t, filepath.Join(baseDir, "wordlists"), dirs.Wordlists)
	assert.Equal(t, filepath.Join(baseDir, "rules"), dirs.Rules)
	assert.Equal(t, filepath.Join(baseDir, "hashlists"), dirs.Hashlists)

	// Verify wordlist subdirectories
	assert.Equal(t, filepath.Join(dirs.Wordlists, "general"), dirs.WordlistGeneral)
	assert.Equal(t, filepath.Join(dirs.Wordlists, "specialized"), dirs.WordlistSpecialized)
	assert.Equal(t, filepath.Join(dirs.Wordlists, "targeted"), dirs.WordlistTargeted)
	assert.Equal(t, filepath.Join(dirs.Wordlists, "custom"), dirs.WordlistCustom)

	// Verify rule subdirectories
	assert.Equal(t, filepath.Join(dirs.Rules, "hashcat"), dirs.RuleHashcat)
	assert.Equal(t, filepath.Join(dirs.Rules, "john"), dirs.RuleJohn)
	assert.Equal(t, filepath.Join(dirs.Rules, "custom"), dirs.RuleCustom)
}