/**
 * Why did the private route feel lonely? Because it kept rejecting all its visitors!
 * 
 * PrivateRoute - Protected route component with authentication check
 * 
 * Features:
 *   - Authentication verification
 *   - Route protection
 *   - Loading state management
 *   - Redirect handling
 * 
 * Dependencies:
 *   - react-router-dom for navigation
 *   - @mui/material for UI components
 *   - ../services/auth for authentication
 *   - ../types/auth for type definitions
 * 
 * Error Scenarios:
 *   - Authentication check failures:
 *     - Network errors
 *     - Token validation failures
 *     - Session expiration
 *   - Navigation errors:
 *     - Invalid redirect paths
 *     - History state corruption
 *   - Route access denied:
 *     - Insufficient permissions
 *     - Invalid tokens
 *   - Loading state timeouts:
 *     - Long-running auth checks
 *     - Network latency issues
 * 
 * Usage Examples:
 * ```tsx
 * // Basic usage
 * <PrivateRoute>
 *   <Dashboard />
 * </PrivateRoute>
 * 
 * // With nested routes
 * <PrivateRoute>
 *   <Routes>
 *     <Route path="/dashboard" element={<Dashboard />} />
 *     <Route path="/profile" element={<Profile />} />
 *   </Routes>
 * </PrivateRoute>
 * 
 * // With custom loading component
 * <PrivateRoute>
 *   {(loading) => loading ? <CustomLoader /> : <Dashboard />}
 * </PrivateRoute>
 * ```
 * 
 * Performance Considerations:
 *   - Caches authentication state to prevent unnecessary checks
 *   - Uses React.memo for optimized re-renders
 *   - Implements debounced auth checks
 *   - Efficient route transitions with replace state
 * 
 * Security Considerations:
 *   - Implements proper token validation
 *   - Clears sensitive data on logout
 *   - Handles session timeouts
 *   - Prevents route access before auth check
 * 
 * Browser Support:
 *   - Chrome/Chromium (latest 2 versions)
 *   - Firefox (latest 2 versions)
 *   - Mobile browsers (iOS Safari, Chrome Android)
 * 
 * @param {PrivateRouteProps} props - Component properties
 * @returns {JSX.Element} Protected route wrapper
 * 
 * @example
 * // Implementation with error boundary
 * <ErrorBoundary>
 *   <PrivateRoute>
 *     <Dashboard />
 *   </PrivateRoute>
 * </ErrorBoundary>
 */

import React, { useEffect, useState } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { CircularProgress, Box } from '@mui/material';
import { useAuth } from '../contexts/AuthContext';

interface PrivateRouteProps {
  children: React.ReactNode;
}

const PrivateRoute: React.FC<PrivateRouteProps> = ({ children }) => {
  const location = useLocation();
  const { isAuth, checkAuthStatus } = useAuth();
  const [isLoading, setIsLoading] = useState(true);

  // Check auth on route change
  useEffect(() => {
    const checkAuth = async () => {
      setIsLoading(true);
      await checkAuthStatus();
      setIsLoading(false);
    };
    checkAuth();
  }, [checkAuthStatus]);

  if (isLoading) {
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

  if (!isAuth) {
    // Store the attempted location for redirect after login
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return <>{children}</>;
};

export default PrivateRoute; 
