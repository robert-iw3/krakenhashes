/**
 * Wordlists Management page for KrakenHashes frontend.
 * 
 * Features:
 *   - View wordlists
 *   - Add new wordlists
 *   - Update wordlist information
 *   - Delete wordlists
 *   - Enable/disable wordlists
 */
import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Button,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  IconButton,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  MenuItem,
  Grid,
  Divider,
  Switch,
  FormControlLabel,
  CircularProgress,
  Alert,
  Tooltip,
  InputAdornment,
  Toolbar,
  alpha,
  Tab,
  Tabs,
  Checkbox,
  FormControl,
  InputLabel,
  Select
} from '@mui/material';
import { 
  Delete as DeleteIcon, 
  Edit as EditIcon, 
  Refresh as RefreshIcon, 
  CloudDownload as DownloadIcon,
  Search as SearchIcon,
  Add as AddIcon,
  Check as CheckIcon,
  Clear as ClearIcon,
  Verified as VerifiedIcon
} from '@mui/icons-material';
import FileUpload from '../components/common/FileUpload';
import { Wordlist, WordlistStatus, WordlistType } from '../types/wordlists';
import * as wordlistService from '../services/wordlists';
import { useSnackbar } from 'notistack';
import { formatFileSize } from '../utils/formatters';

export default function WordlistsManagement() {
  const [wordlists, setWordlists] = useState<Wordlist[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [openUploadDialog, setOpenUploadDialog] = useState(false);
  const [openEditDialog, setOpenEditDialog] = useState(false);
  const [currentWordlist, setCurrentWordlist] = useState<Wordlist | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [nameEdit, setNameEdit] = useState('');
  const [descriptionEdit, setDescriptionEdit] = useState('');
  const [wordlistTypeEdit, setWordlistTypeEdit] = useState<WordlistType>(WordlistType.GENERAL);
  const [formatEdit, setFormatEdit] = useState('plaintext');
  const [tabValue, setTabValue] = useState(0);
  const [sortBy, setSortBy] = useState<keyof Wordlist>('updated_at');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
  const { enqueueSnackbar } = useSnackbar();
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);
  const [selectedWordlistType, setSelectedWordlistType] = useState<WordlistType>(WordlistType.GENERAL);
  const [isLoading, setIsLoading] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [wordlistToDelete, setWordlistToDelete] = useState<{id: string, name: string} | null>(null);

  // Fetch wordlists
  const fetchWordlists = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await wordlistService.getWordlists();
      setWordlists(response.data);
    } catch (err) {
      console.error('Error fetching wordlists:', err);
      setError('Failed to load wordlists');
      enqueueSnackbar('Failed to load wordlists', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [enqueueSnackbar]);

  useEffect(() => {
    fetchWordlists();
  }, [fetchWordlists]);

  // Handle file upload
  const handleUploadWordlist = async (formData: FormData) => {
    try {
      setIsLoading(true);
      
      // Add the wordlist type to the form data
      formData.append('wordlist_type', selectedWordlistType);
      
      // Add required fields if not present
      if (!formData.has('name')) {
        const file = formData.get('file') as File;
        if (file) {
          formData.append('name', file.name.split('.')[0]);
        }
      }
      
      if (!formData.has('format')) {
        const file = formData.get('file') as File;
        if (file) {
          const extension = file.name.split('.').pop()?.toLowerCase() || 'txt';
          // Map file extension to the correct format enum value
          const format = ['gz', 'zip'].includes(extension) ? 'compressed' : 'plaintext';
          formData.append('format', format);
          console.debug(`[Wordlist Upload] Mapped file extension '${extension}' to format '${format}'`);
        } else {
          formData.append('format', 'plaintext');
        }
      }
      
      console.debug('[Wordlist Upload] Sending form data:', 
        Array.from(formData.entries()).reduce((obj, [key, val]) => {
          obj[key] = key === 'file' ? '(file content)' : val;
          return obj;
        }, {} as Record<string, any>)
      );
      
      console.debug('[Wordlist Upload] Authentication cookies before upload:', document.cookie);
      console.debug('[Wordlist Upload] Upload URL:', '/api/wordlists/upload');
      
      try {
        const response = await wordlistService.uploadWordlist(formData, (progress, eta, speed) => {
          // Update progress in the FileUpload component
          const progressEvent = new CustomEvent('upload-progress', { detail: { progress, eta, speed } });
          document.dispatchEvent(progressEvent);
        });
        console.debug('[Wordlist Upload] Upload successful:', response);
        
        // Check if the response indicates a duplicate wordlist
        if (response.data.duplicate) {
          enqueueSnackbar(`Wordlist "${response.data.name}" already exists`, { variant: 'info' });
        } else {
          enqueueSnackbar('Wordlist uploaded successfully', { variant: 'success' });
        }
        
        setUploadDialogOpen(false);
        fetchWordlists();
      } catch (uploadError) {
        console.error('[Wordlist Upload] Upload error details:', uploadError);
        console.debug('[Wordlist Upload] Authentication cookies after error:', document.cookie);
        throw uploadError;
      }
      
      console.debug('[Wordlist Upload] Authentication cookies after upload:', document.cookie);
    } catch (error) {
      console.error('Error uploading wordlist:', error);
      enqueueSnackbar('Failed to upload wordlist', { variant: 'error' });
    } finally {
      setIsLoading(false);
    }
  };

  // Handle wordlist deletion
  const handleDelete = async (id: string, name: string) => {
    try {
      await wordlistService.deleteWordlist(id);
      enqueueSnackbar(`Wordlist "${name}" deleted successfully`, { variant: 'success' });
      fetchWordlists();
    } catch (err: any) {
      console.error('Error deleting wordlist:', err);
      // Extract error message from axios response
      const errorMessage = err.response?.data?.error || 'Failed to delete wordlist';
      enqueueSnackbar(errorMessage, { variant: 'error' });
    } finally {
      setDeleteDialogOpen(false);
      setWordlistToDelete(null);
    }
  };

  // Open delete confirmation dialog
  const openDeleteDialog = (id: string, name: string) => {
    setWordlistToDelete({ id, name });
    setDeleteDialogOpen(true);
  };

  // Close delete confirmation dialog
  const closeDeleteDialog = () => {
    setDeleteDialogOpen(false);
    setWordlistToDelete(null);
  };

  // Handle wordlist download
  const handleDownload = async (id: string, name: string) => {
    try {
      // Direct download without loading into memory
      // This allows streaming of large files
      const link = document.createElement('a');
      link.href = `/api/wordlists/${id}/download`;
      link.setAttribute('download', `${name}.txt`);
      link.style.display = 'none';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } catch (err) {
      console.error('Error downloading wordlist:', err);
      enqueueSnackbar('Failed to download wordlist', { variant: 'error' });
    }
  };

  // Handle edit button click
  const handleEditClick = (wordlist: Wordlist) => {
    setCurrentWordlist(wordlist);
    setNameEdit(wordlist.name);
    setDescriptionEdit(wordlist.description);
    setWordlistTypeEdit(wordlist.wordlist_type);
    setFormatEdit(wordlist.format);
    setOpenEditDialog(true);
  };

  // Handle save edit
  const handleSaveEdit = async () => {
    if (!currentWordlist) return;
    
    try {
      console.debug('[Wordlist Edit] Updating wordlist:', currentWordlist.id, {
        name: nameEdit,
        description: descriptionEdit,
        wordlist_type: wordlistTypeEdit
      });
      
      const response = await wordlistService.updateWordlist(currentWordlist.id, {
        name: nameEdit,
        description: descriptionEdit,
        wordlist_type: wordlistTypeEdit
      });
      
      console.debug('[Wordlist Edit] Update successful:', response);
      enqueueSnackbar('Wordlist updated successfully', { variant: 'success' });
      setOpenEditDialog(false);
      fetchWordlists();
    } catch (err: any) {
      console.error('[Wordlist Edit] Error updating wordlist:', err);
      
      if (err.response?.status === 401) {
        enqueueSnackbar('Your session has expired. Please log in again.', { variant: 'error' });
      } else {
        enqueueSnackbar('Failed to update wordlist: ' + (err.response?.data?.message || err.message), { variant: 'error' });
      }
    }
  };

  // Handle sort change
  const handleSortChange = (column: keyof Wordlist) => {
    if (sortBy === column) {
      // If already sorting by this column, toggle order
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      // Otherwise, sort by this column in ascending order
      setSortBy(column);
      setSortOrder('asc');
    }
  };

  // Render sort label
  const renderSortLabel = (column: keyof Wordlist, label: string) => {
    return (
      <TableSortLabel
        active={sortBy === column}
        direction={sortBy === column ? sortOrder : 'asc'}
        onClick={() => handleSortChange(column)}
      >
        {label}
      </TableSortLabel>
    );
  };

  // Filter and sort wordlists
  const filteredWordlists = wordlists
    .filter(wordlist => {
      // Filter by search term
      const matchesSearch = wordlist.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           wordlist.description.toLowerCase().includes(searchTerm.toLowerCase());
      
      // Filter by tab
      if (tabValue === 0) return matchesSearch; // All
      if (tabValue === 1) return matchesSearch && wordlist.wordlist_type === WordlistType.GENERAL;
      if (tabValue === 2) return matchesSearch && wordlist.wordlist_type === WordlistType.SPECIALIZED;
      if (tabValue === 3) return matchesSearch && wordlist.wordlist_type === WordlistType.TARGETED;
      if (tabValue === 4) return matchesSearch && wordlist.wordlist_type === WordlistType.CUSTOM;
      
      return matchesSearch;
    })
    .sort((a, b) => {
      // Handle special cases for non-string fields
      if (sortBy === 'file_size' || sortBy === 'word_count') {
        return sortOrder === 'asc' 
          ? a[sortBy] - b[sortBy] 
          : b[sortBy] - a[sortBy];
      }
      
      // Handle date fields
      if (sortBy === 'created_at' || sortBy === 'updated_at' || sortBy === 'last_verified_at') {
        const dateA = new Date(a[sortBy] || 0).getTime();
        const dateB = new Date(b[sortBy] || 0).getTime();
        return sortOrder === 'asc' ? dateA - dateB : dateB - dateA;
      }
      
      // Default string comparison
      const valueA = String(a[sortBy] || '').toLowerCase();
      const valueB = String(b[sortBy] || '').toLowerCase();
      return sortOrder === 'asc' 
        ? valueA.localeCompare(valueB) 
        : valueB.localeCompare(valueA);
    });

  // Render status chip based on verification status
  const renderStatusChip = (status: string) => {
    switch (status) {
      case 'verified':
        return <Chip label="Verified" color="success" size="small" />;
      case 'pending':
        return <Chip label="Pending" color="warning" size="small" />;
      case 'failed':
        return <Chip label="Failed" color="error" size="small" />;
      default:
        return <Chip label={status} color="default" size="small" />;
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Grid container spacing={2} alignItems="center" sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6}>
            <Typography variant="h4" component="h1" gutterBottom>
              Wordlist Management
            </Typography>
            <Typography variant="body1" color="text.secondary">
              Manage wordlists for password cracking
            </Typography>
          </Grid>
          <Grid item xs={12} sm={6} sx={{ textAlign: { xs: 'left', sm: 'right' } }}>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={() => setUploadDialogOpen(true)}
              sx={{ mr: 1 }}
              disabled={isLoading}
            >
              Upload Wordlist
            </Button>
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={() => fetchWordlists()}
            >
              Refresh
            </Button>
          </Grid>
        </Grid>

        {error && (
          <Alert severity="error" sx={{ mb: 3 }}>
            {error}
          </Alert>
        )}

        <Paper sx={{ mb: 3, overflow: 'hidden' }}>
          <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
            <Tabs 
              value={tabValue} 
              onChange={(_, newValue) => setTabValue(newValue)}
              aria-label="wordlist tabs"
            >
              <Tab label="All Wordlists" id="tab-0" />
              <Tab label="General" id="tab-1" />
              <Tab label="Specialized" id="tab-2" />
              <Tab label="Targeted" id="tab-3" />
              <Tab label="Custom" id="tab-4" />
            </Tabs>
          </Box>
          
          <Toolbar
            sx={{
              pl: { sm: 2 },
              pr: { xs: 1, sm: 1 },
              display: 'flex',
              justifyContent: 'center'
            }}
          >
            <TextField
              margin="dense"
              placeholder="Search wordlists..."
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon />
                  </InputAdornment>
                ),
                endAdornment: searchTerm && (
                  <InputAdornment position="end">
                    <IconButton size="small" onClick={() => setSearchTerm('')}>
                      <ClearIcon />
                    </IconButton>
                  </InputAdornment>
                )
              }}
              size="small"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              sx={{ width: { xs: '100%', sm: '60%', md: '40%' } }}
            />
          </Toolbar>
          
          <Divider />
          
          <TableContainer>
            <Table sx={{ minWidth: 650 }} aria-label="wordlists table">
              <TableHead>
                <TableRow>
                  <TableCell>
                    {renderSortLabel('name', 'Name')}
                  </TableCell>
                  <TableCell>
                    {renderSortLabel('verification_status', 'Status')}
                  </TableCell>
                  <TableCell>
                    {renderSortLabel('wordlist_type', 'Type')}
                  </TableCell>
                  <TableCell>
                    {renderSortLabel('file_size', 'Size')}
                  </TableCell>
                  <TableCell>
                    {renderSortLabel('word_count', 'Word Count')}
                  </TableCell>
                  <TableCell>
                    {renderSortLabel('updated_at', 'Updated')}
                  </TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={7} align="center" sx={{ py: 3 }}>
                      <CircularProgress size={40} />
                      <Typography variant="body2" sx={{ mt: 1 }}>
                        Loading wordlists...
                      </Typography>
                    </TableCell>
                  </TableRow>
                ) : filteredWordlists.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} align="center" sx={{ py: 3 }}>
                      <Typography variant="body1">
                        No wordlists found
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                        {searchTerm ? 'Try a different search term' : 'Upload a wordlist to get started'}
                      </Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredWordlists.map((wordlist) => (
                    <TableRow key={wordlist.id}>
                      <TableCell>
                        <Box>
                          <Typography variant="body2" fontWeight="medium">
                            {wordlist.name}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            {wordlist.description || 'No description provided'}
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        {renderStatusChip(wordlist.verification_status)}
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={wordlist.wordlist_type}
                          size="small"
                          color="primary"
                          variant="outlined"
                          sx={{ textTransform: 'capitalize' }}
                        />
                      </TableCell>
                      <TableCell>
                        {formatFileSize(wordlist.file_size)}
                      </TableCell>
                      <TableCell>
                        {wordlist.word_count.toLocaleString()}
                      </TableCell>
                      <TableCell>
                        {new Date(wordlist.updated_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell align="right">
                        <Tooltip title="Download">
                          <IconButton
                            onClick={() => handleDownload(wordlist.id, wordlist.name)}
                            disabled={wordlist.verification_status !== 'verified'}
                          >
                            <DownloadIcon />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Edit">
                          <IconButton
                            onClick={() => handleEditClick(wordlist)}
                          >
                            <EditIcon />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Delete">
                          <IconButton
                            color="error"
                            onClick={() => openDeleteDialog(wordlist.id, wordlist.name)}
                          >
                            <DeleteIcon />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>

      {/* Upload Dialog */}
      <Dialog
        open={uploadDialogOpen}
        onClose={() => !isLoading && setUploadDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Upload Wordlist</DialogTitle>
        <DialogContent>
          <FileUpload
            title="Upload a wordlist file"
            description="Select a wordlist file to upload. Supported formats: .txt, .dict, .lst, .gz, .zip"
            acceptedFileTypes=".txt,.dict,.lst,.gz,.zip"
            onUpload={handleUploadWordlist}
            uploadButtonText="Upload Wordlist"
            additionalFields={
              <FormControl fullWidth margin="normal">
                <InputLabel id="wordlist-type-label">Wordlist Type</InputLabel>
                <Select
                  labelId="wordlist-type-label"
                  id="wordlist-type"
                  name="wordlist_type"
                  value={selectedWordlistType}
                  onChange={(e) => setSelectedWordlistType(e.target.value as WordlistType)}
                  label="Wordlist Type"
                >
                  <MenuItem value={WordlistType.GENERAL}>General</MenuItem>
                  <MenuItem value={WordlistType.SPECIALIZED}>Specialized</MenuItem>
                  <MenuItem value={WordlistType.TARGETED}>Targeted</MenuItem>
                  <MenuItem value={WordlistType.CUSTOM}>Custom</MenuItem>
                </Select>
              </FormControl>
            }
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setUploadDialogOpen(false)} color="primary" disabled={isLoading}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog
        open={openEditDialog}
        onClose={() => setOpenEditDialog(false)}
        aria-labelledby="edit-dialog-title"
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle id="edit-dialog-title">Edit Wordlist</DialogTitle>
        <DialogContent>
          <TextField
            margin="dense"
            label="Name"
            fullWidth
            value={nameEdit}
            onChange={(e) => setNameEdit(e.target.value)}
            sx={{ mb: 2 }}
          />
          <TextField
            margin="dense"
            label="Description"
            fullWidth
            multiline
            rows={3}
            value={descriptionEdit}
            onChange={(e) => setDescriptionEdit(e.target.value)}
            sx={{ mb: 2 }}
          />
          <FormControl fullWidth margin="dense" sx={{ mb: 2 }}>
            <InputLabel id="edit-wordlist-type-label">Wordlist Type</InputLabel>
            <Select
              labelId="edit-wordlist-type-label"
              id="edit-wordlist-type"
              value={wordlistTypeEdit}
              onChange={(e) => setWordlistTypeEdit(e.target.value as WordlistType)}
              label="Wordlist Type"
            >
              <MenuItem value={WordlistType.GENERAL}>General</MenuItem>
              <MenuItem value={WordlistType.SPECIALIZED}>Specialized</MenuItem>
              <MenuItem value={WordlistType.TARGETED}>Targeted</MenuItem>
              <MenuItem value={WordlistType.CUSTOM}>Custom</MenuItem>
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenEditDialog(false)}>
            Cancel
          </Button>
          <Button onClick={handleSaveEdit} variant="contained" color="primary">
            Save Changes
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={closeDeleteDialog}
        aria-labelledby="delete-dialog-title"
        aria-describedby="delete-dialog-description"
      >
        <DialogTitle id="delete-dialog-title">Confirm Deletion</DialogTitle>
        <DialogContent>
          <Typography variant="body1" id="delete-dialog-description">
            Are you sure you want to delete wordlist "{wordlistToDelete?.name}"? This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeDeleteDialog}>Cancel</Button>
          <Button 
            onClick={() => wordlistToDelete && handleDelete(wordlistToDelete.id, wordlistToDelete.name)} 
            color="error" 
            variant="contained"
            autoFocus
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
} 