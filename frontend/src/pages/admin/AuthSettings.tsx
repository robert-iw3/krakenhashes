import React, { useState } from 'react';
import { Box, Typography, Button, CircularProgress } from '@mui/material';
import { useSnackbar } from 'notistack';
import { getPasswordPolicy, getAccountSecurity, updateAuthSettings } from '../../services/auth';
import { PasswordPolicy, AccountSecurity, AuthSettingsUpdate } from '../../types/auth';
import AuthSettingsForm from '../../components/admin/AuthSettings';

const AuthSettingsPage = () => {
  const [loading, setLoading] = useState(false);
  const { enqueueSnackbar } = useSnackbar();

  const handleSave = async (settings: AuthSettingsUpdate) => {
    setLoading(true);
    try {
      await updateAuthSettings(settings);
      enqueueSnackbar('Authentication settings updated successfully', { variant: 'success' });
    } catch (error) {
      enqueueSnackbar('Failed to update authentication settings', { variant: 'error' });
      console.error('Failed to update settings:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box>
      <Typography variant="h4" gutterBottom>
        Authentication Settings
      </Typography>
      <AuthSettingsForm onSave={handleSave} loading={loading} />
    </Box>
  );
};

export default AuthSettingsPage; 