import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Switch,
  FormControlLabel,
  Button,
  Alert,
  CircularProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Email as EmailIcon,
  Key as KeyIcon,
  QrCode2 as QrCodeIcon,
  ContentCopy as CopyIcon,
  Check as CheckIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { useAuth } from '../../contexts/AuthContext';
import { 
  getUserMFASettings, 
  enableMFA, 
  disableMFA, 
  verifyMFASetup, 
  generateBackupCodes 
} from '../../services/auth';
import { MFASettings } from '../../types/auth';

interface MFACardProps {
  onMFAChange?: () => void;
}

const MFACard: React.FC<MFACardProps> = ({ onMFAChange }) => {
  const { user, setUser } = useAuth();
  const [loading, setLoading] = useState(true);
  const [mfaSettings, setMFASettings] = useState<MFASettings | null>(null);
  const [showQRDialog, setShowQRDialog] = useState(false);
  const [verificationCode, setVerificationCode] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [showBackupCodes, setShowBackupCodes] = useState(false);
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);
  const [qrCode, setQrCode] = useState<string | null>(null);
  const [secret, setSecret] = useState<string | null>(null);

  useEffect(() => {
    loadMFASettings();
  }, []);

  const loadMFASettings = async () => {
    try {
      const settings = await getUserMFASettings();
      setMFASettings(settings);
      setError(null);
    } catch (err) {
      setError('Failed to load MFA settings');
      console.error('Failed to load MFA settings:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleMFAToggle = async () => {
    try {
      setLoading(true);
      setError(null);
      if (mfaSettings?.mfaEnabled) {
        await disableMFA();
        setSuccess('MFA disabled successfully');
      } else {
        // Check if email is available as a method
        const hasEmailProvider = mfaSettings?.allowedMfaMethods.includes('email');
        const hasAuthenticator = mfaSettings?.allowedMfaMethods.includes('authenticator');

        if (hasEmailProvider) {
          await enableMFA('email');
          // Update user state
          if (setUser && user) {
            setUser({ ...user, mfaEnabled: true, mfaType: 'email' });
          }
          setSuccess('MFA enabled successfully');
          // Reload MFA settings immediately after enabling
          await loadMFASettings();
        } else if (hasAuthenticator) {
          // If email is not available but authenticator is, trigger authenticator setup
          await handleAuthenticatorSetup();
          return; // Return early as handleAuthenticatorSetup will handle the rest
        } else {
          throw new Error('No MFA methods available');
        }
      }
      
      // Reload MFA settings and notify parent if needed
      await loadMFASettings();
      if (onMFAChange) {
        onMFAChange();
      }
    } catch (err) {
      console.error('MFA toggle failed:', err);
      setError(err instanceof Error ? err.message : 'Failed to toggle MFA');
    } finally {
      setLoading(false);
    }
  };

  const handleAuthenticatorSetup = async () => {
    try {
      setError(null);
      const response = await enableMFA('authenticator');
      if (response.qrCode && response.secret) {
        setQrCode(response.qrCode);
        setSecret(response.secret);
        setShowQRDialog(true);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to setup authenticator');
    }
  };

  const handleVerifyCode = async () => {
    if (!verificationCode || !user || !setUser) return;

    try {
      // Clear any existing errors
      setError(null);
      setSuccess(null);
      
      const response = await verifyMFASetup(verificationCode);
      
      // Update user state
      setUser({ ...user, mfaEnabled: true, mfaType: 'authenticator' });
      
      // Reload MFA settings to get the latest state
      await loadMFASettings();
      
      setSuccess('Authenticator app has been set up successfully');
      setShowQRDialog(false);
      setVerificationCode('');
      setQrCode(null);
      setSecret(null);

      // Handle backup codes if they were returned
      if (response?.backupCodes) {
        setBackupCodes(response.backupCodes);
        setShowBackupCodes(true);
      }

      if (onMFAChange) {
        onMFAChange();
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to verify code';
      setError(errorMessage);
      // Don't close dialog on error so user can try again
      setVerificationCode('');
    }
  };

  const handleGenerateBackupCodes = async () => {
    try {
      setError(null);
      const codes = await generateBackupCodes();
      setBackupCodes(codes);
      setShowBackupCodes(true);
      setSuccess('New backup codes have been generated');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate backup codes');
    }
  };

  const handleCopyCode = (code: string, index: number) => {
    navigator.clipboard.writeText(code);
    setCopiedIndex(index);
    setTimeout(() => setCopiedIndex(null), 2000);
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" p={4}>
        <CircularProgress />
      </Box>
    );
  }

  const isEmailRequired = mfaSettings?.mfaEnabled && mfaSettings?.allowedMfaMethods.includes('email');

  return (
    <Card sx={{ mb: 3 }}>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          Multi-Factor Authentication
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {success && (
          <Alert severity="success" sx={{ mb: 2 }}>
            {success}
          </Alert>
        )}

        {mfaSettings?.requireMfa && (
          <Alert severity="info" sx={{ mb: 2 }}>
            MFA is required by your organization's security policy
          </Alert>
        )}

        <FormControlLabel
          control={
            <Switch
              checked={mfaSettings?.mfaEnabled || false}
              onChange={handleMFAToggle}
              disabled={mfaSettings?.requireMfa}
            />
          }
          label="Enable Multi-Factor Authentication"
        />

        {mfaSettings?.mfaEnabled && (
          <Box sx={{ mt: 2 }}>
            <List>
              {/* Email MFA Status */}
              <ListItem>
                <ListItemIcon>
                  <EmailIcon color={isEmailRequired ? "primary" : "disabled"} />
                </ListItemIcon>
                <ListItemText
                  primary="Email Authentication"
                  secondary={isEmailRequired ? "Required for account security" : "Optional"}
                />
                {isEmailRequired && (
                  <Tooltip title="Required">
                    <WarningIcon color="info" />
                  </Tooltip>
                )}
              </ListItem>

              {/* Authenticator App Status */}
              <ListItem>
                <ListItemIcon>
                  <KeyIcon color={mfaSettings?.mfaType === 'authenticator' ? "primary" : "disabled"} />
                </ListItemIcon>
                <ListItemText
                  primary="Authenticator App"
                  secondary={mfaSettings?.mfaType === 'authenticator' ? "Configured" : "Not configured"}
                />
                {mfaSettings?.allowedMfaMethods.includes('authenticator') && mfaSettings?.mfaType !== 'authenticator' && (
                  <Button
                    variant="outlined"
                    onClick={handleAuthenticatorSetup}
                    startIcon={<QrCodeIcon />}
                  >
                    Setup
                  </Button>
                )}
              </ListItem>

              {/* Backup Codes */}
              <ListItem>
                <ListItemIcon>
                  <KeyIcon color={(mfaSettings?.remainingBackupCodes ?? 0) > 0 ? "primary" : "disabled"} />
                </ListItemIcon>
                <ListItemText
                  primary="Backup Codes"
                  secondary={mfaSettings?.remainingBackupCodes 
                    ? `${mfaSettings.remainingBackupCodes} backup ${mfaSettings.remainingBackupCodes === 1 ? 'code' : 'codes'} remaining` 
                    : "No backup codes available"}
                />
              </ListItem>
            </List>
          </Box>
        )}

        {/* QR Code Dialog */}
        <Dialog open={showQRDialog} onClose={() => setShowQRDialog(false)}>
          <DialogTitle>Setup Authenticator App</DialogTitle>
          <DialogContent>
            <Box sx={{ p: 2, textAlign: 'center' }}>
              {error && (
                <Alert severity="error" sx={{ mb: 2 }}>
                  {error}
                </Alert>
              )}
              {qrCode && (
                <Box
                  component="img"
                  src={`data:image/png;base64,${qrCode}`}
                  alt="QR Code"
                  sx={{
                    width: 200,
                    height: 200,
                    mb: 2,
                  }}
                />
              )}
              {secret && (
                <Typography variant="body2" sx={{ mb: 2 }}>
                  If you can't scan the QR code, enter this code manually: <strong>{secret}</strong>
                </Typography>
              )}
              <Typography variant="body2" sx={{ mb: 2 }}>
                Scan this QR code with your authenticator app (e.g., Google Authenticator, Authy)
              </Typography>
              <TextField
                fullWidth
                label="Verification Code"
                value={verificationCode}
                onChange={(e) => setVerificationCode(e.target.value)}
                margin="normal"
                autoComplete="off"
                placeholder="Enter the 6-digit code"
                inputProps={{
                  maxLength: 6,
                  pattern: '[0-9]*',
                }}
              />
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => {
              setShowQRDialog(false);
              setQrCode(null);
              setSecret(null);
              setVerificationCode('');
            }}>
              Cancel
            </Button>
            <Button
              onClick={handleVerifyCode}
              variant="contained"
              disabled={!verificationCode || verificationCode.length !== 6}
            >
              Verify
            </Button>
          </DialogActions>
        </Dialog>

        {/* Backup Codes Dialog */}
        <Dialog
          open={showBackupCodes}
          onClose={() => setShowBackupCodes(false)}
          maxWidth="sm"
          fullWidth
        >
          <DialogTitle>Backup Codes</DialogTitle>
          <DialogContent>
            {backupCodes.length === 0 ? (
              <Box sx={{ textAlign: 'center', py: 2 }}>
                <Typography variant="body2" sx={{ mb: 2 }}>
                  Generate backup codes to use when you can't access your primary authentication method
                </Typography>
                <Button
                  variant="contained"
                  onClick={handleGenerateBackupCodes}
                >
                  Generate Backup Codes
                </Button>
              </Box>
            ) : (
              <Box>
                <Typography variant="body2" color="warning.main" sx={{ mb: 2 }}>
                  Save these codes in a secure location. They will not be shown again!
                </Typography>
                <Box 
                  sx={{ 
                    fontFamily: 'monospace',
                    fontSize: '1.1rem',
                    mb: 3,
                    pl: 2
                  }}
                >
                  {backupCodes.map((code) => (
                    <Typography key={code} sx={{ mb: 1 }}>
                      {code}
                    </Typography>
                  ))}
                </Box>
                <Button
                  fullWidth
                  variant="contained"
                  color="error"
                  startIcon={copiedIndex === -1 ? <CheckIcon /> : <CopyIcon />}
                  onClick={() => {
                    navigator.clipboard.writeText(backupCodes.join('\n'));
                    setCopiedIndex(-1);
                    setTimeout(() => setCopiedIndex(null), 2000);
                  }}
                >
                  {copiedIndex === -1 ? 'Copied!' : 'COPY ALL CODES'}
                </Button>
              </Box>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setShowBackupCodes(false)}>Close</Button>
          </DialogActions>
        </Dialog>
      </CardContent>
    </Card>
  );
};

export default MFACard; 