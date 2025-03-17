/**
 * Types for wordlist management
 */

export enum WordlistType {
  GENERAL = 'general',
  SPECIALIZED = 'specialized',
  TARGETED = 'targeted',
  CUSTOM = 'custom'
}

export enum WordlistStatus {
  READY = 'verified',
  PROCESSING = 'pending',
  ERROR = 'error',
  DELETED = 'deleted'
}

export interface Wordlist {
  id: string;
  name: string;
  description: string;
  wordlist_type: WordlistType;
  format: string;
  file_name: string;
  md5_hash: string;
  file_size: number;
  word_count: number;
  created_at: string;
  updated_at: string;
  verification_status: WordlistStatus;
  created_by: string;
  updated_by?: string;
  last_verified_at?: string;
  tags?: string[];
  is_enabled: boolean;
}

export interface WordlistUploadResponse {
  id: string;
  name: string;
  message: string;
  success: boolean;
  duplicate?: boolean;
}

export interface WordlistFilters {
  search?: string;
  wordlist_type?: WordlistType;
  verification_status?: WordlistStatus;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
} 