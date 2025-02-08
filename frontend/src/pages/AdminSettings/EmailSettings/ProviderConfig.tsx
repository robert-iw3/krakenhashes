import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Button,
  Grid,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  DialogContentText,
} from '@mui/material';
import { LoadingButton } from '@mui/lab';
import { SelectChangeEvent } from '@mui/material/Select';
import { getEmailConfig, updateEmailConfig, testEmailConfig } from '../../../services/api';

interface ProviderConfigProps {
  onNotification: (message: string, severity: 'success' | 'error') => void;
}

interface EmailProviderConfig {
  id?: number;
  provider: 'sendgrid' | 'mailgun';
  apiKey: string;
  fromEmail?: string;
  fromName?: string;
  domain?: string;
  monthlyLimit?: number;
}

const STORAGE_KEY = 'providerConfigState';

const defaultConfig: EmailProviderConfig = {
  provider: 'sendgrid',
  apiKey: '',
  fromEmail: '',
  fromName: '',
  monthlyLimit: undefined,
};

export const ProviderConfig: React.FC<ProviderConfigProps> = ({ onNotification }) => {
  const [loading, setLoading] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [hasLoadedInitialConfig, setHasLoadedInitialConfig] = useState(false);
  const [config, setConfig] = useState<EmailProviderConfig>(defaultConfig);
  const [testEmailOpen, setTestEmailOpen] = useState(false);
  const [testEmail, setTestEmail] = useState('');
  const [saveWithTestOpen, setSaveWithTestOpen] = useState(false);

  const loadConfig = useCallback(async () => {
    try {
      console.debug('[ProviderConfig] Loading configuration...');
      setLoading(true);

      // First check for unsaved changes
      const savedState = localStorage.getItem(STORAGE_KEY);
      if (savedState) {
        try {
          console.debug('[ProviderConfig] Found unsaved changes');
          const parsedState = JSON.parse(savedState);
          setConfig(parsedState);
          setIsEditing(true);
          setHasLoadedInitialConfig(true);
          return; // Don't load from API if we have unsaved changes
        } catch (error) {
          console.error('[ProviderConfig] Failed to parse saved state:', error);
          localStorage.removeItem(STORAGE_KEY);
        }
      }

      // Try to load from API
      const response = await getEmailConfig();
      console.debug('[ProviderConfig] Loaded configuration:', response.data);
      setConfig(response.data);
    } catch (error) {
      console.error('[ProviderConfig] Failed to load configuration:', error);
      // Only show notification if it's not a 404 (expected for new setup)
      if ((error as any).response?.status !== 404) {
        onNotification('Failed to load configuration', 'error');
      }
      // Keep the default config for new setup
      setConfig(defaultConfig);
    } finally {
      setLoading(false);
      setHasLoadedInitialConfig(true);
    }
  }, [onNotification]);

  // Load config on mount
  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  // Save state to localStorage when it changes and we're editing
  useEffect(() => {
    if (isEditing && hasLoadedInitialConfig) {
      console.debug('[ProviderConfig] Saving state:', config);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
    }
  }, [config, isEditing, hasLoadedInitialConfig]);

  const handleChange = (field: keyof EmailProviderConfig) => (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement> | SelectChangeEvent
  ) => {
    const value = event.target.value;
    setIsEditing(true);
    setConfig(prev => {
      const newConfig = {
        ...prev,
        [field]: field === 'monthlyLimit' ? Number(value) || undefined : value,
      };

      // Set default fromEmail for Mailgun when domain changes
      if (field === 'domain' && newConfig.provider === 'mailgun' && (!prev.fromEmail || prev.fromEmail === `noreply@${prev.domain}`)) {
        newConfig.fromEmail = `noreply@${value}`;
      }

      // Set defaults when switching to Mailgun
      if (field === 'provider' && value === 'mailgun') {
        if (newConfig.domain && (!newConfig.fromEmail || newConfig.fromEmail === '')) {
          newConfig.fromEmail = `noreply@${newConfig.domain}`;
        }
        if (!newConfig.fromName) {
          newConfig.fromName = 'KrakenHashes';
        }
      }

      return newConfig;
    });
  };

  const handleTest = async (email: string) => {
    setLoading(true);
    try {
      // If we're testing after save, just send the test email
      if (!isEditing) {
        const payload = {
          test_email: email,
          test_only: true
        };
        await testEmailConfig(payload);
      } else {
        // Use form data for direct testing
        const payload = {
          test_email: email,
          test_only: true,
          config: {
            provider_type: config.provider,
            api_key: config.apiKey,
            additional_config: {
              from_email: config.fromEmail,
              from_name: config.fromName,
              domain: config.domain,
            },
            monthly_limit: config.monthlyLimit,
            is_active: true,
          }
        };
        await testEmailConfig(payload);
      }
      onNotification('Test email sent successfully', 'success');
      setTestEmailOpen(false);
      setTestEmail('');
    } catch (error) {
      console.error('[ProviderConfig] Failed to send test email:', error);
      onNotification(`Error: ${error instanceof Error ? error.message : 'Failed to send test email'}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (withTest: boolean = false) => {
    if (!config.fromEmail) {
      onNotification('From Email is required', 'error');
      return;
    }

    setLoading(true);
    try {
      const payload = {
        config: {
          provider_type: config.provider,
          api_key: config.apiKey,
          additional_config: {
            from_email: config.fromEmail,
            from_name: config.fromName,
            domain: config.domain,
          },
          monthly_limit: config.monthlyLimit,
          is_active: true,
        },
      };

      console.debug('[ProviderConfig] Saving configuration with payload:', payload);

      // Save the config
      await updateEmailConfig(payload);
      onNotification('Configuration saved successfully', 'success');
      
      // Clear form and reload config
      localStorage.removeItem(STORAGE_KEY);
      setIsEditing(false);
      await loadConfig();

      // If testing after save, use a separate call that will use the database config
      if (withTest) {
        setTestEmailOpen(true);
      }
    } catch (error) {
      console.error('[ProviderConfig] Failed to save configuration:', error);
      onNotification(`Error: ${error instanceof Error ? error.message : 'Failed to save configuration'}`, 'error');
    } finally {
      setLoading(false);
      setSaveWithTestOpen(false);
    }
  };

  const handleCancel = () => {
    console.debug('[ProviderConfig] Canceling configuration');
    localStorage.removeItem(STORAGE_KEY);
    setIsEditing(false);
    loadConfig(); // Reload the original config
  };

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Email Provider Configuration
      </Typography>

      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <FormControl fullWidth>
            <InputLabel>Provider</InputLabel>
            <Select
              value={config.provider}
              label="Provider"
              onChange={handleChange('provider')}
            >
              <MenuItem value="sendgrid">SendGrid</MenuItem>
              <MenuItem value="mailgun">Mailgun</MenuItem>
            </Select>
          </FormControl>
        </Grid>

        <Grid item xs={12} md={6}>
          <TextField
            fullWidth
            label="API Key"
            type="password"
            value={config.apiKey}
            onChange={handleChange('apiKey')}
          />
        </Grid>

        {config.provider === 'sendgrid' && (
          <>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="From Email"
                type="email"
                value={config.fromEmail}
                onChange={handleChange('fromEmail')}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="From Name"
                value={config.fromName}
                onChange={handleChange('fromName')}
              />
            </Grid>
          </>
        )}

        {config.provider === 'mailgun' && (
          <>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                variant="filled"
                label="Domain"
                value={config.domain}
                onChange={handleChange('domain')}
                InputLabelProps={{
                  shrink: true,
                }}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                required
                error={!config.fromEmail}
                fullWidth
                variant="filled"
                label="From Email"
                type="email"
                value={config.fromEmail}
                onChange={handleChange('fromEmail')}
                helperText={!config.fromEmail ? "From Email is required" : "Usually noreply@yourdomain"}
                InputLabelProps={{
                  shrink: true,
                }}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                variant="filled"
                label="From Name"
                value={config.fromName}
                onChange={handleChange('fromName')}
                helperText="Display name for emails"
                InputLabelProps={{
                  shrink: true,
                }}
              />
            </Grid>
          </>
        )}

        <Grid item xs={12} md={6}>
          <TextField
            fullWidth
            variant="filled"
            label="Monthly Limit"
            type="number"
            value={config.monthlyLimit || ''}
            onChange={handleChange('monthlyLimit')}
            helperText="Leave empty for unlimited"
            InputLabelProps={{
              shrink: true,
            }}
          />
        </Grid>

        <Grid item xs={12}>
          <Box sx={{ display: 'flex', gap: 2, justifyContent: 'flex-end' }}>
            <Button
              variant="outlined"
              onClick={handleCancel}
              disabled={loading}
            >
              Cancel
            </Button>
            <Button
              variant="outlined"
              onClick={() => setTestEmailOpen(true)}
              disabled={loading}
            >
              Test Connection
            </Button>
            <LoadingButton
              variant="contained"
              onClick={() => setSaveWithTestOpen(true)}
              loading={loading}
            >
              Save Configuration
            </LoadingButton>
          </Box>
        </Grid>
      </Grid>

      {/* Test Email Dialog */}
      <Dialog open={testEmailOpen} onClose={() => setTestEmailOpen(false)}>
        <DialogTitle>Test Email Configuration</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Enter an email address to send a test email to:
          </DialogContentText>
          <TextField
            autoFocus
            margin="dense"
            label="Test Email Address"
            type="email"
            fullWidth
            variant="outlined"
            value={testEmail}
            onChange={(e) => setTestEmail(e.target.value)}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setTestEmailOpen(false)}>Cancel</Button>
          <Button 
            onClick={() => handleTest(testEmail)}
            disabled={!testEmail || loading}
          >
            Send Test Email
          </Button>
        </DialogActions>
      </Dialog>

      {/* Save with Test Dialog */}
      <Dialog open={saveWithTestOpen} onClose={() => setSaveWithTestOpen(false)}>
        <DialogTitle>Save Configuration</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Would you like to test the configuration after saving?
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSaveWithTestOpen(false)}>Cancel</Button>
          <Button onClick={() => handleSave(false)}>Save Only</Button>
          <Button onClick={() => handleSave(true)}>Save and Test</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}; 