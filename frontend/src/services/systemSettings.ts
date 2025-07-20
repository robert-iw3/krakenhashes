import { api } from './api';
import { MaxPriorityConfig } from '../types/systemSettings';

// Admin-only functions
export const getMaxPriority = async (): Promise<MaxPriorityConfig> => {
  const response = await api.get<MaxPriorityConfig>('/api/admin/settings/max-priority');
  return response.data;
};

export const updateMaxPriority = async (maxPriority: number): Promise<MaxPriorityConfig> => {
  const response = await api.put<MaxPriorityConfig>('/api/admin/settings/max-priority', {
    max_priority: maxPriority
  });
  return response.data;
};

// User-accessible function (read-only)
export const getMaxPriorityForUsers = async (): Promise<MaxPriorityConfig> => {
  const response = await api.get<MaxPriorityConfig>('/api/settings/max-priority');
  return response.data;
};

// Agent scheduling settings
export const getSystemSettings = async () => {
  const response = await api.get('/api/admin/settings');
  return response.data;
};

export const updateSystemSetting = async (key: string, value: string) => {
  const response = await api.put(`/api/admin/settings/${key}`, { value });
  return response.data;
};

// Get a specific system setting
export const getSystemSetting = async (key: string) => {
  const response = await api.get(`/api/admin/settings/${key}`);
  return response.data;
}; 