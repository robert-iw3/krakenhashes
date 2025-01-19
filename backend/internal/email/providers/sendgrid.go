package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	emailtypes "github.com/ZerkerEOD/krakenhashes/backend/pkg/email"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridConfig represents SendGrid-specific configuration
type SendGridConfig struct {
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`
}

// sendgridProvider implements the Provider interface for SendGrid
type sendgridProvider struct {
	client    *sendgrid.Client
	fromEmail string
	fromName  string
}

// init registers the SendGrid provider
func init() {
	Register(emailtypes.ProviderSendGrid, func() Provider {
		return &sendgridProvider{}
	})
}

// Initialize sets up the SendGrid client
func (p *sendgridProvider) Initialize(cfg *emailtypes.Config) error {
	if cfg.APIKey == "" {
		debug.Error("sendgrid API key not provided")
		return ErrProviderNotConfigured
	}

	var sgConfig SendGridConfig
	if err := json.Unmarshal(cfg.AdditionalConfig, &sgConfig); err != nil {
		debug.Error("failed to parse sendgrid config: %v", err)
		return fmt.Errorf("invalid sendgrid configuration: %w", err)
	}

	if sgConfig.FromEmail == "" {
		debug.Error("sendgrid from_email not provided")
		return errors.New("sendgrid from_email is required")
	}

	if sgConfig.FromName == "" {
		debug.Error("sendgrid from_name not provided")
		return errors.New("sendgrid from_name is required")
	}

	p.client = sendgrid.NewSendClient(cfg.APIKey)
	p.fromEmail = sgConfig.FromEmail
	p.fromName = sgConfig.FromName

	debug.Info("initialized sendgrid client with from: %s <%s>", p.fromName, p.fromEmail)
	return nil
}

// ValidateConfig validates the SendGrid configuration
func (p *sendgridProvider) ValidateConfig(cfg *emailtypes.Config) error {
	if cfg.APIKey == "" {
		debug.Error("sendgrid API key not provided")
		return errors.New("sendgrid API key is required")
	}

	var sgConfig SendGridConfig
	if err := json.Unmarshal(cfg.AdditionalConfig, &sgConfig); err != nil {
		debug.Error("failed to parse sendgrid config: %v", err)
		return fmt.Errorf("invalid sendgrid configuration: %w", err)
	}

	if sgConfig.FromEmail == "" {
		debug.Error("sendgrid from_email not provided")
		return errors.New("sendgrid from_email is required")
	}

	if sgConfig.FromName == "" {
		debug.Error("sendgrid from_name not provided")
		return errors.New("sendgrid from_name is required")
	}

	debug.Info("validated sendgrid configuration for sender: %s <%s>", sgConfig.FromName, sgConfig.FromEmail)
	return nil
}

// Send sends an email using SendGrid
func (p *sendgridProvider) Send(ctx context.Context, data *emailtypes.EmailData) error {
	if p.client == nil {
		debug.Error("sendgrid client not initialized")
		return ErrProviderNotConfigured
	}

	if data.Template == nil {
		debug.Error("email template not provided")
		return ErrInvalidTemplate
	}

	var textContent, htmlContent string

	// Process template variables
	if len(data.Variables) > 0 {
		debug.Info("processing template variables for email")
		// Create template for both HTML and text content
		htmlTmpl, err := template.New("email_html").Parse(data.Template.HTMLContent)
		if err != nil {
			debug.Error("failed to parse HTML template: %v", err)
			return fmt.Errorf("failed to parse HTML template: %w", err)
		}

		textTmpl, err := template.New("email_text").Parse(data.Template.TextContent)
		if err != nil {
			debug.Error("failed to parse text template: %v", err)
			return fmt.Errorf("failed to parse text template: %w", err)
		}

		// Execute templates with variables
		if err := executeTemplate(htmlTmpl, data.Variables, &htmlContent); err != nil {
			debug.Error("failed to execute HTML template: %v", err)
			return fmt.Errorf("failed to execute HTML template: %w", err)
		}

		if err := executeTemplate(textTmpl, data.Variables, &textContent); err != nil {
			debug.Error("failed to execute text template: %v", err)
			return fmt.Errorf("failed to execute text template: %w", err)
		}
	} else {
		debug.Info("using template content without variables")
		htmlContent = data.Template.HTMLContent
		textContent = data.Template.TextContent
	}

	from := mail.NewEmail(p.fromName, p.fromEmail)
	debug.Info("sending email from %s <%s>", p.fromName, p.fromEmail)

	message := mail.NewV3Mail()
	message.SetFrom(from)
	message.Subject = data.Subject

	personalization := mail.NewPersonalization()
	for _, to := range data.To {
		personalization.AddTos(mail.NewEmail("", to))
	}
	message.AddPersonalizations(personalization)

	// Add content
	message.AddContent(mail.NewContent("text/plain", textContent))
	message.AddContent(mail.NewContent("text/html", htmlContent))

	// Send the email
	response, err := p.client.SendWithContext(ctx, message)
	if err != nil {
		debug.Error("failed to send email: %v", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	if response.StatusCode >= 400 {
		debug.Error("sendgrid API error: %d - %s", response.StatusCode, response.Body)
		return fmt.Errorf("sendgrid API error: %d - %s", response.StatusCode, response.Body)
	}

	debug.Info("successfully sent email with status code: %d", response.StatusCode)
	return nil
}

// TestConnection tests the connection to SendGrid
func (p *sendgridProvider) TestConnection(ctx context.Context, testEmail string) error {
	if p.client == nil {
		debug.Error("sendgrid client not initialized")
		return ErrProviderNotConfigured
	}

	from := mail.NewEmail(p.fromName, p.fromEmail)
	to := mail.NewEmail("", testEmail)
	subject := "KrakenHashes Email Test"
	plainTextContent := "This is a test email from KrakenHashes."
	htmlContent := "This is a test email from KrakenHashes."

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	response, err := p.client.Send(message)
	if err != nil {
		debug.Error("sendgrid test failed: %v", err)
		return fmt.Errorf("sendgrid test failed: %w", err)
	}

	if response.StatusCode >= 400 {
		debug.Error("sendgrid test failed with status code: %d", response.StatusCode)
		return fmt.Errorf("sendgrid test failed with status code: %d", response.StatusCode)
	}

	debug.Info("successfully sent test email to: %s", testEmail)
	return nil
}
