package config

import (
	"os"
	"path/filepath"

	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
)

const (
	// DefaultConfigDir is the default directory name for agent configuration
	// This will be created in the same directory as the executable
	DefaultConfigDir = "config"
)

// GetConfigDir returns the path to the agent's configuration directory
// It checks KH_CONFIG_DIR environment variable first, then falls back to default
// The directory will be created if it doesn't exist
func GetConfigDir() string {
	var configDir string

	// Check environment variable first (useful for testing)
	if envDir := os.Getenv("KH_CONFIG_DIR"); envDir != "" {
		debug.Debug("Using config directory from environment: %s", envDir)
		configDir = envDir
	} else {
		// Get the executable's directory
		execPath, err := os.Executable()
		if err != nil {
			debug.Warning("Could not get executable path: %v", err)
			debug.Debug("Using current directory with default config dir: %s", DefaultConfigDir)
			configDir = DefaultConfigDir
		} else {
			// Use the config directory relative to the executable
			execDir := filepath.Dir(execPath)
			configDir = filepath.Join(execDir, DefaultConfigDir)
			debug.Debug("Using config directory relative to executable: %s", configDir)
		}
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		debug.Error("Failed to create config directory: %v", err)
		// Fall back to current directory if we can't create the intended directory
		configDir = DefaultConfigDir
		if err := os.MkdirAll(configDir, 0700); err != nil {
			debug.Error("Failed to create fallback config directory: %v", err)
		}
	}

	return configDir
}
