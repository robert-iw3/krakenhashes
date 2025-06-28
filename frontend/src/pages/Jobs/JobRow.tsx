import React, { useState } from 'react';
import {
  TableRow,
  TableCell,
  Typography,
  Chip,
  IconButton,
  Box,
  Link,
  Tooltip,
  Alert,
  Collapse,
} from '@mui/material';
import {
  Delete as DeleteIcon,
  Speed as SpeedIcon,
  People as PeopleIcon,
  Refresh as RefreshIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Error as ErrorIcon,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import EditableCell from './EditableCell';
import DeleteConfirm from './DeleteConfirm';
import { JobSummary } from '../../types/jobs';
import { api } from '../../services/api';
import { formatters } from '../../utils/formatters';

interface JobRowProps {
  job: JobSummary;
  onJobUpdated?: () => void;
}

const JobRow: React.FC<JobRowProps> = ({ job, onJobUpdated }) => {
  const navigate = useNavigate();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isRetrying, setIsRetrying] = useState(false);
  const [showError, setShowError] = useState(false);

  const handleJobNameClick = () => {
    navigate(`/jobs/${job.id}`);
  };

  const handleUpdatePriority = async (newPriority: number) => {
    try {
      await api.patch(`/api/jobs/${job.id}`, { priority: newPriority });
      onJobUpdated?.();
    } catch (error) {
      console.error('Failed to update job priority:', error);
      throw error; // Re-throw to show error in EditableCell
    }
  };

  const handleUpdateMaxAgents = async (newMaxAgents: number) => {
    try {
      await api.patch(`/api/jobs/${job.id}`, { max_agents: newMaxAgents });
      onJobUpdated?.();
    } catch (error) {
      console.error('Failed to update max agents:', error);
      throw error; // Re-throw to show error in EditableCell
    }
  };

  const handleDeleteJob = async () => {
    setIsDeleting(true);
    try {
      await api.delete(`/api/jobs/${job.id}`);
      setDeleteDialogOpen(false);
      onJobUpdated?.();
    } catch (error) {
      console.error('Failed to delete job:', error);
    } finally {
      setIsDeleting(false);
    }
  };

  const handleRetryJob = async () => {
    setIsRetrying(true);
    try {
      await api.post(`/api/jobs/${job.id}/retry`);
      onJobUpdated?.();
    } catch (error) {
      console.error('Failed to retry job:', error);
    } finally {
      setIsRetrying(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'running':
        return 'success';
      case 'pending':
        return 'warning';
      case 'completed':
        return 'info';
      case 'failed':
        return 'error';
      case 'paused':
        return 'default';
      case 'cancelled':
        return 'default';
      case 'interrupted':
        return 'warning';
      default:
        return 'default';
    }
  };

  const canRetry = ['failed', 'interrupted', 'cancelled'].includes(job.status.toLowerCase());
  const hasError = job.error_message && (job.status === 'failed' || job.status === 'interrupted');

  return (
    <>
      <TableRow hover>
        {/* Job Name with expand button for errors */}
        <TableCell>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {hasError && (
              <IconButton
                size="small"
                onClick={() => setShowError(!showError)}
                sx={{ p: 0.5 }}
              >
                {showError ? <ExpandLessIcon /> : <ExpandMoreIcon />}
              </IconButton>
            )}
            <Link
              component="button"
              variant="body2"
              onClick={handleJobNameClick}
              sx={{ textAlign: 'left', fontWeight: 'medium' }}
            >
              {job.name}
            </Link>
            <Chip
              label={job.status}
              color={getStatusColor(job.status)}
              size="small"
              variant="outlined"
              icon={hasError ? <ErrorIcon /> : undefined}
            />
          </Box>
        </TableCell>

        {/* Hashlist */}
        <TableCell>
          <Typography variant="body2">{job.hashlist_name}</Typography>
        </TableCell>

        {/* Dispatched / Searched */}
        <TableCell align="center">
          <Typography variant="body2">
            {job.dispatched_percent.toFixed(1)}% / {job.searched_percent.toFixed(1)}%
          </Typography>
        </TableCell>

        {/* Cracked Count */}
        <TableCell align="center">
          <Typography variant="body2" sx={{ fontWeight: 'medium' }}>
            {job.cracked_count.toLocaleString()}
          </Typography>
        </TableCell>

        {/* Agents */}
        <TableCell align="center">
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 0.5 }}>
            <PeopleIcon fontSize="small" color="action" />
            <Typography variant="body2">
              {job.agent_count}
            </Typography>
            {job.total_speed > 0 && (
              <Tooltip title="Combined hash rate">
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, ml: 1 }}>
                  <SpeedIcon fontSize="small" color="action" />
                  <Typography variant="body2" color="text.secondary">
                    {formatters.formatHashRate(job.total_speed)}
                  </Typography>
                </Box>
              </Tooltip>
            )}
          </Box>
        </TableCell>

        {/* Priority */}
        <TableCell align="center">
          {(job.status === 'completed' || job.status === 'cancelled') ? (
            <Typography variant="body2">{job.priority}</Typography>
          ) : (
            <EditableCell
              value={job.priority}
              onSave={handleUpdatePriority}
              type="number"
              min={1}
              max={10}
              validation={(value) => {
                const num = Number(value);
                if (isNaN(num) || num < 1 || num > 10) {
                  return 'Priority must be between 1 and 10';
                }
                return null;
              }}
            />
          )}
        </TableCell>

        {/* Max Agents */}
        <TableCell align="center">
          {(job.status === 'completed' || job.status === 'cancelled') ? (
            <Typography variant="body2">{job.max_agents}</Typography>
          ) : (
            <EditableCell
              value={job.max_agents}
              onSave={handleUpdateMaxAgents}
              type="number"
              min={1}
              max={100}
              validation={(value) => {
                const num = Number(value);
                if (isNaN(num) || num < 1 || num > 100) {
                  return 'Max agents must be between 1 and 100';
                }
                return null;
              }}
            />
          )}
        </TableCell>

        {/* Actions */}
        <TableCell align="center">
          <Box sx={{ display: 'flex', gap: 0.5, justifyContent: 'center' }}>
            {canRetry && (
              <Tooltip title="Retry job">
                <IconButton
                  size="small"
                  color="primary"
                  onClick={handleRetryJob}
                  disabled={isRetrying}
                >
                  <RefreshIcon />
                </IconButton>
              </Tooltip>
            )}
            <Tooltip title="Delete job">
              <IconButton
                size="small"
                color="error"
                onClick={() => setDeleteDialogOpen(true)}
              >
                <DeleteIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </TableCell>
      </TableRow>

      {/* Error message row */}
      {hasError && (
        <TableRow>
          <TableCell colSpan={8} sx={{ py: 0 }}>
            <Collapse in={showError} timeout="auto" unmountOnExit>
              <Alert severity="error" sx={{ m: 2 }}>
                <Typography variant="body2">
                  <strong>Error:</strong> {job.error_message}
                </Typography>
                {job.status === 'interrupted' && (
                  <Typography variant="caption" display="block" sx={{ mt: 1 }}>
                    This job was interrupted (likely due to server restart or agent disconnection).
                    You can retry it to resume from where it left off.
                  </Typography>
                )}
              </Alert>
            </Collapse>
          </TableCell>
        </TableRow>
      )}

      {/* Delete Confirmation Dialog */}
      <DeleteConfirm
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        onConfirm={handleDeleteJob}
        isLoading={isDeleting}
        title="Delete Job"
        message={`Are you sure you want to delete the job "${job.name}"? This action cannot be undone.`}
      />
    </>
  );
};

export default JobRow;