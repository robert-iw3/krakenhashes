package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/db"
	"github.com/ZerkerEOD/hashdom-backend/internal/repository"
	"github.com/ZerkerEOD/hashdom-backend/internal/routes"
	"github.com/ZerkerEOD/hashdom-backend/internal/services"
	"github.com/ZerkerEOD/hashdom-backend/internal/tls"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/gorilla/mux"
)

func main() {
	// Initialize logger
	log.SetOutput(os.Stdout)
	debug.Info("Logger initialized")

	// Initialize database connection
	sqlDB, err := sql.Open("postgres", os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		debug.Error("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	// Initialize repositories
	agentRepo := repository.NewAgentRepository(&db.DB{DB: sqlDB})
	voucherRepo := repository.NewClaimVoucherRepository(&db.DB{DB: sqlDB})

	// Initialize services
	_ = services.NewAgentService(agentRepo, voucherRepo) // Service initialization handled in routes.SetupRoutes

	// Initialize TLS configuration
	tlsConfig := tls.NewConfig()
	if err := tlsConfig.GenerateCertificates(); err != nil {
		debug.Error("Failed to generate certificates: %v", err)
		os.Exit(1)
	}

	serverTLSConfig, err := tlsConfig.LoadTLSConfig()
	if err != nil {
		debug.Error("Failed to load TLS configuration: %v", err)
		os.Exit(1)
	}

	// Create router
	router := mux.NewRouter()

	// Setup routes
	routes.SetupRoutes(router, sqlDB)

	// Create server
	server := &http.Server{
		Addr:      ":8080",
		Handler:   router,
		TLSConfig: serverTLSConfig,
	}

	// Start server in a goroutine
	go func() {
		debug.Info("Starting server on :8080")
		if err := server.ListenAndServeTLS(tlsConfig.CertFile, tlsConfig.KeyFile); err != nil && err != http.ErrServerClosed {
			debug.Error("Failed to start server: %v", err)
			os.Exit(1)
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
		debug.Error("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	debug.Info("Server stopped gracefully")
}
