import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Switch,
  FormControlLabel,
  Alert,
  CircularProgress,
} from '@mui/material';
import {
  Email as EmailIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { getNotificationPreferences, updateNotificationPreferences } from '../../services/user';
import { NotificationPreferences } from '../../types/user';

interface NotificationCardProps {
  onNotificationChange?: () => void;
}

const NotificationCard: React.FC<NotificationCardProps> = ({ onNotificationChange }): JSX.Element => {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [preferences, setPreferences] = useState<NotificationPreferences | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    loadPreferences();
  }, []);

  const loadPreferences = async () => {
    try {
      const prefs = await getNotificationPreferences();
      setPreferences(prefs);
      setError(null);
    } catch (err) {
      setError('Failed to load notification preferences');
      console.error('Failed to load notification preferences:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleNotifications = async () => {
    if (!preferences) return;

    try {
      setSaving(true);
      setError(null);
      setSuccess(null);

      // Check if email is configured
      if (!preferences.emailConfigured && !preferences.notifyOnJobCompletion) {
        setError('Email gateway must be configured before enabling email notifications. Please contact your administrator.');
        return;
      }

      const updatedPrefs: NotificationPreferences = {
        ...preferences,
        notifyOnJobCompletion: !preferences.notifyOnJobCompletion,
      };

      const result = await updateNotificationPreferences(updatedPrefs);
      setPreferences(result);
      setSuccess('Notification preferences updated successfully');

      if (onNotificationChange) {
        onNotificationChange();
      }
    } catch (err: any) {
      console.error('Failed to update notification preferences:', err);
      if (err.response?.data?.code === 'EMAIL_GATEWAY_REQUIRED') {
        setError('Email gateway must be configured before enabling email notifications. Please contact your administrator.');
      } else {
        setError(err.response?.data?.error || 'Failed to update notification preferences');
      }
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Card>
        <CardContent>
          <Box display="flex" justifyContent="center" alignItems="center" minHeight={200}>
            <CircularProgress />
          </Box>
        </CardContent>
      </Card>
    );
  }

  if (!preferences) {
    return (
      <Card>
        <CardContent>
          <Alert severity="error">Failed to load notification preferences</Alert>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center' }}>
          <EmailIcon sx={{ mr: 1 }} />
          Email Notifications
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {success && (
          <Alert severity="success" sx={{ mb: 2 }}>
            {success}
          </Alert>
        )}

        {!preferences.emailConfigured && (
          <Alert severity="warning" sx={{ mb: 2 }} icon={<WarningIcon />}>
            Email gateway is not configured. Email notifications are currently unavailable.
            Please contact your administrator to enable email functionality.
          </Alert>
        )}

        <Box sx={{ mb: 3 }}>
          <FormControlLabel
            control={
              <Switch
                checked={preferences.notifyOnJobCompletion}
                onChange={handleToggleNotifications}
                disabled={!preferences.emailConfigured || saving}
                color="primary"
              />
            }
            label={
              <Box>
                <Typography variant="body1">
                  Job Completion Notifications
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Receive email notifications when your jobs complete
                </Typography>
              </Box>
            }
          />
        </Box>

        {preferences.notifyOnJobCompletion && !preferences.emailConfigured && (
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            Note: Notifications are enabled but will not be sent until an email gateway is configured.
          </Typography>
        )}
      </CardContent>
    </Card>
  );
};

export default NotificationCard;