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
} from '@mui/material';
import { useSnackbar } from 'notistack';
import { api } from '../../services/api';

interface MonitoringSettingsData {
  metrics_retention_days: number;
  enable_aggregation: boolean;
  aggregation_interval: string;
}

const MonitoringSettings: React.FC = () => {
  const [settings, setSettings] = useState<MonitoringSettingsData>({
    metrics_retention_days: 0, // 0 means unlimited
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
      const response = await api.get('/api/system-settings');
      const settingsData = response.data.data || {};
      
      // Extract monitoring-related settings
      setSettings({
        metrics_retention_days: parseInt(settingsData.metrics_retention_days || '0', 10),
        enable_aggregation: settingsData.enable_aggregation === 'true',
        aggregation_interval: settingsData.aggregation_interval || 'daily',
      });
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
      // Save each setting
      const updates = [
        { key: 'metrics_retention_days', value: settings.metrics_retention_days.toString() },
        { key: 'enable_aggregation', value: settings.enable_aggregation.toString() },
        { key: 'aggregation_interval', value: settings.aggregation_interval },
      ];

      for (const update of updates) {
        await api.put('/api/system-settings', update);
      }

      enqueueSnackbar('Monitoring settings saved successfully', { variant: 'success' });
    } catch (err: any) {
      console.error('Failed to save monitoring settings:', err);
      setError(err.response?.data?.error || 'Failed to save settings. Please try again.');
      enqueueSnackbar('Failed to save monitoring settings', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleRetentionChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = parseInt(event.target.value, 10);
    if (!isNaN(value) && value >= 0) {
      setSettings({ ...settings, metrics_retention_days: value });
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
          Metrics Retention
        </Typography>
        
        <TextField
          fullWidth
          label="Retention Period (days)"
          type="number"
          value={settings.metrics_retention_days}
          onChange={handleRetentionChange}
          helperText="Number of days to retain device performance metrics. Set to 0 for unlimited retention."
          inputProps={{ min: 0, step: 1 }}
          sx={{ mb: 3 }}
        />

        <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 'bold', mt: 2 }}>
          Data Aggregation
        </Typography>

        <FormControl fullWidth sx={{ mb: 2 }}>
          <InputLabel>Aggregation Interval</InputLabel>
          <Select
            value={settings.aggregation_interval}
            onChange={(e) => setSettings({ ...settings, aggregation_interval: e.target.value })}
            label="Aggregation Interval"
          >
            <MenuItem value="hourly">Hourly</MenuItem>
            <MenuItem value="daily">Daily</MenuItem>
            <MenuItem value="weekly">Weekly</MenuItem>
          </Select>
          <FormHelperText>
            Historical data will be aggregated at this interval to save storage space
          </FormHelperText>
        </FormControl>

        <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
          Note: Real-time metrics are always retained for the most recent 24 hours regardless of retention settings.
          Older data is aggregated based on the interval selected above.
        </Typography>
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