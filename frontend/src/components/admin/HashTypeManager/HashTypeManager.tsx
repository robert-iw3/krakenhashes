import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  Alert,
  CircularProgress,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { useSnackbar } from 'notistack';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import HashTypeTable from './HashTypeTable';
import HashTypeDialog from './HashTypeDialog';
import { HashType, HashTypeCreateRequest, HashTypeUpdateRequest } from '../../../types/hashType';
import {
  getHashTypes,
  createHashType,
  updateHashType,
  deleteHashType,
} from '../../../services/hashType';

const HashTypeManager: React.FC = () => {
  const { enqueueSnackbar } = useSnackbar();
  const queryClient = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedHashType, setSelectedHashType] = useState<HashType | null>(null);
  const [hashTypeToDelete, setHashTypeToDelete] = useState<HashType | null>(null);

  // Fetch hash types
  const { data: hashTypes = [], isLoading, error } = useQuery<HashType[], Error>({
    queryKey: ['hashTypes'],
    queryFn: () => getHashTypes(false),
  });

  // Create mutation
  const createMutation = useMutation<HashType, Error, HashTypeCreateRequest>({
    mutationFn: createHashType,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['hashTypes'] });
      enqueueSnackbar('Hash type created successfully', { variant: 'success' });
      setDialogOpen(false);
      setSelectedHashType(null);
    },
    onError: (error: any) => {
      const message = error.response?.data?.error || 'Failed to create hash type';
      enqueueSnackbar(message, { variant: 'error' });
    },
  });

  // Update mutation
  const updateMutation = useMutation<HashType, Error, { id: number; data: HashTypeUpdateRequest }>({
    mutationFn: ({ id, data }) => updateHashType(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['hashTypes'] });
      enqueueSnackbar('Hash type updated successfully', { variant: 'success' });
      setDialogOpen(false);
      setSelectedHashType(null);
    },
    onError: (error: any) => {
      const message = error.response?.data?.error || 'Failed to update hash type';
      enqueueSnackbar(message, { variant: 'error' });
    },
  });

  // Delete mutation
  const deleteMutation = useMutation<void, Error, number>({
    mutationFn: deleteHashType,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['hashTypes'] });
      enqueueSnackbar('Hash type deleted successfully', { variant: 'success' });
      setDeleteDialogOpen(false);
      setHashTypeToDelete(null);
    },
    onError: (error: any) => {
      const message = error.response?.data?.error || 'Failed to delete hash type';
      if (message.includes('still referenced')) {
        enqueueSnackbar('Cannot delete: This hash type is in use by existing hashlists', { variant: 'error' });
      } else {
        enqueueSnackbar(message, { variant: 'error' });
      }
    },
  });

  const handleAdd = () => {
    setSelectedHashType(null);
    setDialogOpen(true);
  };

  const handleEdit = (hashType: HashType) => {
    setSelectedHashType(hashType);
    setDialogOpen(true);
  };

  const handleDelete = (hashType: HashType) => {
    setHashTypeToDelete(hashType);
    setDeleteDialogOpen(true);
  };

  const handleSave = async (data: HashTypeCreateRequest | HashTypeUpdateRequest, id?: number) => {
    if (id !== undefined) {
      await updateMutation.mutateAsync({ id, data: data as HashTypeUpdateRequest });
    } else {
      await createMutation.mutateAsync(data as HashTypeCreateRequest);
    }
  };

  const confirmDelete = () => {
    if (hashTypeToDelete) {
      deleteMutation.mutate(hashTypeToDelete.id);
    }
  };

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">
          Failed to load hash types. Please try refreshing the page.
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3 }}>
        <Box>
          <Typography variant="h4" component="h1" gutterBottom>
            Hash Type Management
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Manage supported hash types for password cracking operations
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleAdd}
        >
          Add Hash Type
        </Button>
      </Box>

      {isLoading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
          <CircularProgress />
        </Box>
      ) : (
        <HashTypeTable
          hashTypes={hashTypes}
          onEdit={handleEdit}
          onDelete={handleDelete}
          loading={isLoading}
        />
      )}

      <HashTypeDialog
        open={dialogOpen}
        onClose={() => {
          setDialogOpen(false);
          setSelectedHashType(null);
        }}
        onSave={handleSave}
        hashType={selectedHashType}
        existingIds={hashTypes.map(ht => ht.id)}
      />

      <Dialog
        open={deleteDialogOpen}
        onClose={() => {
          setDeleteDialogOpen(false);
          setHashTypeToDelete(null);
        }}
      >
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete the hash type "{hashTypeToDelete?.name}" (ID: {hashTypeToDelete?.id})?
            {hashTypeToDelete?.is_enabled && (
              <Alert severity="warning" sx={{ mt: 2 }}>
                This hash type is currently enabled. Deleting it may affect existing functionality.
              </Alert>
            )}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => {
              setDeleteDialogOpen(false);
              setHashTypeToDelete(null);
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={confirmDelete}
            color="error"
            variant="contained"
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default HashTypeManager; 