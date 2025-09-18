import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  Typography,
  CircularProgress,
  Alert,
  MenuItem,
  FormControl,
  Select,
  InputLabel,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  InputAdornment,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Search as SearchIcon,
  ContentCopy as CopyIcon,
  Download as DownloadIcon,
} from '@mui/icons-material';
import { api } from '../../services/api';
import { useSnackbar } from 'notistack';
import { CrackedHash, PotResponse } from '../../services/pot';

interface PotTableProps {
  title: string;
  fetchData: (limit: number, offset: number) => Promise<PotResponse>;
  filterParam?: string;
  filterValue?: string;
  contextType: 'master' | 'hashlist' | 'client' | 'job';
  contextName: string;
  contextId?: string;
}

export default function PotTable({ title, fetchData, filterParam, filterValue, contextType, contextName, contextId }: PotTableProps) {
  const [data, setData] = useState<CrackedHash[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(500);
  const [totalCount, setTotalCount] = useState(0);
  const [searchTerm, setSearchTerm] = useState('');
  const [openAllConfirm, setOpenAllConfirm] = useState(false);
  const [hasUsernameData, setHasUsernameData] = useState(false);
  const [checkedForUsernames, setCheckedForUsernames] = useState(false);
  const [downloadingFormat, setDownloadingFormat] = useState<string | null>(null);
  const { enqueueSnackbar } = useSnackbar();

  const pageSizeOptions = [500, 1000, 1500, 2000, -1];

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const limit = rowsPerPage === -1 ? 999999 : rowsPerPage;
      const offset = page * (rowsPerPage === -1 ? 0 : rowsPerPage);
      
      const response = await fetchData(limit, offset);
      setData(response.hashes);
      setTotalCount(response.total_count);
      
      // Check if any hash has username data
      const hasUsername = response.hashes.some(hash => hash.username && hash.username.trim() !== '');
      setHasUsernameData(hasUsername);
    } catch (err) {
      console.error('Error loading pot data:', err);
      setError('Failed to load cracked hashes');
      enqueueSnackbar('Failed to load cracked hashes', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, rowsPerPage, fetchData, enqueueSnackbar]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Check for username data in the entire dataset on mount
  useEffect(() => {
    if (!checkedForUsernames && totalCount > 0) {
      // Make a quick request to check if any usernames exist
      // We'll check the current data, and if we don't find any, we could make a separate call
      // For now, let's just check current data and set it as checked
      setCheckedForUsernames(true);
    }
  }, [totalCount, checkedForUsernames]);

  const handleChangePage = (event: unknown, newPage: number) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newRowsPerPage = parseInt(event.target.value, 10);
    
    if (newRowsPerPage === -1) {
      setOpenAllConfirm(true);
    } else {
      setRowsPerPage(newRowsPerPage);
      setPage(0);
    }
  };

  const handleConfirmAll = () => {
    setRowsPerPage(-1);
    setPage(0);
    setOpenAllConfirm(false);
    enqueueSnackbar('Loading all results. This may take some time...', { variant: 'info' });
  };

  const handleCancelAll = () => {
    setOpenAllConfirm(false);
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    enqueueSnackbar('Copied to clipboard', { variant: 'success' });
  };

  const downloadFormat = async (format: 'hash-pass' | 'user-pass' | 'user' | 'pass') => {
    try {
      setDownloadingFormat(format);
      
      // Build the download URL based on context
      let url = '';
      if (contextType === 'master') {
        url = `/api/pot/download/${format}`;
      } else if (contextType === 'hashlist' && contextId) {
        url = `/api/pot/hashlist/${contextId}/download/${format}`;
      } else if (contextType === 'client' && contextId) {
        url = `/api/pot/client/${contextId}/download/${format}`;
      }
      
      const response = await api.get(url, { responseType: 'blob' });
      
      // Create blob and download
      const blob = new Blob([response.data], { type: 'text/plain' });
      const downloadUrl = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = downloadUrl;
      
      // Get filename from Content-Disposition header or use default
      const contentDisposition = response.headers['content-disposition'];
      let filename = `${contextName}-${format.replace('-', '-')}.lst`;
      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename="?(.+)"?/i);
        if (filenameMatch) {
          filename = filenameMatch[1];
        }
      }
      
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(downloadUrl);
      
      enqueueSnackbar(`Downloaded ${filename}`, { variant: 'success' });
    } catch (err) {
      console.error('Error downloading format:', err);
      enqueueSnackbar('Failed to download file', { variant: 'error' });
    } finally {
      setDownloadingFormat(null);
    }
  };

  const exportData = () => {
    const exportText = data
      .map(hash => `${hash.original_hash}:${hash.password}`)
      .join('\n');
    
    const blob = new Blob([exportText], { type: 'text/plain' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `cracked_hashes_${new Date().toISOString().split('T')[0]}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
    
    enqueueSnackbar('Exported cracked hashes', { variant: 'success' });
  };

  const filteredData = data.filter(hash => {
    if (!searchTerm) return true;
    const searchLower = searchTerm.toLowerCase();
    return (
      hash.original_hash.toLowerCase().includes(searchLower) ||
      hash.password.toLowerCase().includes(searchLower) ||
      (hash.username && hash.username.toLowerCase().includes(searchLower))
    );
  });

  if (loading && data.length === 0) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight={400}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mt: 2 }}>
        {error}
      </Alert>
    );
  }

  return (
    <Paper sx={{ width: '100%', mb: 2 }}>
      <Box sx={{ p: 2 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2, flexWrap: 'wrap', gap: 2 }}>
          <Typography variant="h6" component="div">
            {title}
            {filterParam && filterValue && (
              <Typography variant="body2" color="text.secondary">
                Filtered by {filterParam}: {filterValue}
              </Typography>
            )}
          </Typography>
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', flexWrap: 'wrap' }}>
            <TextField
              size="small"
              placeholder="Search hashes..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon />
                  </InputAdornment>
                ),
              }}
            />
            <Tooltip title="Export visible results">
              <IconButton onClick={exportData} disabled={filteredData.length === 0}>
                <DownloadIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>
        
        <Box sx={{ display: 'flex', gap: 1, mb: 2, flexWrap: 'wrap' }}>
          <Button
            size="small"
            variant="outlined"
            startIcon={<DownloadIcon />}
            onClick={() => downloadFormat('hash-pass')}
            disabled={downloadingFormat !== null}
          >
            Hash:Pass
          </Button>
          <Button
            size="small"
            variant="outlined"
            startIcon={<DownloadIcon />}
            onClick={() => downloadFormat('user-pass')}
            disabled={downloadingFormat !== null || !hasUsernameData}
          >
            User:Pass
          </Button>
          <Button
            size="small"
            variant="outlined"
            startIcon={<DownloadIcon />}
            onClick={() => downloadFormat('user')}
            disabled={downloadingFormat !== null || !hasUsernameData}
          >
            Username
          </Button>
          <Button
            size="small"
            variant="outlined"
            startIcon={<DownloadIcon />}
            onClick={() => downloadFormat('pass')}
            disabled={downloadingFormat !== null}
          >
            Password
          </Button>
        </Box>
        
        <TableContainer>
          <Table size="small" aria-label="cracked hashes table">
            <TableHead>
              <TableRow>
                <TableCell>Original Hash</TableCell>
                <TableCell>Password</TableCell>
                <TableCell>Username</TableCell>
                <TableCell>Hash Type</TableCell>
                <TableCell align="center">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredData.map((hash) => (
                <TableRow key={hash.id} hover>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                    {hash.original_hash}
                  </TableCell>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                    {hash.password}
                  </TableCell>
                  <TableCell>{hash.username || '-'}</TableCell>
                  <TableCell>{hash.hash_type_id}</TableCell>
                  <TableCell align="center">
                    <Tooltip title="Copy hash:password">
                      <IconButton
                        size="small"
                        onClick={() => copyToClipboard(`${hash.original_hash}:${hash.password}`)}
                      >
                        <CopyIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
        
        <TablePagination
          rowsPerPageOptions={pageSizeOptions.map(size => ({
            label: size === -1 ? 'All' : size.toString(),
            value: size,
          }))}
          component="div"
          count={totalCount}
          rowsPerPage={rowsPerPage === -1 ? totalCount : rowsPerPage}
          page={page}
          onPageChange={handleChangePage}
          onRowsPerPageChange={handleChangeRowsPerPage}
        />
      </Box>

      <Dialog open={openAllConfirm} onClose={handleCancelAll}>
        <DialogTitle>Load All Results?</DialogTitle>
        <DialogContent>
          <Typography>
            Loading all {totalCount.toLocaleString()} results may take a significant amount of time 
            and could impact performance. Are you sure you want to continue?
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCancelAll}>Cancel</Button>
          <Button onClick={handleConfirmAll} variant="contained" color="primary">
            Load All
          </Button>
        </DialogActions>
      </Dialog>
    </Paper>
  );
}