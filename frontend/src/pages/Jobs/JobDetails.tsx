import React from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Typography,
  Paper,
  Button,
  Stack
} from '@mui/material';
import { ArrowBack } from '@mui/icons-material';

/**
 * Placeholder page for job management details
 * This page will be fully implemented in a future update
 */
const JobDetails: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  return (
    <Box sx={{ p: 3 }}>
        <Button
          startIcon={<ArrowBack />}
          onClick={() => navigate(-1)}
          sx={{ mb: 3 }}
        >
          Back
        </Button>
        
        <Paper sx={{ p: 4, textAlign: 'center' }}>
          <Typography variant="h4" gutterBottom>
            Job Management
          </Typography>
          
          <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
            Job ID: {id}
          </Typography>
          
          <Stack spacing={2} alignItems="center">
            <Typography variant="h6" color="text.secondary">
              This page is under development
            </Typography>
            
            <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 600 }}>
              The job management page will provide detailed information about job execution,
              task distribution, performance metrics, and control options for managing
              running jobs.
            </Typography>
            
            <Typography variant="body2" color="text.secondary">
              Coming soon in a future update!
            </Typography>
          </Stack>
        </Paper>
    </Box>
  );
};

export default JobDetails;