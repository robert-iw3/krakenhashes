import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Chip,
  LinearProgress,
  Button,
  Divider,
  List,
  ListItem,
  ListItemText,
  ListItemButton,
  Tooltip,
  IconButton,
  Alert
} from '@mui/material';
import {
  Download as DownloadIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  History as HistoryIcon,
  ArrowBack as ArrowBackIcon,
  PlayArrow as PlayArrowIcon
} from '@mui/icons-material';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api } from '../../services/api';
import HashDetailModal from './HashDetailModal';
import CreateJobDialog from './CreateJobDialog';

interface HashDetail {
  id: string;
  hash_value: string;
  original_hash: string;
  username?: string;
  hash_type_id: number;
  is_cracked: boolean;
  password?: string;
  last_updated: string;
  // Frontend friendly aliases
  hash?: string;
  isCracked?: boolean;
  crackedText?: string;
}

export default function HashlistDetailView() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [selectedHash, setSelectedHash] = useState<HashDetail | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [createJobDialogOpen, setCreateJobDialogOpen] = useState(false);
  
  const { data: hashlist, isLoading } = useQuery({
    queryKey: ['hashlist', id],
    queryFn: () => api.get(`/api/hashlists/${id}`).then(res => res.data)
  });

  const { data: hashResponse, isLoading: loadingSamples, refetch: refetchHashes } = useQuery({
    queryKey: ['hashlist-samples', id],
    queryFn: () => api.get(`/api/hashlists/${id}/hashes?limit=10`).then(res => res.data),
    enabled: !!hashlist
  });

  // Extract hashes from response and normalize the data
  const hashSamples: HashDetail[] = React.useMemo(() => {
    if (!hashResponse?.hashes) return [];
    return hashResponse.hashes.map((hash: HashDetail) => ({
      ...hash,
      // Add frontend-friendly aliases
      hash: hash.hash_value,
      isCracked: hash.is_cracked,
      crackedText: hash.password
    }));
  }, [hashResponse]);

  const handleHashClick = (hash: HashDetail) => {
    // Enrich the hash detail with hashlist info
    const enrichedHash = {
      ...hash,
      hashlistName: hashlist?.name,
      hashType: hashlist?.hashTypeName
    };
    setSelectedHash(enrichedHash);
    setModalOpen(true);
  };

  const handleCloseModal = () => {
    setModalOpen(false);
    setSelectedHash(null);
  };

  const handleRefreshHashes = () => {
    refetchHashes();
  };

  if (isLoading) return <LinearProgress />;

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ mb: 2 }}>
        <Button
          startIcon={<ArrowBackIcon />}
          onClick={() => navigate('/hashlists')}
          size="small"
        >
          Back to Hashlists
        </Button>
      </Box>
      
      <Paper sx={{ p: 3, mb: 3 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center">
          <Typography variant="h5">{hashlist.name}</Typography>
          <Box display="flex" gap={1}>
            <Button
              variant="contained"
              startIcon={<PlayArrowIcon />}
              onClick={() => setCreateJobDialogOpen(true)}
              disabled={hashlist.status !== 'ready'}
            >
              Create Job
            </Button>
            <Tooltip title="Download">
              <IconButton>
                <DownloadIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title="Delete">
              <IconButton color="error">
                <DeleteIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>

        <Typography variant="subtitle1" color="text.secondary" sx={{ mt: 1 }}>
          {hashlist.description || 'No description'}
        </Typography>

        <Box display="flex" gap={2} sx={{ mt: 3 }}>
          <Typography>
            Status: <Chip 
              label={hashlist.status} 
              color={
                hashlist.status === 'ready' ? 'success' :
                hashlist.status === 'error' ? 'error' : 'primary'
              }
            />
          </Typography>
          <Typography>
            Hash Type: {hashlist.hashTypeName}
          </Typography>
          <Typography>
            Created: {new Date(hashlist.createdAt).toLocaleString()}
          </Typography>
        </Box>

        <Box sx={{ mt: 3 }}>
          <Typography variant="subtitle2">
            Crack Progress ({hashlist.crackedHashes || 0} of {hashlist.totalHashes || 0})
          </Typography>
          <Box display="flex" alignItems="center" gap={2}>
            <Box width="100%">
              <LinearProgress 
                variant="determinate"
                value={(hashlist.crackedHashes / hashlist.totalHashes) * 100}
              />
            </Box>
            <Typography>
              {Math.round((hashlist.crackedHashes / hashlist.totalHashes) * 100)}%
            </Typography>
          </Box>
        </Box>
      </Paper>

      <Paper sx={{ p: 3, mb: 3 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center">
          <Typography variant="h6">Sample Hashes</Typography>
          <Button startIcon={<RefreshIcon />} size="small" onClick={handleRefreshHashes}>Refresh</Button>
        </Box>
        <Divider sx={{ my: 2 }} />
        {loadingSamples ? (
          <LinearProgress />
        ) : hashSamples.length === 0 ? (
          <Alert severity="info">No hashes found in this hashlist.</Alert>
        ) : (
          <List>
            {hashSamples.map((hash: HashDetail) => (
            <ListItemButton 
              key={hash.id}
              onClick={() => handleHashClick(hash)}
              sx={{ 
                borderRadius: 1,
                mb: 0.5,
                '&:hover': {
                  backgroundColor: 'action.hover'
                }
              }}
            >
              <ListItemText 
                primary={
                  <Typography 
                    variant="body2" 
                    sx={{ 
                      fontFamily: 'monospace',
                      wordBreak: 'break-all',
                      cursor: 'pointer',
                      '&:hover': {
                        textDecoration: 'underline'
                      }
                    }}
                  >
                    {hash.hash_value}
                  </Typography>
                }
                secondary={
                  <Box>
                    {hash.username && <Typography variant="caption" display="block">User: {hash.username}</Typography>}
                    {hash.is_cracked ? `Cracked: ${hash.password}` : 'Not cracked'}
                  </Box>
                }
              />
              <Chip 
                label={hash.is_cracked ? 'Cracked' : 'Pending'}
                color={hash.is_cracked ? 'success' : 'default'}
                size="small"
              />
            </ListItemButton>
            ))}
          </List>
        )}
      </Paper>

      <Paper sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>
          <HistoryIcon sx={{ verticalAlign: 'middle', mr: 1 }} />
          History
        </Typography>
        <Divider sx={{ mb: 2 }} />
        <Typography color="text.secondary">
          History log will appear here
        </Typography>
      </Paper>

      <HashDetailModal 
        open={modalOpen}
        onClose={handleCloseModal}
        hash={selectedHash}
      />
      
      {hashlist && (
        <CreateJobDialog
          open={createJobDialogOpen}
          onClose={() => setCreateJobDialogOpen(false)}
          hashlistId={parseInt(id!)}
          hashlistName={hashlist.name}
          hashTypeId={hashlist.hashTypeID || hashlist.hash_type_id}
        />
      )}
    </Box>
  );
}