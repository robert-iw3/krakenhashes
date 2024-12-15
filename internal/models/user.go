package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username     string    `json:"username" gorm:"not null;unique"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email" gorm:"not null;unique"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	Teams        []Team    `json:"teams,omitempty" gorm:"many2many:user_teams"`
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// SetPassword sets the hashed password for the user
func (u *User) SetPassword(password string) error {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedBytes)
	return nil
}

// CheckPassword checks if the provided password matches the user's hashed password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}
