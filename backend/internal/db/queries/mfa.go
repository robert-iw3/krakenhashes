package queries

import (
	"database/sql"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
)

// MFA-related queries
const (
	// Check if MFA is required by policy
	IsMFARequiredQuery = `
		SELECT require_mfa FROM auth_settings LIMIT 1;
	`

	// Enable MFA for a user
	EnableMFAQuery = `
		UPDATE users 
		SET mfa_enabled = true, mfa_type = $2, mfa_secret = $3, updated_at = NOW() 
		WHERE id = $1;
	`

	// Disable MFA for a user
	DisableMFAQuery = `
		UPDATE users 
		SET mfa_enabled = false, mfa_type = 'email', mfa_secret = NULL, backup_codes = NULL, updated_at = NOW() 
		WHERE id = $1;
	`

	// Store pending MFA setup
	StorePendingMFASetupQuery = `
		INSERT INTO pending_mfa_setup (user_id, method, secret)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) 
		DO UPDATE SET method = $2, secret = $3, created_at = NOW();
	`

	// Get pending MFA setup
	GetPendingMFASetupQuery = `
		SELECT secret 
		FROM pending_mfa_setup 
		WHERE user_id = $1 AND created_at > NOW() - INTERVAL '5 minutes';
	`

	// Store MFA secret
	StoreMFASecretQuery = `
		UPDATE users 
		SET mfa_secret = $2, updated_at = NOW() 
		WHERE id = $1;
	`

	// Store backup codes
	StoreBackupCodesQuery = `
		UPDATE users 
		SET backup_codes = $2, updated_at = NOW() 
		WHERE id = $1;
	`

	// Get user's MFA settings
	GetUserMFASettingsQuery = `
		SELECT mfa_enabled, mfa_type, preferred_mfa_method
		FROM users 
		WHERE id = $1
	`

	// Delete expired pending MFA setups
	CleanupPendingMFASetupQuery = `
		DELETE FROM pending_mfa_setup 
		WHERE created_at < NOW() - INTERVAL '5 minutes';
	`

	// Store email MFA code
	StoreEmailMFACodeQuery = `
		WITH settings AS (
			SELECT mfa_code_expiry_minutes FROM auth_settings LIMIT 1
		)
		INSERT INTO email_mfa_codes (user_id, code, expires_at)
		VALUES (
			$1, 
			$2, 
			NOW() + ((SELECT mfa_code_expiry_minutes FROM settings) || ' minutes')::INTERVAL
		)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			code = $2, 
			expires_at = NOW() + ((SELECT mfa_code_expiry_minutes FROM settings) || ' minutes')::INTERVAL,
			attempts = 0;
	`

	// Verify email MFA code
	VerifyEmailMFACodeQuery = `
		WITH updated AS (
			UPDATE email_mfa_codes 
			SET attempts = attempts + 1
			WHERE user_id = $1 
				AND code = $2 
				AND expires_at > NOW() 
				AND attempts < (SELECT mfa_max_attempts FROM auth_settings)
			RETURNING true AS success
		)
		SELECT COALESCE((SELECT success FROM updated), false);
	`

	// Delete used/expired email MFA codes
	CleanupEmailMFACodesQuery = `
		DELETE FROM email_mfa_codes 
		WHERE expires_at < NOW() 
			OR attempts >= (SELECT mfa_max_attempts FROM auth_settings);
	`

	// Check cooldown period for email MFA codes
	CheckEmailMFACooldownQuery = `
		SELECT EXISTS (
			SELECT 1 
			FROM email_mfa_codes 
			WHERE user_id = $1 
				AND created_at > NOW() - ((SELECT mfa_code_cooldown_minutes FROM auth_settings LIMIT 1) || ' minutes')::INTERVAL
		);
	`

	GetMFAVerifyAttemptsQuery = `
		SELECT attempts
		FROM mfa_sessions
		WHERE session_token = $1
		AND expires_at > NOW()`

	IncrementMFAVerifyAttemptsQuery = `
		UPDATE mfa_sessions 
		SET attempts = attempts + 1 
		WHERE session_token = $1
		AND expires_at > NOW()
		RETURNING attempts`

	ClearMFAVerifyAttemptsQuery = `
		UPDATE mfa_sessions 
		SET attempts = 0 
		WHERE session_token = $1
		AND expires_at > NOW()`

	GetUserByIDQuery = `
		SELECT id, username, email, role, mfa_enabled, mfa_type, mfa_secret, backup_codes,
			last_password_change, failed_login_attempts, last_failed_attempt,
			account_locked, account_locked_until, account_enabled, last_login,
			disabled_reason, disabled_at, disabled_by
		FROM users
		WHERE id = $1`

	// Backup codes queries
	ValidateAndUseBackupCodeQuery = `
		UPDATE users 
		SET backup_codes = array_remove(backup_codes, $2),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 
		AND $2 = ANY(backup_codes)
		RETURNING id`

	GetUnusedBackupCodesCountQuery = `
		SELECT array_length(backup_codes, 1)
		FROM users 
		WHERE id = $1`

	ClearBackupCodesQuery = `
		UPDATE users 
		SET backup_codes = ARRAY[]::text[],
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	// Set preferred MFA method
	SetPreferredMFAMethodQuery = `
		UPDATE users 
		SET preferred_mfa_method = $2,
			updated_at = NOW() 
		WHERE id = $1
	`

	// Get count of remaining backup codes
	GetRemainingBackupCodesCountQuery = `
		SELECT COALESCE(ARRAY_LENGTH(backup_codes, 1), 0)
		FROM users
		WHERE id = $1;
	`

	// Validate and consume a backup code
	ValidateAndConsumeBackupCodeQuery = `
		WITH valid_code AS (
			SELECT backup_codes
			FROM users
			WHERE id = $1
			AND $2 = ANY(backup_codes)
		)
		UPDATE users
		SET backup_codes = ARRAY_REMOVE(backup_codes, $2),
			updated_at = NOW()
		FROM valid_code
		WHERE id = $1
		AND backup_codes = valid_code.backup_codes
		RETURNING id;
	`

	// Store new backup codes
	StoreNewBackupCodesQuery = `
		UPDATE users 
		SET backup_codes = $2,
			updated_at = NOW() 
		WHERE id = $1;
	`

	// MFA Session queries
	// Create MFA session
	CreateMFASessionQuery = `
		INSERT INTO mfa_sessions (user_id, session_token, expires_at, attempts)
		VALUES ($1, $2, NOW() + INTERVAL '5 minutes', 0)
		RETURNING id, expires_at`

	// Get MFA session
	GetMFASessionQuery = `
		SELECT user_id, attempts, expires_at 
		FROM mfa_sessions 
		WHERE session_token = $1 AND expires_at > NOW();
	`

	// Increment MFA session attempts
	IncrementMFASessionAttemptsQuery = `
		UPDATE mfa_sessions 
		SET attempts = attempts + 1
		WHERE session_token = $1 AND expires_at > NOW()
		RETURNING attempts;
	`

	// Delete MFA session
	DeleteMFASessionQuery = `
		DELETE FROM mfa_sessions 
		WHERE session_token = $1;
	`

	// Delete expired MFA sessions
	DeleteExpiredMFASessionsQuery = `
		DELETE FROM mfa_sessions 
		WHERE expires_at < NOW();
	`

	// Check if user requires MFA
	CheckMFARequiredQuery = `
		SELECT 
			CASE 
				WHEN (SELECT require_mfa FROM auth_settings LIMIT 1) THEN true
				WHEN u.mfa_enabled THEN true
				ELSE false
			END as requires_mfa,
			u.mfa_type,
			u.preferred_mfa_method
		FROM users u
		WHERE u.id = $1;
	`
)

// MFASession represents an MFA verification session
type MFASession struct {
	ID           string    `db:"id"`
	UserID       string    `db:"user_id"`
	SessionToken string    `db:"session_token"`
	ExpiresAt    time.Time `db:"expires_at"`
	Attempts     int       `db:"attempts"`
}

// MFARequirement represents whether a user requires MFA and their MFA settings
type MFARequirement struct {
	RequiresMFA        bool   `db:"requires_mfa"`
	MFAType            string `db:"mfa_type"`
	PreferredMFAMethod string `db:"preferred_mfa_method"`
}

// DB represents a database connection
type DB struct {
	*sql.DB
}

// StorePendingMFASetup stores a pending MFA setup for a user
func (db *DB) StorePendingMFASetup(userID, method, secret string) error {
	_, err := db.Exec(StorePendingMFASetupQuery, userID, method, secret)
	return err
}

// GetPendingMFASetup retrieves a pending MFA setup
func (db *DB) GetPendingMFASetup(userID string) (string, error) {
	var secret string
	err := db.QueryRow(GetPendingMFASetupQuery, userID).Scan(&secret)
	if err == sql.ErrNoRows {
		return "", models.ErrNotFound
	}
	return secret, err
}

// StoreEmailMFACode stores an email verification code
func (db *DB) StoreEmailMFACode(userID, code string) error {
	_, err := db.Exec(StoreEmailMFACodeQuery, userID, code)
	return err
}

// GetMFAVerifyAttempts gets the number of verification attempts
func (db *DB) GetMFAVerifyAttempts(userID string) (int, error) {
	var attempts int
	err := db.QueryRow(GetMFAVerifyAttemptsQuery, userID).Scan(&attempts)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return attempts, err
}

// IncrementMFAVerifyAttempts increments the verification attempts counter
func (db *DB) IncrementMFAVerifyAttempts(userID string) error {
	_, err := db.Exec(IncrementMFAVerifyAttemptsQuery, userID)
	return err
}

// ClearMFAVerifyAttempts resets the verification attempts counter
func (db *DB) ClearMFAVerifyAttempts(userID string) error {
	_, err := db.Exec(ClearMFAVerifyAttemptsQuery, userID)
	return err
}

// EnableMFA enables MFA for a user
func (db *DB) EnableMFA(userID, method, secret string) error {
	_, err := db.Exec(EnableMFAQuery, userID, method, secret)
	return err
}

// GetUserByID retrieves a user by their ID
func (db *DB) GetUserByID(userID string) (*models.User, error) {
	user := &models.User{}
	err := db.QueryRow(GetUserByIDQuery, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.MFAEnabled,
		&user.MFAType,
		&user.MFASecret,
		&user.BackupCodes,
		&user.LastPasswordChange,
		&user.FailedLoginAttempts,
		&user.LastFailedAttempt,
		&user.AccountLocked,
		&user.AccountLockedUntil,
		&user.AccountEnabled,
		&user.LastLogin,
		&user.DisabledReason,
		&user.DisabledAt,
		&user.DisabledBy,
	)
	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	return user, err
}

// StoreBackupCodes stores backup codes for a user
func (db *DB) StoreBackupCodes(userID string, codes []string) error {
	_, err := db.Exec(StoreBackupCodesQuery, userID, codes)
	return err
}

// ValidateAndUseBackupCode validates a backup code and removes it from the user's backup codes
func (db *DB) ValidateAndUseBackupCode(userID string, code string) (bool, error) {
	var id string
	err := db.QueryRow(ValidateAndUseBackupCodeQuery, userID, code).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetUnusedBackupCodesCount gets the count of unused backup codes
func (db *DB) GetUnusedBackupCodesCount(userID string) (int, error) {
	var count sql.NullInt64
	err := db.QueryRow(GetUnusedBackupCodesCountQuery, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	if !count.Valid {
		return 0, nil
	}
	return int(count.Int64), nil
}

// ClearBackupCodes removes all backup codes for a user
func (db *DB) ClearBackupCodes(userID string) error {
	_, err := db.Exec(ClearBackupCodesQuery, userID)
	return err
}

// CreateMFASession creates a new MFA session
func (db *DB) CreateMFASession(userID, sessionToken string) (*MFASession, error) {
	session := &MFASession{}
	err := db.QueryRow(CreateMFASessionQuery, userID, sessionToken).Scan(&session.ID, &session.ExpiresAt)
	if err != nil {
		return nil, err
	}
	session.UserID = userID
	session.SessionToken = sessionToken
	return session, nil
}

// GetMFASession retrieves an MFA session by token
func (db *DB) GetMFASession(sessionToken string) (*MFASession, error) {
	session := &MFASession{SessionToken: sessionToken}
	err := db.QueryRow(GetMFASessionQuery, sessionToken).Scan(&session.UserID, &session.Attempts, &session.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	return session, err
}

// IncrementMFASessionAttempts increments the attempts counter for an MFA session
func (db *DB) IncrementMFASessionAttempts(sessionToken string) (int, error) {
	var attempts int
	err := db.QueryRow(IncrementMFASessionAttemptsQuery, sessionToken).Scan(&attempts)
	if err == sql.ErrNoRows {
		return 0, models.ErrNotFound
	}
	return attempts, err
}

// DeleteMFASession deletes an MFA session
func (db *DB) DeleteMFASession(sessionToken string) error {
	_, err := db.Exec(DeleteMFASessionQuery, sessionToken)
	return err
}

// DeleteExpiredMFASessions deletes all expired MFA sessions
func (db *DB) DeleteExpiredMFASessions() error {
	_, err := db.Exec(DeleteExpiredMFASessionsQuery)
	return err
}

// CheckMFARequired checks if a user requires MFA
func (db *DB) CheckMFARequired(userID string) (*MFARequirement, error) {
	req := &MFARequirement{}
	err := db.QueryRow(CheckMFARequiredQuery, userID).Scan(&req.RequiresMFA, &req.MFAType, &req.PreferredMFAMethod)
	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	return req, err
}
