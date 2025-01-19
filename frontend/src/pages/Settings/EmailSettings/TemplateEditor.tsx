import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  TextField,
  Button,
  Grid,
  Typography,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Paper,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TableContainer,
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  CircularProgress,
} from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import { api } from '../../../services/api';

interface TemplateEditorProps {
  onNotification: (message: string, severity: 'success' | 'error') => void;
}

interface Template {
  id?: number;
  templateType: 'security_event' | 'job_completion' | 'admin_error' | 'mfa_code';
  name: string;
  subject: string;
  htmlContent: string;
  textContent: string;
}

const STORAGE_KEY = 'templateEditorState';

// Keep sampleData for both testing and live preview
const sampleData = {
  security_event: {
    EventType: 'Login Attempt',
    Timestamp: new Date().toISOString(),
    Details: 'Failed login attempt from unknown IP',
    IPAddress: '192.168.1.1',
  },
  job_completion: {
    JobName: 'Sample Hash Cracking Job',
    Duration: '2h 15m',
    HashesProcessed: '1,000,000',
    CrackedCount: '750,000',
    SuccessRate: '75',
  },
  admin_error: {
    ErrorType: 'Database Connection',
    Component: 'User Service',
    Timestamp: new Date().toISOString(),
    ErrorMessage: 'Failed to connect to database',
    StackTrace: 'Error: Connection timeout\n  at Database.connect (/app/db.js:25)',
  },
  mfa_code: {
    Code: '123456',
    ExpiryMinutes: '5',
  },
};

