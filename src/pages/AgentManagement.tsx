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

import { useState } from 'react';
import {
  Box,
  Button,
  Container,
  Typography,
  Paper,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Switch,
  FormControlLabel,
} from '@mui/material';
import { Agent, AgentRegistrationForm } from '../types/agent';

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
  const [openDialog, setOpenDialog] = useState(false);
  const [registrationForm, setRegistrationForm] = useState<AgentRegistrationForm>({
    name: '',
    teamId: 0,
    continuous: false,
  });
  const [claimCode, setClaimCode] = useState<string>('');

  const handleRegisterAgent = async () => {
    try {
      const response = await fetch('/api/agents/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(registrationForm),
      });

      const data = await response.json();
      setClaimCode(data.claimCode);
      // Don't close dialog yet, show claim code to user
    } catch (error) {
      console.error('Failed to register agent:', error);
    }
  };

  return (
    <Container maxWidth="lg">
      <Box sx={{ my: 4 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          Agent Management
        </Typography>
        
        <Button
          variant="contained"
          color="primary"
          onClick={() => setOpenDialog(true)}
          sx={{ mb: 3 }}
        >
          Register New Agent
        </Button>

        {/* Agent List */}
        <Box sx={{ display: 'grid', gap: 2 }}>
          {agents.map((agent) => (
            <Paper key={agent.id} sx={{ p: 2 }}>
              <Typography variant="h6">{agent.name}</Typography>
              <Typography>Status: {agent.status}</Typography>
              <Typography>
                Last Heartbeat: {new Date(agent.lastHeartbeat).toLocaleString()}
              </Typography>
            </Paper>
          ))}
        </Box>

        {/* Registration Dialog */}
        <Dialog open={openDialog} onClose={() => setOpenDialog(false)}>
          <DialogTitle>Register New Agent</DialogTitle>
          <DialogContent>
            {!claimCode ? (
              <Box sx={{ pt: 2 }}>
                <TextField
                  fullWidth
                  label="Agent Name"
                  value={registrationForm.name}
                  onChange={(e) =>
                    setRegistrationForm({
                      ...registrationForm,
                      name: e.target.value,
                    })
                  }
                  sx={{ mb: 2 }}
                />
                <FormControlLabel
                  control={
                    <Switch
                      checked={registrationForm.continuous}
                      onChange={(e) =>
                        setRegistrationForm({
                          ...registrationForm,
                          continuous: e.target.checked,
                        })
                      }
                    />
                  }
                  label="Allow Continuous Registration"
                />
              </Box>
            ) : (
              <Box sx={{ pt: 2 }}>
                <Typography>Claim Code:</Typography>
                <Typography variant="h5" sx={{ mt: 1, mb: 2 }}>
                  {claimCode}
                </Typography>
                <Typography color="text.secondary">
                  Use this code to register your agent. 
                  {registrationForm.continuous 
                    ? " This code can be used multiple times until disabled."
                    : " This code can only be used once."}
                </Typography>
              </Box>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => {
              setOpenDialog(false);
              setClaimCode('');
              setRegistrationForm({
                name: '',
                teamId: 0,
                continuous: false,
              });
            }}>
              {claimCode ? 'Close' : 'Cancel'}
            </Button>
            {!claimCode && (
              <Button onClick={handleRegisterAgent} variant="contained">
                Generate Claim Code
              </Button>
            )}
          </DialogActions>
        </Dialog>
      </Box>
    </Container>
  );
} 