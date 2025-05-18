import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  Box, 
  Typography, 
  TextField, 
  Button, 
  Paper, 
  List, 
  ListItem, 
  ListItemText, 
  ListItemIcon, 
  ListItemSecondaryAction, 
  IconButton, 
  Divider, 
  FormControl, 
  InputLabel, 
  Select, 
  MenuItem, 
  Alert, 
  CircularProgress,
  FormHelperText,
  SelectChangeEvent,
  Grid,
  Autocomplete
} from '@mui/material';
import DragHandleIcon from '@mui/icons-material/DragHandle';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { DragDropContext, Droppable, Draggable, DropResult } from 'react-beautiful-dnd';
import { 
  getJobWorkflowFormData, 
  getJobWorkflow, 
  createJobWorkflow, 
  updateJobWorkflow 
} from '../../services/api';
import { 
  PresetJobBasic, 
  JobWorkflowFormData, 
  CreateWorkflowRequest 
} from '../../types/adminJobs';

const JobWorkflowFormPage: React.FC = () => {
  const { jobWorkflowId } = useParams<{ jobWorkflowId?: string }>();
  const navigate = useNavigate();
  const isEditing = Boolean(jobWorkflowId);

  // Form state
  const [formData, setFormData] = useState<JobWorkflowFormData>({
    name: '',
    preset_job_ids: [],
    orderedJobs: []
  });

  // Available preset jobs for selection
  const [availablePresetJobs, setAvailablePresetJobs] = useState<PresetJobBasic[]>([]);
  
  // Currently selected preset job in the dropdown
  const [selectedPresetJobId, setSelectedPresetJobId] = useState<string>('');
  
  // UI state
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Fetch form data and workflow details if editing
  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        
        // Fetch available preset jobs
        const formDataResponse = await getJobWorkflowFormData();
        if (!formDataResponse.preset_jobs?.length) {
          setError('No preset jobs available. Please add preset jobs before creating workflows.');
          setLoading(false);
          return;
        }
        
        setAvailablePresetJobs(formDataResponse.preset_jobs);
        setSelectedPresetJobId(formDataResponse.preset_jobs[0]?.id || '');
        
        // If editing, fetch the workflow data
        if (isEditing && jobWorkflowId) {
          try {
            const workflow = await getJobWorkflow(jobWorkflowId);
            
            // Create mapping of IDs to names for rendering
            const orderedJobs: PresetJobBasic[] = [];
            if (workflow.steps?.length) {
              // Use the step ordering and names from the API response
              workflow.steps.sort((a, b) => a.step_order - b.step_order);
              orderedJobs.push(...workflow.steps.map(step => ({
                id: step.preset_job_id,
                name: step.preset_job_name
              })));
            }
            
            setFormData({
              name: workflow.name,
              preset_job_ids: workflow.steps?.map(step => step.preset_job_id) || [],
              orderedJobs
            });
          } catch (err) {
            console.error('Error fetching job workflow:', err);
            setError('Failed to load workflow. Please try again.');
          }
        }
        
        setLoading(false);
      } catch (err) {
        console.error('Error fetching form data:', err);
        setError('Failed to load form data. Please try again.');
        setLoading(false);
      }
    };

    fetchData();
  }, [isEditing, jobWorkflowId]);

  // Handle form field changes
  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData(prev => ({
      ...prev,
      name: e.target.value
    }));
  };

  // Handle preset job selection
  const handlePresetJobSelect = (e: SelectChangeEvent<string>) => {
    setSelectedPresetJobId(e.target.value);
  };

  // Add selected preset job to workflow
  const handleAddPresetJob = (job: PresetJobBasic | null) => {
    if (!job) return;
    
    // Skip if already in the list
    if (formData.preset_job_ids.includes(job.id)) {
      return;
    }
    
    // Add to orderedJobs and preset_job_ids
    setFormData(prev => {
      const newOrderedJobs = [...prev.orderedJobs, job];
      const newPresetJobIds = [...prev.preset_job_ids, job.id];
      
      return {
        ...prev,
        preset_job_ids: newPresetJobIds,
        orderedJobs: newOrderedJobs
      };
    });
  };

  // Remove preset job from workflow
  const handleRemovePresetJob = (jobId: string) => {
    setFormData(prev => {
      const newOrderedJobs = prev.orderedJobs.filter(job => job.id !== jobId);
      const newPresetJobIds = prev.preset_job_ids.filter(id => id !== jobId);
      
      return {
        ...prev,
        preset_job_ids: newPresetJobIds,
        orderedJobs: newOrderedJobs
      };
    });
  };

  // Handle drag and drop reordering
  const handleDragEnd = (result: DropResult) => {
    if (!result.destination) return;
    
    const { source, destination } = result;
    if (source.index === destination.index) return;
    
    setFormData(prev => {
      const newOrderedJobs = Array.from(prev.orderedJobs);
      const [removed] = newOrderedJobs.splice(source.index, 1);
      newOrderedJobs.splice(destination.index, 0, removed);
      
      // Update preset_job_ids to match new order
      const newPresetJobIds = newOrderedJobs.map(job => job.id);
      
      return {
        ...prev,
        preset_job_ids: newPresetJobIds,
        orderedJobs: newOrderedJobs
      };
    });
  };

  // Form validation
  const validateForm = (): boolean => {
    if (!formData.name.trim()) {
      setError('Workflow name is required');
      return false;
    }
    
    if (formData.preset_job_ids.length === 0) {
      setError('At least one preset job is required');
      return false;
    }
    
    return true;
  };

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }
    
    setSubmitting(true);
    setError(null);
    setSuccessMessage(null);
    
    // Create request payload (only need name and preset_job_ids)
    const payload: CreateWorkflowRequest = {
      name: formData.name,
      preset_job_ids: formData.preset_job_ids
    };
    
    try {
      if (isEditing && jobWorkflowId) {
        await updateJobWorkflow(jobWorkflowId, payload);
        setSuccessMessage('Workflow updated successfully');
      } else {
        await createJobWorkflow(payload);
        setSuccessMessage('Workflow created successfully');
        // Navigate back to list after a short delay
        setTimeout(() => {
          navigate('/admin/job-workflows');
        }, 1500);
      }
    } catch (err) {
      console.error('Error saving workflow:', err);
      setError('Failed to save workflow. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" height="60vh">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Box mb={3} display="flex" alignItems="center">
        <IconButton 
          onClick={() => navigate('/admin/job-workflows')} 
          sx={{ mr: 1 }}
          disabled={submitting}
        >
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="h4">
          {isEditing ? 'Edit Job Workflow' : 'Create New Job Workflow'}
        </Typography>
      </Box>
      
      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}
      
      {successMessage && (
        <Alert severity="success" sx={{ mb: 3 }}>
          {successMessage}
        </Alert>
      )}
      
      <Paper sx={{ p: 3, mb: 3 }}>
        <form onSubmit={handleSubmit}>
          <TextField
            label="Workflow Name"
            value={formData.name}
            onChange={handleNameChange}
            fullWidth
            margin="normal"
            variant="outlined"
            required
            disabled={submitting}
          />
          
          <Box mt={3}>
            <Autocomplete
              options={availablePresetJobs.filter(job => !formData.preset_job_ids.includes(job.id))}
              getOptionLabel={(option) => option.name}
              onChange={(_, value) => handleAddPresetJob(value)}
              disabled={submitting}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Search and add preset jobs"
                  variant="outlined"
                  helperText="Search and select a preset job to add to the workflow"
                  fullWidth
                />
              )}
            />
          </Box>
          
          <Typography variant="h6" sx={{ mt: 4, mb: 2 }}>
            Workflow Steps {formData.orderedJobs.length > 0 && `(${formData.orderedJobs.length})`}
          </Typography>
          
          {formData.orderedJobs.length === 0 ? (
            <Alert severity="info" sx={{ mb: 2 }}>
              No preset jobs added to this workflow yet. Add jobs from the dropdown above.
            </Alert>
          ) : (
            <Paper variant="outlined" sx={{ mb: 3 }}>
              <DragDropContext onDragEnd={handleDragEnd}>
                <Droppable droppableId="workflow-steps">
                  {(provided) => (
                    <List
                      {...provided.droppableProps}
                      ref={provided.innerRef}
                      sx={{ width: '100%', bgcolor: 'background.paper' }}
                    >
                      {formData.orderedJobs.map((job, index) => (
                        <React.Fragment key={job.id}>
                          <Draggable draggableId={job.id} index={index}>
                            {(provided) => (
                              <ListItem
                                ref={provided.innerRef}
                                {...provided.draggableProps}
                                sx={{ pr: 9 }}
                              >
                                <ListItemIcon {...provided.dragHandleProps}>
                                  <DragHandleIcon />
                                </ListItemIcon>
                                <ListItemText
                                  primary={`${index + 1}. ${job.name}`}
                                  secondary="Drag to reorder"
                                />
                                <ListItemSecondaryAction>
                                  <IconButton
                                    edge="end"
                                    onClick={() => handleRemovePresetJob(job.id)}
                                    disabled={submitting}
                                  >
                                    <DeleteIcon />
                                  </IconButton>
                                </ListItemSecondaryAction>
                              </ListItem>
                            )}
                          </Draggable>
                          {index < formData.orderedJobs.length - 1 && <Divider />}
                        </React.Fragment>
                      ))}
                      {provided.placeholder}
                    </List>
                  )}
                </Droppable>
              </DragDropContext>
            </Paper>
          )}
          
          <Box display="flex" justifyContent="flex-end" mt={4}>
            <Button
              variant="outlined"
              color="inherit"
              onClick={() => navigate('/admin/job-workflows')}
              sx={{ mr: 2 }}
              disabled={submitting}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="contained"
              color="primary"
              disabled={submitting || formData.orderedJobs.length === 0}
            >
              {submitting ? <CircularProgress size={24} /> : isEditing ? 'Save Changes' : 'Create Workflow'}
            </Button>
          </Box>
        </form>
      </Paper>
    </Box>
  );
};

export default JobWorkflowFormPage; 