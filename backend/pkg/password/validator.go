package password

import (
	"fmt"
	"unicode"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
)

// ValidationError represents a password validation error
type ValidationError struct {
	Rule    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Rule, e.Message)
}

// Validate checks if a password meets the requirements specified in AuthSettings
func Validate(password string, settings *models.AuthSettings) error {
	// Check minimum length
	if len(password) < settings.MinPasswordLength {
		return &ValidationError{
			Rule:    "Length",
			Message: fmt.Sprintf("Password must be at least %d characters long", settings.MinPasswordLength),
		}
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// Check required character types
	if settings.RequireUppercase && !hasUpper {
		return &ValidationError{
			Rule:    "Uppercase",
			Message: "Password must contain at least one uppercase letter",
		}
	}

	if settings.RequireLowercase && !hasLower {
		return &ValidationError{
			Rule:    "Lowercase",
			Message: "Password must contain at least one lowercase letter",
		}
	}

	if settings.RequireNumbers && !hasNumber {
		return &ValidationError{
			Rule:    "Numbers",
			Message: "Password must contain at least one number",
		}
	}

	if settings.RequireSpecialChars && !hasSpecial {
		return &ValidationError{
			Rule:    "Special",
			Message: "Password must contain at least one special character",
		}
	}

	return nil
}

// GetComplexityDescription returns a human-readable description of password requirements
func GetComplexityDescription(settings *models.AuthSettings) string {
	desc := fmt.Sprintf("Password must be at least %d characters", settings.MinPasswordLength)

	var requirements []string
	if settings.RequireUppercase {
		requirements = append(requirements, "an uppercase letter")
	}
	if settings.RequireLowercase {
		requirements = append(requirements, "a lowercase letter")
	}
	if settings.RequireNumbers {
		requirements = append(requirements, "a number")
	}
	if settings.RequireSpecialChars {
		requirements = append(requirements, "a special character")
	}

	if len(requirements) > 0 {
		desc += " and contain at least"
		for i, req := range requirements {
			if i == len(requirements)-1 && len(requirements) > 1 {
				desc += " and"
			}
			desc += " " + req
			if i < len(requirements)-2 {
				desc += ","
			}
		}
	}

	return desc + "."
}
