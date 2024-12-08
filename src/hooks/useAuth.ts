import React, { useContext, createContext, useState, useEffect, ReactNode } from 'react';
import { isAuthenticated } from '../services/auth';
import { AuthState } from '../types/auth';

interface AuthContextType extends AuthState {
  setAuth: (isAuth: boolean) => void;
}

const defaultContext: AuthContextType = {
  isAuth: false,
  authChecked: false,
  setAuth: () => {}
};

const AuthContext = createContext<AuthContextType>(defaultContext);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [authState, setAuthState] = useState<AuthState>({
    isAuth: false,
    authChecked: false
  });

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const auth = await isAuthenticated();
        setAuthState({
          isAuth: auth,
          authChecked: true
        });
      } catch (error) {
        console.error('Auth check failed:', error);
        setAuthState({
          isAuth: false,
          authChecked: true
        });
      }
    };

    checkAuth();
  }, []);

  const setAuth = (isAuth: boolean): void => {
    setAuthState(prev => ({
      ...prev,
      isAuth,
      authChecked: true
    }));
  };

  return React.createElement(AuthContext.Provider, {
    value: { ...authState, setAuth },
    children
  });
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

export default AuthContext;
