package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"strings"

	"github.com/yourusername/hashdom/internal/models"
	"github.com/yourusername/hashdom/internal/repository"
	"github.com/yourusername/hashdom/pkg/debug"
)

// ClaimVoucherService handles business logic for claim vouchers
type ClaimVoucherService struct {
	repo *repository.ClaimVoucherRepository
}

// NewClaimVoucherService creates a new claim voucher service
func NewClaimVoucherService(repo *repository.ClaimVoucherRepository) *ClaimVoucherService {
	return &ClaimVoucherService{repo: repo}
}

// GenerateVoucher generates a new claim voucher
func (s *ClaimVoucherService) GenerateVoucher(ctx context.Context, userID uint, isContinuous bool) (*models.ClaimVoucher, error) {
	// Generate random code
	code, err := generateCode()
	if err != nil {
		debug.Error("failed to generate code: %v", err)
		return nil, err
	}

	// Store normalized code in database
	voucher := &models.ClaimVoucher{
		Code:         normalizeCode(code), // Store without hyphens
		CreatedByID:  userID,
		IsContinuous: isContinuous,
		IsActive:     true,
	}

	if err := s.repo.Create(ctx, voucher); err != nil {
		debug.Error("failed to create voucher: %v", err)
		return nil, err
	}

	// Return voucher with formatted code for display
	voucher.Code = code // Return with hyphens for display
	return voucher, nil
}

// ListActiveVouchers lists all active vouchers
func (s *ClaimVoucherService) ListActiveVouchers(ctx context.Context) ([]models.ClaimVoucher, error) {
	return s.repo.ListActive(ctx)
}

// DeactivateVoucher deactivates a voucher
func (s *ClaimVoucherService) DeactivateVoucher(ctx context.Context, code string) error {
	// Normalize code by removing hyphens
	normalizedCode := normalizeCode(code)
	return s.repo.Deactivate(ctx, normalizedCode)
}

// GetVoucher retrieves a voucher by code
func (s *ClaimVoucherService) GetVoucher(ctx context.Context, code string) (*models.ClaimVoucher, error) {
	// Normalize code by removing hyphens
	normalizedCode := normalizeCode(code)
	return s.repo.GetByCode(ctx, normalizedCode)
}

// generateCode generates a random claim code
func generateCode() (string, error) {
	// Generate 15 random bytes (will encode to ~24 characters in base32)
	bytes := make([]byte, 15)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to base32 and remove padding
	code := strings.TrimRight(base32.StdEncoding.EncodeToString(bytes), "=")

	// Insert hyphens every 5 characters for readability
	var result strings.Builder
	for i := 0; i < len(code); i += 5 {
		if i > 0 {
			result.WriteString("-")
		}
		end := i + 5
		if end > len(code) {
			end = len(code)
		}
		result.WriteString(code[i:end])
	}

	return result.String(), nil
}

// normalizeCode removes any hyphens from the code
func normalizeCode(code string) string {
	return strings.ReplaceAll(code, "-", "")
}
