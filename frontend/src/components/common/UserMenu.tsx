import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Button,
  Menu,
  MenuItem,
  Typography,
  Divider,
  ListItemIcon,
  Box,
} from '@mui/material';
import {
  KeyboardArrowDown as ArrowDownIcon,
  Settings as SettingsIcon,
  Logout as LogoutIcon,
  Person as PersonIcon,
} from '@mui/icons-material';
import { useAuth } from '../../contexts/AuthContext';
import { logout } from '../../services/auth';

const UserMenu: React.FC = () => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const { user, setAuth } = useAuth();
  const navigate = useNavigate();
  const open = Boolean(anchorEl);

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleLogout = async () => {
    try {
      await logout();
      setAuth(false);
      navigate('/login', { replace: true });
    } catch (error) {
      console.error('Logout failed:', error);
    }
  };

  const handleSettings = () => {
    handleClose();
    navigate('/settings/profile');
  };

  return (
    <Box>
      <Button
        onClick={handleClick}
        endIcon={<ArrowDownIcon />}
        color="inherit"
        sx={{
          textTransform: 'none',
          minWidth: 100,
          '&:hover': {
            backgroundColor: 'rgba(255, 255, 255, 0.08)',
          },
        }}
        startIcon={<PersonIcon />}
      >
        {user?.username || 'User'}
      </Button>
      <Menu
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
        onClick={handleClose}
        PaperProps={{
          elevation: 0,
          sx: {
            overflow: 'visible',
            filter: 'drop-shadow(0px 2px 8px rgba(0,0,0,0.32))',
            mt: 1.5,
            backgroundColor: 'background.paper',
            '& .MuiMenuItem-root': {
              minWidth: 200,
            },
          },
        }}
        transformOrigin={{ horizontal: 'right', vertical: 'top' }}
        anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
      >
        <MenuItem disabled sx={{ opacity: 0.7 }}>
          <Typography variant="body2" color="text.secondary">
            {user?.email || 'user@example.com'}
          </Typography>
        </MenuItem>
        <Divider />
        <MenuItem onClick={handleSettings}>
          <ListItemIcon>
            <SettingsIcon fontSize="small" />
          </ListItemIcon>
          User Settings
        </MenuItem>
        <MenuItem onClick={handleLogout}>
          <ListItemIcon>
            <LogoutIcon fontSize="small" />
          </ListItemIcon>
          Logout
        </MenuItem>
      </Menu>
    </Box>
  );
};

export default UserMenu; 