/**
 * Represents a client entity in the KrakenHashes system.
 */
export interface Client {
  id: string; // Assuming UUID as string
  name: string;
  description?: string;
  contactInfo?: string;
  dataRetentionMonths?: number | null; // Added: number of months, null means use default
  createdAt?: string; // Assuming ISO string format
  updatedAt?: string; // Assuming ISO string format
  cracked_count?: number; // Count of cracked hashes for this client
} 