package env

import (
	"os"

	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
)

// GetOrDefault returns the environment variable value or the default if not set
func GetOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	debug.Debug("%s not set, using default: %s", key, defaultValue)
	return defaultValue
}

// MustGet returns the environment variable value or panics if not set
func MustGet(key string) string {
	value := os.Getenv(key)
	if value == "" {
		debug.Error("Required environment variable %s not set", key)
		panic("Required environment variable " + key + " not set")
	}
	return value
}

// GetBool returns the environment variable as a boolean
// Returns false if the variable is not set or is not "true", "1", "yes", or "y" (case insensitive)
func GetBool(key string) bool {
	value := os.Getenv(key)
	switch value {
	case "true", "1", "yes", "y", "TRUE", "YES", "Y":
		return true
	default:
		return false
	}
}

// GetBoolOrDefault returns the environment variable as a boolean or the default value if not set
func GetBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return GetBool(key)
	}
	return defaultValue
}
