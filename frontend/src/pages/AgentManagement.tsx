/**
 * Agent Management page component for HashDom frontend.
 * 
 * Features:
 *   - Agent registration with claim code generation
 *   - Agent list display and management
 *   - Real-time status monitoring
 *   - Team assignment
 * 
 * @packageDocumentation
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Container,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControlLabel,
  Switch,
  CircularProgress,
  Alert,
} from '@mui/material';
import { Delete as DeleteIcon } from '@mui/icons-material';
import { Agent, ClaimVoucher } from '../types/agent';
import { api } from '../services/api';

/**
 * AgentManagement component handles the display and management of HashDom agents.
 * 
 * Features:
 *   - Register new agents
 *   - Generate claim codes
 *   - View agent status
 *   - Monitor agent health
 * 
 * @returns {JSX.Element} The rendered agent management page
 * 
 * @example
 * <AgentManagement />
 */
export default function AgentManagement() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [claimVouchers, setClaimVouchers] = useState<ClaimVoucher[]>([]);
  const [openDialog, setOpenDialog] = useState(false);
  const [isContinuous, setIsContinuous] = useState(false);
  const [claimCode, setClaimCode] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch data
  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      console.log('Fetching agents and vouchers...');
      const [agentsRes, vouchersRes] = await Promise.all([
        api.get<Agent[]>('/api/agents'),
        api.get<ClaimVoucher[]>('/api/vouchers')
      ]);
      
      console.log('Received agents:', agentsRes.data);
      console.log('Received vouchers:', vouchersRes.data);
      
      setAgents(agentsRes.data || []);
      setClaimVouchers((vouchersRes.data || []).filter(v => v.is_active));
    } catch (error) {
      console.error('Failed to fetch data:', error);
      setError('Failed to load data. Please try again.');
      setAgents([]);
      setClaimVouchers([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    
    // Set up polling for updates
    const interval = setInterval(fetchData, 30000); // Poll every 30 seconds
    
    return () => clearInterval(interval);
  }, []);

  // Handle claim code generation
  const handleCreateClaimCode = async () => {
    try {
      setError(null);
      const response = await api.post<{ code: string }>('/api/vouchers/temp', {
        isContinuous: isContinuous
      });
      setClaimCode(response.data.code);
      await fetchData(); // Refresh the vouchers list
    } catch (error) {
      console.error('Failed to create claim code:', error);
      setError('Failed to generate claim code. Please try again.');
    }
  };

  // Handle voucher deactivation
  const handleDeactivateVoucher = async (code: string) => {
    try {
      setError(null);
      await api.delete(`/api/vouchers/${code}/disable`);
      await fetchData();
    } catch (error) {
      console.error('Failed to deactivate voucher:', error);
      setError('Failed to deactivate voucher. Please try again.');
    }
  };

  // Handle agent removal
  const handleRemoveAgent = async (agentId: string) => {
    try {
      setError(null);
      await api.delete(`/api/agents/${agentId}`);
      await fetchData();
    } catch (error) {
      console.error('Failed to remove agent:', error);
      setError('Failed to remove agent. Please try again.');
    }
  };

  if (loading) {
    return (
      <Container maxWidth="lg">
        <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
          <CircularProgress />
        </Box>
      </Container>
    );
  }

  return (
    <Container maxWidth="lg">
      <Box sx={{ my: 4 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          Agent Management
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        
        <Button
          variant="contained"
          color="primary"
          onClick={() => setOpenDialog(true)}
          sx={{ mb: 3 }}
        >
          Register New Agent
        </Button>

        {/* Active Claim Vouchers Table */}
        <Typography variant="h5" sx={{ mt: 4, mb: 2 }}>
          Active Claim Vouchers
        </Typography>
        <TableContainer component={Paper} sx={{ mb: 4 }}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Claim Code</TableCell>
                <TableCell>Created By</TableCell>
                <TableCell>Created At</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {claimVouchers.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} align="center">
                    No active claim vouchers
                  </TableCell>
                </TableRow>
              ) : (
                claimVouchers.map((voucher) => (
                  <TableRow key={voucher.code}>
                    <TableCell>{voucher.code}</TableCell>
                    <TableCell>{voucher.created_by?.username || 'Unknown'}</TableCell>
                    <TableCell>{new Date(voucher.created_at).toLocaleString()}</TableCell>
                    <TableCell>
                      <Chip 
                        label={voucher.is_continuous ? "Continuous" : "Single Use"}
                        color={voucher.is_continuous ? "primary" : "default"}
                      />
                    </TableCell>
                    <TableCell>
                      <IconButton
                        onClick={() => handleDeactivateVoucher(voucher.code)}
                        color="error"
                        title="Deactivate voucher"
                      >
                        <DeleteIcon />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>

        {/* Active Agents Table */}
        <Typography variant="h5" sx={{ mt: 4, mb: 2 }}>
          Active Agents
        </Typography>
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Agent ID</TableCell>
                <TableCell>Name</TableCell>
                <TableCell>Owner</TableCell>
                <TableCell>Version</TableCell>
                <TableCell>Hardware</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {agents.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} align="center">
                    No active agents
                  </TableCell>
                </TableRow>
              ) : (
                agents.map((agent) => (
                  <TableRow key={agent.id}>
                    <TableCell>{agent.id}</TableCell>
                    <TableCell>{agent.name}</TableCell>
                    <TableCell>{agent.createdBy?.username || 'Unknown'}</TableCell>
                    <TableCell>{agent.version}</TableCell>
                    <TableCell>
                      {agent.hardware?.cpus?.length > 0 && (
                        <Typography variant="body2">
                          CPUs: {agent.hardware.cpus.length} x {agent.hardware.cpus[0]?.model || 'Unknown'}
                        </Typography>
                      )}
                      {agent.hardware?.gpus?.map((gpu, i) => (
                        <Typography key={i} variant="body2">
                          GPU {i + 1}: {gpu.model} ({gpu.memory})
                        </Typography>
                      ))}
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={agent.status}
                        color={agent.status === 'active' ? 'success' : agent.status === 'error' ? 'error' : 'default'}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <IconButton
                        onClick={() => handleRemoveAgent(agent.id)}
                        color="error"
                        title="Remove agent"
                      >
                        <DeleteIcon />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>

        {/* Registration Dialog */}
        <Dialog 
          open={openDialog} 
          onClose={() => {
            setOpenDialog(false);
            setClaimCode('');
            setIsContinuous(false);
            setError(null);
          }}
        >
          <DialogTitle>{claimCode ? 'Generated Code' : 'Register New Agent'}</DialogTitle>
          <DialogContent>
            <Box sx={{ pt: 2 }}>
              {!claimCode && (
                <FormControlLabel
                  control={
                    <Switch
                      checked={isContinuous}
                      onChange={(e) => setIsContinuous(e.target.checked)}
                    />
                  }
                  label="Allow Continuous Registration"
                />
              )}
              {claimCode && (
                <Box sx={{ mt: 2, textAlign: 'center' }}>
                  <Typography variant="subtitle1">Claim Code:</Typography>
                  <Typography variant="h5" sx={{ mt: 1, mb: 2 }}>
                    {claimCode}
                  </Typography>
                  <Typography color="text.secondary">
                    {isContinuous 
                      ? "This code can be used multiple times until disabled."
                      : "This code can only be used once."}
                  </Typography>
                </Box>
              )}
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => {
              setOpenDialog(false);
              setClaimCode('');
              setIsContinuous(false);
              setError(null);
            }}>
              Close
            </Button>
            {!claimCode && (
              <Button onClick={handleCreateClaimCode} variant="contained">
                Generate Code
              </Button>
            )}
          </DialogActions>
        </Dialog>
      </Box>
    </Container>
  );
} 