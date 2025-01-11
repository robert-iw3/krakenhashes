package agent

import (
	"os"
	"path/filepath"

	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
)

// getConfigDir returns the configuration directory path
func getConfigDir() string {
	// Get base directory for certificates from environment variable
	certDir := os.Getenv("KH_CERT_DIR")
	debug.Debug("Initial cert directory from env: %s", certDir)

	if certDir == "" {
		// Default to a directory in the user's home
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if home not found
			certDir = filepath.Join(".", ".krakenhashes")
			debug.Debug("Using fallback cert directory: %s", certDir)
		} else {
			certDir = filepath.Join(home, ".krakenhashes")
			debug.Debug("Using home directory cert path: %s", certDir)
		}
	}

	// If relative path, make it absolute relative to current working directory
	if !filepath.IsAbs(certDir) {
		if cwd, err := os.Getwd(); err == nil {
			certDir = filepath.Join(cwd, certDir)
			debug.Debug("Converted relative path to absolute: %s", certDir)
		} else {
			debug.Warning("Failed to get current working directory: %v", err)
		}
	}

	// Ensure the directory exists
	if err := os.MkdirAll(certDir, 0700); err != nil {
		debug.Error("Failed to create certificate directory: %v", err)
	} else {
		debug.Debug("Certificate directory exists or was created: %s", certDir)
	}

	// Verify directory is accessible
	if _, err := os.Stat(certDir); err != nil {
		debug.Error("Certificate directory is not accessible: %v", err)
	} else {
		debug.Debug("Certificate directory is accessible: %s", certDir)
	}

	return certDir
}
