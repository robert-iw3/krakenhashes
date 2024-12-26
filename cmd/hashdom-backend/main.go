package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/auth"
	"github.com/ZerkerEOD/hashdom-backend/internal/routes"
	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
)

func main() {
	// Initialize logger
	debug.Init(os.Stdout, true)

	// Load environment variables
	caCertPath := os.Getenv("CA_CERT_PATH")
	if caCertPath == "" {
		debug.Fatal("CA_CERT_PATH environment variable is not set")
	}

	caKeyPath := os.Getenv("CA_KEY_PATH")
	if caKeyPath == "" {
		debug.Fatal("CA_KEY_PATH environment variable is not set")
	}

	// Initialize services
	agentService := services.NewAgentService()

	// Initialize CA manager
	caManager, err := auth.NewCAManager(caCertPath, caKeyPath)
	if err != nil {
		debug.Fatal("Failed to initialize CA manager: %v", err)
	}

	// Setup routes
	handler := routes.SetupRoutes(agentService, caManager)

	// Create server
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	// Start server in a goroutine
	go func() {
		debug.Info("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			debug.Fatal("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Shutdown server gracefully
	debug.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		debug.Fatal("Server forced to shutdown: %v", err)
	}

	debug.Info("Server stopped gracefully")
}
