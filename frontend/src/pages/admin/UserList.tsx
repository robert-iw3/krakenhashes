import React, { useState, useEffect, useCallback } from 'react';
import { 
    Box, Typography, Paper, CircularProgress, Alert, Chip, IconButton, Tooltip,
    Button, Dialog, DialogTitle, DialogContent, DialogActions, TextField,
    FormControl, InputLabel, Select, MenuItem, FormHelperText
} from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import EditIcon from '@mui/icons-material/Edit';
import LockIcon from '@mui/icons-material/Lock';
import LockOpenIcon from '@mui/icons-material/LockOpen';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import AddIcon from '@mui/icons-material/Add';
import { useSnackbar } from 'notistack';
import { useNavigate } from 'react-router-dom';
import { format } from 'date-fns';

import { User } from '../../types/user';
import { listAdminUsers, enableAdminUser, disableAdminUser, createAdminUser } from '../../services/api';
import { useAuth } from '../../contexts/AuthContext';

const UserList: React.FC = () => {
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const [actionLoading, setActionLoading] = useState<string | null>(null);
    const [createDialogOpen, setCreateDialogOpen] = useState(false);
    const [createLoading, setCreateLoading] = useState(false);
    const [formData, setFormData] = useState({
        username: '',
        email: '',
        password: '',
        confirmPassword: '',
        role: 'user'
    });
    const [formErrors, setFormErrors] = useState<Record<string, string>>({});

    const { enqueueSnackbar } = useSnackbar();
    const { userRole } = useAuth();
    const navigate = useNavigate();

    const fetchUsers = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const response = await listAdminUsers();
            setUsers(response.data.data || []); 
        } catch (err) {
            console.error("Failed to fetch users:", err);
            setError('Failed to load users. Please try refreshing.');
            enqueueSnackbar('Failed to load users', { variant: 'error' });
        } finally {
            setLoading(false);
        }
    }, [enqueueSnackbar]);

    useEffect(() => {
        if (userRole === 'admin') { 
            fetchUsers();
        }
    }, [userRole, fetchUsers]);

    const handleEnableUser = async (userId: string) => {
        setActionLoading(userId);
        try {
            await enableAdminUser(userId);
            enqueueSnackbar('User enabled successfully', { variant: 'success' });
            fetchUsers(); // Refresh list
        } catch (err) {
            console.error('Failed to enable user:', err);
            enqueueSnackbar('Failed to enable user', { variant: 'error' });
        } finally {
            setActionLoading(null);
        }
    };

    const handleDisableUser = async (userId: string) => {
        const reason = prompt('Please provide a reason for disabling this user:');
        if (!reason) return;

        setActionLoading(userId);
        try {
            await disableAdminUser(userId, { reason });
            enqueueSnackbar('User disabled successfully', { variant: 'success' });
            fetchUsers(); // Refresh list
        } catch (err) {
            console.error('Failed to disable user:', err);
            enqueueSnackbar('Failed to disable user', { variant: 'error' });
        } finally {
            setActionLoading(null);
        }
    };

    const handleCreateUser = async () => {
        // Reset errors
        setFormErrors({});
        
        // Validate form
        const errors: Record<string, string> = {};
        
        if (!formData.username) {
            errors.username = 'Username is required';
        }
        
        if (!formData.email) {
            errors.email = 'Email is required';
        } else if (!formData.email.includes('@')) {
            errors.email = 'Invalid email format';
        }
        
        if (!formData.password) {
            errors.password = 'Password is required';
        } else if (formData.password.length < 8) {
            errors.password = 'Password must be at least 8 characters';
        }
        
        if (!formData.confirmPassword) {
            errors.confirmPassword = 'Please confirm password';
        } else if (formData.password !== formData.confirmPassword) {
            errors.confirmPassword = 'Passwords do not match';
        }
        
        if (Object.keys(errors).length > 0) {
            setFormErrors(errors);
            return;
        }
        
        setCreateLoading(true);
        try {
            await createAdminUser({
                username: formData.username,
                email: formData.email,
                password: formData.password,
                role: formData.role
            });
            
            enqueueSnackbar('User created successfully', { variant: 'success' });
            setCreateDialogOpen(false);
            setFormData({
                username: '',
                email: '',
                password: '',
                confirmPassword: '',
                role: 'user'
            });
            fetchUsers(); // Refresh list
        } catch (err: any) {
            console.error('Failed to create user:', err);
            const message = err.response?.data?.error || 'Failed to create user';
            enqueueSnackbar(message, { variant: 'error' });
        } finally {
            setCreateLoading(false);
        }
    };

    const formatDate = (dateString?: string) => {
        if (!dateString) return '-';
        try {
            return format(new Date(dateString), 'MMM dd, yyyy HH:mm');
        } catch {
            return '-';
        }
    };

    const columns: GridColDef[] = [
        { 
            field: 'username', 
            headerName: 'Username', 
            flex: 1, 
            minWidth: 150 
        },
        { 
            field: 'email', 
            headerName: 'Email', 
            flex: 1.5, 
            minWidth: 200 
        },
        { 
            field: 'role', 
            headerName: 'Role', 
            width: 100,
            renderCell: (params: GridRenderCellParams) => (
                <Chip 
                    label={params.value} 
                    size="small" 
                    color={
                        params.value === 'system' ? 'success' : 
                        params.value === 'admin' ? 'error' : 
                        'default'
                    }
                />
            )
        },
        { 
            field: 'accountStatus', 
            headerName: 'Status', 
            width: 120,
            renderCell: (params: GridRenderCellParams) => {
                const user = params.row as User;
                if (!user.accountEnabled) {
                    return <Chip label="Disabled" size="small" color="error" />;
                }
                if (user.accountLocked) {
                    return <Chip label="Locked" size="small" color="warning" />;
                }
                return <Chip label="Active" size="small" color="success" />;
            }
        },
        { 
            field: 'mfaEnabled', 
            headerName: 'MFA', 
            width: 80,
            renderCell: (params: GridRenderCellParams) => (
                params.value ? 
                    <CheckCircleIcon color="success" fontSize="small" /> : 
                    <CancelIcon color="disabled" fontSize="small" />
            )
        },
        { 
            field: 'lastLogin', 
            headerName: 'Last Login', 
            width: 180,
            renderCell: (params: GridRenderCellParams) => formatDate(params.value as string)
        },
        { 
            field: 'createdAt', 
            headerName: 'Created', 
            width: 180,
            renderCell: (params: GridRenderCellParams) => formatDate(params.value as string)
        },
        {
            field: 'actions',
            headerName: 'Actions',
            width: 150,
            sortable: false,
            renderCell: (params: GridRenderCellParams) => {
                const user = params.row as User;
                const isLoading = actionLoading === user.id;
                
                return (
                    <Box>
                        <Tooltip title="Edit User">
                            <IconButton
                                size="small"
                                onClick={() => navigate(`/admin/users/${user.id}`)}
                                disabled={isLoading}
                            >
                                <EditIcon fontSize="small" />
                            </IconButton>
                        </Tooltip>
                        
                        {user.accountEnabled ? (
                            <Tooltip title="Disable User">
                                <IconButton
                                    size="small"
                                    onClick={() => handleDisableUser(user.id)}
                                    disabled={isLoading}
                                    color="error"
                                >
                                    {isLoading ? <CircularProgress size={16} /> : <LockIcon fontSize="small" />}
                                </IconButton>
                            </Tooltip>
                        ) : (
                            <Tooltip title="Enable User">
                                <IconButton
                                    size="small"
                                    onClick={() => handleEnableUser(user.id)}
                                    disabled={isLoading}
                                    color="success"
                                >
                                    {isLoading ? <CircularProgress size={16} /> : <LockOpenIcon fontSize="small" />}
                                </IconButton>
                            </Tooltip>
                        )}
                    </Box>
                );
            }
        }
    ];

    if (loading) {
        return (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <CircularProgress />
            </Box>
        );
    }

    return (
        <Box sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3 }}>
                <Box>
                    <Typography variant="h4" component="h1" gutterBottom>
                        User Management
                    </Typography>
                    <Typography variant="body1" color="text.secondary">
                        Manage user accounts and permissions
                    </Typography>
                </Box>
                <Button
                    variant="contained"
                    startIcon={<AddIcon />}
                    onClick={() => setCreateDialogOpen(true)}
                >
                    Add User
                </Button>
            </Box>
            
            <Paper sx={{ p: 2, mt: 3 }}>
                {error && (
                    <Alert severity="error" sx={{ mb: 2 }}>
                        {error}
                    </Alert>
                )}
                
                <DataGrid
                    rows={users}
                    columns={columns}
                    initialState={{
                        pagination: {
                            paginationModel: { pageSize: 10 }
                        }
                    }}
                    pageSizeOptions={[10, 25, 50]}
                    autoHeight
                    disableRowSelectionOnClick
                    getRowId={(row) => row.id}
                    sx={{
                        '& .MuiDataGrid-row': {
                            cursor: 'pointer'
                        }
                    }}
                />
            </Paper>

            {/* Create User Dialog */}
            <Dialog 
                open={createDialogOpen} 
                onClose={() => {
                    setCreateDialogOpen(false);
                    setFormData({
                        username: '',
                        email: '',
                        password: '',
                        confirmPassword: '',
                        role: 'user'
                    });
                    setFormErrors({});
                }}
                maxWidth="sm"
                fullWidth
            >
                <DialogTitle>Create New User</DialogTitle>
                <DialogContent>
                    <Box sx={{ mt: 2 }}>
                        <TextField
                            fullWidth
                            label="Username"
                            value={formData.username}
                            onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                            error={!!formErrors.username}
                            helperText={formErrors.username}
                            margin="normal"
                            required
                        />
                        <TextField
                            fullWidth
                            label="Email"
                            type="email"
                            value={formData.email}
                            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                            error={!!formErrors.email}
                            helperText={formErrors.email}
                            margin="normal"
                            required
                        />
                        <TextField
                            fullWidth
                            label="Password"
                            type="password"
                            value={formData.password}
                            onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                            error={!!formErrors.password}
                            helperText={formErrors.password || "Minimum 8 characters"}
                            margin="normal"
                            required
                        />
                        <TextField
                            fullWidth
                            label="Confirm Password"
                            type="password"
                            value={formData.confirmPassword}
                            onChange={(e) => setFormData({ ...formData, confirmPassword: e.target.value })}
                            error={!!formErrors.confirmPassword}
                            helperText={formErrors.confirmPassword}
                            margin="normal"
                            required
                        />
                        <FormControl fullWidth margin="normal" required>
                            <InputLabel>Role</InputLabel>
                            <Select
                                value={formData.role}
                                onChange={(e) => setFormData({ ...formData, role: e.target.value })}
                                label="Role"
                            >
                                <MenuItem value="user">User</MenuItem>
                                <MenuItem value="admin">Admin</MenuItem>
                            </Select>
                            <FormHelperText>Select user's role in the system</FormHelperText>
                        </FormControl>
                    </Box>
                </DialogContent>
                <DialogActions>
                    <Button 
                        onClick={() => {
                            setCreateDialogOpen(false);
                            setFormData({
                                username: '',
                                email: '',
                                password: '',
                                confirmPassword: '',
                                role: 'user'
                            });
                            setFormErrors({});
                        }}
                        disabled={createLoading}
                    >
                        Cancel
                    </Button>
                    <Button 
                        onClick={handleCreateUser}
                        variant="contained"
                        disabled={createLoading}
                    >
                        {createLoading ? <CircularProgress size={24} /> : 'Create'}
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
};

export default UserList;