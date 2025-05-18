// Basic types needed for form data - define here or import if defined elsewhere

export interface WordlistBasic {
  id: number;
  name: string;
}

export interface RuleBasic {
  id: number;
  name: string;
}

export interface BinaryVersionBasic {
  id: number; // Keep as int based on backend model
  name: string;
}

// Corresponds to models.AttackMode
export enum AttackMode {
  Straight = 0,
  Combination = 1,
  BruteForce = 3,
  HybridWordlistMask = 6,
  HybridMaskWordlist = 7,
  Association = 9,
}

// Corresponds to models.PresetJob
export interface PresetJob {
  id: string; // uuid.UUID
  name: string;
  wordlist_ids: string[]; // UUIDs as strings to match backend
  rule_ids: string[]; // UUIDs as strings to match backend
  attack_mode: AttackMode;
  priority: number;
  chunk_size_seconds: number;
  status_updates_enabled: boolean;
  is_small_job: boolean;
  allow_high_priority_override: boolean;
  binary_version_id: number;
  created_at: string; // ISO 8601 date string
  updated_at: string; // ISO 8601 date string
  binary_version_name?: string; // Optional, from JOIN
  mask?: string; // Mask pattern for mask-based attack modes
}

// Internal form state type for use in the UI - keeps IDs as numbers
export interface PresetJobFormData {
  name: string;
  wordlist_ids: number[]; // IDs as numbers for form handling
  rule_ids: number[]; // IDs as numbers for form handling
  attack_mode: AttackMode;
  priority: number;
  chunk_size_seconds: number;
  is_small_job: boolean;
  binary_version_id: number;
  mask?: string; // Mask pattern for mask-based attack modes
  allow_high_priority_override: boolean;
}

// API type for create/update operations - using string UUIDs
export type PresetJobApiData = Omit<PresetJob, 'id' | 'created_at' | 'updated_at' | 'binary_version_name' | 'status_updates_enabled'>;

// Corresponds to models.JobWorkflow
export interface JobWorkflow {
  id: string; // uuid.UUID
  name: string;
  created_at: string; // ISO 8601 date string
  updated_at: string; // ISO 8601 date string
  steps?: JobWorkflowStep[]; // Optional, included in GetByID
}

// Corresponds to models.JobWorkflowStep with PresetJobName always populated
export interface JobWorkflowStep {
  id: number; // int64 in Go
  job_workflow_id: string; // uuid.UUID
  preset_job_id: string; // uuid.UUID
  step_order: number;
  preset_job_name: string; // Always populated when fetched
}

// Corresponds to models.PresetJobBasic - for selection in forms
export interface PresetJobBasic {
  id: string; // uuid.UUID
  name: string;
}

// Form data response for job workflow forms
export interface JobWorkflowFormDataResponse {
  preset_jobs: PresetJobBasic[];
}

// Internal form state for workflow forms
export interface JobWorkflowFormData {
  name: string;
  preset_job_ids: string[]; // Array of preset job UUIDs
  orderedJobs: PresetJobBasic[]; // For UI to manage order
}

// Request type for creating/updating job workflows
export interface CreateWorkflowRequest {
  name: string;
  preset_job_ids: string[]; // Array of preset job UUIDs
}

// Alias for update, same structure
export type UpdateWorkflowRequest = CreateWorkflowRequest;

// Corresponds to repository.PresetJobFormData
export interface PresetJobFormDataResponse {
  wordlists: WordlistBasic[];
  rules: RuleBasic[];
  binary_versions: BinaryVersionBasic[];
}

// Utility type for PresetJob create/update forms
export type PresetJobInput = Omit<PresetJob, 'id' | 'created_at' | 'updated_at' | 'binary_version_name' | 'status_updates_enabled' | 'allow_high_priority_override'> & {
  wordlist_ids: number[] | string[];
  rule_ids: number[] | string[];
}; 