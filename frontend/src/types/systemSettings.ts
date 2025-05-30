export interface MaxPriorityConfig {
  max_priority: number;
}

export interface SystemSettingsFormData {
  max_priority: number | string; // Allow string for empty input state
} 