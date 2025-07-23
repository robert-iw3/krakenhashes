/**
 * Type definitions for job-related data structures
 */

// Job status enum
export type JobStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

// Job summary for list views
export interface JobSummary {
  id: string;
  name: string;
  hashlist_id: number;
  hashlist_name: string;
  status: JobStatus;
  priority: number;
  max_agents: number;
  dispatched_percent: number;
  searched_percent: number;
  cracked_count: number;
  agent_count: number;
  total_speed: number;
  created_at: string;
  updated_at: string;
  completed_at?: string;
  created_by_username?: string;
  error_message?: string;
  // Enhanced chunking fields
  effective_keyspace?: number;
  multiplication_factor?: number;
  uses_rule_splitting?: boolean;
  base_keyspace?: number;
  total_keyspace?: number;
  processed_keyspace?: number;
  dispatched_keyspace?: number;
  overall_progress_percent: number;
}

// Pagination information
export interface PaginationInfo {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

// Job detail for detailed views
export interface JobDetail extends JobSummary {
  workflow_id?: number;
  workflow_name?: string;
  client_id?: number;
  client_name?: string;
  hashlist_count: number;
  hashlist_cracked: number;
  tasks?: JobTask[];
  agents?: JobAgent[];
}

// Job execution information
export interface JobExecution {
  id: string;
  preset_job_id: string;
  preset_job_name?: string;
  hashlist_id: number;
  hashlist_name?: string;
  status: string;
  priority: number;
  max_agents: number;
  total_keyspace?: number;
  processed_keyspace: number;
  attack_mode: number;
  created_by?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  updated_at: string;
  error_message?: string;
  total_hashes?: number;
  cracked_hashes?: number;
  overall_progress_percent?: number;
}

// Job task information
export interface JobTask {
  id: string;
  job_id: string;
  job_execution_id?: string;
  agent_id?: number;
  status: string;
  priority: number;
  chunk_start: number;
  chunk_end: number;
  keyspace_start?: number;
  keyspace_end?: number;
  keyspace_processed?: number;
  benchmark_speed?: number;
  chunk_duration?: number;
  assigned_agent_id?: string;
  assigned_at?: string;
  started_at?: string;
  completed_at?: string;
  last_checkpoint?: string;
  error_message?: string;
  crack_count?: number;
  progress_percent?: number;
}

// Job agent information
export interface JobAgent {
  id: string;
  name: string;
  status: string;
  speed: number;
  last_heartbeat: string;
}

// Job list response
export interface JobListResponse {
  jobs: JobSummary[];
  pagination: PaginationInfo;
  status_counts: Record<string, number>;
}

// Job detail response
export interface JobDetailResponse {
  job: JobDetail;
}