import React, { useState, useEffect, useCallback } from 'react';
import { 
    Box, Typography, Paper, CircularProgress, Alert, Chip, IconButton, Tooltip
} from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import EditIcon from '@mui/icons-material/Edit';
import LockIcon from '@mui/icons-material/Lock';
import LockOpenIcon from '@mui/icons-material/LockOpen';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import { useSnackbar } from 'notistack';
import { useNavigate } from 'react-router-dom';
import { format } from 'date-fns';

import { User } from '../../types/user';
import { listAdminUsers, enableAdminUser, disableAdminUser } from '../../services/api';
import { useAuth } from '../../contexts/AuthContext';

const UserList: React.FC = () => {
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const [actionLoading, setActionLoading] = useState<string | null>(null);

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
            <Typography variant="h4" gutterBottom>
                User Management
            </Typography>
            
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
        </Box>
    );
};

export default UserList;