import React from 'react';
import { Box, Typography, Paper } from '@mui/material';

// TODO: Implement Hash Type Management UI
// - Fetch hash types (useQuery)
// - Display in a table (DataGrid or custom table)
// - Add/Edit/Delete functionality (useMutation)
// - Enable/Disable toggle

const HashTypeManager: React.FC = () => {
  return (
    <Paper sx={{ p: 3 }}>
      <Typography variant="h6" gutterBottom>
        Hash Type Management
      </Typography>
      <Typography variant="body1" color="text.secondary">
        Feature coming soon. Add, edit, enable/disable hash types here.
      </Typography>
      {/* Placeholder for Table/CRUD operations */}
    </Paper>
  );
};

export default HashTypeManager; 