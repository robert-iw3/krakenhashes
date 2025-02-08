import React, { useState } from 'react';
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
  mfaType: string;
  mfaMethods: string[];
  onSuccess: (token: string) => void;
  onError: (error: string) => void;
  expiresAt?: string;
}

const MFAVerification: React.FC<MFAVerificationProps> = ({
  sessionToken,
  mfaType,
  mfaMethods,
  onSuccess,
  onError,
  expiresAt,
}) => {
  const [code, setCode] = useState('');
  const [method, setMethod] = useState(mfaType);
  const [loading, setLoading] = useState(false);
  const [remainingAttempts, setRemainingAttempts] = useState<number | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      const response = await verifyMFA(sessionToken, code, method);
      if (response.success) {
        onSuccess(response.token);
      } else {
        setRemainingAttempts(response.remainingAttempts);
        onError(`Invalid code. ${response.remainingAttempts} attempts remaining.`);
      }
    } catch (error) {
      onError(error instanceof Error ? error.message : 'Verification failed');
    } finally {
      setLoading(false);
    }
  };

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
        
        {mfaMethods.length > 1 && (
          <FormControl fullWidth margin="normal">
            <InputLabel>Authentication Method</InputLabel>
            <Select
              value={method}
              onChange={(e) => setMethod(e.target.value)}
              label="Authentication Method"
            >
              {mfaMethods.map((m) => (
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
        />

        {remainingAttempts !== null && (
          <Typography color="error" sx={{ mt: 1 }}>
            {remainingAttempts} attempts remaining
          </Typography>
        )}

        <Button
          type="submit"
          fullWidth
          variant="contained"
          sx={{ mt: 3, mb: 2 }}
          disabled={loading || !code}
          onClick={handleSubmit}
        >
          {loading ? <CircularProgress size={24} /> : 'Verify'}
        </Button>
      </CardContent>
    </Card>
  );
};

export default MFAVerification; 