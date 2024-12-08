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
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response) {
      throw error.response.data;
    }
    throw new Error('An error occurred during login');
  }
};

export const logout = async (): Promise<void> => {
  try {
    await axios.post(`${API_URL}/api/logout`, {}, { withCredentials: true });
  } catch (error) {
    console.error('Logout failed:', error);
    throw error;
  }
};

export const isAuthenticated = async (): Promise<boolean> => {
  try {
    const response = await axios.get<{ authenticated: boolean }>(
      `${API_URL}/api/check-auth`, 
      { withCredentials: true }
    );
    return response.data.authenticated;
  } catch (error) {
    return false;
  }
}; 