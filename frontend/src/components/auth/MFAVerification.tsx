import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  TextField,
  Typography,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  CircularProgress,
  Card,
  CardContent,
  Alert,
} from '@mui/material';
import { verifyMFA } from '../../services/auth';

interface MFAVerificationProps {
  sessionToken: string;
  mfaType: string[];  // Changed to string[] to match backend type
  preferredMethod: string;
  onSuccess: (token: string) => void;
  onError: (error: string) => void;
  expiresAt?: string;
}

const MFAVerification: React.FC<MFAVerificationProps> = ({
  sessionToken,
  mfaType,
  preferredMethod,
  onSuccess,
  onError,
  expiresAt,
}) => {
  const [code, setCode] = useState('');
  const [method, setMethod] = useState(preferredMethod);
  const [loading, setLoading] = useState(false);
  const [remainingAttempts, setRemainingAttempts] = useState<number>(3);

  // Get available methods including backup if it exists in mfaType
  const getAvailableMethods = () => {
    return mfaType.filter(m => m !== 'backup').concat(mfaType.includes('backup') ? ['backup'] : []);
  };

  const handleMethodChange = async (newMethod: string) => {
    setCode(''); // Clear code when changing methods
    setMethod(newMethod);

    // Request email code when switching to email method
    if (newMethod === 'email') {
      try {
        setLoading(true);
        const response = await verifyMFA(sessionToken, '', 'request_email');
        if (!response.success) {
          onError(response.message || 'Failed to send email code');
        }
      } catch (error) {
        onError(error instanceof Error ? error.message : 'Failed to send email code');
      } finally {
        setLoading(false);
      }
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      const response = await verifyMFA(sessionToken, code, method);
      if (response.success) {
        onSuccess(response.token);
      } else {
        setRemainingAttempts(response.remainingAttempts ?? remainingAttempts - 1);
        onError(response.message || `Invalid code. ${response.remainingAttempts} attempts remaining.`);
        setCode(''); // Clear code on failed attempt
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Verification failed';
      onError(message);
      if (message.includes('No backup codes available')) {
        // Remove backup from available methods if no codes are available
        const updatedMethods = getAvailableMethods().filter(m => m !== 'backup');
        if (method === 'backup') {
          setMethod(preferredMethod); // Switch back to preferred method
        }
        setAvailableMethods(updatedMethods);
      }
    } finally {
      setLoading(false);
    }
  };

  // State for available methods
  const [availableMethods, setAvailableMethods] = useState<string[]>(getAvailableMethods());

  // Update available methods when mfaType changes
  useEffect(() => {
    setAvailableMethods(getAvailableMethods());
  }, [mfaType]);

  // Request email code on initial load if email is the selected method
  useEffect(() => {
    // Only request email code on mount if email is selected but NOT the preferred method
    // This prevents duplicate emails when email is preferred (since backend already sent it)
    if (method === 'email' && method !== preferredMethod && !loading) {
      handleMethodChange('email');
    }
  }, []); // Only run on initial mount

  const getMethodInstructions = () => {
    switch (method) {
      case 'email':
        return (
          <Alert severity="info" sx={{ mb: 2 }}>
            Please enter the verification code sent to your email.
            {expiresAt && (
              <Typography variant="body2" sx={{ mt: 1 }}>
                Code expires at: {new Date(expiresAt).toLocaleTimeString()}
              </Typography>
            )}
          </Alert>
        );
      case 'authenticator':
        return (
          <Alert severity="info" sx={{ mb: 2 }}>
            Please enter the code from your authenticator app.
          </Alert>
        );
      case 'backup':
        return (
          <Alert severity="warning" sx={{ mb: 2 }}>
            Please enter one of your backup codes. Note that each backup code can only be used once.
          </Alert>
        );
      default:
        return null;
    }
  };

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          Two-Factor Authentication Required
        </Typography>
        
        {getMethodInstructions()}
        
        {availableMethods.length > 1 && (
          <FormControl fullWidth margin="normal">
            <InputLabel>Authentication Method</InputLabel>
            <Select
              value={method}
              onChange={(e) => handleMethodChange(e.target.value)}
              label="Authentication Method"
            >
              {availableMethods.map((m) => (
                <MenuItem key={m} value={m}>
                  {m === 'email' ? 'Email Code' : 
                   m === 'authenticator' ? 'Authenticator App' : 
                   m === 'backup' ? 'Backup Code' : m}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        )}

        <TextField
          margin="normal"
          required
          fullWidth
          label={method === 'backup' ? 'Backup Code' : 'Verification Code'}
          value={code}
          onChange={(e) => setCode(e.target.value)}
          disabled={loading}
          autoFocus
          placeholder={method === 'backup' ? 'Enter 8-character backup code' : 'Enter verification code'}
          inputProps={{
            maxLength: method === 'backup' ? 8 : 6,
            pattern: '[0-9]*'
          }}
        />

        <Typography 
          color={remainingAttempts <= 1 ? "error" : "warning"} 
          sx={{ mt: 1 }}
        >
          {remainingAttempts} {remainingAttempts === 1 ? 'attempt' : 'attempts'} remaining
        </Typography>

        <Button
          type="submit"
          fullWidth
          variant="contained"
          sx={{ mt: 3, mb: 2 }}
          disabled={loading || !code || (method === 'backup' ? code.length !== 8 : code.length !== 6)}
          onClick={handleSubmit}
        >
          {loading ? <CircularProgress size={24} /> : 'Verify'}
        </Button>
      </CardContent>
    </Card>
  );
};

export default MFAVerification; 