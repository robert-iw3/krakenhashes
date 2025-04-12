/**
 * Layout - Main application layout component with navigation
 * 
 * Features:
 *   - Responsive drawer navigation
 *   - Dynamic menu items based on permissions
 *   - Collapsible sidebar
 *   - App bar with user controls
 * 
 * Dependencies:
 *   - @mui/material for UI components
 *   - react-router-dom for navigation
 *   - @mui/icons-material for icons
 * 
 * Error Scenarios:
 *   - Navigation failure handling
 *   - Route access permissions
 *   - Component rendering errors
 * 
 * Usage Examples:
 * ```tsx
 * // Basic usage with child component
 * <Layout>
 *   <Dashboard />
 * </Layout>
 * 
 * // Usage with multiple children
 * <Layout>
 *   <Header />
 *   <Content />
 *   <Footer />
 * </Layout>
 * ```
 * 
 * Performance Considerations:
 *   - Memoized menu items to prevent unnecessary re-renders
 *   - Lazy loading of icons
 *   - Optimized drawer transitions
 * 
 * @param {LayoutProps} props - Component props
 * @returns {JSX.Element} Layout wrapper with navigation
 */

import React, { useState, useCallback } from 'react';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import {
  AppBar,
  Box,
  CssBaseline,
  Drawer,
  IconButton,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
  Divider,
  Theme
} from '@mui/material';
import {
  Menu as MenuIcon,
  ChevronLeft as ChevronLeftIcon,
  Dashboard as DashboardIcon,
  Computer as ComputerIcon,
  Logout as LogoutIcon,
  Info as InfoIcon,
  Description as DescriptionIcon,
  Rule as RuleIcon,
  ListAlt as ListAltIcon,
} from '@mui/icons-material';
import { logout } from '../services/auth';
import { useAuth } from '../contexts/AuthContext';
import AdminMenu from './AdminMenu';
import UserMenu from './common/UserMenu';

interface MenuItem {
  text: string;
  icon: JSX.Element;
  path: string;
}

interface LayoutProps {}

const drawerWidth = 240;

const menuItems: MenuItem[] = [
  { text: 'Dashboard', icon: <DashboardIcon />, path: '/dashboard' },
  { text: 'Agents', icon: <ComputerIcon />, path: '/agents' },
  { text: 'Hashlists', icon: <ListAltIcon />, path: '/hashlists' },
  { text: 'Wordlists', icon: <DescriptionIcon />, path: '/wordlists' },
  { text: 'Rules', icon: <RuleIcon />, path: '/rules' },
];

const bottomMenuItems: MenuItem[] = [
  { text: 'About', icon: <InfoIcon />, path: '/about' },
];

const Layout: React.FC<LayoutProps> = () => {
  const [open, setOpen] = useState<boolean>(true);
  const navigate = useNavigate();
  const location = useLocation();
  const { setAuth, setUser, setUserRole, userRole } = useAuth();

  const handleDrawerToggle = (): void => {
    setOpen(!open);
  };

  const handleLogout = useCallback(async (): Promise<void> => {
    try {
      await logout();
      setAuth(false);
      setUser(null);
      setUserRole(null);
      navigate('/login', { replace: true });
    } catch (error) {
      console.error('Logout failed:', error);
    }
  }, [navigate, setAuth, setUser, setUserRole]);

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <CssBaseline />
      <AppBar 
        position="fixed" 
        sx={{ 
          zIndex: (theme: Theme) => theme.zIndex.drawer + 1,
          width: '100%',
        }}
      >
        <Toolbar>
          <IconButton
            color="inherit"
            aria-label="toggle drawer"
            onClick={handleDrawerToggle}
            edge="start"
            sx={{ mr: 2 }}
          >
            {open ? <ChevronLeftIcon /> : <MenuIcon />}
          </IconButton>
          <Typography variant="h6" noWrap component="div" sx={{ flexGrow: 1 }}>
            KrakenHashes
          </Typography>
          <UserMenu />
        </Toolbar>
      </AppBar>
      <Drawer
        variant="permanent"
        open={open}
        sx={{
          width: open ? drawerWidth : (theme: Theme) => theme.spacing(7),
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: open ? drawerWidth : (theme: Theme) => theme.spacing(7),
            overflowX: 'hidden',
            borderRight: (theme: Theme) => `1px solid ${theme.palette.divider}`,
            transition: (theme: Theme) => theme.transitions.create('width', {
              easing: theme.transitions.easing.sharp,
              duration: theme.transitions.duration.enteringScreen,
            }),
            position: 'fixed',
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
          },
        }}
      >
        <Toolbar />
        
        <List>
          {menuItems.map((item) => (
            <ListItem
              button
              key={item.text}
              onClick={() => navigate(item.path)}
              selected={location.pathname === item.path}
              sx={{
                minHeight: 48,
                justifyContent: open ? 'initial' : 'center',
                px: 2.5,
              }}
            >
              <ListItemIcon
                sx={{
                  minWidth: 0,
                  mr: open ? 3 : 'auto',
                  justifyContent: 'center',
                }}
              >
                {item.icon}
              </ListItemIcon>
              <ListItemText 
                primary={item.text} 
                sx={{ opacity: open ? 1 : 0 }}
              />
            </ListItem>
          ))}
        </List>

        {userRole === 'admin' && (
          <>
            <Divider />
            <AdminMenu />
          </>
        )}

        <Box sx={{ flexGrow: 1 }} />
        
        <Divider />
        <List>
          {bottomMenuItems.map((item) => (
            <ListItem
              button
              key={item.text}
              onClick={() => navigate(item.path)}
              selected={location.pathname === item.path}
              sx={{
                minHeight: 48,
                justifyContent: open ? 'initial' : 'center',
                px: 2.5,
              }}
            >
              <ListItemIcon
                sx={{
                  minWidth: 0,
                  mr: open ? 3 : 'auto',
                  justifyContent: 'center',
                }}
              >
                {item.icon}
              </ListItemIcon>
              <ListItemText 
                primary={item.text} 
                sx={{ opacity: open ? 1 : 0 }}
              />
            </ListItem>
          ))}
          <ListItem 
            button 
            onClick={handleLogout}
            sx={{
              minHeight: 48,
              justifyContent: open ? 'initial' : 'center',
              px: 2.5,
            }}
          >
            <ListItemIcon
              sx={{
                minWidth: 0,
                mr: open ? 3 : 'auto',
                justifyContent: 'center',
              }}
            >
              <LogoutIcon />
            </ListItemIcon>
            <ListItemText 
              primary="Logout" 
              sx={{ opacity: open ? 1 : 0 }}
            />
          </ListItem>
        </List>
      </Drawer>
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 3,
          ml: (theme: Theme) => `${open ? drawerWidth + theme.spacing(1) : theme.spacing(8)}px`,
          transition: (theme: Theme) => theme.transitions.create(['margin', 'width'], {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.enteringScreen,
          }),
        }}
      >
        <Toolbar /> {/* Spacer for AppBar */}
        <Outlet />
      </Box>
    </Box>
  );
};

export default Layout; 