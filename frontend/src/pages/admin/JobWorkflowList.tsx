import React, { useState, useEffect } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { 
  Box, 
  Typography, 
  Button, 
  Table, 
  TableBody, 
  TableCell, 
  TableContainer, 
  TableHead, 
  TableRow, 
  Paper, 
  IconButton, 
  CircularProgress, 
  Alert,
  Tooltip
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import { listJobWorkflows, deleteJobWorkflow } from '../../services/api';
import { JobWorkflow } from '../../types/adminJobs';
import { useConfirm } from '../../hooks';

const JobWorkflowListPage: React.FC = () => {
  // State
  const [workflows, setWorkflows] = useState<JobWorkflow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleteInProgress, setDeleteInProgress] = useState(false);
  
  // Dialog hooks
  const { ConfirmDialog, showConfirm } = useConfirm();

  // Load workflows on component mount
  useEffect(() => {
    const fetchWorkflows = async () => {
      try {
        setLoading(true);
        setError(null);
        
        const data = await listJobWorkflows();
        setWorkflows(data);
      } catch (err) {
        console.error('Error fetching job workflows:', err);
        setError('Failed to load workflows. Please try again.');
      } finally {
        setLoading(false);
      }
    };

    fetchWorkflows();
  }, []);

  // Handle workflow deletion
  const handleDelete = async (id: string, name: string) => {
    const confirmed = await showConfirm(
      'Delete Job Workflow',
      `Are you sure you want to delete the workflow "${name}"? This action cannot be undone.`
    );
    
    if (confirmed) {
      try {
        setDeleteInProgress(true);
        await deleteJobWorkflow(id);
        
        // Remove the deleted workflow from state
        setWorkflows(prev => prev.filter(wf => wf.id !== id));
      } catch (err) {
        console.error('Error deleting workflow:', err);
        setError('Failed to delete workflow. Please try again.');
      } finally {
        setDeleteInProgress(false);
      }
    }
  };

  return (
    <Box>
      <ConfirmDialog />
      
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" gutterBottom>
          Job Workflows
        </Typography>
        
        <Button 
          component={RouterLink} 
          to="/admin/job-workflows/new" 
          variant="contained" 
          color="primary" 
          startIcon={<AddIcon />}
          disabled={loading || deleteInProgress}
        >
          Create Workflow
        </Button>
      </Box>
      
      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}
      
      {loading ? (
        <Box display="flex" justifyContent="center" p={3}>
          <CircularProgress />
        </Box>
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Job Count</TableCell>
                <TableCell>Created</TableCell>
                <TableCell>Last Updated</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {workflows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} align="center">
                    <Typography variant="body1" py={2}>
                      No job workflows found. Create your first workflow to get started.
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                workflows.map((workflow) => (
                  <TableRow key={workflow.id}>
                    <TableCell>
                      <RouterLink to={`/admin/job-workflows/${workflow.id}/edit`} style={{ textDecoration: 'none', color: 'inherit' }}>
                        {workflow.name}
                      </RouterLink>
                    </TableCell>
                    <TableCell>{workflow.steps?.length || 0}</TableCell>
                    <TableCell>{new Date(workflow.created_at).toLocaleString()}</TableCell>
                    <TableCell>{new Date(workflow.updated_at).toLocaleString()}</TableCell>
                    <TableCell align="right">
                      <Tooltip title="Edit">
                        <IconButton
                          component={RouterLink}
                          to={`/admin/job-workflows/${workflow.id}/edit`}
                          disabled={deleteInProgress}
                        >
                          <EditIcon />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Delete">
                        <IconButton
                          onClick={() => handleDelete(workflow.id, workflow.name)}
                          disabled={deleteInProgress}
                        >
                          <DeleteIcon />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  );
};

export default JobWorkflowListPage; 