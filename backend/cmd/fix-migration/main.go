package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Get database connection details from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Build from individual components
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "krakenhashes"
		}
		password := os.Getenv("DB_PASSWORD")
		if password == "" {
			password = "krakenhashes"
		}
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "krakenhashes"
		}
		dbURL = "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=disable"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Check current migration state
	var version int
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		log.Fatalf("Failed to check migration state: %v", err)
	}

	log.Printf("Current migration state: version=%d, dirty=%v", version, dirty)

	if dirty && version == 24 {
		// Fix the dirty state
		_, err = db.Exec("UPDATE schema_migrations SET version = 23, dirty = false")
		if err != nil {
			log.Fatalf("Failed to fix migration state: %v", err)
		}
		log.Println("Successfully reset migration state to version 23")

		// Verify the fix
		err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
		if err != nil {
			log.Fatalf("Failed to verify migration state: %v", err)
		}
		log.Printf("New migration state: version=%d, dirty=%v", version, dirty)
	} else {
		log.Println("Migration state is not dirty at version 24, no action needed")
	}
}