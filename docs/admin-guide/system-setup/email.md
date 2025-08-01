# Email Settings Administration

## Overview
KrakenHashes supports email functionality through multiple providers, currently SendGrid and Mailgun. This document covers the configuration and management of email settings through the admin interface.

## Provider Configuration

### SendGrid
To configure SendGrid as your email provider:

1. Select "SendGrid" from the Provider dropdown
2. Configure the following fields:
   - **API Key**: Your SendGrid API key with email sending permissions
   - **From Email**: The verified sender email address
   - **From Name**: Display name for the sender (defaults to "KrakenHashes")
   - **Monthly Limit**: (Optional) Set a monthly email sending limit
   
### Mailgun
To configure Mailgun as your email provider:

1. Select "Mailgun" from the Provider dropdown
2. Configure the following fields:
   - **API Key**: Your Mailgun API key
   - **Domain**: Your verified Mailgun domain
   - **From Email**: The verified sender email address
   - **From Name**: Display name for the sender (defaults to "KrakenHashes")
   - **Monthly Limit**: (Optional) Set a monthly email sending limit

### Monthly Limit
The monthly limit field is optional:
- Leave empty for unlimited emails
- Set a numeric value to limit monthly email sending
- Helps prevent unexpected costs from email service providers

## Testing and Saving Configuration

### Configuration Options
When saving email provider settings, you have three options:

1. **Cancel**: Discard changes and return to previous settings
2. **Save Configuration**: Save settings without testing
3. **Test and Save**: Test the configuration before saving

### Testing Process
When using "Test and Save":

1. Enter a test email address
2. System sends a test email to verify configuration
3. If successful:
   - Configuration is saved
   - Confirmation message displayed
4. If failed:
   - Error message displayed
   - Configuration not saved
   - Troubleshooting information provided

## Email Templates
Email templates are managed separately from provider configuration. For detailed information about email templates and available variables, see [Email Templates Documentation](./email_settings.md).

## Best Practices

1. **Provider Selection**
   - Choose based on your volume needs
   - Consider provider-specific features
   - Review pricing structures

2. **Configuration Testing**
   - Always test configuration before deployment
   - Verify emails are received
   - Check spam folder during testing

3. **Monthly Limits**
   - Set based on expected usage
   - Include buffer for unexpected spikes
   - Monitor usage through provider dashboards

4. **Security Considerations**
   - Store API keys securely
   - Use dedicated sending domains
   - Regularly rotate API keys
   - Monitor for unusual activity

## Troubleshooting

### Common Issues

1. **Emails Not Sending**
   - Verify API key permissions
   - Check monthly limit hasn't been reached
   - Confirm sender email is verified
   - Review provider dashboard for blocks

2. **Test Emails Failing**
   - Verify API key is correct
   - Check domain configuration (Mailgun)
   - Ensure test email address is valid
   - Review error messages in admin interface

3. **Template Issues**
   - Verify template syntax
   - Check variable names match expected format
   - Preview templates before saving
   - Test with various data scenarios 