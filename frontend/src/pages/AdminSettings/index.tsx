import React, { useState } from 'react';
import { Box, Tabs, Tab, Typography, Paper } from '@mui/material';
import { EmailSettings } from './EmailSettings';
import { useAuth } from '../../contexts/AuthContext';
import { Navigate } from 'react-router-dom';
import AuthSettings from '../../components/admin/AuthSettings';
import { useSnackbar } from 'notistack';
import { updateAuthSettings } from '../../services/auth';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel = (props: TabPanelProps) => {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`admin-settings-tabpanel-${index}`}
      aria-labelledby={`admin-settings-tab-${index}`}
      {...other}
    >
      {value === index && (
        <Box sx={{ p: 3 }}>
          {children}
        </Box>
      )}
    </div>
  );
};

export const AdminSettings = () => {
  const [currentTab, setCurrentTab] = useState(() => {
    const savedTab = localStorage.getItem('adminSettingsTab');
    return savedTab ? parseInt(savedTab, 10) : 0;
  });
  
  const [loading, setLoading] = useState(false);
  const { userRole } = useAuth();
  const { enqueueSnackbar } = useSnackbar();

  // Redirect if not admin
  if (userRole !== 'admin') {
    return <Navigate to="/" replace />;
  }

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setCurrentTab(newValue);
    localStorage.setItem('adminSettingsTab', newValue.toString());
  };

  return (
    <Box sx={{ width: '100%', p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Admin Settings
      </Typography>
      
      <Paper sx={{ width: '100%', mt: 3 }}>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs
            value={currentTab}
            onChange={handleTabChange}
            aria-label="admin settings tabs"
          >
            <Tab label="Email Settings" />
            <Tab label="Authentication Settings" />
            <Tab label="System Settings" />
          </Tabs>
        </Box>

        <TabPanel value={currentTab} index={0}>
          <EmailSettings />
        </TabPanel>
        <TabPanel value={currentTab} index={1}>
          <AuthSettings 
            onSave={async (settings) => {
              setLoading(true);
              try {
                await updateAuthSettings(settings);
                enqueueSnackbar('Settings updated successfully', { variant: 'success' });
              } catch (error) {
                console.error('Failed to update settings:', error);
                enqueueSnackbar(error instanceof Error ? error.message : 'Failed to update settings', { variant: 'error' });
                throw error; // Propagate error to trigger form error state
              } finally {
                setLoading(false);
              }
            }}
            loading={loading}
          />
        </TabPanel>
        <TabPanel value={currentTab} index={2}>
          {/* System settings will go here */}
          <Typography>System Settings Coming Soon</Typography>
        </TabPanel>
      </Paper>
    </Box>
  );
}; 