import axios from 'axios';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export const login = async (username, password) => {
  try {
    const response = await axios.post(`${API_URL}/api/login`, { username, password }, { withCredentials: true });
    return response.data;
  } catch (error) {
    throw error.response ? error.response.data : new Error('An error occurred during login');
  }
};

export const logout = async () => {
  try {
    await axios.post(`${API_URL}/api/logout`, {}, { withCredentials: true });
  } catch (error) {
    console.error('Logout failed:', error);
  }
};

export const isAuthenticated = async () => {
  try {
    const response = await axios.get(`${API_URL}/api/check-auth`, { withCredentials: true });
    return response.data.authenticated;
  } catch (error) {
    return false;
  }
};
