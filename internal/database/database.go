package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/models"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

var db *sql.DB

func Connect() (*sql.DB, error) {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func RunMigrations() error {
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
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func GetUserByUsername(username string) (*models.User, error) {
	var u models.User
	query := "SELECT id, username, first_name, last_name, email, password_hash, created_at, updated_at FROM users WHERE username = $1"
	err := db.QueryRow(query, username).Scan(
		&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func StoreToken(userID int, token string) error {
	query := "INSERT INTO auth_tokens (user_id, token, created_at) VALUES ($1, $2, $3)"
	_, err := db.Exec(query, userID, token, time.Now())
	return err
}

func RemoveToken(token string) error {
	query := "DELETE FROM auth_tokens WHERE token = $1"
	_, err := db.Exec(query, token)
	return err
}

func ValidateToken(token string) (int, error) {
	var userID int
	query := "SELECT user_id FROM auth_tokens WHERE token = $1 AND created_at > $2"
	err := db.QueryRow(query, token, time.Now().Add(-7*24*time.Hour)).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return userID, nil
}
