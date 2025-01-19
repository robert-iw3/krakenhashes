package providers

import (
	"context"
	"errors"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	emailtypes "github.com/ZerkerEOD/krakenhashes/backend/pkg/email"
)

var (
	// ErrProviderNotConfigured is returned when the email provider is not properly configured
	ErrProviderNotConfigured = errors.New("email provider not configured")
	// ErrInvalidTemplate is returned when the template is invalid or not found
	ErrInvalidTemplate = errors.New("invalid email template")
	// ErrSendFailed is returned when the email fails to send
	ErrSendFailed = errors.New("failed to send email")
	// ErrMonthlyLimitExceeded is returned when the monthly email limit is exceeded
	ErrMonthlyLimitExceeded = errors.New("monthly email limit exceeded")
)

// Provider defines the interface for email providers
type Provider interface {
	// Initialize sets up the provider with the given configuration
	Initialize(cfg *emailtypes.Config) error

	// Send sends an email using the provider
	Send(ctx context.Context, data *emailtypes.EmailData) error

	// ValidateConfig validates the provider configuration
	ValidateConfig(cfg *emailtypes.Config) error

	// TestConnection tests the connection to the email provider
	TestConnection(ctx context.Context, testEmail string) error
}

// ProviderFactory is a function that creates a new Provider instance
type ProviderFactory func() Provider

// providers is a map of provider types to their factory functions
var providers = make(map[emailtypes.ProviderType]ProviderFactory)

// Register registers a new provider factory for the given provider type
func Register(providerType emailtypes.ProviderType, factory ProviderFactory) {
	debug.Info("registering email provider: %s", providerType)
	providers[providerType] = factory
}

// New creates a new Provider instance for the given provider type
func New(providerType emailtypes.ProviderType) (Provider, error) {
	factory, exists := providers[providerType]
	if !exists {
		debug.Error("unsupported email provider type: %s", providerType)
		return nil, errors.New("unsupported email provider type")
	}
	debug.Info("creating new instance of email provider: %s", providerType)
	return factory(), nil
}
