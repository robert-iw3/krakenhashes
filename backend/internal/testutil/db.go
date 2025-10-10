package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/database"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// SetupTestDB creates a test database connection and runs migrations
func SetupTestDB(t *testing.T) *db.DB {
	t.Helper()

	// Use test database URL from environment or default
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = "postgres://krakenhashes:krakenhashes@localhost:5432/krakenhashes_test?sslmode=disable"
	}

	// Create raw database connection
	rawDB, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Wrap in our DB type
	testDB := &db.DB{DB: rawDB}

	// Set environment variables for migrations
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "krakenhashes")
	os.Setenv("DB_PASSWORD", "krakenhashes")
	os.Setenv("DB_NAME", "krakenhashes_test")

	// Run migrations from the backend directory by temporarily changing directory
	originalDir, _ := os.Getwd()
	// Find the backend directory (go up until we find db/migrations)
	testDir := originalDir
	for {
		migrationsPath := filepath.Join(testDir, "db", "migrations")
		if _, err := os.Stat(migrationsPath); err == nil {
			break
		}
		parent := filepath.Dir(testDir)
		if parent == testDir {
			t.Fatalf("Could not find db/migrations directory from %s", originalDir)
		}
		testDir = parent
	}

	// Change to backend directory for migrations
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to backend directory: %v", err)
	}

	// Run migrations to ensure all tables exist (including MFA tables)
	if err := database.RunMigrations(); err != nil {
		os.Chdir(originalDir) // Restore directory before failing
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Restore original directory
	if err := os.Chdir(originalDir); err != nil {
		t.Fatalf("Failed to restore original directory: %v", err)
	}

	// Always ensure auth_settings has exactly one row
	// First, truncate to start fresh
	_, err = testDB.Exec("TRUNCATE TABLE auth_settings CASCADE")
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("Failed to truncate auth_settings: %v", err)
	}

	// Then insert default auth_settings with all required fields
	_, err = testDB.Exec(`
		INSERT INTO auth_settings (
			min_password_length,
			require_uppercase,
			require_lowercase,
			require_numbers,
			require_special_chars,
			max_failed_attempts,
			lockout_duration_minutes,
			require_mfa,
			jwt_expiry_minutes,
			display_timezone,
			notification_aggregation_minutes
		)
		VALUES (15, true, true, true, true, 5, 60, false, 60, 'UTC', 60)
	`)
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("Failed to insert auth_settings: %v", err)
	}

	// Clean up function
	t.Cleanup(func() {
		// Clean all tables in reverse order of foreign key dependencies
		// Only truncate tables that exist
		tables := []string{
			"email_mfa_codes",      // MFA codes (references users)
			"pending_mfa_setup",    // MFA setup (references users)
			"mfa_sessions",         // MFA sessions (references users)
			"tokens",               // Auth tokens (references users)
			"hashlist_hashes",
			"hashes",
			"hashlists",
			"agents",
			"users",
			"system_settings",
			"auth_settings",
		}

		for _, table := range tables {
			_, err := testDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
			if err != nil {
				// Only log if it's not a "relation does not exist" error
				if !strings.Contains(err.Error(), "does not exist") {
					t.Logf("Warning: Failed to truncate %s: %v", table, err)
				}
			}
		}

		testDB.Close()
	})

	return testDB
}

// CreateTestUser creates a test user with the given attributes
func CreateTestUser(t *testing.T, database *db.DB, username, email, pass string, role string) *models.User {
	t.Helper()

	// Hash the password using bcrypt (for user authentication, not hashcat)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Create user directly with SQL - include ALL fields that GetUserByID expects
	// Note: mfa_type must include 'email' per database constraint
	query := `
		INSERT INTO users (
			username, email, password_hash, role,
			account_enabled, account_locked, failed_login_attempts,
			mfa_enabled, mfa_type, preferred_mfa_method,
			last_password_change, notify_on_job_completion
		)
		VALUES ($1, $2, $3, $4, true, false, 0, false, ARRAY['email']::text[], NULL, NOW(), false)
		RETURNING id, username, email, role, created_at, updated_at,
		          account_enabled, account_locked, failed_login_attempts,
		          mfa_enabled, last_password_change, notify_on_job_completion
	`

	user := &models.User{}
	var notifyOnJobCompletion bool
	err = database.QueryRow(query, username, email, hashedPassword, role).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.AccountEnabled,
		&user.AccountLocked,
		&user.FailedLoginAttempts,
		&user.MFAEnabled,
		&user.LastPasswordChange,
		&notifyOnJobCompletion,
	)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}
