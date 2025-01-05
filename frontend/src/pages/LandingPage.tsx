/**
 * LandingPage - Entry point component that handles authentication routing
 * 
 * Features:
 *   - Authentication state handling
 *   - Conditional routing
 *   - Loading state management
 *   - Responsive layout
 * 
 * Dependencies:
 *   - react-router-dom for navigation
 *   - @mui/material for UI components
 *   - ../types/auth for authentication types
 * 
 * Error Scenarios:
 *   - Authentication state inconsistencies
 *   - Navigation failures
 *   - Route access errors
 *   - Loading state timeouts
 * 
 * Usage Example:
 * ```tsx
 * // In router configuration
 * <Route 
 *   path="/" 
 *   element={
 *     <LandingPage 
 *       authChecked={true} 
 *       isAuth={false} 
 *     />
 *   } 
 * />
 * ```
 * 
 * Performance Considerations:
 *   - Minimal render cycles
 *   - Optimized loading state
 *   - Efficient routing transitions
 * 
 * @param {LandingPageProps} props - Component properties
 * @returns {JSX.Element} Landing page component
 */

import React from 'react';
import { Navigate } from 'react-router-dom';
import { CircularProgress, Box } from '@mui/material';

interface LandingPageProps {
  authChecked: boolean;
  isAuth: boolean;
}

const LandingPage: React.FC<LandingPageProps> = ({ authChecked, isAuth }) => {
  // Show loading spinner while auth check is in progress
  if (!authChecked) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        minHeight="100vh"
      >
        <CircularProgress />
      </Box>
    );
  }

  // Redirect based on auth status
  return isAuth ? (
    <Navigate to="/dashboard" replace />
  ) : (
    <Navigate to="/login" replace />
  );
};

export default LandingPage; 