import { api } from './api';
import { LoginResponse } from '../types/auth';

export const login = async (username: string, password: string): Promise<LoginResponse> => {
  try {
    const response = await api.post<LoginResponse>(
      '/api/login', 
      { username, password }
    );
    return response.data;
  } catch (error: unknown) {
    if (error && typeof error === 'object' && 'response' in error) {
      throw (error as any).response?.data;
    }
    throw new Error('An error occurred during login');
  }
};

export const logout = async (): Promise<void> => {
  try {
    // Let the backend handle cookie cleanup
    await api.post('/api/logout');
  } catch (error) {
    console.error('Logout failed:', error);
    throw error;
  }
};

export const isAuthenticated = async (): Promise<boolean> => {
  try {
    const response = await api.get<{ authenticated: boolean }>('/api/check-auth');
    return response.data.authenticated;
  } catch (error) {
    return false;
  }
}; 