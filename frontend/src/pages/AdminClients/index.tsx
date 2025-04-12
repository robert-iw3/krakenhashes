import React, { useState, useEffect, useCallback } from 'react';
import { 
    Box, Typography, Button, Paper, CircularProgress, Alert,
    Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle, TextField 
} from '@mui/material';
import { DataGrid, GridColDef, GridRowParams, GridActionsCellItem } from '@mui/x-data-grid';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import { useSnackbar } from 'notistack';
import { useNavigate } from 'react-router-dom';

import { Client } from '../../types/client';
import { listAdminClients, createAdminClient, updateAdminClient, deleteAdminClient, getDefaultClientRetentionSetting } from '../../services/api';
import { useAuth } from '../../contexts/AuthContext';
import { Navigate } from 'react-router-dom';

export const AdminClients: React.FC = () => {
    const [clients, setClients] = useState<Client[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const [isAddEditDialogOpen, setIsAddEditDialogOpen] = useState<boolean>(false);
    const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState<boolean>(false);
    const [selectedClient, setSelectedClient] = useState<Client | null>(null);
    const [clientFormData, setClientFormData] = useState<Partial<Client>>({ name: '', description: '', contactInfo: '', dataRetentionMonths: null });
    const [formError, setFormError] = useState<string | null>(null);
    const [isSaving, setIsSaving] = useState<boolean>(false);
    const [defaultRetention, setDefaultRetention] = useState<string | null>(null);
    const [isDefaultRetentionLoading, setIsDefaultRetentionLoading] = useState(true);

    const { enqueueSnackbar } = useSnackbar();
    const { userRole } = useAuth();
    const navigate = useNavigate();

    const fetchClients = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const response = await listAdminClients();
            console.log("[AdminClients] Fetched clients data:", response.data);
            setClients(response.data.data || []); 
        } catch (err) {
            console.error("Failed to fetch clients:", err);
            setError('Failed to load clients. Please try refreshing.');
            enqueueSnackbar('Failed to load clients', { variant: 'error' });
        } finally {
            setLoading(false);
        }
    }, [enqueueSnackbar]);

    const fetchDefaultRetention = useCallback(async () => {
        console.log("[AdminClients] Fetching default retention...");
        setIsDefaultRetentionLoading(true);
        try {
            const response = await getDefaultClientRetentionSetting();
            const defaultValue = response?.data?.data?.value;
            if (defaultValue !== undefined && defaultValue !== null) {
                console.log(`[AdminClients] Default retention fetched: ${defaultValue}`);
                setDefaultRetention(String(defaultValue));
            } else {
                console.warn("[AdminClients] Default retention setting not found or value is null/undefined.");
                setDefaultRetention(null);
            }
        } catch (err) {
            console.error("[AdminClients] Failed to fetch default retention setting:", err);
            setError('Failed to load default client settings.');
            setDefaultRetention(null);
        } finally {
            setIsDefaultRetentionLoading(false);
        }
    }, []);

    useEffect(() => {
        if (userRole === 'admin') { 
            console.log("[AdminClients] User is admin, fetching clients...");
            fetchClients();
            fetchDefaultRetention();
        } else {
             console.log("[AdminClients] User role not admin or not loaded yet:", userRole);
        }
    }, [userRole, fetchClients, fetchDefaultRetention]);

    if (userRole === null) {
        console.log("[AdminClients] userRole is null, showing loading spinner.");
        return (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <CircularProgress />
            </Box>
        );
    }
    
    if (userRole !== 'admin') {
        console.log(`[AdminClients] userRole is '${userRole}', redirecting.`);
        return <Navigate to="/" replace />;
    }

    console.log("[AdminClients] Rendering main content...");
    const columns: GridColDef[] = [
        { field: 'name', headerName: 'Name', flex: 1, minWidth: 150 },
        { field: 'description', headerName: 'Description', flex: 2, minWidth: 200 },
        { field: 'contactInfo', headerName: 'Contact', flex: 1, minWidth: 150 },
        {
            field: 'dataRetentionMonths',
            headerName: 'Retention (Raw)',
            flex: 1,
            minWidth: 150,
        },
        {
            field: 'createdAt',
            headerName: 'Created (Raw)',
            flex: 1,
            minWidth: 180,
        },
        {
            field: 'actions',
            type: 'actions',
            headerName: 'Actions',
            width: 100,
            cellClassName: 'actions',
            getActions: (params: GridRowParams<Client>) => [ 
                <GridActionsCellItem
                    icon={<EditIcon />}
                    label="Edit"
                    onClick={() => handleEditClick(params.row)} 
                    color="inherit"
                />,
                <GridActionsCellItem
                    icon={<DeleteIcon />}
                    label="Delete"
                    onClick={() => handleDeleteClick(params.row)} 
                    color="inherit"
                />,
            ],
        },
    ];
    
    const handleAddClick = () => {
        setSelectedClient(null);
        setFormError(null); 
        setClientFormData({ 
          name: '', 
          description: '', 
          contactInfo: '', 
          dataRetentionMonths: defaultRetention ? parseInt(defaultRetention, 0) : null 
        }); 
        setIsAddEditDialogOpen(true);
    };

    const handleEditClick = (client: Client) => {
        setSelectedClient(client);
        setClientFormData({
            name: client.name,
            description: client.description || '',
            contactInfo: client.contactInfo || '',
            dataRetentionMonths: client.dataRetentionMonths === undefined ? null : client.dataRetentionMonths 
        });
        setFormError(null);
        setIsAddEditDialogOpen(true);
    };

    const handleDeleteClick = (client: Client) => {
        setSelectedClient(client);
        setIsDeleteDialogOpen(true);
    };

    const handleCloseDialog = () => {
        setIsAddEditDialogOpen(false);
        setIsDeleteDialogOpen(false);
        setSelectedClient(null);
        setFormError(null);
    };

    const handleFormChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const { name, value } = event.target;
        setClientFormData(prev => ({
            ...prev,
            [name]: name === 'dataRetentionMonths' ? (value === '' ? null : parseInt(value, 10)) : value
        }));
    };

    const handleSaveClient = async () => {
        setFormError(null);
        setIsSaving(true);

        if (!clientFormData.name?.trim()) {
            setFormError('Client Name is required.');
            setIsSaving(false);
            return;
        }
        const retention = clientFormData.dataRetentionMonths;
        if (retention != null && (isNaN(retention) || retention < 0)) {
            setFormError('Retention Period must be a non-negative number or empty (for default).');
            setIsSaving(false);
            return;
        }

        const payload: Partial<Client> = {
            name: clientFormData.name,
            description: clientFormData.description || undefined,
            contactInfo: clientFormData.contactInfo || undefined,
            dataRetentionMonths: clientFormData.dataRetentionMonths
        };

        try {
            if (selectedClient) { 
                await updateAdminClient(selectedClient.id, payload);
                enqueueSnackbar('Client updated successfully', { variant: 'success' });
            } else { 
                await createAdminClient(payload as Omit<Client, 'id' | 'createdAt' | 'updatedAt'>);
                enqueueSnackbar('Client created successfully', { variant: 'success' });
            }
            fetchClients(); 
            handleCloseDialog();
        } catch (err: any) {
            console.error("Failed to save client:", err);
            const message = err.response?.data?.error || 'Failed to save client.';
            setFormError(message);
            enqueueSnackbar(message, { variant: 'error' });
        } finally {
            setIsSaving(false);
        }
    };

    const handleDeleteConfirm = async () => {
        if (!selectedClient) return;
        setIsSaving(true); 
        try {
            await deleteAdminClient(selectedClient.id);
            enqueueSnackbar('Client deleted successfully', { variant: 'success' });
            fetchClients(); 
            handleCloseDialog();
        } catch (err: any) {
            console.error("Failed to delete client:", err);
            const message = err.response?.data?.error || 'Failed to delete client.';
            enqueueSnackbar(message, { variant: 'error' });
        } finally {
            setIsSaving(false);
            setIsDeleteDialogOpen(false); 
        }
    };
    
    return (
        <Box sx={{ width: '100%', p: 3 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="h4" gutterBottom>
                    Client Management
                </Typography>
                <Button
                    variant="contained"
                    startIcon={<AddIcon />}
                    onClick={handleAddClick}
                >
                    Add Client
                </Button>
            </Box>

            {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

            <Paper sx={{ height: '70vh', width: '100%' }}>
                {loading ? (
                    <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
                        <CircularProgress />
                    </Box>
                ) : (
                    <DataGrid
                        rows={clients}
                        columns={columns}
                        pageSizeOptions={[10, 25, 50]}
                        initialState={{
                            pagination: {
                              paginationModel: { pageSize: 10 },
                            },
                          }}
                        checkboxSelection={false}
                        disableRowSelectionOnClick
                    />
                )}
            </Paper>

            <Dialog open={isAddEditDialogOpen} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
                <DialogTitle>{selectedClient ? 'Edit Client' : 'Add New Client'}</DialogTitle>
                <DialogContent>
                    {formError && <Alert severity="error" sx={{ mb: 2 }}>{formError}</Alert>}
                    <TextField
                        autoFocus
                        margin="dense"
                        name="name"
                        label="Client Name"
                        type="text"
                        fullWidth
                        variant="outlined"
                        value={clientFormData.name || ''}
                        onChange={handleFormChange}
                        required
                    />
                    <TextField
                        margin="dense"
                        name="description"
                        label="Description"
                        type="text"
                        fullWidth
                        multiline
                        rows={3}
                        variant="outlined"
                        value={clientFormData.description || ''}
                        onChange={handleFormChange}
                    />
                    <TextField
                        margin="dense"
                        name="contactInfo"
                        label="Contact Info"
                        type="text"
                        fullWidth
                        variant="outlined"
                        value={clientFormData.contactInfo || ''}
                        onChange={handleFormChange}
                    />
                    <TextField
                        margin="dense"
                        name="dataRetentionMonths"
                        label="Data Retention (Months)"
                        type="number"
                        fullWidth
                        variant="outlined"
                        value={clientFormData.dataRetentionMonths === null ? '' : clientFormData.dataRetentionMonths}
                        onChange={handleFormChange}
                        helperText="Leave empty to use system default. Enter 0 to keep forever."
                        InputProps={{
                            inputProps: { 
                                min: 0 
                            }
                        }}
                    />
                </DialogContent>
                <DialogActions>
                    <Button onClick={handleCloseDialog} disabled={isSaving}>Cancel</Button>
                    <Button onClick={handleSaveClient} disabled={isSaving} variant="contained">
                        {isSaving ? <CircularProgress size={24} /> : (selectedClient ? 'Save Changes' : 'Create Client')}
                    </Button>
                </DialogActions>
            </Dialog>

            <Dialog
                open={isDeleteDialogOpen}
                onClose={handleCloseDialog}
                aria-labelledby="alert-dialog-title"
                aria-describedby="alert-dialog-description"
            >
                <DialogTitle id="alert-dialog-title">
                    {"Confirm Deletion"}
                </DialogTitle>
                <DialogContent>
                    <DialogContentText id="alert-dialog-description">
                        Are you sure you want to delete the client "{selectedClient?.name}"? 
                        This action cannot be undone. Associated hashlists may be deleted immediately 
                        or orphaned based on retention policies.
                    </DialogContentText>
                </DialogContent>
                <DialogActions>
                    <Button onClick={handleCloseDialog} disabled={isSaving}>Cancel</Button>
                    <Button onClick={handleDeleteConfirm} color="error" autoFocus disabled={isSaving}>
                        {isSaving ? <CircularProgress size={24} /> : 'Delete'}
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}; 