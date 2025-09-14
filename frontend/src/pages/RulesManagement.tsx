/**
 * Rules Management page for KrakenHashes frontend.
 * 
 * Features:
 *   - View rules
 *   - Add new rules
 *   - Update rule information
 *   - Delete rules
 *   - Enable/disable rules
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
import { Rule, RuleStatus, RuleType } from '../types/rules';
import * as ruleService from '../services/rules';
import { useSnackbar } from 'notistack';
import { formatFileSize } from '../utils/formatters';

export default function RulesManagement() {
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [openUploadDialog, setOpenUploadDialog] = useState(false);
  const [openEditDialog, setOpenEditDialog] = useState(false);
  const [currentRule, setCurrentRule] = useState<Rule | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [nameEdit, setNameEdit] = useState('');
  const [descriptionEdit, setDescriptionEdit] = useState('');
  const [ruleTypeEdit, setRuleTypeEdit] = useState<RuleType>(RuleType.HASHCAT);
  const [tabValue, setTabValue] = useState(0);
  const [sortBy, setSortBy] = useState<keyof Rule>('updated_at');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
  const { enqueueSnackbar } = useSnackbar();
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);
  const [selectedRuleType, setSelectedRuleType] = useState<RuleType>(RuleType.HASHCAT);
  const [isLoading, setIsLoading] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [ruleToDelete, setRuleToDelete] = useState<{id: string, name: string} | null>(null);

  // Fetch rules
  const fetchRules = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await ruleService.getRules();
      setRules(response.data);
    } catch (err) {
      console.error('Error fetching rules:', err);
      setError('Failed to load rules');
      enqueueSnackbar('Failed to load rules', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [enqueueSnackbar]);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  // Handle file upload
  const handleUploadRule = async (formData: FormData) => {
    try {
      setIsLoading(true);
      
      // Add the rule type to the form data
      formData.append('rule_type', selectedRuleType);
      
      // Add required fields if not present
      if (!formData.has('name')) {
        const file = formData.get('file') as File;
        if (file) {
          // Extract name without extension (everything before the last dot)
          const lastDotIndex = file.name.lastIndexOf('.');
          const nameWithoutExt = lastDotIndex > 0 ? file.name.substring(0, lastDotIndex) : file.name;
          formData.append('name', nameWithoutExt);
        }
      }
      
      // Remove format field as it's not needed for rules
      if (formData.has('format')) {
        formData.delete('format');
      }
      
      console.debug('[Rule Upload] Sending form data with rule_type:', selectedRuleType);
      console.debug('[Rule Upload] Form data contents:', 
        Array.from(formData.entries()).reduce((obj, [key, val]) => {
          obj[key] = key === 'file' ? '(file content)' : val;
          return obj;
        }, {} as Record<string, any>)
      );
      
      const response = await ruleService.uploadRule(formData, (progress, eta, speed) => {
        // Update progress in the FileUpload component
        const progressEvent = new CustomEvent('upload-progress', { detail: { progress, eta, speed } });
        document.dispatchEvent(progressEvent);
      });
      console.debug('[Rule Upload] Upload successful:', response);
      
      // Check if the response indicates a duplicate rule
      if (response.data.duplicate) {
        enqueueSnackbar(`Rule "${response.data.name}" already exists`, { variant: 'info' });
      } else {
        enqueueSnackbar('Rule uploaded successfully', { variant: 'success' });
      }
      
      setUploadDialogOpen(false);
      fetchRules();
    } catch (error) {
      console.error('Error uploading rule:', error);
      enqueueSnackbar('Failed to upload rule', { variant: 'error' });
    } finally {
      setIsLoading(false);
    }
  };

  // Handle rule deletion
  const handleDelete = async (id: string, name: string) => {
    try {
      await ruleService.deleteRule(id);
      enqueueSnackbar(`Rule "${name}" deleted successfully`, { variant: 'success' });
      fetchRules();
    } catch (err) {
      console.error('Error deleting rule:', err);
      enqueueSnackbar('Failed to delete rule', { variant: 'error' });
    } finally {
      setDeleteDialogOpen(false);
      setRuleToDelete(null);
    }
  };

  // Open delete confirmation dialog
  const openDeleteDialog = (id: string, name: string) => {
    setRuleToDelete({ id, name });
    setDeleteDialogOpen(true);
  };

  // Close delete confirmation dialog
  const closeDeleteDialog = () => {
    setDeleteDialogOpen(false);
    setRuleToDelete(null);
  };

  // Handle rule download
  const handleDownload = async (id: string, name: string) => {
    try {
      const response = await ruleService.downloadRule(id);
      
      // Create and trigger download
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `${name}.rule`);
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } catch (err) {
      console.error('Error downloading rule:', err);
      enqueueSnackbar('Failed to download rule', { variant: 'error' });
    }
  };

  // Handle edit button click
  const handleEditClick = (rule: Rule) => {
    setCurrentRule(rule);
    setNameEdit(rule.name);
    setDescriptionEdit(rule.description);
    setRuleTypeEdit(rule.rule_type);
    setOpenEditDialog(true);
  };

  // Handle save edit
  const handleSaveEdit = async () => {
    if (!currentRule) return;
    
    try {
      console.debug('[Rule Edit] Updating rule:', currentRule.id, {
        name: nameEdit,
        description: descriptionEdit,
        rule_type: ruleTypeEdit
      });
      
      const response = await ruleService.updateRule(currentRule.id, {
        name: nameEdit,
        description: descriptionEdit,
        rule_type: ruleTypeEdit
      });
      
      console.debug('[Rule Edit] Update successful:', response);
      enqueueSnackbar('Rule updated successfully', { variant: 'success' });
      setOpenEditDialog(false);
      fetchRules();
    } catch (err: any) {
      console.error('[Rule Edit] Error updating rule:', err);
      
      if (err.response?.status === 401) {
        enqueueSnackbar('Your session has expired. Please log in again.', { variant: 'error' });
      } else {
        enqueueSnackbar('Failed to update rule: ' + (err.response?.data?.message || err.message), { variant: 'error' });
      }
    }
  };

  // Handle rule verification
  const handleVerify = async (id: string, name: string) => {
    try {
      setIsLoading(true);
      await ruleService.verifyRule(id, 'verified');
      enqueueSnackbar(`Rule "${name}" verified successfully`, { variant: 'success' });
      fetchRules();
    } catch (err) {
      console.error('Error verifying rule:', err);
      enqueueSnackbar('Failed to verify rule', { variant: 'error' });
    } finally {
      setIsLoading(false);
    }
  };

  // Handle sort change
  const handleSortChange = (column: keyof Rule) => {
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
  const renderSortLabel = (column: keyof Rule, label: string) => {
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

  // Filter rules based on search term and tab
  const filteredRules = rules
    .filter(rule => {
      // Filter by search term
      const matchesSearch = rule.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           rule.description.toLowerCase().includes(searchTerm.toLowerCase());
      
      // Filter by tab
      if (tabValue === 0) return matchesSearch; // All
      if (tabValue === 1) return matchesSearch && rule.rule_type === RuleType.HASHCAT;
      if (tabValue === 2) return matchesSearch && rule.rule_type === RuleType.JOHN;
      
      return matchesSearch;
    })
    .sort((a, b) => {
      // Handle special cases for non-string fields
      if (sortBy === 'file_size' || sortBy === 'rule_count') {
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

  return (
    <Box sx={{ p: 3 }}>
      <Grid container spacing={2} alignItems="center" sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6}>
            <Typography variant="h4" component="h1" gutterBottom>
              Rule Management
            </Typography>
            <Typography variant="body1" color="text.secondary">
              Manage Hashcat rules for password cracking
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
              Upload Rule
            </Button>
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={() => fetchRules()}
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
              aria-label="rule tabs"
            >
              <Tab label="All Rules" id="tab-0" />
              <Tab label="Hashcat" id="tab-1" />
              <Tab label="John" id="tab-2" />
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
              placeholder="Search rules..."
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
            <Table sx={{ minWidth: 650 }} aria-label="rules table">
              <TableHead>
                <TableRow>
                  <TableCell>
                    {renderSortLabel('name', 'Name')}
                  </TableCell>
                  <TableCell>
                    {renderSortLabel('verification_status', 'Status')}
                  </TableCell>
                  <TableCell>
                    {renderSortLabel('rule_type', 'Type')}
                  </TableCell>
                  <TableCell>
                    {renderSortLabel('file_size', 'Size')}
                  </TableCell>
                  <TableCell>
                    {renderSortLabel('rule_count', 'Rule Count')}
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
                        Loading rules...
                      </Typography>
                    </TableCell>
                  </TableRow>
                ) : filteredRules.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} align="center" sx={{ py: 3 }}>
                      <Typography variant="body1">
                        No rules found
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                        {searchTerm ? 'Try a different search term' : 'Upload a rule to get started'}
                      </Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredRules.map((rule) => (
                    <TableRow key={rule.id}>
                      <TableCell>
                        <Box>
                          <Typography variant="body2" fontWeight="medium">
                            {rule.name}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            {rule.description || 'No description provided'}
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={rule.verification_status}
                          size="small"
                          color={
                            rule.verification_status === RuleStatus.READY
                              ? 'success'
                              : rule.verification_status === RuleStatus.PROCESSING
                              ? 'warning'
                              : 'error'
                          }
                          sx={{ textTransform: 'capitalize' }}
                        />
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={rule.rule_type}
                          size="small"
                          color="primary"
                          variant="outlined"
                          sx={{ textTransform: 'capitalize' }}
                        />
                      </TableCell>
                      <TableCell>
                        {formatFileSize(rule.file_size)}
                      </TableCell>
                      <TableCell>
                        {rule.rule_count.toLocaleString()}
                      </TableCell>
                      <TableCell>
                        {new Date(rule.updated_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell align="right">
                        <Tooltip title="Download">
                          <IconButton
                            onClick={() => handleDownload(rule.id, rule.name)}
                            disabled={rule.verification_status !== 'verified'}
                          >
                            <DownloadIcon />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Edit">
                          <IconButton
                            onClick={() => handleEditClick(rule)}
                          >
                            <EditIcon />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Delete">
                          <IconButton
                            color="error"
                            onClick={() => openDeleteDialog(rule.id, rule.name)}
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
        onClose={() => setUploadDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Upload Rule</DialogTitle>
        <DialogContent>
          <FileUpload
            title="Upload a rule file"
            description="Select a rule file to upload. Supported formats: .rule, .txt"
            acceptedFileTypes=".rule,.txt"
            onUpload={handleUploadRule}
            uploadButtonText="Upload Rule"
            additionalFields={
              <FormControl fullWidth margin="normal">
                <InputLabel id="rule-type-label">Rule Type</InputLabel>
                <Select
                  labelId="rule-type-label"
                  id="rule-type"
                  name="rule_type"
                  value={selectedRuleType}
                  onChange={(e) => setSelectedRuleType(e.target.value as RuleType)}
                  label="Rule Type"
                >
                  <MenuItem value={RuleType.HASHCAT}>Hashcat</MenuItem>
                  <MenuItem value={RuleType.JOHN}>John the Ripper</MenuItem>
                </Select>
              </FormControl>
            }
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setUploadDialogOpen(false)} color="primary">
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
        <DialogTitle id="edit-dialog-title">Edit Rule</DialogTitle>
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
            <InputLabel id="edit-rule-type-label">Rule Type</InputLabel>
            <Select
              labelId="edit-rule-type-label"
              id="edit-rule-type"
              value={ruleTypeEdit}
              onChange={(e) => setRuleTypeEdit(e.target.value as RuleType)}
              label="Rule Type"
            >
              <MenuItem value={RuleType.HASHCAT}>Hashcat</MenuItem>
              <MenuItem value={RuleType.JOHN}>John the Ripper</MenuItem>
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

      {/* Delete Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={closeDeleteDialog}
        aria-labelledby="delete-dialog-title"
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle id="delete-dialog-title">Delete Rule</DialogTitle>
        <DialogContent>
          <Typography variant="body1">
            Are you sure you want to delete rule "{ruleToDelete?.name}"?
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeDeleteDialog}>
            Cancel
          </Button>
          <Button 
            onClick={() => ruleToDelete && handleDelete(ruleToDelete.id, ruleToDelete.name)} 
            variant="contained" 
            color="error"
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
} 