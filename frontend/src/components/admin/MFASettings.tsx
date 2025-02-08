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

const MFASettings: React.FC = () => {
  const [settings, setSettings] = useState<IMFASettings>({
    requireMfa: false,
    allowedMfaMethods: [],
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

  const availableMethods: MFAMethod[] = ['email', 'authenticator', 'passkey'];

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
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
      await updateMFASettings(settings);
      setSuccess(true);
    } catch (err) {
      setError('Failed to update MFA settings');
    } finally {
      setSaving(false);
    }
  };

  const handleMethodToggle = (method: MFAMethod) => {
    setSettings(prev => ({
      ...prev,
      allowedMfaMethods: prev.allowedMfaMethods.includes(method)
        ? prev.allowedMfaMethods.filter(m => m !== method)
        : [...prev.allowedMfaMethods, method],
    }));
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
                onChange={e => setSettings(prev => ({ ...prev, requireMfa: e.target.checked }))}
              />
            }
            label="Require MFA for all users"
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
                  disabled={!settings.requireMfa}
                />
              }
              label={method.charAt(0).toUpperCase() + method.slice(1)}
            />
          ))}

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