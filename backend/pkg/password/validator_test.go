package password

import (
	"strings"
	"testing"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	// Default settings for most tests
	defaultSettings := &models.AuthSettings{
		MinPasswordLength:   8,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireNumbers:      true,
		RequireSpecialChars: true,
	}

	tests := []struct {
		name        string
		password    string
		settings    *models.AuthSettings
		expectError bool
		errorMsg    string
	}{
		// Valid passwords
		{
			name:        "valid password with all requirements",
			password:    "SecurePass123!",
			settings:    defaultSettings,
			expectError: false,
		},
		{
			name:        "valid password with different special chars",
			password:    "TestPass456@#$",
			settings:    defaultSettings,
			expectError: false,
		},
		{
			name:        "valid password exactly 8 chars",
			password:    "Pass123!",
			settings:    defaultSettings,
			expectError: false,
		},
		{
			name:        "valid very long password",
			password:    "ThisIsAVeryLongSecurePassword123!@#$%^&*()",
			settings:    defaultSettings,
			expectError: false,
		},

		// Invalid passwords - too short
		{
			name:        "too short password",
			password:    "Pass1!",
			settings:    defaultSettings,
			expectError: true,
			errorMsg:    "Password must be at least 8 characters long",
		},
		{
			name:        "empty password",
			password:    "",
			settings:    defaultSettings,
			expectError: true,
			errorMsg:    "Password must be at least 8 characters long",
		},

		// Invalid passwords - missing character types
		{
			name:        "missing uppercase",
			password:    "password123!",
			settings:    defaultSettings,
			expectError: true,
			errorMsg:    "Password must contain at least one uppercase letter",
		},
		{
			name:        "missing lowercase",
			password:    "PASSWORD123!",
			settings:    defaultSettings,
			expectError: true,
			errorMsg:    "Password must contain at least one lowercase letter",
		},
		{
			name:        "missing number",
			password:    "PasswordTest!",
			settings:    defaultSettings,
			expectError: true,
			errorMsg:    "Password must contain at least one number",
		},
		{
			name:        "missing special character",
			password:    "Password123",
			settings:    defaultSettings,
			expectError: true,
			errorMsg:    "Password must contain at least one special character",
		},

		// Test with different settings
		{
			name:     "no uppercase required",
			password: "password123!",
			settings: &models.AuthSettings{
				MinPasswordLength:   8,
				RequireUppercase:    false,
				RequireLowercase:    true,
				RequireNumbers:      true,
				RequireSpecialChars: true,
			},
			expectError: false,
		},
		{
			name:     "no special chars required",
			password: "Password123",
			settings: &models.AuthSettings{
				MinPasswordLength:   8,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireNumbers:      true,
				RequireSpecialChars: false,
			},
			expectError: false,
		},
		{
			name:     "minimum requirements only",
			password: "password",
			settings: &models.AuthSettings{
				MinPasswordLength:   8,
				RequireUppercase:    false,
				RequireLowercase:    false,
				RequireNumbers:      false,
				RequireSpecialChars: false,
			},
			expectError: false,
		},

		// Edge cases
		{
			name:        "unicode special characters",
			password:    "Pass123€",
			settings:    defaultSettings,
			expectError: false,
		},
		{
			name:        "spaces should work",
			password:    "Pass 123!",
			settings:    defaultSettings,
			expectError: false,
		},
		{
			name:     "custom minimum length",
			password: "Pass123!Test",
			settings: &models.AuthSettings{
				MinPasswordLength:   15,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireNumbers:      true,
				RequireSpecialChars: true,
			},
			expectError: true,
			errorMsg:    "Password must be at least 15 characters long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.password, tt.settings)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				// Check that it's a ValidationError
				var valErr *ValidationError
				assert.ErrorAs(t, err, &valErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetComplexityDescription(t *testing.T) {
	tests := []struct {
		name     string
		settings *models.AuthSettings
		expected string
	}{
		{
			name: "all requirements",
			settings: &models.AuthSettings{
				MinPasswordLength:   8,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireNumbers:      true,
				RequireSpecialChars: true,
			},
			expected: "Password must be at least 8 characters and contain at least an uppercase letter, a lowercase letter, a number and a special character.",
		},
		{
			name: "no requirements",
			settings: &models.AuthSettings{
				MinPasswordLength:   6,
				RequireUppercase:    false,
				RequireLowercase:    false,
				RequireNumbers:      false,
				RequireSpecialChars: false,
			},
			expected: "Password must be at least 6 characters.",
		},
		{
			name: "some requirements",
			settings: &models.AuthSettings{
				MinPasswordLength:   10,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireNumbers:      false,
				RequireSpecialChars: false,
			},
			expected: "Password must be at least 10 characters and contain at least an uppercase letter and a lowercase letter.",
		},
		{
			name: "single requirement",
			settings: &models.AuthSettings{
				MinPasswordLength:   12,
				RequireUppercase:    false,
				RequireLowercase:    false,
				RequireNumbers:      true,
				RequireSpecialChars: false,
			},
			expected: "Password must be at least 12 characters and contain at least a number.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := GetComplexityDescription(tt.settings)
			assert.Equal(t, tt.expected, description)
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Rule:    "Length",
		Message: "Password must be at least 8 characters long",
	}

	assert.Equal(t, "Length: Password must be at least 8 characters long", err.Error())
}

func TestPasswordComplexityCombinations(t *testing.T) {
	// Test various combinations of settings to ensure all paths are covered
	passwords := map[string]struct {
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	}{
		"UPPERCASE":        {true, false, false, false},
		"lowercase":        {false, true, false, false},
		"12345678":         {false, false, true, false},
		"!@#$%^&*":         {false, false, false, true},
		"UPPER123":         {true, false, true, false},
		"lower123":         {false, true, true, false},
		"UPPER!@#":         {true, false, false, true},
		"lower!@#":         {false, true, false, true},
		"123!@#$%":         {false, false, true, true},
		"UpperLower":       {true, true, false, false},
		"UpperLower123":    {true, true, true, false},
		"UpperLower!":      {true, true, false, true},
		"UPPER123!":        {true, false, true, true},
		"lower123!":        {false, true, true, true},
		"UpperLower123!":   {true, true, true, true},
	}

	for password, expected := range passwords {
		t.Run(password, func(t *testing.T) {
			// Test with each requirement individually
			settings := &models.AuthSettings{
				MinPasswordLength:   len(password),
				RequireUppercase:    true,
				RequireLowercase:    false,
				RequireNumbers:      false,
				RequireSpecialChars: false,
			}

			err := Validate(password, settings)
			if expected.hasUpper {
				assert.NoError(t, err, "Should pass uppercase requirement")
			} else {
				assert.Error(t, err, "Should fail uppercase requirement")
			}

			// Test lowercase requirement
			settings.RequireUppercase = false
			settings.RequireLowercase = true
			err = Validate(password, settings)
			if expected.hasLower {
				assert.NoError(t, err, "Should pass lowercase requirement")
			} else {
				assert.Error(t, err, "Should fail lowercase requirement")
			}

			// Test number requirement
			settings.RequireLowercase = false
			settings.RequireNumbers = true
			err = Validate(password, settings)
			if expected.hasNumber {
				assert.NoError(t, err, "Should pass number requirement")
			} else {
				assert.Error(t, err, "Should fail number requirement")
			}

			// Test special char requirement
			settings.RequireNumbers = false
			settings.RequireSpecialChars = true
			err = Validate(password, settings)
			if expected.hasSpecial {
				assert.NoError(t, err, "Should pass special char requirement")
			} else {
				assert.Error(t, err, "Should fail special char requirement")
			}
		})
	}
}

func TestUnicodeCharacterClassification(t *testing.T) {
	// Test that various unicode characters are properly classified
	tests := []struct {
		password string
		settings *models.AuthSettings
		shouldPass bool
		description string
	}{
		{
			password: "Pass123€", // Euro symbol
			settings: &models.AuthSettings{
				MinPasswordLength:   8,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireNumbers:      true,
				RequireSpecialChars: true,
			},
			shouldPass: true,
			description: "Euro symbol should count as special character",
		},
		{
			password: "Pass123中文", // Chinese characters
			settings: &models.AuthSettings{
				MinPasswordLength:   8,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireNumbers:      true,
				RequireSpecialChars: false,
			},
			shouldPass: true,
			description: "Chinese characters should be allowed",
		},
		{
			password: "Pass¹²³!", // Superscript numbers
			settings: &models.AuthSettings{
				MinPasswordLength:   8,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireNumbers:      true,
				RequireSpecialChars: true,
			},
			shouldPass: true,
			description: "Superscript numbers should count as numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := Validate(tt.password, tt.settings)
			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidate(b *testing.B) {
	password := "TestPassword123!"
	settings := &models.AuthSettings{
		MinPasswordLength:   8,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireNumbers:      true,
		RequireSpecialChars: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validate(password, settings)
	}
}

func BenchmarkValidateLongPassword(b *testing.B) {
	password := strings.Repeat("TestPassword123!", 10)
	settings := &models.AuthSettings{
		MinPasswordLength:   8,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireNumbers:      true,
		RequireSpecialChars: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validate(password, settings)
	}
}