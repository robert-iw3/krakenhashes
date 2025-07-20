/**
 * Timezone utilities for agent scheduling
 * 
 * Handles conversion between local time and UTC for schedule management
 */

/**
 * Convert local time to UTC time string
 * @param localTime Time in HH:MM format
 * @param dayOfWeek Day of week (0-6, Sunday-Saturday)
 * @returns UTC time in HH:MM format
 */
export const convertLocalTimeToUTC = (localTime: string, dayOfWeek: number): string => {
  const [hours, minutes] = localTime.split(':').map(Number);
  
  // Create a date object for the next occurrence of this day
  const now = new Date();
  const currentDay = now.getDay();
  let daysUntilTarget = dayOfWeek - currentDay;
  
  if (daysUntilTarget < 0) {
    daysUntilTarget += 7;
  }
  
  const targetDate = new Date(now);
  targetDate.setDate(now.getDate() + daysUntilTarget);
  targetDate.setHours(hours, minutes, 0, 0);
  
  // Get UTC time
  const utcHours = targetDate.getUTCHours().toString().padStart(2, '0');
  const utcMinutes = targetDate.getUTCMinutes().toString().padStart(2, '0');
  
  return `${utcHours}:${utcMinutes}`;
};

/**
 * Convert UTC time to local time string
 * @param utcTime Time in HH:MM format (UTC)
 * @param dayOfWeek Day of week (0-6, Sunday-Saturday)
 * @returns Local time in HH:MM format
 */
export const convertUTCTimeToLocal = (utcTime: string, dayOfWeek: number): string => {
  const [hours, minutes] = utcTime.split(':').map(Number);
  
  // Create a date object for the next occurrence of this day
  const now = new Date();
  const currentDay = now.getDay();
  let daysUntilTarget = dayOfWeek - currentDay;
  
  if (daysUntilTarget < 0) {
    daysUntilTarget += 7;
  }
  
  const targetDate = new Date(now);
  targetDate.setDate(now.getDate() + daysUntilTarget);
  targetDate.setUTCHours(hours, minutes, 0, 0);
  
  // Get local time
  const localHours = targetDate.getHours().toString().padStart(2, '0');
  const localMinutes = targetDate.getMinutes().toString().padStart(2, '0');
  
  return `${localHours}:${localMinutes}`;
};

/**
 * Get the user's timezone
 * @returns Timezone string (e.g., "America/New_York")
 */
export const getUserTimezone = (): string => {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
};

/**
 * Get timezone abbreviation (e.g., EST, PST)
 * @returns Timezone abbreviation
 */
export const getTimezoneAbbreviation = (): string => {
  const date = new Date();
  const options: Intl.DateTimeFormatOptions = { 
    timeZoneName: 'short' 
  };
  const formatter = new Intl.DateTimeFormat('en-US', options);
  const parts = formatter.formatToParts(date);
  const timeZonePart = parts.find(part => part.type === 'timeZoneName');
  return timeZonePart?.value || 'UTC';
};

/**
 * Get UTC offset string (e.g., "UTC-5")
 * @returns UTC offset string
 */
export const getUTCOffset = (): string => {
  const date = new Date();
  const offsetMinutes = date.getTimezoneOffset();
  const offsetHours = Math.abs(Math.floor(offsetMinutes / 60));
  const offsetSign = offsetMinutes > 0 ? '-' : '+';
  return `UTC${offsetSign}${offsetHours}`;
};

/**
 * Format time for display with timezone info
 * @param time Time in HH:MM format
 * @param includeTimezone Whether to include timezone info
 * @returns Formatted time string
 */
export const formatTimeDisplay = (time: string, includeTimezone = false): string => {
  if (!includeTimezone) {
    return time;
  }
  
  const tzAbbr = getTimezoneAbbreviation();
  const utcOffset = getUTCOffset();
  return `${time} ${tzAbbr} (${utcOffset})`;
};

/**
 * Check if a schedule spans overnight (e.g., 22:00 - 02:00)
 * @param startTime Start time in HH:MM format
 * @param endTime End time in HH:MM format
 * @returns True if schedule spans midnight
 */
export const isOvernightSchedule = (startTime: string, endTime: string): boolean => {
  const [startHours, startMinutes] = startTime.split(':').map(Number);
  const [endHours, endMinutes] = endTime.split(':').map(Number);
  
  const startTotalMinutes = startHours * 60 + startMinutes;
  const endTotalMinutes = endHours * 60 + endMinutes;
  
  return endTotalMinutes < startTotalMinutes;
};

/**
 * Get the days of the week with names
 * @returns Array of day objects
 */
export const getDaysOfWeek = () => [
  { value: 0, label: 'Sunday', short: 'Sun' },
  { value: 1, label: 'Monday', short: 'Mon' },
  { value: 2, label: 'Tuesday', short: 'Tue' },
  { value: 3, label: 'Wednesday', short: 'Wed' },
  { value: 4, label: 'Thursday', short: 'Thu' },
  { value: 5, label: 'Friday', short: 'Fri' },
  { value: 6, label: 'Saturday', short: 'Sat' }
];

/**
 * Validate time format
 * @param time Time string to validate
 * @returns True if valid HH:MM format
 */
export const isValidTimeFormat = (time: string): boolean => {
  const timeRegex = /^([0-1][0-9]|2[0-3]):[0-5][0-9]$/;
  return timeRegex.test(time);
};

/**
 * Get default working hours in local time
 * @returns Object with start and end times
 */
export const getDefaultWorkingHours = () => ({
  start: '09:00',
  end: '17:00'
});