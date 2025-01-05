package main

import (
	"log"

	"github.com/ZerkerEOD/hashdom-backend/internal/database"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Run migrations
	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Error running migrations: %v", err)
	}

	log.Println("Migrations completed successfully")
}
