package models

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NullUUID represents a UUID that may be null
type NullUUID struct {
	UUID  uuid.UUID
	Valid bool
}

// Scan implements the Scanner interface
func (nu *NullUUID) Scan(value interface{}) error {
	if value == nil {
		nu.UUID, nu.Valid = uuid.Nil, false
		return nil
	}

	switch v := value.(type) {
	case []byte:
		parsed, err := uuid.ParseBytes(v)
		if err != nil {
			return err
		}
		nu.UUID, nu.Valid = parsed, true
		return nil
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return err
		}
		nu.UUID, nu.Valid = parsed, true
		return nil
	default:
		return fmt.Errorf("unsupported type for UUID: %T", value)
	}
}

// Value implements the driver.Valuer interface
func (nu NullUUID) Value() (driver.Value, error) {
	if !nu.Valid {
		return nil, nil
	}
	return nu.UUID.String(), nil
}

// ClaimVoucher represents a voucher that can be claimed by an agent
type ClaimVoucher struct {
	Code         string       `json:"code"`
	IsActive     bool         `json:"isActive"`
	IsContinuous bool         `json:"isContinuous"`
	ExpiresAt    sql.NullTime `json:"expiresAt,omitempty"`
	CreatedByID  uuid.UUID    `json:"createdById"`
	CreatedBy    *User        `json:"createdBy,omitempty"`
	UsedByID     NullUUID     `json:"usedById,omitempty"`
	UsedBy       *User        `json:"usedBy,omitempty"`
	UsedAt       sql.NullTime `json:"usedAt,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

// ClaimVoucherUsage tracks usage attempts of claim vouchers
type ClaimVoucherUsage struct {
	ID            uint           `json:"id"`
	VoucherCode   string         `json:"voucherCode"`
	AttemptedByID uuid.UUID      `json:"attemptedById"`
	AttemptedBy   *User          `json:"attemptedBy,omitempty"`
	AttemptedAt   time.Time      `json:"attemptedAt"`
	Success       bool           `json:"success"`
	IPAddress     string         `json:"ipAddress"`
	UserAgent     string         `json:"userAgent"`
	ErrorMessage  sql.NullString `json:"errorMessage,omitempty"`
}

// IsValid checks if the voucher is valid for use
func (v *ClaimVoucher) IsValid() bool {
	if !v.IsActive {
		return false
	}

	if v.ExpiresAt.Valid && time.Now().After(v.ExpiresAt.Time) {
		return false
	}

	if !v.IsContinuous && v.UsedByID.Valid {
		return false
	}

	return true
}

// Use marks the voucher as used
func (v *ClaimVoucher) Use(userID uuid.UUID) {
	if !v.IsContinuous {
		v.IsActive = false
		now := time.Now()
		v.UsedAt = sql.NullTime{Time: now, Valid: true}
		v.UsedByID = NullUUID{UUID: userID, Valid: true}
	}
}
