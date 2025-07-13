import { api } from './api';

export interface JobExecutionSettings {
  default_chunk_duration: number;
  chunk_fluctuation_percentage: number;
  agent_hashlist_retention_hours: number;
  progress_reporting_interval: number;
  max_concurrent_jobs_per_agent: number;
  job_interruption_enabled: boolean;
  benchmark_cache_duration_hours: number;
  speedtest_timeout_seconds: number;
  enable_realtime_crack_notifications: boolean;
  metrics_retention_realtime_days: number;
  metrics_retention_daily_days: number;
  metrics_retention_weekly_days: number;
  job_refresh_interval_seconds: number;
  max_chunk_retry_attempts: number;
  jobs_per_page_default: number;
  // Rule splitting settings
  rule_split_enabled: boolean;
  rule_split_threshold: number;
  rule_split_min_rules: number;
  rule_split_max_chunks: number;
  rule_chunk_temp_dir: string;
}

export const getJobExecutionSettings = async (): Promise<JobExecutionSettings> => {
  const response = await api.get('/api/admin/settings/job-execution');
  return response.data;
};

export const updateJobExecutionSettings = async (settings: JobExecutionSettings): Promise<void> => {
  await api.put('/api/admin/settings/job-execution', settings);
};