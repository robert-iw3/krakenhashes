package testutil

import (
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/google/uuid"
)

// Test user fixtures
var (
	TestAdminUser = models.User{
		ID:       uuid.New(),
		Username: "testadmin",
		Email:    "admin@test.com",
		Role:     "admin",
	}

	TestRegularUser = models.User{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    "user@test.com",
		Role:     "user",
	}

	TestAgentUser = models.User{
		ID:       uuid.New(),
		Username: "testagent",
		Email:    "agent@test.com",
		Role:     "agent",
	}
)

// Default test passwords
const (
	DefaultTestPassword = "TestPassword123!"
	WeakTestPassword    = "weak"
	StrongTestPassword  = "SuperSecure123!@#$%^&*()"
)

// MFA test codes
const (
	ValidTOTPSecret = "JBSWY3DPEHPK3PXP" // Base32 encoded test secret
	ValidEmailCode  = "123456"
	ValidBackupCode = "ABCD1234"
	InvalidMFACode  = "000000"
)

// Test JWT secret
const TestJWTSecret = "test-jwt-secret-for-testing-only"

// ValidUser returns a valid user model for testing
func ValidUser() *models.User {
	return &models.User{
		Username: "validuser",
		Email:    "valid@example.com",
		Role:     "user",
	}
}

// ValidLoginRequest returns a valid login request
func ValidLoginRequest() map[string]string {
	return map[string]string{
		"username": "testuser",
		"password": DefaultTestPassword,
	}
}

// ValidMFAVerifyRequest returns a valid MFA verification request
func ValidMFAVerifyRequest(method, code, sessionToken string) map[string]string {
	return map[string]string{
		"method":       method,
		"code":         code,
		"sessionToken": sessionToken,
	}
}
