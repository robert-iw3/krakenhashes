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
  Tooltip,
  IconButton
} from '@mui/material';
import {
  Download as DownloadIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  History as HistoryIcon
} from '@mui/icons-material';
import { useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api } from '../../services/api';

interface HashDetail {
  hash: string;
  isCracked: boolean;
  crackedText?: string;
}

export default function HashlistDetailView() {
  const { id } = useParams();
  const { data: hashlist, isLoading } = useQuery({
    queryKey: ['hashlist', id],
    queryFn: () => api.get(`/hashlists/${id}`).then(res => res.data)
  });

  const { data: hashSamples = [], isLoading: loadingSamples } = useQuery({
    queryKey: ['hashlist-samples', id],
    queryFn: () => api.get(`/hashlists/${id}/hashes?limit=10`).then(res => res.data),
    enabled: !!hashlist
  });

  if (isLoading) return <LinearProgress />;

  return (
    <Box sx={{ p: 3 }}>
      <Paper sx={{ p: 3, mb: 3 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center">
          <Typography variant="h5">{hashlist.name}</Typography>
          <Box>
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
          <Button startIcon={<RefreshIcon />} size="small">Refresh</Button>
        </Box>
        <Divider sx={{ my: 2 }} />
        <List>
          {hashSamples.map((hash: HashDetail) => (
            <ListItem key={hash.hash}>
              <ListItemText 
                primary={hash.hash}
                secondary={hash.isCracked ? `Cracked: ${hash.crackedText}` : 'Not cracked'}
              />
              <Chip 
                label={hash.isCracked ? 'Cracked' : 'Pending'}
                color={hash.isCracked ? 'success' : 'default'}
                size="small"
              />
            </ListItem>
          ))}
        </List>
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
    </Box>
  );
}