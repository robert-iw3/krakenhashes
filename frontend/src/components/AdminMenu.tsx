import React from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { List, ListItem, ListItemIcon, ListItemText } from '@mui/material';
import { Settings as SettingsIcon } from '@mui/icons-material';

const AdminMenu: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();

  return (
    <List>
      <ListItem
        button
        onClick={() => navigate('/admin/settings')}
        selected={location.pathname.startsWith('/admin/settings')}
        sx={{
          minHeight: 48,
          px: 2.5,
        }}
      >
        <ListItemIcon
          sx={{
            minWidth: 0,
            mr: 3,
            justifyContent: 'center',
          }}
        >
          <SettingsIcon />
        </ListItemIcon>
        <ListItemText primary="Admin Settings" />
      </ListItem>
    </List>
  );
};

export default AdminMenu; 