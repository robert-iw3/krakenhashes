package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db/queries"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// GetUserByUsername retrieves a user by their username
func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := db.QueryRow(queries.GetUserByUsername, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		debug.Error("Failed to get user by username: %v", err)
		return nil, err
	}
	return user, nil
}

// StoreToken stores a JWT token for a user
func (db *DB) StoreToken(userID, token string) error {
	// Calculate expiration time (24 hours from now)
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err := db.Exec(queries.StoreToken, userID, token, expiresAt)
	if err != nil {
		debug.Error("Failed to store token: %v", err)
		return err
	}
	return nil
}

// RemoveToken removes a JWT token from storage
func (db *DB) RemoveToken(token string) error {
	_, err := db.Exec(queries.RemoveToken, token)
	if err != nil {
		debug.Error("Failed to remove token: %v", err)
		return err
	}
	return nil
}

// TokenExists checks if a JWT token exists in storage
func (db *DB) TokenExists(token string) (bool, error) {
	var exists bool
	err := db.QueryRow(queries.TokenExists, token).Scan(&exists)
	if err != nil {
		debug.Error("Failed to check token existence: %v", err)
		return false, err
	}
	return exists, nil
}

// UpdateTokenActivity updates the last_activity timestamp for a token
func (db *DB) UpdateTokenActivity(token string) error {
	_, err := db.Exec(queries.UpdateTokenActivity, token)
	if err != nil {
		debug.Error("Failed to update token activity: %v", err)
		return err
	}
	return nil
}

// IsTokenExpiredByIdleTimeout checks if a token has exceeded the idle timeout
func (db *DB) IsTokenExpiredByIdleTimeout(token string) (bool, error) {
	var expired bool
	err := db.QueryRow(queries.IsTokenExpiredByIdleTimeout, token).Scan(&expired)
	if err != nil {
		debug.Error("Failed to check token idle timeout: %v", err)
		return false, err
	}
	return expired, nil
}

// CreateLoginAttempt records a login attempt
func (db *DB) CreateLoginAttempt(attempt *models.LoginAttempt) error {
	_, err := db.Exec(queries.CreateLoginAttempt,
		attempt.UserID,
		attempt.Username,
		attempt.IPAddress,
		attempt.UserAgent,
		attempt.Success,
		attempt.FailureReason)
	return err
}

// GetUserLoginAttempts retrieves login attempts for a user
func (db *DB) GetUserLoginAttempts(userID uuid.UUID, limit int) ([]*models.LoginAttempt, error) {
	rows, err := db.Query(queries.GetUserLoginAttempts, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []*models.LoginAttempt
	for rows.Next() {
		attempt := &models.LoginAttempt{}
		err := rows.Scan(
			&attempt.ID,
			&attempt.UserID,
			&attempt.Username,
			&attempt.IPAddress,
			&attempt.UserAgent,
			&attempt.Success,
			&attempt.FailureReason,
			&attempt.AttemptedAt,
			&attempt.Notified,
		)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, attempt)
	}
	return attempts, rows.Err()
}

// GetUnnotifiedFailedAttempts retrieves unnotified failed login attempts
func (db *DB) GetUnnotifiedFailedAttempts(since string) ([]*models.LoginAttempt, error) {
	rows, err := db.Query(queries.GetUnnotifiedFailedAttempts, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []*models.LoginAttempt
	for rows.Next() {
		attempt := &models.LoginAttempt{}
		err := rows.Scan(
			&attempt.ID,
			&attempt.UserID,
			&attempt.Username,
			&attempt.IPAddress,
			&attempt.UserAgent,
			&attempt.Success,
			&attempt.FailureReason,
			&attempt.AttemptedAt,
			&attempt.Notified,
		)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, attempt)
	}
	return attempts, rows.Err()
}

// MarkAttemptsAsNotified marks login attempts as notified
func (db *DB) MarkAttemptsAsNotified(ids []uuid.UUID) error {
	_, err := db.Exec(queries.MarkAttemptsAsNotified, ids)
	return err
}

// CreateSession creates a new active session
func (db *DB) CreateSession(session *models.ActiveSession) error {
	_, err := db.Exec(queries.CreateSession,
		session.UserID,
		session.IPAddress,
		session.UserAgent)
	return err
}

// UpdateSessionActivity updates the last active time for a session
func (db *DB) UpdateSessionActivity(sessionID uuid.UUID) error {
	_, err := db.Exec(queries.UpdateSessionActivity, sessionID)
	return err
}

// DeleteSession deletes a specific session
func (db *DB) DeleteSession(sessionID uuid.UUID) error {
	_, err := db.Exec(queries.DeleteSession, sessionID)
	return err
}

// DeleteUserSessions deletes all sessions for a user
func (db *DB) DeleteUserSessions(userID uuid.UUID) error {
	_, err := db.Exec(queries.DeleteUserSessions, userID)
	return err
}

