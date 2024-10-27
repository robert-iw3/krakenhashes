package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/database"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/routes"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Error running migrations: %v", err)
	}

	// Create a new router
	r := mux.NewRouter()

	// Setup routes
	routes.SetupRoutes(r)

	// Get port from environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default port if not specified
	}

	// Start the server
	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
