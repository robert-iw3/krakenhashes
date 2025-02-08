import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

interface AuthContextType {
  isAuthenticated: boolean;
  isLoading: boolean;
  userRole: string | null;
  setAuth: (value: boolean, role?: string | null) => void;
  checkAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType>({
  isAuthenticated: false,
  isLoading: true,
  userRole: null,
  setAuth: () => {},
  checkAuth: async () => {}
});

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [userRole, setUserRole] = useState<string | null>(null);

  const checkAuth = useCallback(async () => {
    try {
      setIsLoading(true);
      const response = await api.get('/api/check-auth');
      setIsAuthenticated(response.data.authenticated);
      if (response.data.role) {
        setUserRole(response.data.role);
      }
    } catch (error) {
      console.error('Auth check failed:', error);
      setIsAuthenticated(false);
      setUserRole(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Check auth on mount
  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  // Recheck auth when window regains focus
  useEffect(() => {
    const handleFocus = () => {
      checkAuth();
    };

    window.addEventListener('focus', handleFocus);
    return () => {
      window.removeEventListener('focus', handleFocus);
    };
  }, [checkAuth]);

  const setAuth = useCallback((value: boolean, role: string | null = null) => {
    setIsAuthenticated(value);
    setUserRole(role);
  }, []);

  return (
    <AuthContext.Provider value={{ isAuthenticated, isLoading, userRole, setAuth, checkAuth }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}; 