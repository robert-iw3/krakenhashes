package models

import (
	"time"
)

// ClaimVoucher represents a claim code that can be used to register agents
type ClaimVoucher struct {
	Code         string    `json:"code" gorm:"primaryKey"`
	CreatedByID  uint      `json:"createdById" gorm:"not null"`
	CreatedBy    User      `json:"createdBy" gorm:"foreignKey:CreatedByID"`
	CreatedAt    time.Time `json:"createdAt"`
	IsContinuous bool      `json:"isContinuous"`
	IsActive     bool      `json:"isActive" gorm:"default:true"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty"` // Optional expiration time
	UsedAt       time.Time `json:"usedAt,omitempty"`    // When the code was used (for single-use codes)
	UsedByID     *uint     `json:"usedById,omitempty"`  // Who used the code (for auditing)
	UsedBy       *User     `json:"usedBy,omitempty" gorm:"foreignKey:UsedByID"`
}

// ClaimVoucherUsage tracks usage attempts of claim vouchers
type ClaimVoucherUsage struct {
	ID            uint         `json:"id" gorm:"primaryKey"`
	VoucherCode   string       `json:"voucherCode"`
	Voucher       ClaimVoucher `json:"-" gorm:"foreignKey:VoucherCode"`
	AttemptedByID uint         `json:"attemptedById"`
	AttemptedBy   User         `json:"attemptedBy" gorm:"foreignKey:AttemptedByID"`
	AttemptedAt   time.Time    `json:"attemptedAt"`
	Success       bool         `json:"success"`
	IPAddress     string       `json:"ipAddress"`
	UserAgent     string       `json:"userAgent"`
	ErrorMessage  string       `json:"errorMessage,omitempty"`
}

// TableName specifies the table name for ClaimVoucherUsage
func (ClaimVoucherUsage) TableName() string {
	return "claim_voucher_usage"
}

// IsValid checks if the claim voucher can be used
func (v *ClaimVoucher) IsValid() bool {
	if !v.IsActive {
		return false
	}

	// Check expiration if set
	if !v.ExpiresAt.IsZero() && time.Now().After(v.ExpiresAt) {
		return false
	}

	// For single-use vouchers, check if already used
	if !v.IsContinuous && !v.UsedAt.IsZero() {
		return false
	}

	return true
}

// Use marks the voucher as used
func (v *ClaimVoucher) Use(userID uint) {
	if !v.IsContinuous {
		v.IsActive = false
		v.UsedAt = time.Now()
		v.UsedByID = &userID
	}
}
