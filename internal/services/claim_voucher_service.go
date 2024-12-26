package services

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/models"
	"github.com/ZerkerEOD/hashdom-backend/internal/repository"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/google/uuid"
)

// ClaimVoucherService handles business logic for claim vouchers
type ClaimVoucherService struct {
	repo *repository.ClaimVoucherRepository
}

// NewClaimVoucherService creates a new claim voucher service
func NewClaimVoucherService(repo *repository.ClaimVoucherRepository) *ClaimVoucherService {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
	return &ClaimVoucherService{repo: repo}
}

// generateClaimCode generates a random 20-character alphanumeric claim code with hyphens for readability
func generateClaimCode() string {
	// Define the character set (0-9 and A-Z)
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := make([]byte, 20)
	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}

	// Format with hyphens for display (XXXXX-XXXXX-XXXXX-XXXXX)
	return fmt.Sprintf("%s-%s-%s-%s",
		string(code[0:5]),
		string(code[5:10]),
		string(code[10:15]),
		string(code[15:20]))
}

// normalizeClaimCode removes hyphens from a claim code and converts to uppercase
func normalizeClaimCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(code, "-", ""))
}

// formatClaimCode adds hyphens to a claim code for display
func formatClaimCode(code string) string {
	code = normalizeClaimCode(code)
	if len(code) != 20 {
		return code
	}
	return fmt.Sprintf("%s-%s-%s-%s",
		code[0:5],
		code[5:10],
		code[10:15],
		code[15:20])
}

// CreateTempVoucher creates a temporary claim voucher
func (s *ClaimVoucherService) CreateTempVoucher(ctx context.Context, userID string, expiresIn time.Duration, isContinuous bool) (*models.ClaimVoucher, error) {
	// Parse user ID to UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		debug.Error("failed to parse user ID: %v", err)
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Generate expiration time
	expiresAt := time.Now().Add(expiresIn)

	// Create voucher with normalized code for storage
	code := generateClaimCode()
	voucher := &models.ClaimVoucher{
		Code:         normalizeClaimCode(code),
		IsActive:     true,
		IsContinuous: isContinuous,
		ExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
		CreatedByID:  userUUID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save voucher
	if err := s.repo.Create(ctx, voucher); err != nil {
		debug.Error("failed to create voucher: %v", err)
		return nil, fmt.Errorf("failed to create voucher: %w", err)
	}

	// Format code for display before returning
	voucher.Code = formatClaimCode(voucher.Code)
	return voucher, nil
}

// ListVouchers retrieves all active vouchers
func (s *ClaimVoucherService) ListVouchers(ctx context.Context) ([]models.ClaimVoucher, error) {
	vouchers, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	// Format codes for display
	for i := range vouchers {
		vouchers[i].Code = formatClaimCode(vouchers[i].Code)
	}
	return vouchers, nil
}

// DisableVoucher disables a claim voucher
func (s *ClaimVoucherService) DisableVoucher(ctx context.Context, code string) error {
	// Normalize code before disabling
	normalizedCode := normalizeClaimCode(code)
	return s.repo.Deactivate(ctx, normalizedCode)
}

// GetVoucher retrieves a voucher by code
func (s *ClaimVoucherService) GetVoucher(ctx context.Context, code string) (*models.ClaimVoucher, error) {
	// Normalize code before lookup
	normalizedCode := normalizeClaimCode(code)
	voucher, err := s.repo.GetByCode(ctx, normalizedCode)
	if err != nil {
		return nil, err
	}

	// Format code for display before returning
	voucher.Code = formatClaimCode(voucher.Code)
	return voucher, nil
}

// UseVoucher marks a voucher as used by a specific user
func (s *ClaimVoucherService) UseVoucher(ctx context.Context, code string, userID string) error {
	// Parse user ID to UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		debug.Error("failed to parse user ID: %v", err)
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Normalize code before using
	normalizedCode := normalizeClaimCode(code)
	return s.repo.Use(ctx, normalizedCode, userUUID)
}
