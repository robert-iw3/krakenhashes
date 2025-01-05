import axios from 'axios';
import { LoginResponse } from '../types/auth';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export const login = async (username: string, password: string): Promise<LoginResponse> => {
  try {
    const response = await axios.post<LoginResponse>(
      `${API_URL}/api/login`, 
      { username, password }, 
      { withCredentials: true }
    );
    
    // Store token if login was successful
    if (response.data.success && response.data.token) {
      localStorage.setItem('token', response.data.token);
    }
    
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
    await axios.post(`${API_URL}/api/logout`, {}, { withCredentials: true });
    // Clear token on logout
    localStorage.removeItem('token');
  } catch (error) {
    console.error('Logout failed:', error);
    throw error;
  }
};

export const isAuthenticated = async (): Promise<boolean> => {
  try {
    const token = localStorage.getItem('token');
    const response = await axios.get<{ authenticated: boolean }>(
      `${API_URL}/api/check-auth`, 
      { 
        withCredentials: true,
        headers: token ? { Authorization: `Bearer ${token}` } : undefined
      }
    );
    return response.data.authenticated;
  } catch (error) {
    return false;
  }
}; 