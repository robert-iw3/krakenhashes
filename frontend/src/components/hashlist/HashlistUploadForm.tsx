import React, { useState, useCallback } from 'react';
import { useDropzone } from 'react-dropzone';
import { Box, Button, CircularProgress, TextField, Typography } from '@mui/material';
import ClientAutocomplete from './ClientAutocomplete';
import HashTypeSelect from './HashTypeSelect';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../../services/api';
import { useNavigate } from 'react-router-dom';

const schema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional(),
  hashTypeId: z.number().min(1, 'Hash type is required'),
  clientName: z.string().nullish(),
});

type FormData = z.infer<typeof schema>;

interface HashlistUploadFormProps {
  onSuccess?: () => void;
}

export default function HashlistUploadForm({ onSuccess }: HashlistUploadFormProps) {
  const [file, setFile] = useState<File | null>(null);
  const [uploadProgress, setUploadProgress] = useState(0);
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  
  const { control, handleSubmit, reset, formState: { errors } } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      description: '',
      hashTypeId: undefined,
      clientName: null,
    }
  });

  const onDrop = useCallback((acceptedFiles: File[]) => {
    if (acceptedFiles.length > 0) {
      setFile(acceptedFiles[0]);
    }
  }, []);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'text/plain': ['.txt'],
      'text/csv': ['.csv']
    },
    maxFiles: 1
  });

  const uploadMutation = useMutation({
    mutationFn: async (data: FormData) => {
      if (!file) throw new Error('No file selected');
      
      const formData = new FormData();
      formData.append('hashlist_file', file);
      formData.append('name', data.name);
      formData.append('hash_type_id', data.hashTypeId.toString());
      if (data.description) formData.append('description', data.description);
      if (data.clientName) formData.append('client_name', data.clientName);

      console.log("FormData object before sending:");
      formData.forEach((value, key) => {
        console.log(`${key}: ${value}`);
      });

      return api.post('/api/hashlists', formData, {
        onUploadProgress: (progressEvent) => {
          const percentCompleted = Math.round(
            (progressEvent.loaded * 100) / (progressEvent.total || 1)
          );
          setUploadProgress(percentCompleted);
        }
      });
    },
    onSuccess: (response) => {
      // The backend returns the created hashlist data
      const hashlistId = response.data?.id || response.data?.data?.id;
      
      if (hashlistId) {
        // Navigate to the hashlist detail page
        navigate(`/hashlists/${hashlistId}`);
      } else if (onSuccess) {
        // Fallback to the provided callback if no ID is returned
        onSuccess();
      }
      
      reset();
      setFile(null);
      setUploadProgress(0);
    },
    onError: (error) => {
      console.error("Upload failed:", error);
      setUploadProgress(0);
    }
  });

  const onSubmit = (data: FormData): void => {
    console.log("Submitting form data:", data);
    console.log("Client Name in onSubmit:", data.clientName);
    uploadMutation.mutate(data);
  };

  return (
    <Box component="form" onSubmit={handleSubmit(onSubmit)} sx={{ mt: 3 }}>
      <Controller
        name="name"
        control={control}
        render={({ field }) => (
          <TextField
            {...field}
            label="Hashlist Name"
            fullWidth
            margin="normal"
            error={!!errors.name}
            helperText={errors.name?.message}
          />
        )}
      />

      <HashTypeSelect
        control={control}
        name="hashTypeId"
        label="Hash Type"
      />

      <Controller
        name="description"
        control={control}
        render={({ field }) => (
          <TextField
            {...field}
            label="Description"
            fullWidth
            margin="normal"
            multiline
            rows={3}
          />
        )}
      />

      <Controller
        name="clientName"
        control={control}
        render={({ field }) => (
          <ClientAutocomplete
            value={field.value ?? null}
            onChange={field.onChange}
          />
        )}
      />

      <Box 
        {...getRootProps()}
        sx={{
          border: '2px dashed',
          borderColor: isDragActive ? 'primary.main' : 'grey.400',
          borderRadius: 1,
          p: 4,
          textAlign: 'center',
          cursor: 'pointer',
          my: 2
        }}
      >
        <input {...getInputProps()} />
        {file ? (
          <Typography>{file.name}</Typography>
        ) : isDragActive ? (
          <Typography>Drop the hashlist file here...</Typography>
        ) : (
          <Typography>Drag and drop a hashlist file, or click to select</Typography>
        )}
      </Box>

      {uploadProgress > 0 && uploadProgress < 100 && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <CircularProgress variant="determinate" value={uploadProgress} />
          <Typography>{uploadProgress}%</Typography>
        </Box>
      )}

      <Button
        type="submit"
        variant="contained"
        disabled={uploadMutation.isPending || !file}
        sx={{ mt: 2 }}
      >
        {uploadMutation.isPending ? 'Uploading...' : 'Upload Hashlist'}
      </Button>

      {uploadMutation.isError && (
        <Typography color="error" sx={{ mt: 2 }}>
          Error uploading hashlist: {(uploadMutation.error as Error)?.message || 'An unknown error occurred'}
        </Typography>
      )}
    </Box>
  );
}