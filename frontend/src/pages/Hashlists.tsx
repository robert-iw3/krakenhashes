import React, { useState } from 'react';
import { Box, Typography, Button } from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';
import HashlistsDashboard from '../components/hashlist/HashlistsDashboard';

const Hashlists: React.FC = () => {
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3 }}>
        <Box>
          <Typography variant="h4" component="h1" gutterBottom>
            Hashlist Management
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Upload and manage password hash lists for cracking operations
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setUploadDialogOpen(true)}
        >
          Upload Hashlist
        </Button>
      </Box>
      <HashlistsDashboard 
        uploadDialogOpen={uploadDialogOpen}
        setUploadDialogOpen={setUploadDialogOpen}
      />
    </Box>
  );
};

export default Hashlists;