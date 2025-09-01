import React, { useEffect, useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Switch,
  FormControlLabel,
  FormGroup,
  TextField,
  Button,
  Checkbox,
  Alert,
  CircularProgress,
} from '@mui/material';
import { MFASettings as IMFASettings, MFAMethod } from '../../types/auth';
import { getAdminMFASettings, updateMFASettings } from '../../services/auth';
import { getEmailConfig } from '../../services/api';

const MFASettings: React.FC = () => {
  const [settings, setSettings] = useState<IMFASettings>({
    requireMfa: false,
    allowedMfaMethods: ['email'],
    emailCodeValidity: 5,
    backupCodesCount: 8,
    mfaCodeCooldownMinutes: 1,
    mfaCodeExpiryMinutes: 5,
    mfaMaxAttempts: 3,
    mfaEnabled: false
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [hasEmailGateway, setHasEmailGateway] = useState(false);

  // Note: passkey is not yet implemented, so we exclude it from available methods
  const availableMethods: MFAMethod[] = ['email', 'authenticator'];

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      // Check email gateway status
      try {
        const emailConfig = await getEmailConfig();
        setHasEmailGateway(!!emailConfig?.data?.provider_type && emailConfig?.data?.is_active !== false);
      } catch (err) {
        // If we can't fetch email config, assume no gateway
        setHasEmailGateway(false);
      }

      const data = await getAdminMFASettings();
      setSettings(data);
      setError(null);
    } catch (err) {
      setError('Failed to load MFA settings');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    setSuccess(false);

    try {
      // Ensure email is always included in allowed methods if MFA is required
      const updatedSettings = {
        ...settings,
        allowedMfaMethods: settings.requireMfa 
          ? Array.from(new Set([...settings.allowedMfaMethods, 'email']))
          : settings.allowedMfaMethods
      };

      await updateMFASettings(updatedSettings);
      setSuccess(true);
      setSettings(updatedSettings);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Failed to update MFA settings');
      }
    } finally {
      setSaving(false);
    }
  };

  const handleMethodToggle = (method: MFAMethod) => {
    setSettings(prev => {
      const newMethods = prev.allowedMfaMethods.includes(method)
        ? prev.allowedMfaMethods.filter(m => m !== method)
        : [...prev.allowedMfaMethods, method];

      // Ensure email is always included if MFA is required
      if (prev.requireMfa && method !== 'email') {
        if (!newMethods.includes('email')) {
          newMethods.push('email');
        }
      }

      return {
        ...prev,
        allowedMfaMethods: newMethods,
      };
    });
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" p={4}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="h5" gutterBottom>
          Multi-Factor Authentication Settings
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {success && (
          <Alert severity="success" sx={{ mb: 2 }}>
            Settings updated successfully
          </Alert>
        )}

        <FormGroup>
          <FormControlLabel
            control={
              <Switch
                checked={settings.requireMfa}
                onChange={e => {
                  const newValue = e.target.checked;
                  setSettings(prev => ({
                    ...prev,
                    requireMfa: newValue,
                    // Ensure email is included when enabling required MFA
                    allowedMfaMethods: newValue 
                      ? Array.from(new Set([...prev.allowedMfaMethods, 'email']))
                      : prev.allowedMfaMethods
                  }));
                }}
                disabled={!hasEmailGateway}
              />
            }
            label={hasEmailGateway ? "Require MFA for all users" : "Require MFA for all users (Email config required)"}
          />

          <Typography variant="subtitle1" sx={{ mt: 2, mb: 1 }}>
            Allowed MFA Methods
          </Typography>

          {availableMethods.map(method => (
            <FormControlLabel
              key={method}
              control={
                <Checkbox
                  checked={settings.allowedMfaMethods.includes(method)}
                  onChange={() => handleMethodToggle(method)}
                  disabled={!settings.requireMfa || (method === 'email' && settings.requireMfa)}
                />
              }
              label={method.charAt(0).toUpperCase() + method.slice(1)}
            />
          ))}
          
          {/* Show passkey as not currently implemented */}
          <FormControlLabel
            control={
              <Checkbox
                checked={false}
                disabled={true}
              />
            }
            label="Passkey (Not currently implemented)"
          />

          <Box sx={{ mt: 2 }}>
            <TextField
              label="Email Code Validity (minutes)"
              type="number"
              value={settings.emailCodeValidity}
              onChange={e => setSettings(prev => ({ ...prev, emailCodeValidity: parseInt(e.target.value) || 5 }))}
              disabled={!settings.requireMfa || !settings.allowedMfaMethods.includes('email')}
              fullWidth
              margin="normal"
            />

            <TextField
              label="Number of Backup Codes"
              type="number"
              value={settings.backupCodesCount}
              onChange={e => setSettings(prev => ({ ...prev, backupCodesCount: parseInt(e.target.value) || 8 }))}
              disabled={!settings.requireMfa}
              fullWidth
              margin="normal"
            />

            <TextField
              label="Code Cooldown (minutes)"
              type="number"
              value={settings.mfaCodeCooldownMinutes}
              onChange={e => setSettings(prev => ({ ...prev, mfaCodeCooldownMinutes: parseInt(e.target.value) || 1 }))}
              disabled={!settings.requireMfa}
              fullWidth
              margin="normal"
            />

            <TextField
              label="Code Expiry (minutes)"
              type="number"
              value={settings.mfaCodeExpiryMinutes}
              onChange={e => setSettings(prev => ({ ...prev, mfaCodeExpiryMinutes: parseInt(e.target.value) || 5 }))}
              disabled={!settings.requireMfa}
              fullWidth
              margin="normal"
            />

            <TextField
              label="Maximum Code Attempts"
              type="number"
              value={settings.mfaMaxAttempts}
              onChange={e => setSettings(prev => ({ ...prev, mfaMaxAttempts: parseInt(e.target.value) || 3 }))}
              disabled={!settings.requireMfa}
              fullWidth
              margin="normal"
            />
          </Box>

          <Box sx={{ mt: 3 }}>
            <Button
              variant="contained"
              color="primary"
              onClick={handleSave}
              disabled={saving}
              startIcon={saving && <CircularProgress size={20} color="inherit" />}
            >
              {saving ? 'Saving...' : 'Save Changes'}
            </Button>
          </Box>
        </FormGroup>
      </CardContent>
    </Card>
  );
};

export default MFASettings; 