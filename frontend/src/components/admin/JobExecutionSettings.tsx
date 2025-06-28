import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  TextField,
  Button,
  Alert,
  CircularProgress,
  Grid,
  FormControlLabel,
  Switch,
  Divider,
  Paper,
  InputAdornment,
} from '@mui/material';
import { useSnackbar } from 'notistack';
import { getJobExecutionSettings, updateJobExecutionSettings, JobExecutionSettings } from '../../services/jobSettings';

const JobExecutionSettingsComponent: React.FC = () => {
  const [settings, setSettings] = useState<JobExecutionSettings | null>(null);
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
      const data = await getJobExecutionSettings();
      setSettings(data);
    } catch (err: any) {
      console.error('Failed to fetch job execution settings:', err);
      setError(err.response?.data?.error || 'Failed to load settings');
      enqueueSnackbar('Failed to load job execution settings', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!settings) return;
    
    setError(null);
    setSaving(true);
    
    try {
      await updateJobExecutionSettings(settings);
      enqueueSnackbar('Job execution settings updated successfully', { variant: 'success' });
    } catch (err: any) {
      console.error('Failed to update job execution settings:', err);
      const message = err.response?.data?.error || 'Failed to save settings';
      setError(message);
      enqueueSnackbar(message, { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleChange = (field: keyof JobExecutionSettings) => (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    if (!settings) return;
    
    const value = event.target.type === 'checkbox' 
      ? event.target.checked 
      : parseInt(event.target.value, 10);
    
    setSettings({
      ...settings,
      [field]: value,
    });
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (!settings) {
    return (
      <Alert severity="error">Failed to load job execution settings</Alert>
    );
  }

  const convertSecondsToMinutes = (seconds: number) => Math.floor(seconds / 60);
  const convertMinutesToSeconds = (minutes: number) => minutes * 60;
  const convertHoursToDays = (hours: number) => Math.floor(hours / 24);
  const convertDaysToHours = (days: number) => days * 24;

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Job Execution Settings
      </Typography>
      <Typography variant="body2" color="textSecondary" gutterBottom>
        Configure how jobs are executed and distributed across agents
      </Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Grid container spacing={3}>
        {/* Chunking Settings */}
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="subtitle1" gutterBottom fontWeight="bold">
              Job Chunking
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Grid container spacing={2}>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  type="number"
                  label="Default Chunk Duration"
                  value={convertSecondsToMinutes(settings.default_chunk_duration)}
                  onChange={(e) => {
                    const minutes = parseInt(e.target.value, 10);
                    setSettings({
                      ...settings,
                      default_chunk_duration: convertMinutesToSeconds(minutes),
                    });
                  }}
                  helperText="Duration for each job chunk"
                  InputProps={{
                    inputProps: { min: 1 },
                    endAdornment: <InputAdornment position="end">minutes</InputAdornment>,
                  }}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  type="number"
                  label="Chunk Fluctuation Percentage"
                  value={settings.chunk_fluctuation_percentage}
                  onChange={handleChange('chunk_fluctuation_percentage')}
                  helperText="Allowed fluctuation for final chunks"
                  InputProps={{
                    inputProps: { min: 0, max: 100 },
                    endAdornment: <InputAdornment position="end">%</InputAdornment>,
                  }}
                />
              </Grid>
            </Grid>
          </Paper>
        </Grid>

        {/* Agent Settings */}
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="subtitle1" gutterBottom fontWeight="bold">
              Agent Configuration
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Grid container spacing={2}>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  type="number"
                  label="Hashlist Retention"
                  value={convertHoursToDays(settings.agent_hashlist_retention_hours)}
                  onChange={(e) => {
                    const days = parseInt(e.target.value, 10);
                    setSettings({
                      ...settings,
                      agent_hashlist_retention_hours: convertDaysToHours(days),
                    });
                  }}
                  helperText="How long agents retain hashlists"
                  InputProps={{
                    inputProps: { min: 1 },
                    endAdornment: <InputAdornment position="end">days</InputAdornment>,
                  }}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  type="number"
                  label="Max Concurrent Jobs per Agent"
                  value={settings.max_concurrent_jobs_per_agent}
                  onChange={handleChange('max_concurrent_jobs_per_agent')}
                  helperText="Maximum jobs an agent can run simultaneously"
                  InputProps={{
                    inputProps: { min: 1 },
                  }}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  type="number"
                  label="Progress Reporting Interval"
                  value={settings.progress_reporting_interval}
                  onChange={handleChange('progress_reporting_interval')}
                  helperText="How often agents report progress"
                  InputProps={{
                    inputProps: { min: 1 },
                    endAdornment: <InputAdornment position="end">seconds</InputAdornment>,
                  }}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  type="number"
                  label="Benchmark Cache Duration"
                  value={convertHoursToDays(settings.benchmark_cache_duration_hours)}
                  onChange={(e) => {
                    const days = parseInt(e.target.value, 10);
                    setSettings({
                      ...settings,
                      benchmark_cache_duration_hours: convertDaysToHours(days),
                    });
                  }}
                  helperText="How long to cache agent benchmarks"
                  InputProps={{
                    inputProps: { min: 1 },
                    endAdornment: <InputAdornment position="end">days</InputAdornment>,
                  }}
                />
              </Grid>
            </Grid>
          </Paper>
        </Grid>

        {/* Job Control Settings */}
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="subtitle1" gutterBottom fontWeight="bold">
              Job Control
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Grid container spacing={2}>
              <Grid item xs={12} md={6}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={settings.job_interruption_enabled}
                      onChange={handleChange('job_interruption_enabled')}
                    />
                  }
                  label="Allow Job Interruption"
                />
                <Typography variant="caption" color="textSecondary" display="block">
                  Higher priority jobs can interrupt running jobs
                </Typography>
              </Grid>
              <Grid item xs={12} md={6}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={settings.enable_realtime_crack_notifications}
                      onChange={handleChange('enable_realtime_crack_notifications')}
                    />
                  }
                  label="Real-time Crack Notifications"
                />
                <Typography variant="caption" color="textSecondary" display="block">
                  Send notifications when hashes are cracked
                </Typography>
              </Grid>
              <Grid item xs={12} md={4}>
                <TextField
                  fullWidth
                  type="number"
                  label="Job Refresh Interval"
                  value={settings.job_refresh_interval_seconds}
                  onChange={handleChange('job_refresh_interval_seconds')}
                  helperText="How often to refresh job status in UI"
                  InputProps={{
                    inputProps: { min: 1, max: 60 },
                    endAdornment: <InputAdornment position="end">seconds</InputAdornment>,
                  }}
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <TextField
                  fullWidth
                  type="number"
                  label="Max Chunk Retry Attempts"
                  value={settings.max_chunk_retry_attempts}
                  onChange={handleChange('max_chunk_retry_attempts')}
                  helperText="Number of times to retry failed chunks"
                  InputProps={{
                    inputProps: { min: 0, max: 10 },
                  }}
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <TextField
                  fullWidth
                  type="number"
                  label="Jobs Per Page"
                  value={settings.jobs_per_page_default}
                  onChange={handleChange('jobs_per_page_default')}
                  helperText="Default pagination size for job lists"
                  InputProps={{
                    inputProps: { min: 5, max: 100 },
                  }}
                />
              </Grid>
            </Grid>
          </Paper>
        </Grid>

        {/* Metrics Retention Settings */}
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="subtitle1" gutterBottom fontWeight="bold">
              Metrics Retention
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Grid container spacing={2}>
              <Grid item xs={12} md={4}>
                <TextField
                  fullWidth
                  type="number"
                  label="Real-time Metrics"
                  value={settings.metrics_retention_realtime_days}
                  onChange={handleChange('metrics_retention_realtime_days')}
                  helperText="Days to retain real-time metrics"
                  InputProps={{
                    inputProps: { min: 1 },
                    endAdornment: <InputAdornment position="end">days</InputAdornment>,
                  }}
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <TextField
                  fullWidth
                  type="number"
                  label="Daily Aggregated Metrics"
                  value={settings.metrics_retention_daily_days}
                  onChange={handleChange('metrics_retention_daily_days')}
                  helperText="Days to retain daily metrics"
                  InputProps={{
                    inputProps: { min: 7 },
                    endAdornment: <InputAdornment position="end">days</InputAdornment>,
                  }}
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <TextField
                  fullWidth
                  type="number"
                  label="Weekly Aggregated Metrics"
                  value={settings.metrics_retention_weekly_days}
                  onChange={handleChange('metrics_retention_weekly_days')}
                  helperText="Days to retain weekly metrics"
                  InputProps={{
                    inputProps: { min: 30 },
                    endAdornment: <InputAdornment position="end">days</InputAdornment>,
                  }}
                />
              </Grid>
            </Grid>
          </Paper>
        </Grid>

        {/* Save Button */}
        <Grid item xs={12}>
          <Box display="flex" justifyContent="flex-end">
            <Button
              variant="contained"
              color="primary"
              onClick={handleSave}
              disabled={saving || loading}
              size="large"
            >
              {saving ? <CircularProgress size={24} /> : 'Save Settings'}
            </Button>
          </Box>
        </Grid>
      </Grid>
    </Box>
  );
};

export default JobExecutionSettingsComponent;