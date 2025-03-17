package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
)

const (
	// DefaultConfigDir is the default directory name for agent configuration
	// This will be created in the same directory as the executable
	DefaultConfigDir = "config"
	// DefaultDataDir is the default directory name for agent data
	// This will be created in the same directory as the executable
	DefaultDataDir = "data"
)

// DataDirs represents the data directories used by the agent
type DataDirs struct {
	Binaries  string
	Wordlists string
	Rules     string
	Hashlists string
}

// GetDataDirs returns the paths to the agent's data directories
// It checks KH_DATA_DIR environment variable first, then falls back to default
// The directories will be created if they don't exist
func GetDataDirs() (*DataDirs, error) {
	var baseDataDir string

	// Print current working directory for debugging
	cwd, err := os.Getwd()
	if err != nil {
		debug.Error("Failed to get current working directory: %v", err)
	} else {
		debug.Info("Current working directory in GetDataDirs: %s", cwd)
	}

	// Check environment variable first (useful for testing)
	if envDir := os.Getenv("KH_DATA_DIR"); envDir != "" {
		debug.Info("Using data directory from environment: %s", envDir)

		// Check if the path is relative or absolute
		if filepath.IsAbs(envDir) {
			debug.Info("Data directory path is absolute")
			baseDataDir = envDir
		} else {
			debug.Info("Data directory path is relative, resolving from current directory")
			absPath, err := filepath.Abs(envDir)
			if err != nil {
				debug.Error("Failed to resolve absolute path: %v", err)
				baseDataDir = envDir
			} else {
				debug.Info("Resolved absolute path: %s", absPath)
				baseDataDir = absPath
			}
		}
	} else {
		// Get the executable's directory
		execPath, err := os.Executable()
		if err != nil {
			debug.Error("Could not get executable path: %v", err)
			debug.Info("Using current directory with default data dir: %s", DefaultDataDir)
			baseDataDir = DefaultDataDir
		} else {
			// Use the data directory relative to the executable
			execDir := filepath.Dir(execPath)
			baseDataDir = filepath.Join(execDir, DefaultDataDir)
			debug.Info("Using data directory relative to executable: %s", baseDataDir)
		}
	}

	// Create base data directory if it doesn't exist
	if err := os.MkdirAll(baseDataDir, 0750); err != nil {
		debug.Error("Failed to create base data directory %s: %v", baseDataDir, err)
		return nil, fmt.Errorf("failed to create base data directory: %w", err)
	}

	// Create data directories structure
	dirs := &DataDirs{
		Binaries:  filepath.Join(baseDataDir, "binaries"),
		Wordlists: filepath.Join(baseDataDir, "wordlists"),
		Rules:     filepath.Join(baseDataDir, "rules"),
		Hashlists: filepath.Join(baseDataDir, "hashlists"),
	}

	// Create each directory with appropriate permissions
	for name, dir := range map[string]string{
		"binaries":  dirs.Binaries,
		"wordlists": dirs.Wordlists,
		"rules":     dirs.Rules,
		"hashlists": dirs.Hashlists,
	} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			debug.Error("Failed to create %s directory %s: %v", name, dir, err)
			return nil, fmt.Errorf("failed to create %s directory: %w", name, err)
		}
		debug.Info("Created %s directory: %s", name, dir)
	}

	return dirs, nil
}

// GetConfigDir returns the path to the agent's configuration directory
// It checks KH_CONFIG_DIR environment variable first, then falls back to default
// The directory will be created if it doesn't exist
func GetConfigDir() string {
	var configDir string

	// Print current working directory for debugging
	cwd, err := os.Getwd()
	if err != nil {
		debug.Error("Failed to get current working directory: %v", err)
	} else {
		debug.Info("Current working directory in GetConfigDir: %s", cwd)
	}

	// Check environment variable first (useful for testing)
	if envDir := os.Getenv("KH_CONFIG_DIR"); envDir != "" {
		debug.Info("Using config directory from environment: %s", envDir)

		// Check if the path is relative or absolute
		if filepath.IsAbs(envDir) {
			debug.Info("Config directory path is absolute")
			configDir = envDir
		} else {
			debug.Info("Config directory path is relative, resolving from current directory")
			absPath, err := filepath.Abs(envDir)
			if err != nil {
				debug.Error("Failed to resolve absolute path: %v", err)
				configDir = envDir
			} else {
				debug.Info("Resolved absolute path: %s", absPath)
				configDir = absPath
			}
		}
	} else {
		// Get the executable's directory
		execPath, err := os.Executable()
		if err != nil {
			debug.Error("Could not get executable path: %v", err)
			debug.Info("Using current directory with default config dir: %s", DefaultConfigDir)
			configDir = DefaultConfigDir
		} else {
			// Use the config directory relative to the executable
			execDir := filepath.Dir(execPath)
			configDir = filepath.Join(execDir, DefaultConfigDir)
			debug.Info("Using config directory relative to executable: %s", configDir)
		}
	}

	// Create the directory if it doesn't exist
	debug.Debug("Creating config directory if it doesn't exist: %s", configDir)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		debug.Error("Failed to create config directory %s: %v", configDir, err)
		// Fall back to current directory if we can't create the intended directory
		configDir = DefaultConfigDir
		debug.Warning("Falling back to default config directory: %s", configDir)
		if err := os.MkdirAll(configDir, 0700); err != nil {
			debug.Error("Failed to create fallback config directory: %v", err)
		}
	}

	debug.Info("Using config directory: %s", configDir)
	return configDir
}
