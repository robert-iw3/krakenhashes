package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/db"
	"github.com/ZerkerEOD/hashdom-backend/internal/db/queries"
	"github.com/ZerkerEOD/hashdom-backend/internal/models"
	"github.com/google/uuid"
)

// ClaimVoucherRepository handles database operations for claim vouchers
type ClaimVoucherRepository struct {
	db *db.DB
}

// NewClaimVoucherRepository creates a new claim voucher repository
func NewClaimVoucherRepository(db *db.DB) *ClaimVoucherRepository {
	return &ClaimVoucherRepository{db: db}
}

// Create creates a new claim voucher
func (r *ClaimVoucherRepository) Create(ctx context.Context, voucher *models.ClaimVoucher) error {
	err := r.db.QueryRowContext(ctx, queries.CreateClaimVoucher,
		voucher.Code,
		voucher.IsActive,
		voucher.IsContinuous,
		sql.NullTime{Time: voucher.ExpiresAt.Time, Valid: voucher.ExpiresAt.Valid},
		voucher.CreatedByID,
		voucher.CreatedAt,
		voucher.UpdatedAt,
	).Scan(&voucher.Code)

	if err != nil {
		return fmt.Errorf("failed to create claim voucher: %w", err)
	}

	return nil
}

// GetByCode retrieves a claim voucher by code
func (r *ClaimVoucherRepository) GetByCode(ctx context.Context, code string) (*models.ClaimVoucher, error) {
	voucher := &models.ClaimVoucher{}
	var createdByUser, usedByUser models.User
	var usedByID models.NullUUID
	var usedAt, expiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, queries.GetClaimVoucherByCode, code).Scan(
		&voucher.Code,
		&voucher.IsActive,
		&voucher.IsContinuous,
		&expiresAt,
		&voucher.CreatedByID,
		&usedByID,
		&usedAt,
		&voucher.CreatedAt,
		&voucher.UpdatedAt,
		&createdByUser.ID,
		&createdByUser.Username,
		&createdByUser.Email,
		&createdByUser.Role,
		&usedByUser.ID,
		&usedByUser.Username,
		&usedByUser.Email,
		&usedByUser.Role,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("claim voucher not found with code: %s", code)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get claim voucher: %w", err)
	}

	voucher.ExpiresAt = expiresAt
	voucher.UsedAt = usedAt
	voucher.UsedByID = usedByID
	voucher.CreatedBy = &createdByUser
	if usedByID.Valid {
		voucher.UsedBy = &usedByUser
	}

	return voucher, nil
}

// Use marks a claim voucher as used
func (r *ClaimVoucherRepository) Use(ctx context.Context, code string, userID uuid.UUID) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, queries.UseClaimVoucher,
		code,
		userID,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to use claim voucher: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("claim voucher not found or already used: %s", code)
	}

	return nil
}

// Deactivate deactivates a claim voucher
func (r *ClaimVoucherRepository) Deactivate(ctx context.Context, code string) error {
	result, err := r.db.ExecContext(ctx, queries.DeactivateClaimVoucher, code)
	if err != nil {
		return fmt.Errorf("failed to deactivate claim voucher: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("claim voucher not found: %s", code)
	}

	return nil
}

// ListActive retrieves all active claim vouchers
func (r *ClaimVoucherRepository) ListActive(ctx context.Context) ([]models.ClaimVoucher, error) {
	rows, err := r.db.QueryContext(ctx, queries.ListActiveVouchers)
	if err != nil {
		return nil, fmt.Errorf("failed to list active claim vouchers: %w", err)
	}
	defer rows.Close()

	var vouchers []models.ClaimVoucher
	for rows.Next() {
		var voucher models.ClaimVoucher
		var createdByUser, usedByUser models.User
		var usedByID models.NullUUID
		var usedAt, expiresAt sql.NullTime
		var usedByUsername, usedByEmail, usedByRole sql.NullString
		var createdByUsername, createdByEmail, createdByRole sql.NullString

		err := rows.Scan(
			&voucher.Code,
			&voucher.IsActive,
			&voucher.IsContinuous,
			&expiresAt,
			&voucher.CreatedByID,
			&usedByID,
			&usedAt,
			&voucher.CreatedAt,
			&voucher.UpdatedAt,
			&createdByUser.ID,
			&createdByUsername,
			&createdByEmail,
			&createdByRole,
			&usedByUser.ID,
			&usedByUsername,
			&usedByEmail,
			&usedByRole,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan claim voucher: %w", err)
		}

		voucher.ExpiresAt = expiresAt
		voucher.UsedAt = usedAt
		voucher.UsedByID = usedByID

		// Only set the created by user if we have valid data
		if createdByUsername.Valid {
			createdByUser.Username = createdByUsername.String
			createdByUser.Email = createdByEmail.String
			createdByUser.Role = createdByRole.String
			voucher.CreatedBy = &createdByUser
		}

		// Only set the used by user if we have valid data
		if usedByID.Valid && usedByUsername.Valid {
			usedByUser.Username = usedByUsername.String
			usedByUser.Email = usedByEmail.String
			usedByUser.Role = usedByRole.String
			voucher.UsedBy = &usedByUser
		}

		vouchers = append(vouchers, voucher)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating claim vouchers: %w", err)
	}

	return vouchers, nil
}

// LogUsageAttempt logs an attempt to use a claim voucher
func (r *ClaimVoucherRepository) LogUsageAttempt(ctx context.Context, usage *models.ClaimVoucherUsage) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO claimVoucherUsage (
			voucherCode, attemptedById, attemptedAt,
			success, ipAddress, userAgent, errorMessage
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		usage.VoucherCode,
		usage.AttemptedByID,
		usage.AttemptedAt,
		usage.Success,
		usage.IPAddress,
		usage.UserAgent,
		usage.ErrorMessage,
	)

	if err != nil {
		return fmt.Errorf("failed to log claim voucher usage: %w", err)
	}

	return nil
}
