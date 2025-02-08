package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	Username            string     `json:"username" db:"username"`
	FirstName           string     `json:"firstName,omitempty"`
	LastName            string     `json:"lastName,omitempty"`
	Email               string     `json:"email" db:"email"`
	PasswordHash        string     `json:"-" db:"password_hash"`
	Role                string     `json:"role" db:"role"`
	Teams               []Team     `json:"teams,omitempty"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
	MFAEnabled          bool       `json:"mfa_enabled" db:"mfa_enabled"`
	MFAType             string     `json:"mfa_type" db:"mfa_type"`
	MFASecret           string     `json:"-" db:"mfa_secret"`
	BackupCodes         []string   `json:"-" db:"backup_codes"`
	PreferredMFAMethod  string     `json:"preferred_mfa_method" db:"preferred_mfa_method"`
	LastPasswordChange  time.Time  `json:"last_password_change" db:"last_password_change"`
	FailedLoginAttempts int        `json:"failed_login_attempts" db:"failed_login_attempts"`
	LastFailedAttempt   *time.Time `json:"last_failed_attempt" db:"last_failed_attempt"`
	AccountLocked       bool       `json:"account_locked" db:"account_locked"`
	AccountLockedUntil  *time.Time `json:"account_locked_until" db:"account_locked_until"`
	AccountEnabled      bool       `json:"account_enabled" db:"account_enabled"`
	LastLogin           *time.Time `json:"last_login" db:"last_login"`
	DisabledReason      *string    `json:"disabled_reason" db:"disabled_reason"`
	DisabledAt          *time.Time `json:"disabled_at" db:"disabled_at"`
	DisabledBy          *uuid.UUID `json:"disabled_by" db:"disabled_by"`
}

// UserMFAData represents sensitive MFA data for a user
type UserMFAData struct {
	ID                 string   `json:"id" db:"id"`
	MFAEnabled         bool     `json:"mfa_enabled" db:"mfa_enabled"`
	MFASecret          string   `json:"mfa_secret" db:"mfa_secret"`
	BackupCodes        []string `json:"backup_codes" db:"backup_codes"`
	PreferredMFAMethod string   `json:"preferred_mfa_method" db:"preferred_mfa_method"`
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

// ScanBackupCodes scans a backup codes array from the database
func (u *User) ScanBackupCodes(value interface{}) error {
	if value == nil {
		u.BackupCodes = []string{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		// Try parsing as JSON array first
		if err := json.Unmarshal(v, &u.BackupCodes); err == nil {
			return nil
		}
		// If JSON parsing fails, try parsing as Postgres array string
		str := string(v)
		if str[0] == '{' && str[len(str)-1] == '}' {
			// Remove the curly braces and split by comma
			str = str[1 : len(str)-1]
			if str == "" {
				u.BackupCodes = []string{}
				return nil
			}
			u.BackupCodes = strings.Split(str, ",")
			return nil
		}
		return fmt.Errorf("invalid backup codes format: %s", str)
	case string:
		// Try parsing as JSON array first
		if err := json.Unmarshal([]byte(v), &u.BackupCodes); err == nil {
			return nil
		}
		// If JSON parsing fails, try parsing as Postgres array string
		if v[0] == '{' && v[len(v)-1] == '}' {
			// Remove the curly braces and split by comma
			v = v[1 : len(v)-1]
			if v == "" {
				u.BackupCodes = []string{}
				return nil
			}
			u.BackupCodes = strings.Split(v, ",")
			return nil
		}
		return fmt.Errorf("invalid backup codes format: %s", v)
	case []string:
		u.BackupCodes = v
		return nil
	default:
		return fmt.Errorf("unsupported type for backup codes: %T", value)
	}
}

// BackupCodesValue returns the backup codes in a format suitable for database storage
func (u User) BackupCodesValue() (driver.Value, error) {
	if len(u.BackupCodes) == 0 {
		return nil, nil
	}
	return u.BackupCodes, nil
}
