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

import React, { useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import theme from './styles/theme';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import AgentManagement from './pages/AgentManagement';
import PrivateRoute from './components/PrivateRoute';
import CertificateCheck from './components/CertificateCheck';
import { AuthProvider } from './contexts/AuthContext';
import About from './pages/About';
import AuthSettingsPage from './pages/admin/AuthSettings';
import { SnackbarProvider } from 'notistack';
import { AdminSettings } from './pages/AdminSettings';
import ProfileSettings from './pages/settings/ProfileSettings';

const App: React.FC = () => {
  const [certVerified, setCertVerified] = useState(() => {
    // Check if we've already verified the cert
    return localStorage.getItem('cert_valid') === 'true';
  });

  if (!certVerified) {
    return (
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <CertificateCheck onCertVerified={() => setCertVerified(true)} />
      </ThemeProvider>
    );
  }

  return (
    <AuthProvider>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <SnackbarProvider maxSnack={3}>
          <Router>
            <Routes>
              {/* Public routes */}
              <Route 
                path="/login" 
                element={<Login />} 
              />

              {/* Protected routes */}
              <Route element={<PrivateRoute><Layout /></PrivateRoute>}>
                <Route path="/dashboard" element={<Dashboard />} />
                <Route path="/agents" element={<AgentManagement />} />
                <Route path="/admin/settings" element={<AdminSettings />} />
                <Route path="/about" element={<About />} />
                <Route path="/settings/profile" element={<ProfileSettings />} />
              </Route>
              <Route
                path="/admin/auth/settings"
                element={
                  <PrivateRoute>
                    <AuthSettingsPage />
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
        </SnackbarProvider>
      </ThemeProvider>
    </AuthProvider>
  );
};

export default App; 