export const TemplateEditor: React.FC<TemplateEditorProps> = ({ onNotification }) => {
  const [loading, setLoading] = useState(false);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null);
  const [isEditing, setIsEditing] = useState(false);

  const loadData = useCallback(async () => {
    try {
      console.debug('[TemplateEditor] Loading templates...');
      setLoading(true);
      const response = await api.get('/api/email/templates');
      console.debug('[TemplateEditor] Templates loaded:', response.data);

      // Transform the data to match our frontend model
      const transformedTemplates = response.data.map((template: any) => ({
        id: template.id,
        templateType: template.template_type,
        name: template.name,
        subject: template.subject,
        htmlContent: template.html_content,
        textContent: template.text_content,
      }));

      console.debug('[TemplateEditor] Transformed templates:', transformedTemplates);
      setTemplates(transformedTemplates);
    } catch (error) {
      console.error('[TemplateEditor] Failed to load templates:', error);
      onNotification('Failed to load templates', 'error');
    } finally {
      setLoading(false);
    }
  }, [onNotification]);

  // Load templates and restore edit state on mount
  useEffect(() => {
    const restoreState = () => {
      const savedState = localStorage.getItem(STORAGE_KEY);
      if (savedState) {
        const { template, editing } = JSON.parse(savedState);
        if (template && editing) {
          console.debug('[TemplateEditor] Restoring edit state:', template);
          setSelectedTemplate(template);
          setIsEditing(true);
        }
      }
    };

    loadData();
    restoreState();
  }, []);

  // Save edit state to localStorage whenever it changes
  useEffect(() => {
    if (selectedTemplate && isEditing) {
      console.debug('[TemplateEditor] Saving edit state:', selectedTemplate);
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        template: selectedTemplate,
        editing: isEditing,
      }));
    } else {
      localStorage.removeItem(STORAGE_KEY);
    }
  }, [selectedTemplate, isEditing]);

  const handleEditTemplate = (template: Template) => {
    console.debug('[TemplateEditor] Editing template:', template);
    setSelectedTemplate(template);
    setIsEditing(true);
  };

  const handleDeleteTemplate = async (id: number) => {
    try {
      setLoading(true);
      await api.delete(`/api/email/templates/${id}`);
      onNotification('Template deleted successfully', 'success');
      await loadData();
    } catch (error) {
      console.error('[TemplateEditor] Failed to delete template:', error);
      onNotification('Failed to delete template', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!selectedTemplate) return;

    try {
      setLoading(true);
      const payload = {
        template_type: selectedTemplate.templateType,
        name: selectedTemplate.name,
        subject: selectedTemplate.subject,
        html_content: selectedTemplate.htmlContent,
        text_content: selectedTemplate.textContent,
      };

      if (selectedTemplate.id) {
        await api.put(`/api/email/templates/${selectedTemplate.id}`, payload);
        onNotification('Template updated successfully', 'success');
      } else {
        await api.post('/api/email/templates', payload);
        onNotification('Template created successfully', 'success');
      }

      await loadData();
      setIsEditing(false);
      setSelectedTemplate(null);
    } catch (error) {
      console.error('[TemplateEditor] Failed to save template:', error);
      onNotification('Failed to save template', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleTest = async () => {
    if (!selectedTemplate) return;

    try {
      setLoading(true);
      // TODO: Implement test email sending
      await new Promise(resolve => setTimeout(resolve, 1000));
      onNotification('Test email sent successfully', 'success');
    } catch (error) {
      console.error('[TemplateEditor] Failed to send test email:', error);
      onNotification('Failed to send test email', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = () => {
    setIsEditing(false);
    setSelectedTemplate(null);
    localStorage.removeItem(STORAGE_KEY);
  };

  const getPreviewContent = () => {
    if (!selectedTemplate?.templateType || !selectedTemplate?.htmlContent) {
      return '';
    }

    let content = selectedTemplate.htmlContent;
    const data = sampleData[selectedTemplate.templateType] || {};

    // Replace template variables with sample data
    Object.entries(data).forEach(([key, value]) => {
      const regex = new RegExp(`{{\\s*\\.${key}\\s*}}`, 'g');
      content = content.replace(regex, String(value));
    });

    return content;
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" p={3}>
        <CircularProgress />
      </Box>
    );
  }

  if (isEditing && selectedTemplate) {
    return (
      <Box>
        <Box mb={3} display="flex" justifyContent="space-between" alignItems="center">
          <Typography variant="h6">
            {selectedTemplate.id ? 'Edit Template' : 'Create Template'}
          </Typography>
          <Box>
            <Button
              variant="outlined"
              onClick={handleCancel}
              sx={{ mr: 1 }}
            >
              Cancel
            </Button>
            <Button
              variant="outlined"
              onClick={handleTest}
              sx={{ mr: 1 }}
            >
              Test
            </Button>
            <LoadingButton
              variant="contained"
              onClick={handleSave}
              loading={loading}
            >
              Save
            </LoadingButton>
          </Box>
        </Box>

        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <FormControl fullWidth>
              <InputLabel>Template Type</InputLabel>
              <Select
                value={selectedTemplate.templateType}
                label="Template Type"
                onChange={(e) => setSelectedTemplate(prev => prev ? {
                  ...prev,
                  templateType: e.target.value as Template['templateType']
                } : null)}
              >
                <MenuItem value="security_event">Security Event</MenuItem>
                <MenuItem value="job_completion">Job Completion</MenuItem>
                <MenuItem value="admin_error">Admin Error</MenuItem>
                <MenuItem value="mfa_code">MFA Code</MenuItem>
              </Select>
            </FormControl>
          </Grid>

          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              label="Name"
              value={selectedTemplate.name}
              onChange={(e) => setSelectedTemplate(prev => prev ? {
                ...prev,
                name: e.target.value
              } : null)}
            />
          </Grid>

          <Grid item xs={12}>
            <TextField
              fullWidth
              label="Subject"
              value={selectedTemplate.subject}
              onChange={(e) => setSelectedTemplate(prev => prev ? {
                ...prev,
                subject: e.target.value
              } : null)}
            />
          </Grid>

          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              label="HTML Content"
              value={selectedTemplate.htmlContent}
              onChange={(e) => setSelectedTemplate(prev => prev ? {
                ...prev,
                htmlContent: e.target.value
              } : null)}
              multiline
              rows={15}
            />
          </Grid>

          <Grid item xs={12} md={6}>
            <Paper 
              sx={{ 
                p: 2, 
                height: '100%', 
                maxHeight: '500px', 
                overflow: 'auto',
                backgroundColor: '#1a1a1a',
                color: '#ffffff',
                '& a': { color: '#4fc3f7' },
                '& *': { maxWidth: '100%' }
              }}
            >
              <Typography variant="subtitle2" gutterBottom sx={{ color: '#ffffff' }}>
                Live Preview (with sample data):
              </Typography>
              <Box 
                sx={{ 
                  mt: 2,
                }}
                dangerouslySetInnerHTML={{ __html: getPreviewContent() }} 
              />
            </Paper>
          </Grid>

          <Grid item xs={12}>
            <TextField
              fullWidth
              label="Text Content"
              value={selectedTemplate.textContent}
              onChange={(e) => setSelectedTemplate(prev => prev ? {
                ...prev,
                textContent: e.target.value
              } : null)}
              multiline
              rows={8}
            />
          </Grid>
        </Grid>
      </Box>
    );
  }

  return (
    <Box>
      <Box mb={3} display="flex" justifyContent="space-between" alignItems="center">
        <Typography variant="h6">Email Templates</Typography>
        <Button
          variant="contained"
          onClick={() => handleEditTemplate({
            templateType: 'security_event',
            name: '',
            subject: '',
            htmlContent: '',
            textContent: '',
          })}
        >
          Create Template
        </Button>
      </Box>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Type</TableCell>
              <TableCell>Subject</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {templates.map((template) => (
              <TableRow key={template.id}>
                <TableCell>{template.name}</TableCell>
                <TableCell>{template.templateType}</TableCell>
                <TableCell>{template.subject}</TableCell>
                <TableCell align="right">
                  <IconButton
                    onClick={() => handleEditTemplate(template)}
                    size="small"
                  >
                    <EditIcon />
                  </IconButton>
                  <IconButton
                    onClick={() => handleDeleteTemplate(template.id!)}
                    size="small"
                  >
                    <DeleteIcon />
                  </IconButton>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}; 