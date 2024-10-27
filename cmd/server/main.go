package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/database"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/routes"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/pkg/debug"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file first
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}

	// Reinitialize debug package with loaded environment variables
	debug.Reinitialize()

	debug.Info("Initializing application...")
	debug.Info("Environment variables loaded successfully")

	db, err := database.Connect()
	if err != nil {
		debug.Error("Database connection failed: %v", err)
		os.Exit(1)
	}
	defer db.Close()
	debug.Info("Database connection established")

	// Run migrations
	if err := database.RunMigrations(); err != nil {
		debug.Error("Database migrations failed: %v", err)
		os.Exit(1)
	}
	debug.Info("Database migrations completed successfully")

	// Create a new router
	r := mux.NewRouter()
	debug.Debug("Router initialized")

	// Setup routes
	routes.SetupRoutes(r)
	debug.Info("Routes configured successfully")

	// Get port from environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default port if not specified
		debug.Warning("No PORT environment variable found, using default: %s", port)
	}

	// Start the server
	debug.Info("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		debug.Error("Server failed to start: %v", err)
		os.Exit(1)
	}
}
