import React, { useState } from 'react';
import {
  Box,
  Button,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  MenuItem,
  CircularProgress,
} from '@mui/material';
import { useSnackbar } from 'notistack';
import { AddBinaryRequest, addBinary } from '../../services/binary';

interface AddBinaryFormProps {
  onSuccess: () => void;
  onCancel: () => void;
}

const initialFormData: AddBinaryRequest = {
  binary_type: 'hashcat',
  compression_type: '7z',
  source_url: '',
  file_name: '',
};

const AddBinaryForm: React.FC<AddBinaryFormProps> = ({ onSuccess, onCancel }) => {
  const [formData, setFormData] = useState<AddBinaryRequest>(initialFormData);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { enqueueSnackbar } = useSnackbar();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    if (name === 'source_url' && value) {
      // Extract file name from URL
      try {
        const url = new URL(value);
        const fileName = decodeURIComponent(url.pathname.split('/').pop() || '');
        setFormData((prev) => ({ ...prev, file_name: fileName }));
      } catch (error) {
        console.warn('Failed to parse URL:', error);
      }
    }
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      setIsSubmitting(true);
      await addBinary(formData);
      enqueueSnackbar('Binary added successfully', { variant: 'success' });
      onSuccess();
    } catch (error) {
      console.error('Error adding binary:', error);
      enqueueSnackbar('Failed to add binary', { variant: 'error' });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <DialogTitle>Add New Binary</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
          <TextField
            select
            label="Binary Type"
            name="binary_type"
            value={formData.binary_type}
            onChange={handleChange}
            required
            fullWidth
          >
            <MenuItem value="hashcat">Hashcat</MenuItem>
            <MenuItem value="john">John the Ripper</MenuItem>
          </TextField>
          <TextField
            select
            label="Compression Type"
            name="compression_type"
            value={formData.compression_type}
            onChange={handleChange}
            required
            fullWidth
          >
            <MenuItem value="7z">7z</MenuItem>
            <MenuItem value="zip">ZIP</MenuItem>
            <MenuItem value="tar.gz">TAR.GZ</MenuItem>
            <MenuItem value="tar.xz">TAR.XZ</MenuItem>
          </TextField>
          <TextField
            label="Source URL"
            name="source_url"
            value={formData.source_url}
            onChange={handleChange}
            required
            fullWidth
            type="url"
            helperText="URL to download the binary (e.g., https://hashcat.net/beta/hashcat-6.2.6%2B813.7z)"
          />
          <TextField
            label="File Name"
            name="file_name"
            value={formData.file_name}
            onChange={handleChange}
            required
            fullWidth
            helperText="Auto-filled from URL, but can be modified if needed"
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onCancel} disabled={isSubmitting}>
          Cancel
        </Button>
        <Button
          type="submit"
          variant="contained"
          disabled={isSubmitting}
          startIcon={isSubmitting ? <CircularProgress size={20} /> : null}
        >
          Add Binary
        </Button>
      </DialogActions>
    </form>
  );
};

export default AddBinaryForm; 