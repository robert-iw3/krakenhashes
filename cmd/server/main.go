package main

import (
	"net/http"
	"os"

	"github.com/ZerkerEOD/hashdom-backend/internal/database"
	"github.com/ZerkerEOD/hashdom-backend/internal/routes"
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
	routes.SetupRoutes(r, db)
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
