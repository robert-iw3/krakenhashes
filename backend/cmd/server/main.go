package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/config"
	"github.com/ZerkerEOD/hashdom-backend/internal/database"
	"github.com/ZerkerEOD/hashdom-backend/internal/routes"
	"github.com/ZerkerEOD/hashdom-backend/internal/tls"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Initialize debug package first with default settings
	debug.Reinitialize()
	debug.Info("Debug logging initialized with default settings")

	// Get and log current working directory
	cwd, err := os.Getwd()
	if err != nil {
		debug.Error("Failed to get working directory: %v", err)
		os.Exit(1)
	}
	debug.Info("Current working directory: %s", cwd)

	// Load .env file
	err = godotenv.Load()
	if err != nil {
		debug.Info("Attempting to load .env from current directory: %s", cwd)
		debug.Warning("Failed to load .env file from current directory: %v", err)

		debug.Info("Attempting to load .env from project root")
		err = godotenv.Load("../../.env")
		if err != nil {
			debug.Error("Failed to load .env file from project root: %v", err)
			os.Exit(1)
		}
		debug.Info("Successfully loaded .env file from project root")
	} else {
		debug.Info("Successfully loaded .env file from current directory")
	}

	// Reinitialize debug package with loaded environment variables
	debug.Reinitialize()
	debug.Info("Debug logging reinitialized with environment variables")

	debug.Info("Initializing application...")
	debug.Info("Environment variables loaded successfully")

	// Initialize application configuration
	appConfig := config.NewConfig()
	debug.Info("Application configuration initialized")

	// Initialize TLS provider
	debug.Info("Initializing TLS provider")
	tlsProvider, err := tls.InitializeProvider(appConfig)
	if err != nil {
		debug.Error("Failed to initialize TLS provider: %v", err)
		os.Exit(1)
	}

	// Get TLS configuration for server
	serverTLSConfig, err := tlsProvider.GetTLSConfig()
	if err != nil {
		debug.Error("Failed to get TLS configuration: %v", err)
		os.Exit(1)
	}

	// Initialize database connection
	debug.Info("Initializing database connection")
	sqlDB, err := database.Connect()
	if err != nil {
		debug.Error("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	// Create router
	debug.Info("Creating router")
	r := mux.NewRouter()

	// Setup routes
	debug.Info("Setting up routes")
	routes.SetupRoutes(r, sqlDB, tlsProvider)

	// Create server
	debug.Info("Creating server")
	server := &http.Server{
		Addr:      appConfig.GetHTTPSAddress(),
		Handler:   r,
		TLSConfig: serverTLSConfig,
	}

	// Create HTTP server for CA certificate
	httpServer := &http.Server{
		Addr:    appConfig.GetHTTPAddress(), // Use configured HTTP port
		Handler: r,
	}

	// Start HTTP server in a goroutine for CA certificate
	go func() {
		debug.Info("Starting HTTP server for CA certificate on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			debug.Error("HTTP server error: %v", err)
		}
	}()

	// Channel to wait for server errors
	serverErr := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		debug.Info("Starting HTTPS server on %s", server.Addr)
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			debug.Error("HTTPS server error: %v", err)
			serverErr <- err
		}
	}()

	// Run migrations
	if err := database.RunMigrations(); err != nil {
		debug.Error("Database migrations failed: %v", err)
		os.Exit(1)
	}
	debug.Info("Database migrations completed successfully")

	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	debug.Info("Server is ready to handle requests")

	// Block until we receive a signal or server error
	select {
	case err := <-serverErr:
		debug.Error("Server error: %v", err)
		os.Exit(1)
	case sig := <-sigChan:
		debug.Info("Received signal: %v", sig)
		debug.Info("Shutting down server...")

		// Create a deadline for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Shutdown both servers
		if err := httpServer.Shutdown(ctx); err != nil {
			debug.Error("Error during HTTP server shutdown: %v", err)
		}
		if err := server.Shutdown(ctx); err != nil {
			debug.Error("Error during HTTPS server shutdown: %v", err)
		}
		debug.Info("Server shutdown complete")
	}
}
