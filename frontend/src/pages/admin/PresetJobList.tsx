import React from 'react';
import { Box, Typography, Button, CircularProgress, Alert, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, IconButton } from '@mui/material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link as RouterLink } from 'react-router-dom';
import { Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon } from '@mui/icons-material';
import { useSnackbar } from 'notistack';

// Import types and API functions from existing services
// Ensure AttackMode enum is imported if needed for display formatting
import { PresetJob, AttackMode } from '../../types/adminJobs'; 
import { listPresetJobs, deletePresetJob } from '../../services/api';

// Helper function to format AttackMode enum for display
const formatAttackMode = (mode: AttackMode): string => {
  switch (mode) {
    case AttackMode.Straight: return 'Straight';
    case AttackMode.Combination: return 'Combination';
    case AttackMode.BruteForce: return 'Brute-Force';
    case AttackMode.HybridWordlistMask: return 'Hybrid (Wordlist + Mask)';
    case AttackMode.HybridMaskWordlist: return 'Hybrid (Mask + Wordlist)';
    case AttackMode.Association: return 'Association';
    default: return `Unknown (${mode})`;
  }
};

const PresetJobListPage: React.FC = () => {
  const queryClient = useQueryClient();
  const { enqueueSnackbar } = useSnackbar();

  // Correct useQuery signature: options object only
  const { data: presetJobs, isLoading, error } = useQuery<PresetJob[], Error>({
    queryKey: ['presetJobs'],
    queryFn: listPresetJobs,
  });

  // Correct useMutation signature: options object with mutationFn
  const deleteMutation = useMutation<void, Error, string>({
    mutationFn: deletePresetJob, // Specify mutation function here
    onSuccess: () => {
      enqueueSnackbar('Preset job deleted successfully', { variant: 'success' });
      queryClient.invalidateQueries({ queryKey: ['presetJobs'] });
    },
    onError: (err: Error) => {
      enqueueSnackbar(`Failed to delete preset job: ${err.message}`, { variant: 'error' });
    },
  });

  const handleDelete = (id: string) => {
    if (window.confirm('Are you sure you want to delete this preset job?')) {
      deleteMutation.mutate(id);
    }
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">
          Preset Jobs Management
        </Typography>
        <Button
          variant="contained"
          component={RouterLink}
          to="/admin/preset-jobs/new"
          startIcon={<AddIcon />}
          disabled={deleteMutation.isPending} 
        >
          Create New Preset Job
        </Button>
      </Box>

      {(isLoading || deleteMutation.isPending) && <CircularProgress />} 
      {error && <Alert severity="error">Error fetching preset jobs: {error.message}</Alert>}
      {deleteMutation.error && <Alert severity="error">Error deleting preset job: {deleteMutation.error.message}</Alert>} 

      {!isLoading && !error && presetJobs && (
        <TableContainer component={Paper}>
          <Table sx={{ minWidth: 650 }} aria-label="preset jobs table">
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Attack Mode</TableCell>
                <TableCell>Priority</TableCell>
                <TableCell>Binary Version</TableCell>
                <TableCell>Wordlists</TableCell>
                <TableCell>Rules</TableCell>
                <TableCell>Created At</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {Array.isArray(presetJobs) && presetJobs.length === 0 && ( 
                <TableRow>
                  <TableCell colSpan={8} align="center">
                    No preset jobs found.
                  </TableCell>
                </TableRow>
              )}
              {Array.isArray(presetJobs) && presetJobs?.map((job: PresetJob) => ( 
                <TableRow
                  key={job.id}
                  sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
                >
                  <TableCell component="th" scope="row">
                    {job.name}
                  </TableCell>
                  <TableCell>{formatAttackMode(job.attack_mode)}</TableCell>
                  <TableCell>{job.priority}</TableCell>
                  <TableCell>{job.binary_version_name || job.binary_version_id}</TableCell>
                  <TableCell>{job.wordlist_ids?.length || 0}</TableCell>
                  <TableCell>{job.rule_ids?.length || 0}</TableCell>
                  <TableCell>{new Date(job.created_at).toLocaleString()}</TableCell>
                  <TableCell align="right">
                    <IconButton 
                      component={RouterLink} 
                      to={`/admin/preset-jobs/${job.id}/edit`} 
                      aria-label="edit"
                      disabled={deleteMutation.isPending}
                    >
                      <EditIcon />
                    </IconButton>
                    <IconButton 
                      onClick={() => handleDelete(job.id)} 
                      aria-label="delete"
                      disabled={deleteMutation.isPending}
                    >
                      <DeleteIcon />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  );
};

export default PresetJobListPage; 