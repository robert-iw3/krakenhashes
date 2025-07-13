package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	_ "github.com/lib/pq"
)

// SetupTestDB creates a test database connection and runs migrations
func SetupTestDB(t *testing.T) *db.DB {
	t.Helper()

	// Use test database URL from environment or default
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = "postgres://postgres:postgres@localhost:5432/krakenhashes_test?sslmode=disable"
	}

	// Create raw database connection
	rawDB, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Wrap in our DB type
	testDB := db.NewDB(rawDB)

	// Clean up function
	t.Cleanup(func() {
		// Clean all tables in reverse order of foreign key dependencies
		tables := []string{
			"email_mfa_codes",
			"mfa_verify_attempts",
			"pending_mfa_setup",
			"backup_codes",
			"mfa_sessions",
			"auth_tokens",
			"email_logs",
			"vouchers",
			"agents",
			"users",
			"email_templates",
			"email_providers",
			"auth_settings",
			"mfa_settings",
		}

		for _, table := range tables {
			_, err := testDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
			if err != nil {
				t.Logf("Warning: Failed to truncate %s: %v", table, err)
			}
		}

		testDB.Close()
	})

	return testDB
}

// CreateTestUser creates a test user with the given attributes
func CreateTestUser(t *testing.T, db *db.DB, username, email, password string, role string) *db.User {
	t.Helper()

	user := &db.User{
		Username: username,
		Email:    email,
		Role:     role,
	}

	// Create user with password
	createdUser, err := db.CreateUserWithPassword(user, password)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return createdUser
}
