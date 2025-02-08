/**
 * Login - Authentication component for KrakenHashes frontend
 * 
 * Features:
 *   - User authentication
 *   - Password strength validation
 *   - Remember me functionality
 *   - Rate limiting protection
 * 
 * Dependencies:
 *   - react-router-dom for navigation
 *   - @mui/material for UI components
 *   - ../services/auth for authentication
 *   - ../types/auth for type definitions
 * 
 * Browser Support:
 *   - Chrome/Chromium based (Chrome, Edge, Brave)
 *   - Firefox
 *   - Mobile responsive design
 * 
 * Error Scenarios:
 *   - Invalid credentials
 *   - Network failures
 *   - Rate limit exceeded
 *   - Password policy violations
 * 
 * TODOs:
 *   - Implement forgot password functionality (requires email service)
 *   - Add 2FA support
 *   - Implement CAPTCHA for failed login attempts
 * 
 * @param {LoginProps} props - Component properties
 * @returns {JSX.Element} Login form component
 */

import React, { useState, useCallback, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  Box, 
  Button, 
  TextField, 
  Typography, 
  Container,
  FormControlLabel,
  Checkbox,
  CircularProgress
} from '@mui/material';
import { login } from '../services/auth';
import { useAuth } from '../contexts/AuthContext';
import { LoginCredentials } from '../types/auth';
import MFAVerification from '../components/auth/MFAVerification';

// Rate limiting configuration
const RATE_LIMIT = {
  maxRequests: 10,
  timeWindow: 1000, // 1 second
};

const Login: React.FC = () => {
  const { setAuth, setUserRole, checkAuthStatus } = useAuth();
  const [credentials, setCredentials] = useState<LoginCredentials>({
    username: '',
    password: ''
  });
  const [error, setError] = useState<string>('');
  const [rememberMe, setRememberMe] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const [mfaRequired, setMfaRequired] = useState<boolean>(false);
  const [mfaSession, setMfaSession] = useState<{
    sessionToken: string;
    mfaType: string;
    mfaMethods: string[];
  } | null>(null);
  const requestCount = useRef<number>(0);
  const lastRequestTime = useRef<number>(Date.now());
  const navigate = useNavigate();

  /**
   * Handles rate limiting for login attempts
   * 
   * @returns {boolean} Whether request should be allowed
   * @throws {Error} When rate limit is exceeded
   */
  const checkRateLimit = useCallback((): boolean => {
    const now = Date.now();
    if (now - lastRequestTime.current > RATE_LIMIT.timeWindow) {
      requestCount.current = 0;
      lastRequestTime.current = now;
    }
    
    if (requestCount.current >= RATE_LIMIT.maxRequests) {
      throw new Error('Too many login attempts. Please try again later.');
    }
    
    requestCount.current++;
    return true;
  }, []);

  /**
   * Handles form submission and authentication
   * 
   * @param {React.FormEvent} e - Form event
   * @returns {Promise<void>}
   */
  const handleSubmit = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      checkRateLimit();

      const response = await login(credentials.username, credentials.password);
      
      // Check if MFA is required
      if (response.mfa_required) {
        // Verify required MFA fields are present
        if (!response.session_token || !response.mfa_type || !response.preferred_method) {
          throw new Error('Invalid MFA response from server');
        }
        
        setMfaRequired(true);
        setMfaSession({
          sessionToken: response.session_token,
          mfaType: response.mfa_type,
          mfaMethods: [response.preferred_method]
        });
      } else if (response.token) {
        handleLoginSuccess(response.token);
      } else {
        setError(response.message || 'Login failed');
      }
    } catch (error) {
      setError(error instanceof Error ? error.message : 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  const handleMFASuccess = (token: string) => {
    handleLoginSuccess(token);
  };

  const handleLoginSuccess = (token: string) => {
    if (rememberMe) {
      localStorage.setItem('rememberMe', 'true');
    }
    setAuth(true);
    checkAuthStatus(); // This will fetch the user profile and set the role
    navigate('/dashboard', { replace: true });
  };

  const handleMFAError = (error: string) => {
    setError(error);
  };

  if (mfaRequired && mfaSession) {
    return (
      <Container component="main" maxWidth="xs">
        <Box
          sx={{
            marginTop: 8,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
          }}
        >
          <MFAVerification
            sessionToken={mfaSession.sessionToken}
            mfaType={mfaSession.mfaType}
            mfaMethods={mfaSession.mfaMethods}
            onSuccess={handleMFASuccess}
            onError={handleMFAError}
          />
        </Box>
      </Container>
    );
  }

  return (
    <Container component="main" maxWidth="xs">
      <Box
        sx={{
          marginTop: 8,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
        }}
      >
        <Typography component="h1" variant="h5">
          Log in to KrakenHashes
        </Typography>
        <Box component="form" onSubmit={handleSubmit} noValidate sx={{ mt: 1 }}>
          {error && (
            <Typography color="error" align="center">
              {error}
            </Typography>
          )}
          <TextField
            margin="normal"
            required
            fullWidth
            id="username"
            label="Username"
            name="username"
            autoComplete="username"
            autoFocus
            value={credentials.username}
            onChange={(e) => setCredentials((prev) => ({
              ...prev,
              username: e.target.value
            }))}
            disabled={loading}
          />
          <TextField
            margin="normal"
            required
            fullWidth
            name="password"
            label="Password"
            type="password"
            id="password"
            autoComplete="current-password"
            value={credentials.password}
            onChange={(e) => {
              setCredentials((prev) => ({
                ...prev,
                password: e.target.value
              }));
            }}
            disabled={loading}
          />
          <FormControlLabel
            control={
              <Checkbox
                value="remember"
                color="primary"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                disabled={loading}
              />
            }
            label="Remember me"
          />
          <Button
            type="submit"
            fullWidth
            variant="contained"
            sx={{ mt: 3, mb: 2 }}
            disabled={loading || !credentials.username || !credentials.password}
          >
            {loading ? <CircularProgress size={24} /> : 'Log In'}
          </Button>
        </Box>
      </Box>
    </Container>
  );
};

export default Login; 