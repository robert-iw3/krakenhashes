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

// ClaimVoucher represents a claim voucher in the system
type ClaimVoucher struct {
	Code          string        `json:"code"`
	IsActive      bool          `json:"is_active"`
	IsContinuous  bool          `json:"is_continuous"`
	CreatedByID   uuid.UUID     `json:"created_by_id"`
	CreatedBy     *User         `json:"created_by,omitempty"`
	UsedByAgentID sql.NullInt64 `json:"used_by_agent_id,omitempty"`
	UsedByAgent   *Agent        `json:"used_by_agent,omitempty"`
	UsedAt        sql.NullTime  `json:"used_at,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
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
	// Only check if the voucher is active
	if !v.IsActive {
		return false
	}

	// For single-use codes, check if they've been used
	if !v.IsContinuous && v.UsedByAgentID.Valid {
		return false
	}

	return true
}
