package repository

import (
	"database/sql"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/models"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := "SELECT id, username, first_name, last_name, email, password_hash, created_at, updated_at FROM users WHERE username = $1"
	err := r.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) CreateUser(user *models.User) error {
	query := "INSERT INTO users (username, first_name, last_name, email, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)"
	_, err := r.db.Exec(query, user.Username, user.FirstName, user.LastName, user.Email, user.PasswordHash, time.Now(), time.Now())
	return err
}

func (r *Repository) UpdateUser(user *models.User) error {
	query := "UPDATE users SET first_name = $1, last_name = $2, email = $3, updated_at = $4 WHERE id = $5"
	_, err := r.db.Exec(query, user.FirstName, user.LastName, user.Email, time.Now(), user.ID)
	return err
}

func (r *Repository) DeleteUser(userID int) error {
	query := "DELETE FROM users WHERE id = $1"
	_, err := r.db.Exec(query, userID)
	return err
}

// Add more methods as needed, such as UpdateUser, DeleteUser, etc.
