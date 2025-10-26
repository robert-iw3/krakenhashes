/**
 * Password Analytics page for KrakenHashes frontend.
 *
 * Features:
 *   - Select client and date range for analysis
 *   - Generate new analytics reports
 *   - View previous reports
 *   - Display comprehensive password analytics
 *   - Queue management and status tracking
 */
import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Button,
  Typography,
  Paper,
  TextField,
  MenuItem,
  Grid,
  CircularProgress,
  Alert,
  Card,
  CardContent,
  CardHeader,
  Divider,
  Tab,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  LinearProgress,
  AlertTitle,
  Tooltip,
} from '@mui/material';
import {
  Add as AddIcon,
  Refresh as RefreshIcon,
  Delete as DeleteIcon,
  Replay as RetryIcon,
  Visibility as VisibilityIcon,
} from '@mui/icons-material';
import { useSnackbar } from 'notistack';
import analyticsService from '../services/analytics';
import { AnalyticsReport, CreateAnalyticsReportRequest } from '../types/analytics';
import { api } from '../services/api';

// Import display components
import AnalyticsReportDisplay from '../components/analytics/AnalyticsReportDisplay';

interface Client {
  id: string;
  name: string;
}

export default function Analytics() {
  const [clients, setClients] = useState<Client[]>([]);
  const [selectedClient, setSelectedClient] = useState<string>('');
  const [reportType, setReportType] = useState<'new' | 'previous'>('new');
  // Use string dates for simplicity
  const thirtyDaysAgo = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000);
  const [startDate, setStartDate] = useState<string>(thirtyDaysAgo.toISOString().slice(0, 16));
  const [endDate, setEndDate] = useState<string>(new Date().toISOString().slice(0, 16));
  const [customPatterns, setCustomPatterns] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [clientReports, setClientReports] = useState<AnalyticsReport[]>([]);
  const [currentReport, setCurrentReport] = useState<AnalyticsReport | null>(null);
  const [reportStatus, setReportStatus] = useState<string>('');
  const [pollInterval, setPollInterval] = useState<NodeJS.Timeout | null>(null);
  const { enqueueSnackbar } = useSnackbar();

  // Helper function to format dates
  const formatDate = (date: Date | string, formatStr: string): string => {
    const d = typeof date === 'string' ? new Date(date) : date;
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const year = d.getFullYear();
    const hours = String(d.getHours()).padStart(2, '0');
    const minutes = String(d.getMinutes()).padStart(2, '0');

    if (formatStr === 'MMM d, yyyy') {
      const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
      return `${monthNames[d.getMonth()]} ${d.getDate()}, ${year}`;
    } else if (formatStr === 'MMM d, yyyy HH:mm') {
      const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
      return `${monthNames[d.getMonth()]} ${d.getDate()}, ${year} ${hours}:${minutes}`;
    }
    return d.toLocaleString();
  };

  // Fetch clients on mount
  useEffect(() => {
    fetchClients();
  }, []);

  // Poll for report status when viewing a report
  useEffect(() => {
    if (currentReport && (currentReport.status === 'queued' || currentReport.status === 'processing')) {
      const interval = setInterval(() => {
        fetchReportStatus(currentReport.id);
      }, 5000); // Poll every 5 seconds
      setPollInterval(interval);

      return () => {
        if (interval) clearInterval(interval);
      };
    } else {
      if (pollInterval) {
        clearInterval(pollInterval);
        setPollInterval(null);
      }
    }
  }, [currentReport?.id, currentReport?.status]);

  const fetchClients = async () => {
    try {
      const response = await api.get('/api/analytics/clients');
      setClients(response.data);
    } catch (error) {
      console.error('Error fetching clients:', error);
      enqueueSnackbar('Failed to load clients', { variant: 'error' });
    }
  };

  const fetchClientReports = async (clientId: string) => {
    try {
      setLoading(true);
      const reports = await analyticsService.getClientReports(clientId);
      setClientReports(reports);
    } catch (error) {
      console.error('Error fetching client reports:', error);
      enqueueSnackbar('Failed to load reports', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const fetchReportStatus = async (reportId: string) => {
    try {
      const response = await analyticsService.getReport(reportId);
      setReportStatus(response.status);
      setCurrentReport(response.report);

      // Stop polling if completed or failed
      if (response.status === 'completed' || response.status === 'failed') {
        if (pollInterval) {
          clearInterval(pollInterval);
          setPollInterval(null);
        }
      }
    } catch (error) {
      console.error('Error fetching report status:', error);
    }
  };

  const handleClientChange = (clientId: string) => {
    setSelectedClient(clientId);
    setCurrentReport(null);
    setReportStatus('');
    if (reportType === 'previous') {
      fetchClientReports(clientId);
    }
  };

  const handleReportTypeChange = (event: React.SyntheticEvent, newValue: number) => {
    const type = newValue === 0 ? 'new' : 'previous';
    setReportType(type);
    setCurrentReport(null);
    setReportStatus('');

    if (type === 'previous' && selectedClient) {
      fetchClientReports(selectedClient);
    }
  };

  const handleGenerateReport = async () => {
    if (!selectedClient) {
      enqueueSnackbar('Please select a client', { variant: 'warning' });
      return;
    }

    try {
      setLoading(true);

      const patterns = customPatterns
        ? customPatterns.split(',').map(p => p.trim()).filter(p => p)
        : [];

      // Append time components to dates (00:00:00 for start, 23:59:59 for end)
      const startDateTime = `${startDate}T00:00:00`;
      const endDateTime = `${endDate}T23:59:59`;

      const request: CreateAnalyticsReportRequest = {
        client_id: selectedClient,
        start_date: new Date(startDateTime).toISOString(),
        end_date: new Date(endDateTime).toISOString(),
        custom_patterns: patterns,
      };

      const report = await analyticsService.createReport(request);
      setCurrentReport(report);
      setReportStatus('queued');
      enqueueSnackbar(`Report queued (Position: ${report.queue_position})`, { variant: 'success' });
    } catch (error: any) {
      console.error('Error generating report:', error);
      enqueueSnackbar(error.response?.data?.error || 'Failed to generate report', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const handleViewReport = async (reportId: string) => {
    try {
      setLoading(true);
      const response = await analyticsService.getReport(reportId);
      setCurrentReport(response.report);
      setReportStatus(response.status);
    } catch (error) {
      console.error('Error viewing report:', error);
      enqueueSnackbar('Failed to load report', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteReport = async (reportId: string) => {
    try {
      await analyticsService.deleteReport(reportId);
      enqueueSnackbar('Report deleted successfully', { variant: 'success' });
      if (selectedClient) {
        fetchClientReports(selectedClient);
      }
      if (currentReport?.id === reportId) {
        setCurrentReport(null);
        setReportStatus('');
      }
    } catch (error) {
      console.error('Error deleting report:', error);
      enqueueSnackbar('Failed to delete report', { variant: 'error' });
    }
  };

  const handleRetryReport = async (reportId: string) => {
    try {
      const report = await analyticsService.retryReport(reportId);
      setCurrentReport(report);
      setReportStatus('queued');
      enqueueSnackbar(`Report queued for retry (Position: ${report.queue_position})`, { variant: 'success' });
    } catch (error) {
      console.error('Error retrying report:', error);
      enqueueSnackbar('Failed to retry report', { variant: 'error' });
    }
  };

  const getStatusChip = (status: string) => {
    const statusColors: Record<string, any> = {
      queued: { color: 'info', label: 'Queued' },
      processing: { color: 'warning', label: 'Processing' },
      completed: { color: 'success', label: 'Completed' },
      failed: { color: 'error', label: 'Failed' },
    };

    const config = statusColors[status] || { color: 'default', label: status };
    return <Chip label={config.label} color={config.color} size="small" />;
  };

  return (
      <Box sx={{ p: 3 }}>
        {/* Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3 }}>
          <Box>
            <Typography variant="h4" component="h1" gutterBottom>
              Password Analytics
            </Typography>
            <Typography variant="body1" color="text.secondary">
              Generate comprehensive password analytics reports for security assessments
            </Typography>
          </Box>
        </Box>

        {/* Client Selection */}
        <Paper sx={{ p: 3, mb: 3 }}>
          <Typography variant="h6" gutterBottom>
            Select Client
          </Typography>
          <TextField
            select
            fullWidth
            label="Client"
            value={selectedClient}
            onChange={(e) => handleClientChange(e.target.value)}
            sx={{ mb: 2 }}
          >
            <MenuItem value="">
              <em>Select a client</em>
            </MenuItem>
            {clients.map((client) => (
              <MenuItem key={client.id} value={client.id}>
                {client.name}
              </MenuItem>
            ))}
          </TextField>
        </Paper>

        {/* Report Type Tabs */}
        {selectedClient && (
          <Paper sx={{ mb: 3 }}>
            <Tabs value={reportType === 'new' ? 0 : 1} onChange={handleReportTypeChange}>
              <Tab label="Generate New Report" />
              <Tab label="View Previous Reports" />
            </Tabs>

            <Divider />

            {/* New Report Form */}
            {reportType === 'new' && (
              <Box sx={{ p: 3 }}>
                <Grid container spacing={3}>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Start Date"
                      type="date"
                      value={startDate}
                      onChange={(e) => setStartDate(e.target.value)}
                      InputLabelProps={{ shrink: true }}
                      sx={{
                        '& .MuiInputBase-root': {
                          backgroundColor: '#121212',
                        },
                        '& input[type="date"]': {
                          colorScheme: 'dark',
                        },
                        '& input[type="date"]::-webkit-calendar-picker-indicator': {
                          filter: 'invert(1)',
                          cursor: 'pointer',
                        },
                      }}
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="End Date"
                      type="date"
                      value={endDate}
                      onChange={(e) => setEndDate(e.target.value)}
                      InputLabelProps={{ shrink: true }}
                      sx={{
                        '& .MuiInputBase-root': {
                          backgroundColor: '#121212',
                        },
                        '& input[type="date"]': {
                          colorScheme: 'dark',
                        },
                        '& input[type="date"]::-webkit-calendar-picker-indicator': {
                          filter: 'invert(1)',
                          cursor: 'pointer',
                        },
                      }}
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      label="Custom Patterns (comma-separated)"
                      placeholder="e.g., goog, ggl, micro, soft"
                      value={customPatterns}
                      onChange={(e) => setCustomPatterns(e.target.value)}
                      helperText="Add custom organization name variations to check"
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <Button
                      variant="contained"
                      startIcon={loading ? <CircularProgress size={20} /> : <AddIcon />}
                      onClick={handleGenerateReport}
                      disabled={loading || !selectedClient}
                      fullWidth
                    >
                      Generate Report
                    </Button>
                  </Grid>
                </Grid>
              </Box>
            )}

            {/* Previous Reports List */}
            {reportType === 'previous' && (
              <Box sx={{ p: 3 }}>
                {loading ? (
                  <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
                    <CircularProgress />
                  </Box>
                ) : clientReports.length === 0 ? (
                  <Alert severity="info">No reports found for this client</Alert>
                ) : (
                  <TableContainer>
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell>Date Range</TableCell>
                          <TableCell>Generated On</TableCell>
                          <TableCell>Status</TableCell>
                          <TableCell align="right">Hashes</TableCell>
                          <TableCell align="right">Cracked</TableCell>
                          <TableCell align="right">Actions</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {clientReports.map((report) => (
                          <TableRow key={report.id}>
                            <TableCell>
                              {formatDate(report.start_date, 'MMM d, yyyy')} - {formatDate(report.end_date, 'MMM d, yyyy')}
                            </TableCell>
                            <TableCell>{formatDate(report.created_at, 'MMM d, yyyy HH:mm')}</TableCell>
                            <TableCell>{getStatusChip(report.status)}</TableCell>
                            <TableCell align="right">{report.total_hashes.toLocaleString()}</TableCell>
                            <TableCell align="right">{report.total_cracked.toLocaleString()}</TableCell>
                            <TableCell align="right">
                              <Tooltip title="View Report">
                                <IconButton
                                  size="small"
                                  onClick={() => handleViewReport(report.id)}
                                  color="primary"
                                >
                                  <VisibilityIcon />
                                </IconButton>
                              </Tooltip>
                              {report.status === 'failed' && (
                                <Tooltip title="Retry Report">
                                  <IconButton
                                    size="small"
                                    onClick={() => handleRetryReport(report.id)}
                                    color="warning"
                                  >
                                    <RetryIcon />
                                  </IconButton>
                                </Tooltip>
                              )}
                              <Tooltip title="Delete Report">
                                <IconButton
                                  size="small"
                                  onClick={() => handleDeleteReport(report.id)}
                                  color="error"
                                >
                                  <DeleteIcon />
                                </IconButton>
                              </Tooltip>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Box>
            )}
          </Paper>
        )}

        {/* Report Display */}
        {currentReport && (
          <AnalyticsReportDisplay
            report={currentReport}
            status={reportStatus}
            onRetry={() => handleRetryReport(currentReport.id)}
            onDelete={() => handleDeleteReport(currentReport.id)}
          />
        )}
      </Box>
  );
}
