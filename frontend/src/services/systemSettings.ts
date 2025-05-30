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