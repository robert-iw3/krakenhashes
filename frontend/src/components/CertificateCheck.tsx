import React, { useEffect, useState, useCallback } from 'react';
import {
  Box,
  Typography,
  Button,
  Paper,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  CircularProgress,
  Alert
} from '@mui/material';
import {
  Security as SecurityIcon,
  Download as DownloadIcon,
  Refresh as RefreshIcon,
  CheckCircle as CheckCircleIcon
} from '@mui/icons-material';

interface CertificateCheckProps {
  onCertVerified: () => void;
}

const CertificateCheck: React.FC<CertificateCheckProps> = ({ onCertVerified }) => {
  const [checking, setChecking] = useState(true);
  const [certValid, setCertValid] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const checkCertificate = useCallback(async () => {
    try {
      setChecking(true);
      setError(null);

      // Get the configured API URLs
      const httpApiUrl = process.env.REACT_APP_HTTP_API_URL || 'http://localhost:1337';
      const apiUrl = process.env.REACT_APP_API_URL || 'https://localhost:31337';

      // First try to fetch the CA cert to see if it exists
      const caResponse = await fetch(`http://${window.location.hostname}:1337/ca.crt`, {
        method: 'HEAD',
        credentials: 'include',
        mode: 'cors',
        headers: {
          'Accept': 'application/x-x509-ca-cert'
        }
      });

      if (!caResponse.ok) {
        console.error('CA certificate not available');
        setCertValid(false);
        setError('CA certificate not available from server. Please contact support.');
        return;
      }

      // Now try to make a request to a secure endpoint using the HTTPS API URL
      const response = await fetch(`${apiUrl}/api/health`, {
        method: 'GET',
        credentials: 'include',
        headers: {
          'Accept': 'application/json',
        }
      });
      
      if (response.ok) {
        setCertValid(true);
        localStorage.setItem('cert_valid', 'true');
        onCertVerified();
      } else {
        console.error('Health check failed:', response.status, response.statusText);
        setCertValid(false);
        localStorage.removeItem('cert_valid');
        setError('Failed to verify certificate. Please ensure you have installed the CA certificate and try again.');
      }
    } catch (error) {
      console.debug('[CertCheck] Certificate validation failed:', error);
      // If we get a security error, it means the certificate isn't trusted
      setCertValid(false);
      localStorage.removeItem('cert_valid');
      setError('Certificate not trusted. Please install the CA certificate and ensure it is trusted by your browser.');
    } finally {
      setChecking(false);
    }
  }, [onCertVerified]);

  const downloadCertificates = async () => {
    try {
      setError(null);
      // Use dedicated HTTP endpoint for certificate download
      const httpApiUrl = process.env.REACT_APP_HTTP_API_URL || 'http://localhost:1337';

      // Download CA certificate
      const caResponse = await fetch(`http://${window.location.hostname}:1337/ca.crt`, {
        method: 'GET',
        credentials: 'include',
        mode: 'cors',
        headers: {
          'Accept': 'application/x-x509-ca-cert'
        }
      });

      if (!caResponse.ok) {
        throw new Error(`Failed to download CA certificate: ${caResponse.statusText}`);
      }

      const caBlob = await caResponse.blob();
      const caUrl = window.URL.createObjectURL(caBlob);
      const caLink = document.createElement('a');
      caLink.href = caUrl;
      caLink.download = 'krakenhashes-ca.crt';
      document.body.appendChild(caLink);
      caLink.click();
      window.URL.revokeObjectURL(caUrl);
      document.body.removeChild(caLink);

      // After downloading, show instructions for installation
      setError(`Please follow these steps:
1. Open the downloaded certificate (krakenhashes-ca.crt)
2. When prompted, select "Trust this CA to identify websites"
3. Complete the installation
4. Restart your browser
5. Click Verify to check the installation`);

    } catch (error) {
      console.error('Failed to download certificates:', error);
      setError('Failed to download certificates. Please try again or contact support.');
    }
  };

  useEffect(() => {
    // Don't use stored validation state anymore - always check
    checkCertificate();
  }, [checkCertificate]);

  if (checking) {
    return (
      <Box
        display="flex"
        flexDirection="column"
        alignItems="center"
        justifyContent="center"
        minHeight="100vh"
        gap={2}
      >
        <CircularProgress />
        <Typography>Checking SSL Certificate...</Typography>
      </Box>
    );
  }

  if (!certValid) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        minHeight="100vh"
        padding={3}
      >
        <Paper elevation={3} sx={{ p: 4, maxWidth: 600 }}>
          <Box display="flex" flexDirection="column" gap={3}>
            <Box display="flex" alignItems="center" gap={2}>
              <SecurityIcon color="primary" sx={{ fontSize: 40 }} />
              <Typography variant="h5" component="h1">
                Security Certificate Required
              </Typography>
            </Box>

            {error && (
              <Alert severity={error.includes('Please follow these steps') ? 'info' : 'error'} onClose={() => setError(null)}>
                {error}
              </Alert>
            )}

            <Typography>
              To ensure secure communication with KrakenHashes, you need to install our certificate authority (CA) certificate.
              Please follow these steps:
            </Typography>

            <List>
              <ListItem>
                <ListItemIcon>
                  <DownloadIcon />
                </ListItemIcon>
                <ListItemText 
                  primary="Download CA Certificate"
                  secondary="Click the button below to download our CA certificate"
                />
              </ListItem>
              <ListItem>
                <ListItemIcon>
                  <SecurityIcon />
                </ListItemIcon>
                <ListItemText 
                  primary="Install CA Certificate"
                  secondary="Import the CA certificate (krakenhashes-ca.crt) into your browser's trusted certificate authorities"
                />
              </ListItem>
              <ListItem>
                <ListItemIcon>
                  <RefreshIcon />
                </ListItemIcon>
                <ListItemText 
                  primary="Verify Installation"
                  secondary="Click verify below after installing the certificate"
                />
              </ListItem>
            </List>

            <Box display="flex" gap={2}>
              <Button
                variant="contained"
                startIcon={<DownloadIcon />}
                onClick={downloadCertificates}
              >
                Download CA Certificate
              </Button>
              <Button
                variant="outlined"
                startIcon={<RefreshIcon />}
                onClick={checkCertificate}
              >
                Verify Certificate
              </Button>
            </Box>
          </Box>
        </Paper>
      </Box>
    );
  }

  return (
    <Box
      display="flex"
      flexDirection="column"
      alignItems="center"
      justifyContent="center"
      minHeight="100vh"
      gap={2}
    >
      <CheckCircleIcon color="success" sx={{ fontSize: 48 }} />
      <Typography>Certificates verified! Redirecting...</Typography>
    </Box>
  );
};

export default CertificateCheck; 