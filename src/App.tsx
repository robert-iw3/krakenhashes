/**
 * App - Root application component with routing and authentication
 * 
 * Features:
 *   - Protected routes
 *   - Authentication state management
 *   - Theme provider
 *   - Global layout
 * 
 * Dependencies:
 *   - react-router-dom for routing
 *   - @mui/material for theming
 *   - ./services/auth for authentication
 *   - ./components/* for page components
 * 
 * Error Scenarios:
 *   - Authentication failures
 *   - Route access errors
 *   - Component loading errors
 *   - Theme initialization failures
 * 
 * Usage Example:
 * ```tsx
 * // In index.tsx
 * ReactDOM.render(
 *   <React.StrictMode>
 *     <App />
 *   </React.StrictMode>,
 *   document.getElementById('root')
 * );
 * ```
 * 
 * Performance Considerations:
 *   - Lazy loading of routes
 *   - Optimized authentication checks
 *   - Memoized theme provider
 * 
 * @returns {JSX.Element} Root application component
 */

import React from 'react';
import { BrowserRouter as Router, Route, Routes, Navigate } from 'react-router-dom';
import { ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import theme from './styles/theme';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import AgentManagement from './pages/AgentManagement';
import PrivateRoute from './components/PrivateRoute';
import { AuthProvider } from './hooks/useAuth';

const App: React.FC = () => {
  return (
    <AuthProvider>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Router>
          <Routes>
            {/* Public routes */}
            <Route 
              path="/login" 
              element={<Login />} 
            />

            {/* Protected routes */}
            <Route
              path="/dashboard"
              element={
                <PrivateRoute>
                  <Layout>
                    <Dashboard />
                  </Layout>
                </PrivateRoute>
              }
            />
            <Route
              path="/agents"
              element={
                <PrivateRoute>
                  <Layout>
                    <AgentManagement />
                  </Layout>
                </PrivateRoute>
              }
            />

            {/* Root route */}
            <Route
              path="/"
              element={<Navigate to="/dashboard" replace />}
            />

            {/* Catch all route */}
            <Route
              path="*"
              element={<Navigate to="/dashboard" replace />}
            />
          </Routes>
        </Router>
      </ThemeProvider>
    </AuthProvider>
  );
};

export default App; 