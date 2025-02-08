import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { User } from '../types/auth';
import { getUserProfile } from '../services/user';
import { isAuthenticated } from '../services/auth';

interface AuthContextType {
  isAuth: boolean;
  setAuth: (isAuth: boolean) => void;
  user: User | null;
  setUser: (user: User | null) => void;
  userRole: string | null;
  setUserRole: (role: string | null) => void;
  checkAuthStatus: () => Promise<boolean>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isAuth, setAuth] = useState(false);
  const [user, setUser] = useState<User | null>(null);
  const [userRole, setUserRole] = useState<string | null>(null);

  const checkAuthStatus = useCallback(async () => {
    try {
      const authCheck = await isAuthenticated();
      setAuth(authCheck.authenticated);
      setUserRole(authCheck.role || null);
      
      if (authCheck.authenticated) {
        const profile = await getUserProfile();
        setUser(profile);
      } else {
        setUser(null);
        setUserRole(null);
      }
      
      return authCheck.authenticated;
    } catch (error) {
      console.error('Auth check failed:', error);
      setAuth(false);
      setUser(null);
      setUserRole(null);
      return false;
    }
  }, []);

  // Initial auth check
  useEffect(() => {
    checkAuthStatus();
  }, [checkAuthStatus]);

  // Check auth on window focus
  useEffect(() => {
    const handleFocus = () => {
      checkAuthStatus();
    };

    window.addEventListener('focus', handleFocus);
    return () => window.removeEventListener('focus', handleFocus);
  }, [checkAuthStatus]);

  // Check auth on visibility change (tab switch)
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        checkAuthStatus();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, [checkAuthStatus]);

  // Periodic auth check when authenticated
  useEffect(() => {
    if (!isAuth) return;

    const interval = setInterval(() => {
      checkAuthStatus();
    }, 5 * 60 * 1000); // Check every 5 minutes

    return () => clearInterval(interval);
  }, [isAuth, checkAuthStatus]);

  return (
    <AuthContext.Provider 
      value={{ 
        isAuth, 
        setAuth, 
        user, 
        setUser, 
        userRole, 
        setUserRole,
        checkAuthStatus 
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}; 