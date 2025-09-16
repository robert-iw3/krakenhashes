import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Button,
  Grid,
  Slider,
  Alert,
  CircularProgress,
  Tooltip,
  InputAdornment
} from '@mui/material';
import { Save as SaveIcon } from '@mui/icons-material';
import { useSnackbar } from 'notistack';
import { api } from '../../services/api';

interface AgentDownloadSettings {
  max_concurrent_downloads: number;
  download_timeout_minutes: number;
  download_retry_attempts: number;
  progress_interval_seconds: number;
  chunk_size_mb: number;
}

const AgentDownloadSettings: React.FC = () => {
  const { enqueueSnackbar } = useSnackbar();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [settings, setSettings] = useState<AgentDownloadSettings>({
    max_concurrent_downloads: 3,
    download_timeout_minutes: 60,
    download_retry_attempts: 3,
    progress_interval_seconds: 10,
    chunk_size_mb: 10
  });

  useEffect(() => {
    fetchSettings();
  }, []);

  const fetchSettings = async () => {
    try {
      const response = await api.get<AgentDownloadSettings>('/api/admin/settings/agent-download');
      setSettings(response.data);
    } catch (error) {
      console.error('Failed to fetch agent download settings:', error);
      enqueueSnackbar('Failed to load agent download settings', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await api.put('/api/admin/settings/agent-download', settings);
      enqueueSnackbar('Agent download settings updated successfully', { variant: 'success' });
    } catch (error) {
      console.error('Failed to update agent download settings:', error);
      enqueueSnackbar('Failed to update agent download settings', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleChange = (field: keyof AgentDownloadSettings) => (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const value = parseInt(event.target.value, 10);
    if (!isNaN(value)) {
      setSettings(prev => ({ ...prev, [field]: value }));
    }
  };

  const handleSliderChange = (field: keyof AgentDownloadSettings) => (
    event: Event,
    value: number | number[]
  ) => {
    setSettings(prev => ({ ...prev, [field]: value as number }));
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Agent File Transfer Settings
      </Typography>
      <Typography variant="body2" color="text.secondary" gutterBottom sx={{ mb: 3 }}>
        Configure how agents download files from the backend. These settings apply to all connected agents.
      </Typography>

      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" gutterBottom fontWeight="bold">
                Concurrent Downloads
              </Typography>
              <Typography variant="body2" color="text.secondary" gutterBottom>
                Maximum number of files that can be downloaded simultaneously
              </Typography>
              <Box sx={{ px: 2, pt: 2 }}>
                <Slider
                  value={settings.max_concurrent_downloads}
                  onChange={handleSliderChange('max_concurrent_downloads')}
                  valueLabelDisplay="on"
                  step={1}
                  marks
                  min={1}
                  max={10}
                />
              </Box>
              <TextField
                type="number"
                value={settings.max_concurrent_downloads}
                onChange={handleChange('max_concurrent_downloads')}
                fullWidth
                size="small"
                inputProps={{ min: 1, max: 10 }}
                sx={{ mt: 2 }}
              />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" gutterBottom fontWeight="bold">
                Download Timeout
              </Typography>
              <Typography variant="body2" color="text.secondary" gutterBottom>
                Maximum time allowed for a single file download
              </Typography>
              <TextField
                type="number"
                value={settings.download_timeout_minutes}
                onChange={handleChange('download_timeout_minutes')}
                fullWidth
                size="small"
                InputProps={{
                  endAdornment: <InputAdornment position="end">minutes</InputAdornment>,
                }}
                inputProps={{ min: 1, max: 1440 }}
                helperText="Between 1 minute and 24 hours (1440 minutes)"
                sx={{ mt: 2 }}
              />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" gutterBottom fontWeight="bold">
                Retry Attempts
              </Typography>
              <Typography variant="body2" color="text.secondary" gutterBottom>
                Number of times to retry failed downloads
              </Typography>
              <Box sx={{ px: 2, pt: 2 }}>
                <Slider
                  value={settings.download_retry_attempts}
                  onChange={handleSliderChange('download_retry_attempts')}
                  valueLabelDisplay="on"
                  step={1}
                  marks
                  min={0}
                  max={10}
                />
              </Box>
              <TextField
                type="number"
                value={settings.download_retry_attempts}
                onChange={handleChange('download_retry_attempts')}
                fullWidth
                size="small"
                inputProps={{ min: 0, max: 10 }}
                sx={{ mt: 2 }}
              />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" gutterBottom fontWeight="bold">
                Progress Report Interval
              </Typography>
              <Typography variant="body2" color="text.secondary" gutterBottom>
                How often agents report download progress
              </Typography>
              <TextField
                type="number"
                value={settings.progress_interval_seconds}
                onChange={handleChange('progress_interval_seconds')}
                fullWidth
                size="small"
                InputProps={{
                  endAdornment: <InputAdornment position="end">seconds</InputAdornment>,
                }}
                inputProps={{ min: 1, max: 300 }}
                helperText="Between 1 second and 5 minutes (300 seconds)"
                sx={{ mt: 2 }}
              />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" gutterBottom fontWeight="bold">
                Chunk Size
              </Typography>
              <Tooltip title="Size of download chunks for resume capability (future feature)">
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  Download chunk size for future resume capability
                </Typography>
              </Tooltip>
              <TextField
                type="number"
                value={settings.chunk_size_mb}
                onChange={handleChange('chunk_size_mb')}
                fullWidth
                size="small"
                InputProps={{
                  endAdornment: <InputAdornment position="end">MB</InputAdornment>,
                }}
                inputProps={{ min: 1, max: 100 }}
                helperText="Between 1 MB and 100 MB"
                sx={{ mt: 2 }}
              />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12}>
          <Alert severity="info" sx={{ mb: 2 }}>
            Changes will apply to agents on their next connection. Currently connected agents will receive updates automatically.
          </Alert>

          <Box display="flex" justifyContent="flex-end">
            <Button
              variant="contained"
              color="primary"
              onClick={handleSave}
              disabled={saving}
              startIcon={saving ? <CircularProgress size={20} /> : <SaveIcon />}
            >
              {saving ? 'Saving...' : 'Save Settings'}
            </Button>
          </Box>
        </Grid>
      </Grid>
    </Box>
  );
};

export default AgentDownloadSettings;