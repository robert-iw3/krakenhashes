package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db/queries"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/google/uuid"
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *db.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *db.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	err := r.db.QueryRowContext(ctx, queries.CreateUser,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	var mfaType interface{}
	var backupCodes interface{}
	var mfaSecret sql.NullString
	var preferredMFAMethod sql.NullString
	var lastFailedAttempt, accountLockedUntil, lastLogin, disabledAt sql.NullTime
	var disabledReason sql.NullString

	err := r.db.QueryRowContext(ctx, queries.GetUserByID, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.MFAEnabled,
		&mfaType,
		&mfaSecret,
		&backupCodes,
		&preferredMFAMethod,
		&user.LastPasswordChange,
		&user.FailedLoginAttempts,
		&lastFailedAttempt,
		&user.AccountLocked,
		&accountLockedUntil,
		&user.AccountEnabled,
		&lastLogin,
		&disabledReason,
		&disabledAt,
		&user.DisabledBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", id)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Handle nullable fields
	if mfaSecret.Valid {
		user.MFASecret = mfaSecret.String
	}
	if preferredMFAMethod.Valid {
		user.PreferredMFAMethod = preferredMFAMethod.String
	}
	if lastFailedAttempt.Valid {
		user.LastFailedAttempt = &lastFailedAttempt.Time
	}
	if accountLockedUntil.Valid {
		user.AccountLockedUntil = &accountLockedUntil.Time
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	if disabledReason.Valid {
		user.DisabledReason = &disabledReason.String
	}
	if disabledAt.Valid {
		user.DisabledAt = &disabledAt.Time
	}

	// Handle MFA type array
	if err := user.ScanMFAType(mfaType); err != nil {
		return nil, fmt.Errorf("failed to scan MFA type: %w", err)
	}

	// Handle backup codes array
	if err := user.ScanBackupCodes(backupCodes); err != nil {
		return nil, fmt.Errorf("failed to scan backup codes: %w", err)
	}

	// Get user's teams
	teams, err := r.getUserTeams(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user teams: %w", err)
	}
	user.Teams = teams

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}

	err := r.db.QueryRowContext(ctx, queries.GetUserByEmail, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found with email: %s", email)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Get user's teams
	teams, err := r.getUserTeams(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user teams: %w", err)
	}
	user.Teams = teams

	return user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	result, err := r.db.ExecContext(ctx, queries.UpdateUser,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found: %s", user.ID)
	}

	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, queries.DeleteUser, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	return nil
}

// AddToTeam adds a user to a team
func (r *UserRepository) AddToTeam(ctx context.Context, userID, teamID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, queries.AddUserToTeam, userID, teamID)
	if err != nil {
		return fmt.Errorf("failed to add user to team: %w", err)
	}
	return nil
}

// RemoveFromTeam removes a user from a team
func (r *UserRepository) RemoveFromTeam(ctx context.Context, userID, teamID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, queries.RemoveUserFromTeam, userID, teamID)
	if err != nil {
		return fmt.Errorf("failed to remove user from team: %w", err)
	}
	return nil
}

// getUserTeams retrieves all teams for a user
func (r *UserRepository) getUserTeams(ctx context.Context, userID uuid.UUID) ([]models.Team, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.description, t.created_at, t.updated_at
		FROM teams t
		JOIN user_teams ut ON t.id = ut.team_id
		WHERE ut.user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user teams: %w", err)
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var team models.Team
		err := rows.Scan(
			&team.ID,
			&team.Name,
			&team.Description,
			&team.CreatedAt,
			&team.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, team)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating teams: %w", err)
	}

	return teams, nil
}

