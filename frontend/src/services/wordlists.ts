/**
 * API services for wordlist management
 */
import { api } from './api';
import { Wordlist, WordlistFilters, WordlistUploadResponse } from '../types/wordlists';

// Get all wordlists with optional filtering
export const getWordlists = (filters?: WordlistFilters) => 
  api.get<Wordlist[]>('/api/wordlists', { params: filters });

// Get a single wordlist by ID
export const getWordlist = (id: string) => 
  api.get<Wordlist>(`/api/wordlists/${id}`);

// Upload a new wordlist
export const uploadWordlist = (formData: FormData, onProgress?: (progress: number) => void) => 
  api.post<WordlistUploadResponse>('/api/wordlists/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    },
    withCredentials: true, // Ensure cookies are sent with the request
    onUploadProgress: (progressEvent) => {
      const percentCompleted = Math.round((progressEvent.loaded * 100) / (progressEvent.total || 1));
      console.debug(`[Wordlist Upload] ${percentCompleted}% completed`);
      if (onProgress) {
        onProgress(percentCompleted);
      }
    },
    timeout: 86400000 // 24 hours timeout for extremely large files
  });

// Update wordlist metadata
export const updateWordlist = (id: string, data: Partial<Wordlist>) => 
  api.put<Wordlist>(`/api/wordlists/${id}`, data, {
    withCredentials: true // Ensure cookies are sent with the request
  });

// Delete a wordlist
export const deleteWordlist = (id: string) => 
  api.delete(`/api/wordlists/${id}`, { withCredentials: true });

// Verify a wordlist
export const verifyWordlist = (id: string, status: 'verified' | 'failed' | 'pending', wordCount?: number) => 
  api.post(`/api/wordlists/${id}/verify`, { 
    status, 
    word_count: wordCount 
  });

// Enable/disable a wordlist
export const toggleWordlistStatus = (id: string, isEnabled: boolean) => 
  api.put<Wordlist>(`/api/wordlists/${id}/status`, { is_enabled: isEnabled });

// Download a wordlist
export const downloadWordlist = (id: string) => 
  api.get(`/api/wordlists/${id}/download`, { responseType: 'blob' }); 