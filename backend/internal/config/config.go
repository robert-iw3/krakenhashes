package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/env"
)

// Config holds the application configuration
type Config struct {
	Host      string
	HTTPPort  int // Port for HTTP (CA certificate)
	HTTPSPort int // Port for HTTPS (API)
	ConfigDir string
}

// NewConfig creates a new Config instance with values from environment variables
func NewConfig() *Config {
	httpsPort := 31337 // Default HTTPS port
	if portStr := os.Getenv("KH_HTTPS_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			httpsPort = p
		}
	}

	httpPort := 1337 // Default HTTP port
	if portStr := os.Getenv("KH_HTTP_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			httpPort = p
		}
	}

	host := os.Getenv("KH_HOST")
	if host == "" {
		// In Docker, bind to all interfaces
		if env.GetBool("KH_IN_DOCKER") {
			host = "0.0.0.0"
		} else {
			host = "localhost" // Default host for local development
		}
	}

	// Get config directory from environment or use default
	configDir := os.Getenv("KH_CONFIG_DIR")
	if configDir == "" {
		// Try to get user's home directory
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory
			configDir = ".krakenhashes"
		} else {
			configDir = filepath.Join(home, ".krakenhashes")
		}
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		debug.Error("Failed to create config directory: %v", err)
		// Fallback to current directory
		configDir = ".krakenhashes"
		if err := os.MkdirAll(configDir, 0755); err != nil {
			debug.Error("Failed to create fallback config directory: %v", err)
		}
	}

	debug.Info("Using config directory: %s", configDir)

	return &Config{
		Host:      host,
		HTTPPort:  httpPort,
		HTTPSPort: httpsPort,
		ConfigDir: configDir,
	}
}

// GetHTTPAddress returns the address for the HTTP server
func (c *Config) GetHTTPAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.HTTPPort)
}

// GetHTTPSAddress returns the address for the HTTPS server
func (c *Config) GetHTTPSAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.HTTPSPort)
}

// GetWSEndpoint returns the WebSocket endpoint URL
func (c *Config) GetWSEndpoint() string {
	return fmt.Sprintf("ws://%s:%d/ws", c.Host, c.HTTPSPort)
}

// GetAPIEndpoint returns the API endpoint URL
func (c *Config) GetAPIEndpoint() string {
	return fmt.Sprintf("http://%s:%d/api", c.Host, c.HTTPSPort)
}

// GetAddress returns the full address for the server to listen on
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.HTTPSPort)
}

// GetCertsDir returns the path to the certificates directory
func (c *Config) GetCertsDir() string {
	// Check if running in Docker
	if env.GetBool("KH_IN_DOCKER") {
		return "/etc/krakenhashes/certs"
	}

	// For non-Docker installations, use the configured path
	certsDir := env.GetOrDefault("KH_CERTS_DIR", filepath.Join(c.ConfigDir, "certs"))

	// If the path is relative, make it absolute based on config directory
	if !filepath.IsAbs(certsDir) {
		certsDir = filepath.Join(c.ConfigDir, certsDir)
	}

	return certsDir
}

// GetAdditionalDNSNames returns a list of additional DNS names from environment variables
func GetAdditionalDNSNames() []string {
	// Get comma-separated list of DNS names from environment variable
	dnsNamesStr := env.GetOrDefault("KH_ADDITIONAL_DNS_NAMES", "")
	if dnsNamesStr == "" {
		return nil
	}

	// Split and trim whitespace
	dnsNames := strings.Split(dnsNamesStr, ",")
	for i := range dnsNames {
		dnsNames[i] = strings.TrimSpace(dnsNames[i])
	}

	return dnsNames
}

// GetAdditionalIPAddresses returns a list of additional IP addresses from environment variables
func GetAdditionalIPAddresses() []string {
	// Get comma-separated list of IP addresses from environment variable
	ipAddressesStr := env.GetOrDefault("KH_ADDITIONAL_IP_ADDRESSES", "")
	if ipAddressesStr == "" {
		return nil
	}

	// Split and trim whitespace
	ipAddresses := strings.Split(ipAddressesStr, ",")
	for i := range ipAddresses {
		ipAddresses[i] = strings.TrimSpace(ipAddresses[i])
	}

	return ipAddresses
}

// GetTLSConfig returns the TLS configuration including additional DNS names and IP addresses
func (c *Config) GetTLSConfig() ([]string, []string) {
	// Get additional DNS names
	dnsNames := GetAdditionalDNSNames()

	// Get additional IP addresses
	ipAddresses := GetAdditionalIPAddresses()

	return dnsNames, ipAddresses
}
