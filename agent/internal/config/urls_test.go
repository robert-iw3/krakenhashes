package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewURLConfig(t *testing.T) {
	tests := []struct {
		name                string
		envVars             map[string]string
		expectedWebSocketURL string
		expectedBaseURL     string
		expectedHTTPPort    string
	}{
		{
			name:                "default configuration",
			envVars:             map[string]string{},
			expectedWebSocketURL: "ws://localhost:31337",
			expectedBaseURL:     "http://localhost:31337",
			expectedHTTPPort:    "1337",
		},
		{
			name: "custom host and port",
			envVars: map[string]string{
				"KH_HOST": "example.com",
				"KH_PORT": "8080",
			},
			expectedWebSocketURL: "ws://example.com:8080",
			expectedBaseURL:     "http://example.com:8080",
			expectedHTTPPort:    "1337",
		},
		{
			name: "with TLS enabled",
			envVars: map[string]string{
				"KH_HOST":  "secure.example.com",
				"KH_PORT":  "443",
				"USE_TLS":  "true",
			},
			expectedWebSocketURL: "wss://secure.example.com:443",
			expectedBaseURL:     "https://secure.example.com:443",
			expectedHTTPPort:    "1337",
		},
		{
			name: "custom HTTP port",
			envVars: map[string]string{
				"KH_HTTP_PORT": "8081",
			},
			expectedWebSocketURL: "ws://localhost:31337",
			expectedBaseURL:     "http://localhost:31337",
			expectedHTTPPort:    "8081",
		},
		{
			name: "all custom values with TLS",
			envVars: map[string]string{
				"KH_HOST":      "api.krakenhashes.com",
				"KH_PORT":      "9443",
				"KH_HTTP_PORT": "9080",
				"USE_TLS":      "true",
			},
			expectedWebSocketURL: "wss://api.krakenhashes.com:9443",
			expectedBaseURL:     "https://api.krakenhashes.com:9443",
			expectedHTTPPort:    "9080",
		},
		{
			name: "TLS false explicitly",
			envVars: map[string]string{
				"USE_TLS": "false",
			},
			expectedWebSocketURL: "ws://localhost:31337",
			expectedBaseURL:     "http://localhost:31337",
			expectedHTTPPort:    "1337",
		},
		{
			name: "invalid TLS value treated as false",
			envVars: map[string]string{
				"USE_TLS": "yes", // Not "true", so should be false
			},
			expectedWebSocketURL: "ws://localhost:31337",
			expectedBaseURL:     "http://localhost:31337",
			expectedHTTPPort:    "1337",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			config := NewURLConfig()
			assert.NotNil(t, config)
			assert.Equal(t, tt.expectedWebSocketURL, config.WebSocketURL)
			assert.Equal(t, tt.expectedBaseURL, config.BaseURL)
			assert.Equal(t, tt.expectedHTTPPort, config.HTTPPort)
		})
	}
}

