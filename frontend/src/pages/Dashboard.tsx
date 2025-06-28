/**
 * Dashboard - Main dashboard component for authenticated users
 * 
 * Features:
 *   - Overview of system status
 *   - Quick access to key features
 *   - User session management
 *   - System metrics display
 * 
 * Dependencies:
 *   - react-router-dom for navigation
 *   - @mui/material for UI components
 *   - ../services/auth for authentication
 *   - ../types/auth for type definitions
 * 
 * Error Scenarios:
 *   - Session expiration handling
 *   - Logout failures: Network errors, server errors
 *   - Navigation errors: Route access denied
 *   - Data loading failures: API timeouts, invalid responses
 * 
 * Usage Example:
 * ```tsx
 * // In protected route
 * <Route 
 *   path="/dashboard" 
 *   element={<Dashboard />} 
 * />
 * 
 * // With error boundary
 * <ErrorBoundary>
 *   <Dashboard />
 * </ErrorBoundary>
 * ```
 * 
 * Performance Considerations:
 *   - Lazy loading of dashboard widgets using React.lazy
 *   - Data fetching with caching and invalidation
 *   - Memoized component state using useMemo
 *   - Debounced logout handling to prevent multiple calls
 * 
 * @returns {JSX.Element} Dashboard component
 */

import React, { useMemo } from 'react';
import {
  Box,
  Typography,
  Grid,
  Paper,
  Container,
  Divider
} from '@mui/material';
// import JobStatusMonitor from '../components/JobStatusMonitor'; // Removed to improve page load performance

/**
 * Dashboard component for displaying system overview and metrics
 * 
 * @component
 * @example
 * return (
 *   <Dashboard />
 * )
 */
const Dashboard: React.FC = () => {
  // Memoize grid items to prevent unnecessary re-renders
  const gridItems = useMemo(() => (
    <>
      <Grid item xs={12} md={4}>
        <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column' }}>
          <Typography variant="h6" gutterBottom>
            System Status
          </Typography>
          {/* 
            TODO: Implement system status widget
            - Add real-time status updates
            - Include error state handling
            - Add refresh mechanism
          */}
        </Paper>
      </Grid>

      <Grid item xs={12} md={4}>
        <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column' }}>
          <Typography variant="h6" gutterBottom>
            Active Agents
          </Typography>
          {/* 
            TODO: Implement active agents list
            - Add pagination
            - Include search/filter
            - Add sorting capabilities
          */}
        </Paper>
      </Grid>

      <Grid item xs={12} md={4}>
        <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column' }}>
          <Typography variant="h6" gutterBottom>
            Recent Activity
          </Typography>
          {/* 
            TODO: Implement activity feed
            - Add real-time updates
            - Include activity filtering
            - Add timestamp sorting
          */}
        </Paper>
      </Grid>
    </>
  ), []);

  return (
    <Container maxWidth="lg">
      <Box sx={{ mt: 4, mb: 4 }}>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Typography variant="h4" component="h1" gutterBottom>
              Dashboard
            </Typography>
          </Grid>
          {gridItems}
          
          <Grid item xs={12}>
            <Divider sx={{ my: 2 }} />
            <Typography variant="h5" component="h2" gutterBottom>
              Job Management
            </Typography>
            <Paper sx={{ p: 2 }}>
              <Typography variant="body1" gutterBottom>
                Visit the <strong>Jobs</strong> page in the navigation menu to view and manage password cracking jobs in real-time.
              </Typography>
              <Typography variant="body2" color="text.secondary">
                The Jobs page provides live updates, job monitoring, and comprehensive management features.
              </Typography>
            </Paper>
          </Grid>
        </Grid>
      </Box>
    </Container>
  );
};

export default Dashboard; 