/**
 * Types for agent scheduling system
 */

export interface AgentSchedule {
  id?: number;
  agentId: number;
  dayOfWeek: number; // 0-6 (Sunday-Saturday)
  startTime: string; // "HH:MM" in user's timezone
  endTime: string;   // "HH:MM" in user's timezone
  timezone: string;  // User's timezone (e.g., "America/New_York")
  isActive: boolean;
}

export interface AgentScheduleDTO {
  // What we send to backend (UTC times)
  dayOfWeek: number;
  startTimeUTC: string; // "HH:MM" in UTC
  endTimeUTC: string;   // "HH:MM" in UTC
  timezone: string;     // User's timezone for reference
  isActive: boolean;
}

export interface AgentSchedulingInfo {
  agentId: number;
  schedulingEnabled: boolean;
  scheduleTimezone: string;
  schedules: AgentSchedule[];
}

export interface ScheduleEditDialogProps {
  open: boolean;
  onClose: () => void;
  agentId: number;
  schedules: AgentSchedule[];
  onSave: (schedules: AgentSchedule[]) => void;
}

export interface DayScheduleProps {
  dayOfWeek: number;
  schedule?: AgentSchedule;
  onChange: (schedule: AgentSchedule | null) => void;
  disabled?: boolean;
}