func TestURLConfig_GetWebSocketURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *URLConfig
		expected string
	}{
		{
			name: "HTTP WebSocket",
			config: &URLConfig{
				WebSocketURL: "ws://localhost:31337",
			},
			expected: "ws://localhost:31337/ws/agent",
		},
		{
			name: "HTTPS WebSocket",
			config: &URLConfig{
				WebSocketURL: "wss://secure.example.com:443",
			},
			expected: "wss://secure.example.com:443/ws/agent",
		},
		{
			name: "Custom port",
			config: &URLConfig{
				WebSocketURL: "ws://api.local:8080",
			},
			expected: "ws://api.local:8080/ws/agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.config.GetWebSocketURL()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestURLConfig_GetRegistrationURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *URLConfig
		expected string
	}{
		{
			name: "HTTP registration",
			config: &URLConfig{
				BaseURL: "http://localhost:31337",
			},
			expected: "http://localhost:31337/api/agent/register",
		},
		{
			name: "HTTPS registration",
			config: &URLConfig{
				BaseURL: "https://secure.example.com:443",
			},
			expected: "https://secure.example.com:443/api/agent/register",
		},
		{
			name: "Custom port",
			config: &URLConfig{
				BaseURL: "http://api.local:8080",
			},
			expected: "http://api.local:8080/api/agent/register",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.config.GetRegistrationURL()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestURLConfig_GetAPIBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *URLConfig
		expected string
	}{
		{
			name: "HTTP API base",
			config: &URLConfig{
				BaseURL: "http://localhost:31337",
			},
			expected: "http://localhost:31337/api",
		},
		{
			name: "HTTPS API base",
			config: &URLConfig{
				BaseURL: "https://secure.example.com:443",
			},
			expected: "https://secure.example.com:443/api",
		},
		{
			name: "Custom port",
			config: &URLConfig{
				BaseURL: "http://api.local:8080",
			},
			expected: "http://api.local:8080/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.config.GetAPIBaseURL()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestURLConfig_GetCACertURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *URLConfig
		expected string
	}{
		{
			name: "standard HTTP base URL",
			config: &URLConfig{
				BaseURL:  "http://localhost:31337",
				HTTPPort: "1337",
			},
			expected: "http://localhost:1337/ca.crt",
		},
		{
			name: "HTTPS base URL should use HTTP for CA cert",
			config: &URLConfig{
				BaseURL:  "https://secure.example.com:443",
				HTTPPort: "8080",
			},
			expected: "http://secure.example.com:8080/ca.crt",
		},
		{
			name: "custom domain and port",
			config: &URLConfig{
				BaseURL:  "https://api.krakenhashes.com:9443",
				HTTPPort: "9080",
			},
			expected: "http://api.krakenhashes.com:9080/ca.crt",
		},
		{
			name: "IPv4 address",
			config: &URLConfig{
				BaseURL:  "https://192.168.1.100:8443",
				HTTPPort: "8080",
			},
			expected: "http://192.168.1.100:8080/ca.crt",
		},
		{
			name: "IPv6 address",
			config: &URLConfig{
				BaseURL:  "https://[::1]:8443",
				HTTPPort: "8080",
			},
			expected: "http://::1:8080/ca.crt",
		},
		{
			name: "invalid base URL falls back to localhost",
			config: &URLConfig{
				BaseURL:  "not-a-valid-url",
				HTTPPort: "1337",
			},
			expected: "http://localhost:1337/ca.crt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.config.GetCACertURL()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestURLConfig_Integration(t *testing.T) {
	// Test that all URL methods produce consistent results
	tests := []struct {
		name   string
		config *URLConfig
	}{
		{
			name: "HTTP configuration",
			config: &URLConfig{
				WebSocketURL: "ws://api.local:8080",
				BaseURL:      "http://api.local:8080",
				HTTPPort:     "8081",
			},
		},
		{
			name: "HTTPS configuration",
			config: &URLConfig{
				WebSocketURL: "wss://secure.api.local:8443",
				BaseURL:      "https://secure.api.local:8443",
				HTTPPort:     "8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all URLs are properly formed
			wsURL := tt.config.GetWebSocketURL()
			assert.Contains(t, wsURL, "/ws/agent")
			assert.Contains(t, wsURL, tt.config.WebSocketURL)

			regURL := tt.config.GetRegistrationURL()
			assert.Contains(t, regURL, "/api/agent/register")
			assert.Contains(t, regURL, tt.config.BaseURL)

			apiURL := tt.config.GetAPIBaseURL()
			assert.Contains(t, apiURL, "/api")
			assert.Contains(t, apiURL, tt.config.BaseURL)

			caURL := tt.config.GetCACertURL()
			assert.Contains(t, caURL, "/ca.crt")
			assert.Contains(t, caURL, "http://") // Always HTTP
			assert.Contains(t, caURL, tt.config.HTTPPort)
		})
	}
}

func TestURLConstruction_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		validate func(t *testing.T, config *URLConfig)
	}{
		{
			name: "empty host defaults to localhost",
			envVars: map[string]string{
				"KH_HOST": "",
			},
			validate: func(t *testing.T, config *URLConfig) {
				assert.Contains(t, config.BaseURL, "localhost")
				assert.Contains(t, config.WebSocketURL, "localhost")
			},
		},
		{
			name: "empty port defaults to 31337",
			envVars: map[string]string{
				"KH_PORT": "",
			},
			validate: func(t *testing.T, config *URLConfig) {
				assert.Contains(t, config.BaseURL, ":31337")
				assert.Contains(t, config.WebSocketURL, ":31337")
			},
		},
		{
			name: "empty HTTP port defaults to 1337",
			envVars: map[string]string{
				"KH_HTTP_PORT": "",
			},
			validate: func(t *testing.T, config *URLConfig) {
				assert.Equal(t, "1337", config.HTTPPort)
			},
		},
		{
			name: "special characters in host",
			envVars: map[string]string{
				"KH_HOST": "api-server.kraken_hashes.com",
				"KH_PORT": "9000",
			},
			validate: func(t *testing.T, config *URLConfig) {
				assert.Equal(t, "ws://api-server.kraken_hashes.com:9000", config.WebSocketURL)
				assert.Equal(t, "http://api-server.kraken_hashes.com:9000", config.BaseURL)
			},
		},
		{
			name: "numeric host (IP address)",
			envVars: map[string]string{
				"KH_HOST": "10.0.0.1",
				"KH_PORT": "8080",
			},
			validate: func(t *testing.T, config *URLConfig) {
				assert.Equal(t, "ws://10.0.0.1:8080", config.WebSocketURL)
				assert.Equal(t, "http://10.0.0.1:8080", config.BaseURL)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			config := NewURLConfig()
			assert.NotNil(t, config)
			
			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

// Benchmark URL construction
func BenchmarkNewURLConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewURLConfig()
	}
}

func BenchmarkGetCACertURL(b *testing.B) {
	config := &URLConfig{
		BaseURL:  "https://api.example.com:8443",
		HTTPPort: "8080",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetCACertURL()
	}
}