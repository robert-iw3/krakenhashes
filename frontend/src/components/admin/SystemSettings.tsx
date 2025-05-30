import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Button,
  Alert,
  CircularProgress,
  Grid,
  Tooltip,
  IconButton,
} from '@mui/material';
import { Info as InfoIcon } from '@mui/icons-material';
import { useSnackbar } from 'notistack';
import { getMaxPriority, updateMaxPriority } from '../../services/systemSettings';
import { MaxPriorityConfig, SystemSettingsFormData } from '../../types/systemSettings';

interface SystemSettingsProps {
  onSave?: (settings: SystemSettingsFormData) => Promise<void>;
  loading?: boolean;
}

const SystemSettings: React.FC<SystemSettingsProps> = ({ onSave, loading = false }) => {
  const [formData, setFormData] = useState<SystemSettingsFormData>({
    max_priority: 1000,
  });
  const [loadingData, setLoadingData] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { enqueueSnackbar } = useSnackbar();

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      setLoadingData(true);
      const data = await getMaxPriority();
      setFormData({
        max_priority: data.max_priority,
      });
      setError(null);
    } catch (error) {
      console.error('Failed to load system settings:', error);
      setError('Failed to load system settings');
    } finally {
      setLoadingData(false);
    }
  };

  const handleSave = async () => {
    if (typeof formData.max_priority === 'string' && formData.max_priority.trim() === '') {
      setError('Maximum priority is required');
      return;
    }

    const maxPriority = typeof formData.max_priority === 'string' 
      ? parseInt(formData.max_priority) 
      : formData.max_priority;

    if (isNaN(maxPriority) || maxPriority < 1) {
      setError('Maximum priority must be a positive number');
      return;
    }

    if (maxPriority > 1000000) {
      setError('Maximum priority cannot exceed 1,000,000');
      return;
    }

    try {
      setSaving(true);
      setError(null);
      
      if (onSave) {
        await onSave({ max_priority: maxPriority });
      } else {
        await updateMaxPriority(maxPriority);
      }
      
      enqueueSnackbar('System settings updated successfully', { variant: 'success' });
      
      // Reload settings to get the updated values
      await loadSettings();
    } catch (error: any) {
      console.error('Failed to save system settings:', error);
      
      // Handle specific error responses
      if (error.response?.status === 409) {
        const errorData = error.response.data;
        if (errorData.conflicting_jobs) {
          setError(`${errorData.message}\n\nConflicting preset jobs:\n${errorData.conflicting_jobs.join('\n')}`);
        } else {
          setError(errorData.message || 'Cannot update maximum priority due to conflicts');
        }
      } else {
        setError(error.response?.data?.message || error.message || 'Failed to save system settings');
      }
    } finally {
      setSaving(false);
    }
  };

  const handleMaxPriorityChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData(prev => ({
      ...prev,
      max_priority: e.target.value,
    }));
    setError(null);
  };

  if (loadingData) {
    return (
      <Box display="flex" justifyContent="center" p={4}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      {error && (
        <Alert severity="error" sx={{ mb: 3 }} style={{ whiteSpace: 'pre-line' }}>
          {error}
        </Alert>
      )}

      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Box display="flex" alignItems="center" mb={2}>
                <Typography variant="h6" component="h3">
                  Priority Settings
                </Typography>
                <Tooltip title="Configure the maximum priority value that can be assigned to jobs and preset jobs. This helps maintain consistent priority ranges across your organization.">
                  <IconButton size="small" sx={{ ml: 1 }}>
                    <InfoIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </Box>
              
              <TextField
                fullWidth
                label="Maximum Job Priority"
                type="number"
                value={formData.max_priority}
                onChange={handleMaxPriorityChange}
                disabled={loading || saving}
                inputProps={{
                  min: 1,
                  max: 1000000,
                }}
                helperText="Set the maximum priority value (1-1,000,000). Jobs and preset jobs cannot exceed this priority."
                sx={{ mb: 3 }}
              />

              <Box display="flex" gap={2}>
                <Button
                  variant="contained"
                  onClick={handleSave}
                  disabled={loading || saving || loadingData}
                  startIcon={saving ? <CircularProgress size={20} /> : null}
                >
                  {saving ? 'Saving...' : 'Save Settings'}
                </Button>
                
                <Button
                  variant="outlined"
                  onClick={loadSettings}
                  disabled={loading || saving || loadingData}
                >
                  Reset
                </Button>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="h6" component="h3" gutterBottom>
                Priority System Information
              </Typography>
              
              <Typography variant="body2" color="text.secondary" paragraph>
                The priority system uses a range from 0 to your configured maximum priority. 
                Higher numbers indicate higher priority.
              </Typography>
              
              <Typography variant="body2" color="text.secondary" paragraph>
                <strong>Current Maximum:</strong> {typeof formData.max_priority === 'string' ? formData.max_priority : formData.max_priority.toLocaleString()}
              </Typography>
              
              <Typography variant="body2" color="text.secondary" paragraph>
                <strong>Note:</strong> You cannot set a maximum priority lower than any existing 
                preset job priorities. Update or remove high-priority preset jobs first if needed.
              </Typography>
              
              <Typography variant="body2" color="text.secondary">
                <strong>Recommended ranges by organization size:</strong>
                <br />• Small organization: 0-100
                <br />• Medium/large organization: 0-1,000
                <br />• Ridiculous workload organization: 0-10,000
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default SystemSettings; 