// GetUserSessions retrieves all active sessions for a user
func (db *DB) GetUserSessions(userID uuid.UUID) ([]*models.ActiveSession, error) {
	rows, err := db.Query(queries.GetUserSessions, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.ActiveSession
	for rows.Next() {
		session := &models.ActiveSession{}
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.IPAddress,
			&session.UserAgent,
			&session.CreatedAt,
			&session.LastActiveAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

// UpdateUserAuthInfo updates a user's authentication information
func (db *DB) UpdateUserAuthInfo(userID uuid.UUID, info *models.UserAuthInfo) error {
	_, err := db.Exec(queries.UpdateUserAuthInfo,
		info.MFAEnabled,
		info.MFAType,
		info.MFASecret,
		info.BackupCodes,
		userID)
	return err
}

// IncrementFailedAttempts increments the failed login attempts for a user
func (db *DB) IncrementFailedAttempts(userID uuid.UUID) (int, error) {
	var attempts int
	err := db.QueryRow(queries.IncrementFailedAttempts, userID).Scan(&attempts)
	return attempts, err
}

// ResetFailedAttempts resets the failed login attempts for a user
func (db *DB) ResetFailedAttempts(userID uuid.UUID) error {
	_, err := db.Exec(queries.ResetFailedAttempts, userID)
	return err
}

// LockUserAccount locks a user account
func (db *DB) LockUserAccount(userID uuid.UUID, durationMinutes int) error {
	_, err := db.Exec(queries.LockUserAccount, durationMinutes, userID)
	return err
}

// DisableUserAccount disables a user account
func (db *DB) DisableUserAccount(userID uuid.UUID, reason string, disabledBy uuid.UUID) error {
	_, err := db.Exec(queries.DisableUserAccount, reason, disabledBy, userID)
	return err
}

// EnableUserAccount enables a user account
func (db *DB) EnableUserAccount(userID uuid.UUID) error {
	_, err := db.Exec(queries.EnableUserAccount, userID)
	return err
}

// UpdateLastLogin updates the last login time for a user
func (db *DB) UpdateLastLogin(userID uuid.UUID) error {
	_, err := db.Exec(queries.UpdateLastLogin, userID)
	return err
}

// GetUserByID retrieves a user by their ID, excluding sensitive MFA data
func (db *DB) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	var skipMFASecret, skipBackupCodes, disabledReason sql.NullString
	var lastFailedAttempt, accountLockedUntil, lastLogin, disabledAt sql.NullTime
	var disabledBy sql.NullString
	var mfaType []string

	err := db.QueryRow(queries.GetUserByID, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.MFAEnabled,
		pq.Array(&mfaType),
		&skipMFASecret,
		&skipBackupCodes,
		&user.PreferredMFAMethod,
		&user.LastPasswordChange,
		&user.FailedLoginAttempts,
		&lastFailedAttempt,
		&user.AccountLocked,
		&accountLockedUntil,
		&user.AccountEnabled,
		&lastLogin,
		&disabledReason,
		&disabledAt,
		&disabledBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		debug.Error("Failed to get user by ID: %v", err)
		return nil, err
	}

	// Set the scanned mfa_type array
	user.MFAType = mfaType

	// Handle nullable fields
	if lastFailedAttempt.Valid {
		user.LastFailedAttempt = &lastFailedAttempt.Time
	}
	if accountLockedUntil.Valid {
		user.AccountLockedUntil = &accountLockedUntil.Time
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	if disabledAt.Valid {
		user.DisabledAt = &disabledAt.Time
	}
	if disabledReason.Valid {
		user.DisabledReason = &disabledReason.String
	}
	if disabledBy.Valid {
		id, err := uuid.Parse(disabledBy.String)
		if err == nil {
			user.DisabledBy = &id
		}
	}

	return &user, nil
}

// GetUserWithMFAData gets a user with sensitive MFA data
func (db *DB) GetUserWithMFAData(userID string) (*models.UserMFAData, error) {
	var user models.UserMFAData
	var backupCodes []string

	query := `
		SELECT 
			id,
			mfa_enabled,
			mfa_secret,
			backup_codes,
			preferred_mfa_method
		FROM users 
		WHERE id = $1
	`

	err := db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.MFAEnabled,
		&user.MFASecret,
		pq.Array(&backupCodes),
		&user.PreferredMFAMethod,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user MFA data: %w", err)
	}

	user.BackupCodes = backupCodes
	return &user, nil
}

// SetPreferredMFAMethod sets the user's preferred MFA method
func (db *DB) SetPreferredMFAMethod(userID string, method string) error {
	_, err := db.Exec(queries.SetPreferredMFAMethodQuery, userID, method)
	return err
}

// GetUserMFASettings gets a user's MFA settings
func (db *DB) GetUserMFASettings(userID string) (*models.UserMFAData, error) {
	var settings models.UserMFAData
	var mfaType []string
	var backupCodes []string
	var mfaSecret sql.NullString
	var preferredMethod sql.NullString

	err := db.QueryRow(queries.GetUserMFASettingsQuery, userID).Scan(
		&settings.MFAEnabled,
		pq.Array(&mfaType),
		&preferredMethod,
		&mfaSecret,
		pq.Array(&backupCodes),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user MFA settings: %w", err)
	}

	settings.MFAType = mfaType
	settings.BackupCodes = backupCodes
	settings.ID = userID

	// Handle NULL values
	if mfaSecret.Valid {
		settings.MFASecret = mfaSecret.String
	}
	if preferredMethod.Valid {
		settings.PreferredMFAMethod = preferredMethod.String
	} else {
		settings.PreferredMFAMethod = "email" // Default to email if not set
	}

	return &settings, nil
}

// GetUserIDFromMFASession retrieves the user ID associated with an MFA session token
func (db *DB) GetUserIDFromMFASession(sessionToken string) (string, error) {
	var userID string
	err := db.QueryRow(`
		SELECT user_id 
		FROM mfa_sessions 
		WHERE session_token = $1 
		AND expires_at > NOW()`,
		sessionToken,
	).Scan(&userID)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("invalid or expired session token")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user ID from MFA session: %w", err)
	}

	return userID, nil
}

// ClearMFASession removes an MFA session after successful verification
func (db *DB) ClearMFASession(sessionToken string) error {
	_, err := db.Exec(`
		DELETE FROM mfa_sessions 
		WHERE session_token = $1`,
		sessionToken,
	)
	if err != nil {
		return fmt.Errorf("failed to clear MFA session: %w", err)
	}
	return nil
}
