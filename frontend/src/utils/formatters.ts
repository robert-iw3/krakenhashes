/**
 * Utility functions for formatting data
 */

/**
 * Format a file size in bytes to a human-readable string
 * @param bytes File size in bytes
 * @param decimals Number of decimal places to show
 * @returns Formatted string with appropriate unit (B, KB, MB, GB, TB)
 */
export const formatFileSize = (bytes: number, decimals: number = 2): string => {
  if (bytes === 0) return '0 Bytes';

  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  
  // Determine the appropriate unit
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  // Format the number with the appropriate unit
  return parseFloat((bytes / Math.pow(k, i)).toFixed(decimals)) + ' ' + sizes[i];
}; 