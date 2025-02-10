# Email Template System

## Overview
The email template system allows for customizable email notifications using predefined templates. Each template type has specific variables that can be used to insert dynamic content.

## Template Types and Variables

### Security Event Template
Used for security-related notifications.

**Variables:**
- `{{ .EventType }}` - Type of security event (e.g., "Failed Login", "Password Changed")
- `{{ .Timestamp }}` - When the event occurred
- `{{ .Details }}` - Detailed description of the event
- `{{ .IPAddress }}` - IP address associated with the event

### Job Completion Template
Used for notifying users about completed hash cracking jobs.

**Variables:**
- `{{ .JobName }}` - Name of the completed job
- `{{ .Duration }}` - How long the job took to complete
- `{{ .HashesProcessed }}` - Total number of hashes processed
- `{{ .CrackedCount }}` - Number of hashes successfully cracked
- `{{ .SuccessRate }}` - Percentage of hashes cracked

### Admin Error Template
Used for system error notifications to administrators.

**Variables:**
- `{{ .ErrorType }}` - Type of error encountered
- `{{ .Component }}` - System component where the error occurred
- `{{ .Timestamp }}` - When the error occurred
- `{{ .ErrorMessage }}` - Detailed error message
- `{{ .StackTrace }}` - Stack trace for debugging

### MFA Code Template
Used for sending multi-factor authentication codes.

**Variables:**
- `{{ .Code }}` - The authentication code
- `{{ .ExpiryMinutes }}` - Minutes until the code expires

## Usage Guidelines
1. All variables must be enclosed in double curly braces with a dot prefix
2. Variable names are case-sensitive
3. Templates support both HTML and plain text versions
4. Use the preview feature to test variable substitution
5. Always test templates with the "Send Test Email" feature before saving

## Example Usage
```html
<h2>Job Completed: {{ .JobName }}</h2>
<p>Duration: {{ .Duration }}</p>
<p>Success Rate: {{ .SuccessRate }}%</p>
```

## Best Practices
1. Include both HTML and plain text versions
2. Keep subject lines concise and descriptive
3. Test templates with various data scenarios
4. Use consistent styling across templates
5. Include error handling for missing variables 