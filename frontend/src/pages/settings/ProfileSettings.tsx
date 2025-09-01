import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Button,
  Grid,
  Alert,
  CircularProgress,
  Divider,
} from '@mui/material';
import { useAuth } from '../../contexts/AuthContext';
import { updateUserProfile, ProfileUpdate } from '../../services/user';
import { getPasswordPolicy } from '../../services/auth';
import { PasswordPolicy } from '../../types/auth';
import PasswordValidation from '../../components/common/PasswordValidation';
import MFACard from '../../components/settings/MFACard';
import NotificationCard from '../../components/settings/NotificationCard';

interface UserProfile {
  username: string;
  email: string;
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}

const ProfileSettings: React.FC = () => {
  const { user, setUser } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [policy, setPolicy] = useState<PasswordPolicy | null>(null);
  const [profile, setProfile] = useState<UserProfile>({
    username: user?.username || '',
    email: user?.email || '',
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  });

  useEffect(() => {
    const loadPolicy = async () => {
      try {
        const policyData = await getPasswordPolicy();
        setPolicy(policyData);
      } catch (error) {
        console.error('Failed to load password policy:', error);
      }
    };
    loadPolicy();
  }, []);

  const validatePassword = (password: string): boolean => {
    if (!policy) return false;

    const validation = {
      length: password.length >= (policy.minPasswordLength || 15),
      uppercase: !policy.requireUppercase || /[A-Z]/.test(password),
      lowercase: !policy.requireLowercase || /[a-z]/.test(password),
      numbers: !policy.requireNumbers || /[0-9]/.test(password),
      specialChars: !policy.requireSpecialChars || /[!@#$%^&*(),.?":{}|<>]/.test(password),
    };

    return Object.values(validation).every(Boolean);
  };

  const handleChange = (field: keyof UserProfile) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setProfile(prev => ({ ...prev, [field]: event.target.value }));
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const updates: ProfileUpdate = {};

      // Handle email update
      if (profile.email !== user?.email) {
        updates.email = profile.email;
      }

      // Only handle password update if current password is provided
      if (profile.currentPassword) {
        if (profile.newPassword) {
          if (profile.newPassword !== profile.confirmPassword) {
            throw new Error('New passwords do not match');
          }
          if (!validatePassword(profile.newPassword)) {
            throw new Error('New password does not meet requirements');
          }
          updates.currentPassword = profile.currentPassword;
          updates.newPassword = profile.newPassword;
        } else {
          throw new Error('Please enter a new password or clear the current password field');
        }
      } else if (profile.newPassword || profile.confirmPassword) {
        throw new Error('Current password is required to change password');
      }

      if (Object.keys(updates).length === 0) {
        setSuccess('No changes to save');
        setLoading(false);
        return;
      }

      await updateUserProfile(updates);
      setSuccess('Profile updated successfully');
      
      if (updates.email && setUser && user) {
        setUser({ ...user, email: updates.email });
      }

      // Clear password fields after successful update
      setProfile(prev => ({
        ...prev,
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      }));
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to update profile');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Profile Settings
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

      <form onSubmit={handleSubmit}>
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Account Information
            </Typography>
            <Grid container spacing={2}>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Username"
                  value={profile.username}
                  disabled
                  margin="normal"
                  helperText="Username cannot be changed"
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Email"
                  value={profile.email}
                  onChange={handleChange('email')}
                  type="email"
                  margin="normal"
                />
              </Grid>
            </Grid>

            <Divider sx={{ my: 3 }} />

            <Typography variant="h6" gutterBottom>
              Change Password
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              To change your password, enter your current password and a new password below.
              Leave these fields empty if you only want to update your email.
            </Typography>
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Current Password"
                  value={profile.currentPassword}
                  onChange={handleChange('currentPassword')}
                  type="password"
                  margin="normal"
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="New Password"
                  value={profile.newPassword}
                  onChange={handleChange('newPassword')}
                  type="password"
                  margin="normal"
                  disabled={!profile.currentPassword}
                  helperText={!profile.currentPassword ? "Enter current password first" : ""}
                />
                {profile.newPassword && (
                  <PasswordValidation password={profile.newPassword} />
                )}
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Confirm New Password"
                  value={profile.confirmPassword}
                  onChange={handleChange('confirmPassword')}
                  type="password"
                  margin="normal"
                  disabled={!profile.currentPassword}
                  error={profile.newPassword !== profile.confirmPassword && profile.confirmPassword !== ''}
                  helperText={
                    !profile.currentPassword ? "Enter current password first" :
                    profile.confirmPassword !== '' && (
                      profile.newPassword !== profile.confirmPassword 
                        ? 'Passwords do not match'
                        : profile.newPassword === profile.confirmPassword
                          ? 'Passwords match'
                          : ''
                    )
                  }
                  FormHelperTextProps={{
                    sx: {
                      color: profile.confirmPassword !== '' && profile.newPassword === profile.confirmPassword 
                        ? 'success.main' 
                        : 'error.main'
                    }
                  }}
                />
              </Grid>
            </Grid>

            <Box sx={{ mt: 3, display: 'flex', justifyContent: 'flex-end' }}>
              <Button
                variant="contained"
                color="primary"
                type="submit"
                disabled={loading}
                startIcon={loading && <CircularProgress size={20} color="inherit" />}
              >
                {loading ? 'Saving...' : 'Save Changes'}
              </Button>
            </Box>
          </CardContent>
        </Card>

        <MFACard onMFAChange={() => {
          // Refresh user data when MFA settings change
          if (setUser && user) {
            setUser({ ...user });
          }
        }} />

        <NotificationCard onNotificationChange={() => {
          // You can add any refresh logic here if needed
          console.log('Notification preferences updated');
        }} />
      </form>
    </Box>
  );
}

export default ProfileSettings; 