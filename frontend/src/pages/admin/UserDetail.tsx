import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
    Box, Typography, Paper, TextField, Button, CircularProgress,
    Alert, Grid, Card, CardContent, Divider, Chip, IconButton,
    Dialog, DialogTitle, DialogContent, DialogActions, FormControlLabel,
    Checkbox, List, ListItem, ListItemText, ListItemIcon, Select,
    MenuItem, FormControl, InputLabel
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import SaveIcon from '@mui/icons-material/Save';
import LockResetIcon from '@mui/icons-material/LockReset';
import SecurityIcon from '@mui/icons-material/Security';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import PersonIcon from '@mui/icons-material/Person';
import EmailIcon from '@mui/icons-material/Email';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import CloseIcon from '@mui/icons-material/Close';
import { useSnackbar, closeSnackbar } from 'notistack';
import { format } from 'date-fns';

import { User } from '../../types/user';
import { 
    getAdminUser, 
    updateAdminUser, 
    resetAdminUserPassword,
    disableAdminUserMFA,
    enableAdminUser,
    disableAdminUser,
    unlockAdminUser
} from '../../services/api';

const UserDetail: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { enqueueSnackbar } = useSnackbar();

    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Form state
    const [username, setUsername] = useState('');
    const [email, setEmail] = useState('');
    const [role, setRole] = useState('');
    const [hasChanges, setHasChanges] = useState(false);

    // Dialog states
    const [resetPasswordOpen, setResetPasswordOpen] = useState(false);
    const [disableMFAOpen, setDisableMFAOpen] = useState(false);
    const [disableAccountOpen, setDisableAccountOpen] = useState(false);

    // Password reset state
    const [newPassword, setNewPassword] = useState('');
    const [temporaryPassword, setTemporaryPassword] = useState(true);

    const fetchUser = useCallback(async () => {
        if (!id) return;
        
        setLoading(true);
        setError(null);
        try {
            const response = await getAdminUser(id);
            setUser(response.data.data);
            setUsername(response.data.data.username);
            setEmail(response.data.data.email);
            setRole(response.data.data.role);
        } catch (err) {
            console.error("Failed to fetch user:", err);
            setError('Failed to load user details');
            enqueueSnackbar('Failed to load user details', { variant: 'error' });
        } finally {
            setLoading(false);
        }
    }, [id, enqueueSnackbar]);

    useEffect(() => {
        fetchUser();
    }, [fetchUser]);

    useEffect(() => {
        if (user) {
            setHasChanges(
                username !== user.username || 
                email !== user.email || 
                role !== user.role
            );
        }
    }, [username, email, role, user]);

    const handleSave = async () => {
        if (!user || !hasChanges) return;

        setSaving(true);
        try {
            const updateData: any = { username, email };
            // Only include role if it's different and not trying to set to system
            if (role !== user.role && role !== 'system') {
                updateData.role = role;
            }
            await updateAdminUser(user.id, updateData);
            enqueueSnackbar('User details updated successfully', { variant: 'success' });
            fetchUser(); // Refresh data
        } catch (err: any) {
            console.error('Failed to update user:', err);
            const message = err.response?.data?.error || 'Failed to update user';
            enqueueSnackbar(message, { variant: 'error' });
        } finally {
            setSaving(false);
        }
    };

    const handleResetPassword = async () => {
        if (!user) return;

        setSaving(true);
        try {
            const response = await resetAdminUserPassword(user.id, {
                password: newPassword || undefined,
                temporary: temporaryPassword
            });
            
            if (response.data.data.temporary_password) {
                // Show temporary password in a dialog
                const tempPassword = response.data.data.temporary_password;
                
                // Create a custom notification with copy functionality
                const message = (
                    <Box>
                        <Typography variant="body2" gutterBottom>
                            Password reset successfully!
                        </Typography>
                        <Typography variant="body2" sx={{ fontWeight: 'bold', my: 1 }}>
                            Temporary password: {tempPassword}
                        </Typography>
                        <Button
                            size="small"
                            variant="outlined"
                            onClick={() => {
                                navigator.clipboard.writeText(tempPassword);
                                enqueueSnackbar('Password copied to clipboard', { variant: 'info' });
                            }}
                            sx={{ mt: 1 }}
                        >
                            Copy to Clipboard
                        </Button>
                        <Typography variant="caption" display="block" sx={{ mt: 2 }}>
                            Please share this password securely with the user.
                        </Typography>
                    </Box>
                );
                
                enqueueSnackbar(message, { 
                    variant: 'success', 
                    persist: true,
                    action: (key) => (
                        <IconButton
                            size="small"
                            color="inherit"
                            onClick={() => closeSnackbar(key)}
                        >
                            <CloseIcon />
                        </IconButton>
                    )
                });
            } else {
                enqueueSnackbar('Password reset successfully', { variant: 'success' });
            }
            
            setResetPasswordOpen(false);
            setNewPassword('');
            setTemporaryPassword(true);
        } catch (err) {
            console.error('Failed to reset password:', err);
            enqueueSnackbar('Failed to reset password', { variant: 'error' });
        } finally {
            setSaving(false);
        }
    };

    const handleDisableMFA = async () => {
        if (!user) return;

        setSaving(true);
        try {
            await disableAdminUserMFA(user.id);
            enqueueSnackbar('MFA disabled successfully', { variant: 'success' });
            setDisableMFAOpen(false);
            fetchUser(); // Refresh data
        } catch (err) {
            console.error('Failed to disable MFA:', err);
            enqueueSnackbar('Failed to disable MFA', { variant: 'error' });
        } finally {
            setSaving(false);
        }
    };

    const handleToggleAccount = async (reason?: string) => {
        if (!user) return;

        setSaving(true);
        try {
            if (user.accountEnabled) {
                await disableAdminUser(user.id, { reason: reason! });
                enqueueSnackbar('User account disabled', { variant: 'success' });
            } else {
                await enableAdminUser(user.id);
                enqueueSnackbar('User account enabled', { variant: 'success' });
            }
            setDisableAccountOpen(false);
            fetchUser(); // Refresh data
        } catch (err) {
            console.error('Failed to toggle account:', err);
            enqueueSnackbar('Failed to update account status', { variant: 'error' });
        } finally {
            setSaving(false);
        }
    };

    const handleUnlockAccount = async () => {
        if (!user) return;

        setSaving(true);
        try {
            await unlockAdminUser(user.id);
            enqueueSnackbar('User account unlocked', { variant: 'success' });
            fetchUser(); // Refresh data
        } catch (err) {
            console.error('Failed to unlock account:', err);
            enqueueSnackbar('Failed to unlock account', { variant: 'error' });
        } finally {
            setSaving(false);
        }
    };

    const formatDate = (dateString?: string) => {
        if (!dateString) return 'Never';
        try {
            return format(new Date(dateString), 'MMM dd, yyyy HH:mm:ss');
        } catch {
            return 'Invalid date';
        }
    };

    if (loading) {
        return (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <CircularProgress />
            </Box>
        );
    }

    if (error || !user) {
        return (
            <Box sx={{ p: 3 }}>
                <Alert severity="error">{error || 'User not found'}</Alert>
                <Button startIcon={<ArrowBackIcon />} onClick={() => navigate('/admin/users')} sx={{ mt: 2 }}>
                    Back to Users
                </Button>
            </Box>
        );
    }

    return (
        <Box sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
                <IconButton onClick={() => navigate('/admin/users')} sx={{ mr: 2 }}>
                    <ArrowBackIcon />
                </IconButton>
                <Typography variant="h4">User Details</Typography>
            </Box>

            <Grid container spacing={3}>
                {/* User Information Card */}
                <Grid item xs={12} md={8}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6" gutterBottom>User Information</Typography>
                            <Divider sx={{ mb: 2 }} />
                            
                            <Grid container spacing={2}>
                                <Grid item xs={12} sm={6}>
                                    <TextField
                                        fullWidth
                                        label="Username"
                                        value={username}
                                        onChange={(e) => setUsername(e.target.value)}
                                        disabled={user.role === 'system'}
                                        InputProps={{
                                            startAdornment: <PersonIcon sx={{ mr: 1, color: 'action.active' }} />
                                        }}
                                    />
                                </Grid>
                                <Grid item xs={12} sm={6}>
                                    <TextField
                                        fullWidth
                                        label="Email"
                                        value={email}
                                        onChange={(e) => setEmail(e.target.value)}
                                        disabled={user.role === 'system'}
                                        InputProps={{
                                            startAdornment: <EmailIcon sx={{ mr: 1, color: 'action.active' }} />
                                        }}
                                    />
                                </Grid>
                                <Grid item xs={12} sm={6}>
                                    {user.role === 'system' ? (
                                        <TextField
                                            fullWidth
                                            label="Role"
                                            value={user.role}
                                            disabled
                                            InputProps={{
                                                endAdornment: (
                                                    <Chip 
                                                        label={user.role} 
                                                        size="small" 
                                                        color="success"
                                                    />
                                                )
                                            }}
                                        />
                                    ) : (
                                        <FormControl fullWidth>
                                            <InputLabel>Role</InputLabel>
                                            <Select
                                                value={role}
                                                label="Role"
                                                onChange={(e) => setRole(e.target.value)}
                                            >
                                                <MenuItem value="user">User</MenuItem>
                                                <MenuItem value="admin">Admin</MenuItem>
                                            </Select>
                                        </FormControl>
                                    )}
                                </Grid>
                                <Grid item xs={12} sm={6}>
                                    <TextField
                                        fullWidth
                                        label="User ID"
                                        value={user.id}
                                        disabled
                                    />
                                </Grid>
                            </Grid>

                            <Box sx={{ mt: 3, display: 'flex', gap: 2 }}>
                                <Button
                                    variant="contained"
                                    startIcon={<SaveIcon />}
                                    onClick={handleSave}
                                    disabled={!hasChanges || saving || user.role === 'system'}
                                >
                                    Save Changes
                                </Button>
                                {user.role === 'system' && (
                                    <Typography variant="caption" color="text.secondary" sx={{ alignSelf: 'center' }}>
                                        System users cannot be modified
                                    </Typography>
                                )}
                            </Box>
                        </CardContent>
                    </Card>

                    {/* Account Status Card */}
                    <Card sx={{ mt: 3 }}>
                        <CardContent>
                            <Typography variant="h6" gutterBottom>Account Status</Typography>
                            <Divider sx={{ mb: 2 }} />
                            
                            <List>
                                <ListItem>
                                    <ListItemIcon>
                                        {user.accountEnabled ? <CheckCircleIcon color="success" /> : <CancelIcon color="error" />}
                                    </ListItemIcon>
                                    <ListItemText
                                        primary="Account Status"
                                        secondary={user.accountEnabled ? 'Enabled' : `Disabled - ${user.disabledReason || 'No reason provided'}`}
                                    />
                                    <Button
                                        variant="outlined"
                                        color={user.accountEnabled ? 'error' : 'success'}
                                        onClick={() => user.accountEnabled ? setDisableAccountOpen(true) : handleToggleAccount()}
                                        disabled={user.role === 'system'}
                                    >
                                        {user.accountEnabled ? 'Disable' : 'Enable'}
                                    </Button>
                                </ListItem>

                                <ListItem>
                                    <ListItemIcon>
                                        {user.accountLocked ? <CancelIcon color="warning" /> : <CheckCircleIcon color="success" />}
                                    </ListItemIcon>
                                    <ListItemText
                                        primary="Lock Status"
                                        secondary={user.accountLocked ? 
                                            `Locked until ${formatDate(user.accountLockedUntil)}` : 
                                            'Not locked'}
                                    />
                                    {user.accountLocked && (
                                        <Button
                                            variant="outlined"
                                            color="warning"
                                            onClick={handleUnlockAccount}
                                        >
                                            Unlock
                                        </Button>
                                    )}
                                </ListItem>

                                <ListItem>
                                    <ListItemIcon>
                                        <SecurityIcon color={user.mfaEnabled ? 'success' : 'disabled'} />
                                    </ListItemIcon>
                                    <ListItemText
                                        primary="MFA Status"
                                        secondary={user.mfaEnabled ? 
                                            `Enabled (${user.mfaType.join(', ')})` : 
                                            'Disabled'}
                                    />
                                    {user.mfaEnabled && (
                                        <Button
                                            variant="outlined"
                                            color="warning"
                                            onClick={() => setDisableMFAOpen(true)}
                                        >
                                            Disable MFA
                                        </Button>
                                    )}
                                </ListItem>
                            </List>

                            <Box sx={{ mt: 3 }}>
                                <Button
                                    variant="outlined"
                                    startIcon={<LockResetIcon />}
                                    onClick={() => setResetPasswordOpen(true)}
                                    disabled={user.role === 'system'}
                                >
                                    Reset Password
                                </Button>
                            </Box>
                        </CardContent>
                    </Card>
                </Grid>

                {/* Activity Card */}
                <Grid item xs={12} md={4}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6" gutterBottom>Activity Information</Typography>
                            <Divider sx={{ mb: 2 }} />
                            
                            <List dense>
                                <ListItem>
                                    <ListItemIcon>
                                        <CalendarTodayIcon fontSize="small" />
                                    </ListItemIcon>
                                    <ListItemText
                                        primary="Created"
                                        secondary={formatDate(user.createdAt)}
                                    />
                                </ListItem>
                                <ListItem>
                                    <ListItemIcon>
                                        <CalendarTodayIcon fontSize="small" />
                                    </ListItemIcon>
                                    <ListItemText
                                        primary="Last Login"
                                        secondary={formatDate(user.lastLogin)}
                                    />
                                </ListItem>
                                <ListItem>
                                    <ListItemIcon>
                                        <CalendarTodayIcon fontSize="small" />
                                    </ListItemIcon>
                                    <ListItemText
                                        primary="Password Changed"
                                        secondary={formatDate(user.lastPasswordChange)}
                                    />
                                </ListItem>
                                {user.failedLoginAttempts > 0 && (
                                    <ListItem>
                                        <ListItemText
                                            primary="Failed Login Attempts"
                                            secondary={`${user.failedLoginAttempts} attempts`}
                                        />
                                    </ListItem>
                                )}
                                {user.disabledAt && (
                                    <ListItem>
                                        <ListItemText
                                            primary="Disabled At"
                                            secondary={formatDate(user.disabledAt)}
                                        />
                                    </ListItem>
                                )}
                            </List>
                        </CardContent>
                    </Card>
                </Grid>
            </Grid>

            {/* Reset Password Dialog */}
            <Dialog open={resetPasswordOpen} onClose={() => setResetPasswordOpen(false)} maxWidth="sm" fullWidth>
                <DialogTitle>Reset User Password</DialogTitle>
                <DialogContent>
                    <FormControlLabel
                        control={
                            <Checkbox
                                checked={temporaryPassword}
                                onChange={(e) => setTemporaryPassword(e.target.checked)}
                            />
                        }
                        label="Generate temporary password"
                        sx={{ mb: 2 }}
                    />
                    {!temporaryPassword && (
                        <TextField
                            fullWidth
                            type="password"
                            label="New Password"
                            value={newPassword}
                            onChange={(e) => setNewPassword(e.target.value)}
                            helperText="Must be at least 8 characters"
                        />
                    )}
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setResetPasswordOpen(false)}>Cancel</Button>
                    <Button 
                        onClick={handleResetPassword} 
                        variant="contained" 
                        disabled={saving || (!temporaryPassword && newPassword.length < 8)}
                    >
                        Reset Password
                    </Button>
                </DialogActions>
            </Dialog>

            {/* Disable MFA Dialog */}
            <Dialog open={disableMFAOpen} onClose={() => setDisableMFAOpen(false)}>
                <DialogTitle>Disable MFA</DialogTitle>
                <DialogContent>
                    <Typography>
                        Are you sure you want to disable MFA for this user? 
                        They will need to set it up again if required.
                    </Typography>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setDisableMFAOpen(false)}>Cancel</Button>
                    <Button onClick={handleDisableMFA} variant="contained" color="warning" disabled={saving}>
                        Disable MFA
                    </Button>
                </DialogActions>
            </Dialog>

            {/* Disable Account Dialog */}
            <Dialog open={disableAccountOpen} onClose={() => setDisableAccountOpen(false)} maxWidth="sm" fullWidth>
                <DialogTitle>Disable User Account</DialogTitle>
                <DialogContent>
                    <TextField
                        fullWidth
                        multiline
                        rows={3}
                        label="Reason for disabling"
                        placeholder="Please provide a reason..."
                        sx={{ mt: 2 }}
                        onChange={(e) => setUser({ ...user, disabledReason: e.target.value })}
                    />
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setDisableAccountOpen(false)}>Cancel</Button>
                    <Button 
                        onClick={() => handleToggleAccount(user.disabledReason)} 
                        variant="contained" 
                        color="error"
                        disabled={saving || !user.disabledReason}
                    >
                        Disable Account
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
};

export default UserDetail;