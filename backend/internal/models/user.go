package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	FirstName    string    `json:"firstName,omitempty"`
	LastName     string    `json:"lastName,omitempty"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Teams        []Team    `json:"teams,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// SetPassword hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedBytes)
	return nil
}

// CheckPassword verifies if the provided password matches the user's hashed password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// NewUser creates a new user with a generated UUID
func NewUser(username, email string) *User {
	return &User{
		ID:        uuid.New(),
		Username:  username,
		Email:     email,
		Role:      "user", // Default role
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ScanTeams scans a JSON-encoded teams string into the Teams slice
func (u *User) ScanTeams(value interface{}) error {
	if value == nil {
		u.Teams = []Team{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &u.Teams)
	case string:
		return json.Unmarshal([]byte(v), &u.Teams)
	default:
		return fmt.Errorf("unsupported type for teams: %T", value)
	}
}

// Value returns the JSON encoding of Teams for database storage
func (u User) TeamsValue() (driver.Value, error) {
	if len(u.Teams) == 0 {
		return nil, nil
	}
	return json.Marshal(u.Teams)
}
