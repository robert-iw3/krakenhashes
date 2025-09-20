import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Typography,
  Paper,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  Chip,
  CircularProgress,
  Alert,
  Skeleton,
  TextField,
  IconButton,
  Link
} from '@mui/material';
import {
  ArrowBack,
  Edit as EditIcon,
  Save as SaveIcon,
  Cancel as CancelIcon,
  Refresh as RefreshIcon,
  Replay as ReplayIcon
} from '@mui/icons-material';
import { getJobDetails, api } from '../../services/api';
import { JobDetailsResponse, JobTask } from '../../types/jobs';
import JobProgressBar from '../../components/JobProgressBar';

const JobDetails: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  
  const [jobData, setJobData] = useState<JobDetailsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(true);
  
  // Edit states
  const [editingPriority, setEditingPriority] = useState(false);
  const [editingMaxAgents, setEditingMaxAgents] = useState(false);
  const [tempPriority, setTempPriority] = useState<number>(0);
  const [tempMaxAgents, setTempMaxAgents] = useState<number>(1);
  const [saving, setSaving] = useState(false);
  
  // Completed tasks pagination state
  const [completedTasksPage, setCompletedTasksPage] = useState(0);
  const [completedTasksPageSize, setCompletedTasksPageSize] = useState(25);
  
  // Refs to track current state for polling
  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const currentStatusRef = useRef<string>('');
  const isEditingRef = useRef<boolean>(false);

  // Update editing ref when editing state changes
  useEffect(() => {
    isEditingRef.current = editingPriority || editingMaxAgents;
  }, [editingPriority, editingMaxAgents]);
  
  // Update status ref when job data changes
  useEffect(() => {
    if (jobData) {
      currentStatusRef.current = jobData.status;
    }
  }, [jobData?.status]);
  
  // Fetch job details
  const fetchJobDetails = useCallback(async () => {
    if (!id) return;
    
    // Don't fetch if user is editing
    if (isEditingRef.current) {
      return;
    }
    
    try {
      const data = await getJobDetails(id);
      setJobData(data);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch job details:', err);
      setError('Failed to load job details');
    } finally {
      setLoading(false);
    }
  }, [id]);

  // Initial fetch
  useEffect(() => {
    fetchJobDetails();
  }, [fetchJobDetails]);
  
  // Setup and manage polling
  useEffect(() => {
    // Clear any existing interval
    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }
    
    // Determine if we should poll
    const shouldPoll = jobData && 
                      ['pending', 'running', 'paused'].includes(jobData.status) &&
                      autoRefreshEnabled &&
                      !isEditingRef.current;
    
    if (shouldPoll) {
      // Set up polling interval
      const interval = setInterval(() => {
        // Check conditions again inside the interval
        const activeStatuses = ['pending', 'running', 'paused'];
        if (activeStatuses.includes(currentStatusRef.current) && 
            !isEditingRef.current &&
            autoRefreshEnabled) {
          fetchJobDetails();
        }
      }, 5000);
      
      pollingIntervalRef.current = interval;
    }
    
    // Cleanup on unmount or when dependencies change
    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
        pollingIntervalRef.current = null;
      }
    };
  }, [jobData?.status, autoRefreshEnabled, fetchJobDetails]);

  // Handle priority edit
  const handleEditPriority = () => {
    setTempPriority(jobData?.priority || 0);
    setEditingPriority(true);
    setAutoRefreshEnabled(false); // Pause auto-refresh during edit
  };

  const handleSavePriority = async () => {
    if (!id) return;
    
    setSaving(true);
    try {
      await api.patch(`/api/jobs/${id}`, { priority: tempPriority });
      await fetchJobDetails();
      setEditingPriority(false);
      setAutoRefreshEnabled(true); // Resume auto-refresh after save
    } catch (err) {
      console.error('Failed to update priority:', err);
      setError('Failed to update priority');
    } finally {
      setSaving(false);
    }
  };
  
  const handleCancelPriority = () => {
    setEditingPriority(false);
    setAutoRefreshEnabled(true); // Resume auto-refresh after cancel
  };

  // Handle max agents edit
  const handleEditMaxAgents = () => {
    setTempMaxAgents(jobData?.max_agents || 1);
    setEditingMaxAgents(true);
    setAutoRefreshEnabled(false); // Pause auto-refresh during edit
  };

  const handleSaveMaxAgents = async () => {
    if (!id) return;
    
    setSaving(true);
    try {
      await api.patch(`/api/jobs/${id}`, { max_agents: tempMaxAgents });
      await fetchJobDetails();
      setEditingMaxAgents(false);
      setAutoRefreshEnabled(true); // Resume auto-refresh after save
    } catch (err) {
      console.error('Failed to update max agents:', err);
      setError('Failed to update max agents');
    } finally {
      setSaving(false);
    }
  };
  
  const handleCancelMaxAgents = () => {
    setEditingMaxAgents(false);
    setAutoRefreshEnabled(true); // Resume auto-refresh after cancel
  };

  // Handle retry task
  const handleRetryTask = async (taskId: string) => {
    if (!id) return;

    try {
      await api.post(`/api/jobs/${id}/tasks/${taskId}/retry`);
      await fetchJobDetails();
    } catch (err) {
      console.error('Failed to retry task:', err);
      setError('Failed to retry task');
    }
  };

  // Format helpers
  const formatDate = (dateString?: string) => {
    if (!dateString) return 'N/A';
    return new Date(dateString).toLocaleString();
  };

  const formatKeyspace = (value?: number): string => {
    if (!value) return 'N/A';
    if (value >= 1e12) return `${(value / 1e12).toFixed(2)}T`;
    if (value >= 1e9) return `${(value / 1e9).toFixed(2)}B`;
    if (value >= 1e6) return `${(value / 1e6).toFixed(2)}M`;
    if (value >= 1e3) return `${(value / 1e3).toFixed(2)}K`;
    return value.toString();
  };

  const formatSpeed = (speed?: number): string => {
    if (!speed) return 'N/A';
    if (speed >= 1e12) return `${(speed / 1e12).toFixed(2)} TH/s`;
    if (speed >= 1e9) return `${(speed / 1e9).toFixed(2)} GH/s`;
    if (speed >= 1e6) return `${(speed / 1e6).toFixed(2)} MH/s`;
    if (speed >= 1e3) return `${(speed / 1e3).toFixed(2)} KH/s`;
    return `${speed} H/s`;
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'running': return 'success';
      case 'pending': return 'warning';
      case 'reconnect_pending': return 'warning';
      case 'completed': return 'info';
      case 'failed': return 'error';
      case 'cancelled': return 'default';
      default: return 'default';
    }
  };

  const getAttackModeName = (mode?: number): string => {
    const modes: Record<number, string> = {
      0: 'Dictionary',
      3: 'Brute-force',
      6: 'Hybrid Wordlist + Mask',
      7: 'Hybrid Mask + Wordlist',
    };
    return mode !== undefined ? modes[mode] || `Mode ${mode}` : 'N/A';
  };

  if (loading) {
    return (
      <Box sx={{ p: 3 }}>
        <Skeleton variant="rectangular" height={60} sx={{ mb: 3 }} />
        <Skeleton variant="rectangular" height={400} sx={{ mb: 3 }} />
        <Skeleton variant="rectangular" height={200} />
      </Box>
    );
  }

  if (error && !jobData) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
        <Button startIcon={<ArrowBack />} onClick={() => navigate(-1)}>
          Back
        </Button>
      </Box>
    );
  }

  if (!jobData) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">Job not found</Alert>
        <Button startIcon={<ArrowBack />} onClick={() => navigate(-1)} sx={{ mt: 2 }}>
          Back
        </Button>
      </Box>
    );
  }

  // Get active tasks for agent table
  const activeTasks = jobData.tasks.filter(task => 
    ['running', 'pending', 'reconnect_pending'].includes(task.status)
  );

  // Get failed tasks
  const failedTasks = jobData.tasks.filter(task => task.status === 'failed');

  // Get completed tasks and sort by completion time (most recent first)
  const completedTasks = jobData.tasks
    .filter(task => task.status === 'completed')
    .sort((a, b) => {
      if (!a.completed_at || !b.completed_at) return 0;
      return new Date(b.completed_at).getTime() - new Date(a.completed_at).getTime();
    });

  // Paginate completed tasks
  const paginatedCompletedTasks = completedTasks.slice(
    completedTasksPage * completedTasksPageSize,
    (completedTasksPage + 1) * completedTasksPageSize
  );

  const totalKeyspace = jobData.effective_keyspace || jobData.total_keyspace || 0;

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Button
            startIcon={<ArrowBack />}
            onClick={() => navigate(-1)}
          >
            Back
          </Button>
          <Typography variant="h4" component="h1">
            Job Details
          </Typography>
          <Chip 
            label={jobData.status} 
            color={getStatusColor(jobData.status) as any}
            size="small"
          />
          {['pending', 'running', 'paused'].includes(jobData.status) && (
            <Chip
              label={autoRefreshEnabled && !isEditingRef.current ? 'Auto-refresh: ON' : 'Auto-refresh: PAUSED'}
              color={autoRefreshEnabled && !isEditingRef.current ? 'success' : 'warning'}
              size="small"
              variant="outlined"
            />
          )}
        </Box>
        <IconButton onClick={fetchJobDetails} disabled={loading} title="Refresh now">
          <RefreshIcon />
        </IconButton>
      </Box>

      {/* Error Alert */}
      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Job Information Table */}
      <Paper sx={{ mb: 3 }}>
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Typography variant="h6">Job Information</Typography>
        </Box>
        <TableContainer>
          <Table>
            <TableBody>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold', width: '30%' }}>ID</TableCell>
                <TableCell>{jobData.id}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Name</TableCell>
                <TableCell>{jobData.name}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Status</TableCell>
                <TableCell>
                  <Chip 
                    label={jobData.status} 
                    color={getStatusColor(jobData.status) as any}
                    size="small"
                  />
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Priority</TableCell>
                <TableCell>
                  {editingPriority ? (
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <TextField
                        type="number"
                        value={tempPriority}
                        onChange={(e) => setTempPriority(parseInt(e.target.value) || 0)}
                        inputProps={{ min: 0, max: 10 }}
                        size="small"
                        sx={{ width: 100 }}
                        disabled={saving}
                      />
                      <IconButton onClick={handleSavePriority} disabled={saving} size="small" title="Save">
                        <SaveIcon />
                      </IconButton>
                      <IconButton onClick={handleCancelPriority} disabled={saving} size="small" title="Cancel">
                        <CancelIcon />
                      </IconButton>
                    </Box>
                  ) : (
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      {jobData.priority}
                      <IconButton onClick={handleEditPriority} size="small">
                        <EditIcon />
                      </IconButton>
                    </Box>
                  )}
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Max Agents</TableCell>
                <TableCell>
                  {editingMaxAgents ? (
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <TextField
                        type="number"
                        value={tempMaxAgents}
                        onChange={(e) => setTempMaxAgents(parseInt(e.target.value) || 1)}
                        inputProps={{ min: 1 }}
                        size="small"
                        sx={{ width: 100 }}
                        disabled={saving}
                      />
                      <IconButton onClick={handleSaveMaxAgents} disabled={saving} size="small" title="Save">
                        <SaveIcon />
                      </IconButton>
                      <IconButton onClick={handleCancelMaxAgents} disabled={saving} size="small" title="Cancel">
                        <CancelIcon />
                      </IconButton>
                    </Box>
                  ) : (
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      {jobData.max_agents}
                      <IconButton onClick={handleEditMaxAgents} size="small">
                        <EditIcon />
                      </IconButton>
                    </Box>
                  )}
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Hashlist</TableCell>
                <TableCell>{jobData.hashlist_name} (ID: {jobData.hashlist_id})</TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Attack Mode</TableCell>
                <TableCell>{getAttackModeName(jobData.attack_mode)}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Keyspace</TableCell>
                <TableCell>{formatKeyspace(jobData.base_keyspace)}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Effective Keyspace</TableCell>
                <TableCell>
                  {formatKeyspace(jobData.effective_keyspace)}
                  {jobData.multiplication_factor && jobData.multiplication_factor > 1 && (
                    <Chip 
                      label={`×${jobData.multiplication_factor}${jobData.uses_rule_splitting ? ' (rules)' : ''}`} 
                      size="small" 
                      color="error" 
                      variant="filled"
                      sx={{ ml: 1 }}
                    />
                  )}
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Processed Keyspace</TableCell>
                <TableCell>{formatKeyspace(jobData.processed_keyspace)}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Dispatched Keyspace</TableCell>
                <TableCell>{formatKeyspace(jobData.dispatched_keyspace)}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Progress</TableCell>
                <TableCell>{jobData.overall_progress_percent?.toFixed(2) || 0}%</TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Cracks Found</TableCell>
                <TableCell>
                  {jobData.cracked_count > 0 ? (
                    <Link
                      component="button"
                      variant="body2"
                      onClick={() => navigate(`/pot/hashlist/${jobData.hashlist_id}`)}
                      sx={{ fontWeight: 'medium' }}
                    >
                      {jobData.cracked_count}
                    </Link>
                  ) : (
                    jobData.cracked_count
                  )}
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Created At</TableCell>
                <TableCell>{formatDate(jobData.created_at)}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Started At</TableCell>
                <TableCell>{formatDate(jobData.started_at)}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Completed At</TableCell>
                <TableCell>{formatDate(jobData.completed_at)}</TableCell>
              </TableRow>
              {jobData.error_message && (
                <TableRow>
                  <TableCell sx={{ fontWeight: 'bold' }}>Error</TableCell>
                  <TableCell>
                    <Alert severity="error" sx={{ py: 0.5 }}>
                      {jobData.error_message}
                    </Alert>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      {/* Visual Progress Tracking */}
      <Paper sx={{ p: 3, mb: 3 }}>
        <Typography variant="h6" sx={{ mb: 2 }}>
          Task Progress Visualization
        </Typography>
        <JobProgressBar 
          tasks={jobData.tasks} 
          totalKeyspace={totalKeyspace}
          height={50}
        />
      </Paper>

      {/* Agent Performance Table */}
      <Paper>
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Typography variant="h6">
            Active Tasks ({activeTasks.length} running)
          </Typography>
        </Box>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Agent ID</TableCell>
                <TableCell>Task ID</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Keyspace Range</TableCell>
                <TableCell>Progress</TableCell>
                <TableCell>Speed</TableCell>
                <TableCell>Cracks</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {activeTasks.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} align="center">
                    <Typography color="text.secondary" sx={{ py: 2 }}>
                      No active tasks
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                activeTasks.map((task) => (
                  <TableRow key={task.id}>
                    <TableCell>{task.agent_id || 'Unassigned'}</TableCell>
                    <TableCell>{task.id.slice(0, 8)}...</TableCell>
                    <TableCell>
                      <Chip 
                        label={task.status} 
                        color={getStatusColor(task.status) as any}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      {formatKeyspace(task.effective_keyspace_start || task.keyspace_start)} - {formatKeyspace(task.effective_keyspace_end || task.keyspace_end)}
                    </TableCell>
                    <TableCell>{task.progress_percent?.toFixed(2) || 0}%</TableCell>
                    <TableCell>{formatSpeed(task.benchmark_speed)}</TableCell>
                    <TableCell>
                      {task.crack_count > 0 ? (
                        <Link
                          component="button"
                          variant="body2"
                          onClick={() => navigate(`/pot/hashlist/${jobData.hashlist_id}`)}
                          sx={{ fontWeight: 'medium' }}
                        >
                          {task.crack_count}
                        </Link>
                      ) : (
                        task.crack_count
                      )}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      {/* Failed Tasks Table */}
      {failedTasks.length > 0 && (
        <Paper sx={{ mt: 3 }}>
          <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <Typography variant="h6" color="error">
              Failed Tasks ({failedTasks.length} total)
            </Typography>
          </Box>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Agent ID</TableCell>
                  <TableCell>Task ID</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Retry Count</TableCell>
                  <TableCell>Error Message</TableCell>
                  <TableCell>Last Updated</TableCell>
                  <TableCell align="center">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {failedTasks.map((task) => (
                  <TableRow key={task.id}>
                    <TableCell>{task.agent_id || 'Unassigned'}</TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.85rem' }}>
                        {task.id.substring(0, 8)}...
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={task.status}
                        color="error"
                        size="small"
                      />
                    </TableCell>
                    <TableCell>{task.retry_count || 0}</TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ maxWidth: 300 }}>
                        {task.error_message || 'No error message'}
                      </Typography>
                    </TableCell>
                    <TableCell>{formatDate(task.updated_at)}</TableCell>
                    <TableCell align="center">
                      <Button
                        variant="outlined"
                        size="small"
                        startIcon={<ReplayIcon />}
                        onClick={() => handleRetryTask(task.id)}
                        sx={{ textTransform: 'none' }}
                      >
                        Retry
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      )}

      {/* Completed Tasks Table */}
      {completedTasks.length > 0 && (
        <Paper sx={{ mt: 3 }}>
          <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <Typography variant="h6">
              Completed Tasks ({completedTasks.length} total)
            </Typography>
          </Box>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Agent ID</TableCell>
                  <TableCell>Task ID</TableCell>
                  <TableCell>Completed At</TableCell>
                  <TableCell>Keyspace Range</TableCell>
                  <TableCell>Final Progress</TableCell>
                  <TableCell>Average Speed</TableCell>
                  <TableCell>Cracks Found</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {paginatedCompletedTasks.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} align="center">
                      <Typography color="text.secondary" sx={{ py: 2 }}>
                        No completed tasks
                      </Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  paginatedCompletedTasks.map((task) => (
                    <TableRow key={task.id}>
                      <TableCell>{task.agent_id || 'Unassigned'}</TableCell>
                      <TableCell>{task.id.slice(0, 8)}...</TableCell>
                      <TableCell>{formatDate(task.completed_at)}</TableCell>
                      <TableCell>
                        {formatKeyspace(task.effective_keyspace_start || task.keyspace_start)} - {formatKeyspace(task.effective_keyspace_end || task.keyspace_end)}
                      </TableCell>
                      <TableCell>{task.progress_percent?.toFixed(2) || 100}%</TableCell>
                      <TableCell>{formatSpeed(task.benchmark_speed)}</TableCell>
                      <TableCell>
                        {task.crack_count > 0 ? (
                          <Link
                            component="button"
                            variant="body2"
                            onClick={() => navigate(`/pot/hashlist/${jobData.hashlist_id}`)}
                            sx={{ fontWeight: 'medium' }}
                          >
                            {task.crack_count}
                          </Link>
                        ) : (
                          task.crack_count
                        )}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
          {completedTasks.length > completedTasksPageSize && (
            <TablePagination
              rowsPerPageOptions={[25, 50, 100, 200]}
              component="div"
              count={completedTasks.length}
              rowsPerPage={completedTasksPageSize}
              page={completedTasksPage}
              onPageChange={(event, newPage) => setCompletedTasksPage(newPage)}
              onRowsPerPageChange={(event) => {
                setCompletedTasksPageSize(parseInt(event.target.value, 10));
                setCompletedTasksPage(0);
              }}
              showFirstButton
              showLastButton
            />
          )}
        </Paper>
      )}
    </Box>
  );
};

export default JobDetails;