// List retrieves all users with optional filters
func (r *UserRepository) List(ctx context.Context, filters map[string]interface{}) ([]models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	argPos := 1

	// Add filters
	if role, ok := filters["role"].(string); ok {
		query += fmt.Sprintf(" AND role = $%d", argPos)
		args = append(args, role)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// Get user's teams
		teams, err := r.getUserTeams(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user teams: %w", err)
		}
		user.Teams = teams

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// ListAll retrieves all users for admin view
func (r *UserRepository) ListAll(ctx context.Context) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx, queries.ListAllUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to list all users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var mfaType interface{}
		var preferredMFAMethod sql.NullString
		var lastLogin, disabledAt, lockedUntil sql.NullTime
		var disabledReason sql.NullString
		var disabledBy *uuid.UUID

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Role,
			&user.AccountEnabled,
			&user.AccountLocked,
			&lockedUntil,
			&user.MFAEnabled,
			&mfaType,
			&preferredMFAMethod,
			&user.CreatedAt,
			&user.UpdatedAt,
			&lastLogin,
			&disabledReason,
			&disabledAt,
			&disabledBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// Handle nullable fields
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}
		if disabledAt.Valid {
			user.DisabledAt = &disabledAt.Time
		}
		if lockedUntil.Valid {
			user.AccountLockedUntil = &lockedUntil.Time
		}
		if disabledReason.Valid {
			user.DisabledReason = &disabledReason.String
		}
		if preferredMFAMethod.Valid {
			user.PreferredMFAMethod = preferredMFAMethod.String
		}
		user.DisabledBy = disabledBy

		// Handle MFA type array
		if err := user.ScanMFAType(mfaType); err != nil {
			return nil, fmt.Errorf("failed to scan MFA type: %w", err)
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// GetDetails retrieves detailed user information for admin view
func (r *UserRepository) GetDetails(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	var mfaType interface{}
	var preferredMFAMethod sql.NullString
	var lastLogin, lastPasswordChange, lastFailedAttempt, disabledAt, lockedUntil sql.NullTime
	var disabledReason sql.NullString
	var disabledBy *uuid.UUID

	err := r.db.QueryRowContext(ctx, queries.GetUserDetails, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.AccountEnabled,
		&user.AccountLocked,
		&lockedUntil,
		&user.MFAEnabled,
		&mfaType,
		&preferredMFAMethod,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLogin,
		&lastPasswordChange,
		&user.FailedLoginAttempts,
		&lastFailedAttempt,
		&disabledReason,
		&disabledAt,
		&disabledBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", id)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}

	// Handle nullable fields
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	if lastPasswordChange.Valid {
		user.LastPasswordChange = lastPasswordChange.Time
	}
	if lastFailedAttempt.Valid {
		user.LastFailedAttempt = &lastFailedAttempt.Time
	}
	if disabledAt.Valid {
		user.DisabledAt = &disabledAt.Time
	}
	if lockedUntil.Valid {
		user.AccountLockedUntil = &lockedUntil.Time
	}
	if disabledReason.Valid {
		user.DisabledReason = &disabledReason.String
	}
	if preferredMFAMethod.Valid {
		user.PreferredMFAMethod = preferredMFAMethod.String
	}
	user.DisabledBy = disabledBy

	// Handle MFA type array
	if err := user.ScanMFAType(mfaType); err != nil {
		return nil, fmt.Errorf("failed to scan MFA type: %w", err)
	}

	// Get user's teams
	teams, err := r.getUserTeams(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user teams: %w", err)
	}
	user.Teams = teams

	return user, nil
}

// DisableAccount disables a user account
func (r *UserRepository) DisableAccount(ctx context.Context, userID uuid.UUID, reason string, adminID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, queries.DisableUserAccount, userID, reason, adminID)
	if err != nil {
		return fmt.Errorf("failed to disable user account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}

	return nil
}

// EnableAccount re-enables a user account
func (r *UserRepository) EnableAccount(ctx context.Context, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, queries.EnableUserAccount, userID)
	if err != nil {
		return fmt.Errorf("failed to enable user account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}

	return nil
}

// ResetPassword updates a user's password (admin action)
func (r *UserRepository) ResetPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	result, err := r.db.ExecContext(ctx, queries.AdminResetUserPassword, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to reset user password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}

	return nil
}

// DisableMFA disables MFA for a user
func (r *UserRepository) DisableMFA(ctx context.Context, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, queries.DisableUserMFA, userID)
	if err != nil {
		return fmt.Errorf("failed to disable user MFA: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}

	return nil
}

// UpdateDetails updates username and/or email and/or role
func (r *UserRepository) UpdateDetails(ctx context.Context, userID uuid.UUID, username, email, role *string) error {
	// Check for duplicate username
	if username != nil {
		var exists bool
		err := r.db.QueryRowContext(ctx, queries.CheckUsernameExists, *username, userID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check username: %w", err)
		}
		if exists {
			return fmt.Errorf("username already exists: %s", *username)
		}
	}

	// Check for duplicate email
	if email != nil {
		var exists bool
		err := r.db.QueryRowContext(ctx, queries.CheckEmailExists, *email, userID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check email: %w", err)
		}
		if exists {
			return fmt.Errorf("email already exists: %s", *email)
		}
	}

	result, err := r.db.ExecContext(ctx, queries.UpdateUserDetails, userID, username, email, role)
	if err != nil {
		return fmt.Errorf("failed to update user details: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}

	return nil
}

// UnlockAccount unlocks a locked user account
func (r *UserRepository) UnlockAccount(ctx context.Context, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, queries.UnlockUserAccount, userID)
	if err != nil {
		return fmt.Errorf("failed to unlock user account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}

	return nil
}
