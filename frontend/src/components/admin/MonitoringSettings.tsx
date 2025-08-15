import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  TextField,
  Button,
  Alert,
  CircularProgress,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  FormHelperText,
  Paper,
  Divider,
  FormControlLabel,
  Switch,
} from '@mui/material';
import { useSnackbar } from 'notistack';
import { getMonitoringSettings, updateMonitoringSettings, MonitoringSettings as MonitoringSettingsData } from '../../services/monitoringSettings';

const MonitoringSettings: React.FC = () => {
  const [settings, setSettings] = useState<MonitoringSettingsData>({
    metrics_retention_realtime_days: 7,
    metrics_retention_daily_days: 30,
    metrics_retention_weekly_days: 365,
    enable_aggregation: true,
    aggregation_interval: 'daily',
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { enqueueSnackbar } = useSnackbar();

  useEffect(() => {
    fetchSettings();
  }, []);

  const fetchSettings = async () => {
    setLoading(true);
    setError(null);
    try {
      const monitoringSettings = await getMonitoringSettings();
      setSettings(monitoringSettings);
    } catch (err) {
      console.error('Failed to fetch monitoring settings:', err);
      setError('Failed to load settings. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setError(null);
    setSaving(true);

    try {
      await updateMonitoringSettings(settings);
      enqueueSnackbar('Monitoring settings saved successfully', { variant: 'success' });
    } catch (err: any) {
      console.error('Failed to save monitoring settings:', err);
      setError(err.response?.data?.error || 'Failed to save settings. Please try again.');
      enqueueSnackbar('Failed to save monitoring settings', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleRetentionChange = (field: keyof MonitoringSettingsData) => (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = parseInt(event.target.value, 10);
    if (!isNaN(value) && value >= 0) {
      setSettings({ ...settings, [field]: value });
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight={200}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Monitoring Settings
      </Typography>
      
      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Paper sx={{ p: 3, mb: 3 }}>
        <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 'bold' }}>
          📊 Metrics Retention Periods
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          Configure how long to retain metrics at different levels of detail. Data cascades from fine-grained to aggregated over time.
        </Typography>
        
        <Box sx={{ mb: 3 }}>
          <TextField
            fullWidth
            label="Real-time Data Retention"
            type="number"
            value={settings.metrics_retention_realtime_days}
            onChange={handleRetentionChange('metrics_retention_realtime_days')}
            helperText="Days to keep fine-grained metrics for recent activity (e.g., 7 days)"
            inputProps={{ min: 0, step: 1 }}
            sx={{ mb: 2 }}
          />
          
          <TextField
            fullWidth
            label="Daily Aggregates Retention"
            type="number"
            value={settings.metrics_retention_daily_days}
            onChange={handleRetentionChange('metrics_retention_daily_days')}
            helperText="Days to store daily summaries after real-time period (e.g., 30 days)"
            inputProps={{ min: 0, step: 1 }}
            sx={{ mb: 2 }}
          />
          
          <TextField
            fullWidth
            label="Weekly Aggregates Retention"
            type="number"
            value={settings.metrics_retention_weekly_days}
            onChange={handleRetentionChange('metrics_retention_weekly_days')}
            helperText="Days to preserve long-term trends as weekly summaries (e.g., 365 days)"
            inputProps={{ min: 0, step: 1 }}
          />
        </Box>

        <Divider sx={{ my: 3 }} />

        <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 'bold' }}>
          ⚙️ Aggregation Settings
        </Typography>

        <FormControlLabel
          control={
            <Switch
              checked={settings.enable_aggregation}
              onChange={(e) => setSettings({ ...settings, enable_aggregation: e.target.checked })}
            />
          }
          label="Enable Automatic Aggregation"
          sx={{ mb: 2 }}
        />
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Automatically consolidate older metrics to save storage space while preserving trends
        </Typography>

        <FormControl fullWidth sx={{ mb: 2 }}>
          <InputLabel>Aggregation Interval</InputLabel>
          <Select
            value={settings.aggregation_interval}
            onChange={(e) => setSettings({ ...settings, aggregation_interval: e.target.value })}
            label="Aggregation Interval"
            disabled={!settings.enable_aggregation}
          >
            <MenuItem value="hourly">Hourly</MenuItem>
            <MenuItem value="daily">Daily</MenuItem>
            <MenuItem value="weekly">Weekly</MenuItem>
          </Select>
          <FormHelperText>
            How often to run the aggregation process
          </FormHelperText>
        </FormControl>

        <Alert severity="info" sx={{ mt: 2 }}>
          <Typography variant="body2">
            <strong>How cascading aggregation works:</strong>
            <br />
            • Real-time data provides detailed metrics for recent activity
            <br />
            • After the real-time period, data is aggregated to daily summaries
            <br />
            • After the daily period, data is further aggregated to weekly summaries
            <br />
            • This approach keeps charts readable while preserving historical trends
          </Typography>
        </Alert>
      </Paper>

      <Box display="flex" justifyContent="flex-end">
        <Button
          variant="contained"
          color="primary"
          onClick={handleSave}
          disabled={saving}
          startIcon={saving && <CircularProgress size={20} />}
        >
          {saving ? 'Saving...' : 'Save Settings'}
        </Button>
      </Box>
    </Box>
  );
};

export default MonitoringSettings;