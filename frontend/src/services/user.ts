import { User } from '../types/auth';
import { NotificationPreferences, ProfileUpdate } from '../types/user';
import { api } from './api';

export type { ProfileUpdate } from '../types/user';

export const getUserProfile = async (): Promise<User> => {
  const response = await api.get('/api/user/profile');
  return response.data;
};

export const updateUserProfile = async (update: ProfileUpdate): Promise<void> => {
  await api.put('/api/user/profile', update);
};

export const getNotificationPreferences = async (): Promise<NotificationPreferences> => {
  const response = await api.get('/api/user/notification-preferences');
  return response.data;
};

export const updateNotificationPreferences = async (prefs: NotificationPreferences): Promise<NotificationPreferences> => {
  const response = await api.put('/api/user/notification-preferences', prefs);
  return response.data;
}; 