import { User } from '../types/auth';
import { api } from './api';

export interface ProfileUpdate {
  email?: string;
  currentPassword?: string;
  newPassword?: string;
}

export const getUserProfile = async (): Promise<User> => {
  const response = await api.get('/api/user/profile');
  return response.data;
};

export const updateUserProfile = async (update: ProfileUpdate): Promise<void> => {
  await api.put('/api/user/profile', update);
}; 