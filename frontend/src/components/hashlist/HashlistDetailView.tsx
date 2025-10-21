import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Chip,
  LinearProgress,
  Button,
  Divider,
  Tooltip,
  IconButton,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle
} from '@mui/material';
import {
  Download as DownloadIcon,
  Delete as DeleteIcon,
  History as HistoryIcon,
  ArrowBack as ArrowBackIcon,
  PlayArrow as PlayArrowIcon
} from '@mui/icons-material';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../../services/api';
import CreateJobDialog from './CreateJobDialog';
import HashlistHashesTable from './HashlistHashesTable';
import { useSnackbar } from 'notistack';
import { AxiosResponse, AxiosError } from 'axios';

interface HashDetail {
  id: string;
  hash_value: string;
  original_hash: string;
  username?: string;
  domain?: string;
  hash_type_id: number;
  is_cracked: boolean;
  password?: string;
  last_updated: string;
  // Frontend friendly aliases
  hash?: string;
  isCracked?: boolean;
  crackedText?: string;
}

export default function HashlistDetailView() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [createJobDialogOpen, setCreateJobDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const queryClient = useQueryClient();
  const { enqueueSnackbar } = useSnackbar();

  const { data: hashlist, isLoading } = useQuery({
    queryKey: ['hashlist', id],
    queryFn: () => api.get(`/api/hashlists/${id}`).then(res => res.data)
  });

  // Delete Mutation
  const deleteMutation = useMutation<AxiosResponse, AxiosError, string>({
    mutationFn: (hashlistId: string) => api.delete(`/api/hashlists/${hashlistId}`),
    onSuccess: () => {
      enqueueSnackbar('Hashlist deleted successfully', { variant: 'success' });
      queryClient.invalidateQueries({ queryKey: ['hashlists'] });
      navigate('/hashlists'); // Redirect to list after deletion
    },
    onError: (error) => {
      const errorMsg = (error.response?.data as any)?.error || error.message || 'Failed to delete hashlist';
      enqueueSnackbar(errorMsg, { variant: 'error' });
      setDeleteDialogOpen(false);
    },
  });

  const handleDeleteClick = () => {
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = () => {
    if (id) {
      deleteMutation.mutate(id);
    }
  };

  const handleDeleteCancel = () => {
    setDeleteDialogOpen(false);
  };

  if (isLoading) return <LinearProgress />;

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ mb: 2 }}>
        <Button
          startIcon={<ArrowBackIcon />}
          onClick={() => navigate('/hashlists')}
          size="small"
        >
          Back to Hashlists
        </Button>
      </Box>
      
      <Paper sx={{ p: 3, mb: 3 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center">
          <Typography variant="h5">{hashlist.name}</Typography>
          <Box display="flex" gap={1}>
            <Button
              variant="contained"
              startIcon={<PlayArrowIcon />}
              onClick={() => setCreateJobDialogOpen(true)}
              disabled={hashlist.status !== 'ready'}
            >
              Create Job
            </Button>
            <Tooltip title="Download">
              <IconButton>
                <DownloadIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title="Delete">
              <IconButton color="error" onClick={handleDeleteClick}>
                <DeleteIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>

        <Typography variant="subtitle1" color="text.secondary" sx={{ mt: 1 }}>
          {hashlist.description || 'No description'}
        </Typography>

        <Box display="flex" gap={2} sx={{ mt: 3 }}>
          <Typography>
            Status: <Chip 
              label={hashlist.status} 
              color={
                hashlist.status === 'ready' ? 'success' :
                hashlist.status === 'error' ? 'error' : 'primary'
              }
            />
          </Typography>
          <Typography>
            Hash Type: {hashlist.hashTypeName}
          </Typography>
          <Typography>
            Created: {new Date(hashlist.createdAt).toLocaleString()}
          </Typography>
        </Box>

        <Box sx={{ mt: 3 }}>
          <Typography variant="subtitle2">
            Crack Progress ({hashlist.cracked_hashes || 0} of {hashlist.total_hashes || 0})
          </Typography>
          <Box display="flex" alignItems="center" gap={2}>
            <Box width="100%">
              <LinearProgress
                variant="determinate"
                value={hashlist.total_hashes > 0
                  ? ((hashlist.cracked_hashes || 0) / hashlist.total_hashes) * 100
                  : 0
                }
              />
            </Box>
            <Typography>
              {hashlist.total_hashes > 0
                ? Math.round(((hashlist.cracked_hashes || 0) / hashlist.total_hashes) * 100)
                : 0
              }%
            </Typography>
          </Box>
        </Box>
      </Paper>

      {hashlist && (
        <HashlistHashesTable
          hashlistId={id!}
          hashlistName={hashlist.name}
          totalHashes={hashlist.total_hashes || 0}
          crackedHashes={hashlist.cracked_hashes || 0}
        />
      )}

      <Paper sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>
          <HistoryIcon sx={{ verticalAlign: 'middle', mr: 1 }} />
          History
        </Typography>
        <Divider sx={{ mb: 2 }} />
        <Typography color="text.secondary">
          History log will appear here
        </Typography>
      </Paper>

      {hashlist && (
        <CreateJobDialog
          open={createJobDialogOpen}
          onClose={() => setCreateJobDialogOpen(false)}
          hashlistId={parseInt(id!)}
          hashlistName={hashlist.name}
          hashTypeId={hashlist.hashTypeID || hashlist.hash_type_id}
        />
      )}

      <Dialog
        open={deleteDialogOpen}
        onClose={handleDeleteCancel}
        aria-labelledby="alert-dialog-title"
        aria-describedby="alert-dialog-description"
      >
        <DialogTitle id="alert-dialog-title">
          Confirm Deletion
        </DialogTitle>
        <DialogContent>
          <DialogContentText id="alert-dialog-description">
            Are you sure you want to delete the hashlist "{hashlist?.name || ''}"?
            This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleDeleteCancel} color="primary">
            Cancel
          </Button>
          <Button onClick={handleDeleteConfirm} color="error" autoFocus disabled={deleteMutation.isPending}>
            {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}