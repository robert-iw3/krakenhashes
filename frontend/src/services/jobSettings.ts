import { api } from './api';

export interface JobExecutionSettings {
  default_chunk_duration: number;
  chunk_fluctuation_percentage: number;
  agent_hashlist_retention_hours: number;
  progress_reporting_interval: number;
  max_concurrent_jobs_per_agent: number;
  job_interruption_enabled: boolean;
  benchmark_cache_duration_hours: number;
  enable_realtime_crack_notifications: boolean;
  metrics_retention_realtime_days: number;
  metrics_retention_daily_days: number;
  metrics_retention_weekly_days: number;
}

export const getJobExecutionSettings = async (): Promise<JobExecutionSettings> => {
  const response = await api.get('/admin/settings/job-execution');
  return response.data.data;
};

export const updateJobExecutionSettings = async (settings: JobExecutionSettings): Promise<void> => {
  await api.put('/admin/settings/job-execution', settings);
};