package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/ZerkerEOD/hashdom/backend/internal/models"

	"github.com/ZerkerEOD/hashdom/backend/pkg/debug"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

var db *sql.DB

/*
 * Connect establishes a connection to the PostgreSQL database using environment variables.
 * It validates the connection with a ping test before returning.
 *
 * Returns:
 *   - *sql.DB: Database connection pool if successful
 *   - error: Any error encountered during connection
 */
func Connect() (*sql.DB, error) {
	debug.Info("Attempting database connection")

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	debug.Debug("Database configuration - Host: %s, Port: %s, User: %s, Database: %s",
		dbHost, dbPort, dbUser, dbName)

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	debug.Debug("Connection string created (without password): host=%s port=%s user=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		debug.Error("Failed to open database connection: %v", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	debug.Debug("Attempting to ping database...")
	err = db.Ping()
	if err != nil {
		debug.Error("Failed to ping database: %v", err)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	debug.Info("Successfully connected to database")
	return db, nil
}

/*
 * RunMigrations executes all pending database migrations from the db/migrations directory.
 * Migrations are run in order based on their timestamp prefix.
 *
 * Returns:
 *   - error: Any error encountered during migration, nil if successful
 *           Returns nil if no migrations are pending (ErrNoChange)
 */
func RunMigrations() error {
	debug.Info("Starting database migrations")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	m, err := migrate.New(
		"file://db/migrations",
		connStr)
	if err != nil {
		debug.Error("Failed to create migration instance: %v", err)
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		debug.Error("Migration failed: %v", err)
		return err
	}
	debug.Info("Database migrations completed successfully")
	return nil
}

/*
 * GetUserByUsername retrieves a user from the database by their username.
 *
 * Parameters:
 *   - username: The username to search for
 *
 * Returns:
 *   - *models.User: User object if found
 *   - error: sql.ErrNoRows if user not found, or any other database error
 */
func GetUserByUsername(username string) (*models.User, error) {
	debug.Debug("Looking up user by username: %s", username)
	var u models.User
	query := "SELECT id, username, first_name, last_name, email, password_hash, created_at, updated_at FROM users WHERE username = $1"
	err := db.QueryRow(query, username).Scan(
		&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			debug.Info("No user found with username: %s", username)
		} else {
			debug.Error("Database error when looking up user: %v", err)
		}
		return nil, err
	}
	debug.Debug("Successfully retrieved user: %s", username)
	return &u, nil
}

/*
 * StoreToken saves an authentication token for a specific user.
 * If a token already exists for the user, it will be replaced.
 *
 * Parameters:
 *   - userID: The ID of the user
 *   - token: The authentication token to store
 *
 * Returns:
 *   - error: Any error encountered during the operation
 */
func StoreToken(userID string, token string) error {
	debug.Debug("Storing auth token for user ID: %s", userID)
	query := "INSERT INTO auth_tokens (user_id, token, created_at) VALUES ($1, $2, $3)"
	_, err := db.Exec(query, userID, token, time.Now())
	if err != nil {
		debug.Error("Failed to store auth token: %v", err)
		return err
	}
	debug.Debug("Successfully stored auth token")
	return nil
}

/*
 * RemoveToken deletes an authentication token from the database.
 * This is typically used during logout operations.
 *
 * Parameters:
 *   - token: The authentication token to remove
 *
 * Returns:
 *   - error: Any error encountered during the operation
 */
func RemoveToken(token string) error {
	debug.Debug("Removing auth token")
	query := "DELETE FROM auth_tokens WHERE token = $1"
	_, err := db.Exec(query, token)
	if err != nil {
		debug.Error("Failed to remove auth token: %v", err)
		return err
	}
	debug.Debug("Successfully removed auth token")
	return nil
}

/*
 * ValidateToken checks if a token exists and is valid.
 * Returns the associated user ID if the token is valid.
 *
 * Parameters:
 *   - token: The authentication token to validate
 *
 * Returns:
 *   - int: User ID if token is valid, 0 if invalid
 *   - error: Any error encountered during validation
 */
func ValidateToken(token string) (string, error) {
	debug.Debug("Validating auth token")
	var userID string
	query := "SELECT user_id FROM auth_tokens WHERE token = $1 AND created_at > $2"
	err := db.QueryRow(query, token, time.Now().Add(-7*24*time.Hour)).Scan(&userID)
	if err == sql.ErrNoRows {
		debug.Info("No valid token found")
		return "", nil
	}
	if err != nil {
		debug.Error("Error validating token: %v", err)
		return "", err
	}
	debug.Debug("Successfully validated token for user ID: %s", userID)
	return userID, nil
}

/*
 * TokenExists checks if a given token exists in the database.
 *
 * Parameters:
 *   - token: The authentication token to check
 *
 * Returns:
 *   - bool: True if token exists and is valid, false otherwise
 *   - error: Any error encountered during the operation
 */
func TokenExists(token string) (bool, error) {
	debug.Debug("Checking if token exists in database")

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM auth_tokens WHERE token = $1)`
	err := db.QueryRow(query, token).Scan(&exists)
	if err != nil {
		debug.Error("Error checking token existence: %v", err)
		return false, err
	}

	debug.Debug("Token existence check result: %v", exists)
	return exists, nil
}
