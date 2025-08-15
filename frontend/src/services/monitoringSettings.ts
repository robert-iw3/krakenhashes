import { api } from './api';

export interface MonitoringSettings {
  metrics_retention_realtime_days: number;
  metrics_retention_daily_days: number;
  metrics_retention_weekly_days: number;
  enable_aggregation: boolean;
  aggregation_interval: string;
}

export const getMonitoringSettings = async (): Promise<MonitoringSettings> => {
  const response = await api.get('/api/admin/settings/monitoring');
  return response.data;
};

export const updateMonitoringSettings = async (settings: MonitoringSettings): Promise<void> => {
  await api.put('/api/admin/settings/monitoring', settings);
};