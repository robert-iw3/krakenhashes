package repository

import (
	"context"
	"time"

	"github.com/yourusername/hashdom/internal/models"
	"github.com/yourusername/hashdom/pkg/debug"
	"gorm.io/gorm"
)

// ClaimVoucherRepository handles database operations for claim vouchers
type ClaimVoucherRepository struct {
	db *gorm.DB
}

// NewClaimVoucherRepository creates a new claim voucher repository
func NewClaimVoucherRepository(db *gorm.DB) *ClaimVoucherRepository {
	return &ClaimVoucherRepository{db: db}
}

// Create creates a new claim voucher
func (r *ClaimVoucherRepository) Create(ctx context.Context, voucher *models.ClaimVoucher) error {
	if err := r.db.WithContext(ctx).Create(voucher).Error; err != nil {
		debug.Error("failed to create claim voucher: %v", err)
		return err
	}
	return nil
}

// GetByCode retrieves a claim voucher by code
func (r *ClaimVoucherRepository) GetByCode(ctx context.Context, code string) (*models.ClaimVoucher, error) {
	var voucher models.ClaimVoucher
	if err := r.db.WithContext(ctx).
		Preload("CreatedBy").
		Preload("UsedBy").
		Where("code = ?", code).
		First(&voucher).Error; err != nil {
		debug.Error("failed to get claim voucher by code: %v", err)
		return nil, err
	}
	return &voucher, nil
}

// ListActive retrieves all active claim vouchers
func (r *ClaimVoucherRepository) ListActive(ctx context.Context) ([]models.ClaimVoucher, error) {
	var vouchers []models.ClaimVoucher
	if err := r.db.WithContext(ctx).
		Preload("CreatedBy").
		Where("is_active = ?", true).
		Find(&vouchers).Error; err != nil {
		debug.Error("failed to list active claim vouchers: %v", err)
		return nil, err
	}
	return vouchers, nil
}

// Deactivate deactivates a claim voucher
func (r *ClaimVoucherRepository) Deactivate(ctx context.Context, code string) error {
	if err := r.db.WithContext(ctx).
		Model(&models.ClaimVoucher{}).
		Where("code = ?", code).
		Update("is_active", false).Error; err != nil {
		debug.Error("failed to deactivate claim voucher: %v", err)
		return err
	}
	return nil
}

// Use marks a claim voucher as used
func (r *ClaimVoucherRepository) Use(ctx context.Context, code string, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var voucher models.ClaimVoucher
		if err := tx.
			Where("code = ?", code).
			First(&voucher).Error; err != nil {
			debug.Error("failed to get claim voucher for use: %v", err)
			return err
		}

		if !voucher.IsValid() {
			debug.Error("attempt to use invalid claim voucher: %s", code)
			return ErrInvalidVoucher
		}

		voucher.Use(userID)
		if err := tx.Save(&voucher).Error; err != nil {
			debug.Error("failed to save used claim voucher: %v", err)
			return err
		}

		return nil
	})
}

// LogUsageAttempt logs an attempt to use a claim voucher
func (r *ClaimVoucherRepository) LogUsageAttempt(ctx context.Context, attempt *models.ClaimVoucherUsage) error {
	if err := r.db.WithContext(ctx).Create(attempt).Error; err != nil {
		debug.Error("failed to log claim voucher usage attempt: %v", err)
		return err
	}
	return nil
}

// GetUsageAttempts retrieves usage attempts for a voucher
func (r *ClaimVoucherRepository) GetUsageAttempts(ctx context.Context, code string) ([]models.ClaimVoucherUsage, error) {
	var attempts []models.ClaimVoucherUsage
	if err := r.db.WithContext(ctx).
		Preload("AttemptedBy").
		Where("voucher_code = ?", code).
		Order("attempted_at DESC").
		Find(&attempts).Error; err != nil {
		debug.Error("failed to get claim voucher usage attempts: %v", err)
		return nil, err
	}
	return attempts, nil
}

// CleanupExpired removes expired claim vouchers
func (r *ClaimVoucherRepository) CleanupExpired(ctx context.Context) error {
	if err := r.db.WithContext(ctx).
		Where("expires_at < ? AND is_active = ?", time.Now(), true).
		Delete(&models.ClaimVoucher{}).Error; err != nil {
		debug.Error("failed to cleanup expired claim vouchers: %v", err)
		return err
	}
	return nil
}

// GetUserVouchers retrieves all claim vouchers created by a user
func (r *ClaimVoucherRepository) GetUserVouchers(ctx context.Context, userID uint) ([]models.ClaimVoucher, error) {
	var vouchers []models.ClaimVoucher
	if err := r.db.WithContext(ctx).
		Preload("CreatedBy").
		Where("created_by_id = ?", userID).
		Find(&vouchers).Error; err != nil {
		debug.Error("failed to get user claim vouchers: %v", err)
		return nil, err
	}
	return vouchers, nil
}
