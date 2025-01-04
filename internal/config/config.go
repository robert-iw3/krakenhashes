package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	Host string
	Port int
}

// NewConfig creates a new Config instance with values from environment variables
func NewConfig() *Config {
	port := 8080 // Default port
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost" // Default host
	}

	return &Config{
		Host: host,
		Port: port,
	}
}

// GetWSEndpoint returns the WebSocket endpoint URL
func (c *Config) GetWSEndpoint() string {
	return fmt.Sprintf("ws://%s:%d/ws", c.Host, c.Port)
}

// GetAPIEndpoint returns the API endpoint URL
func (c *Config) GetAPIEndpoint() string {
	return fmt.Sprintf("http://%s:%d/api", c.Host, c.Port)
}

// GetAddress returns the full address for the server to listen on
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
