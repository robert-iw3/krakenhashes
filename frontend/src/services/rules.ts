/**
 * API services for rule management
 */
import { api } from './api';
import { Rule, RuleFilters, RuleUploadResponse } from '../types/rules';

// Get all rules with optional filtering
export const getRules = (filters?: RuleFilters) => 
  api.get<Rule[]>('/api/rules', { params: filters });

// Get a single rule by ID
export const getRule = (id: string) => 
  api.get<Rule>(`/api/rules/${id}`);

// Upload a new rule
export const uploadRule = (formData: FormData) => 
  api.post<RuleUploadResponse>('/api/rules/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    },
    withCredentials: true, // Ensure cookies are sent with the request
    onUploadProgress: (progressEvent) => {
      const percentCompleted = Math.round((progressEvent.loaded * 100) / (progressEvent.total || 1));
      console.debug(`[Rule Upload] ${percentCompleted}% completed`);
    },
    timeout: 86400000 // 24 hours timeout for extremely large files
  });

// Update rule metadata
export const updateRule = (id: string, data: Partial<Rule>) => 
  api.put<Rule>(`/api/rules/${id}`, data, {
    withCredentials: true // Ensure cookies are sent with the request
  });

// Delete a rule
export const deleteRule = (id: string) => 
  api.delete(`/api/rules/${id}`);

// Enable/disable a rule
export const toggleRuleStatus = (id: string, isEnabled: boolean) => 
  api.put<Rule>(`/api/rules/${id}/status`, { is_enabled: isEnabled });

// Download a rule
export const downloadRule = (id: string) => 
  api.get(`/api/rules/${id}/download`, { responseType: 'blob' });

// Verify a rule
export const verifyRule = (id: string, status: 'verified' | 'failed' | 'pending', ruleCount?: number) => 
  api.post(`/api/rules/${id}/verify`, { 
    status, 
    rule_count: ruleCount 
  }, { withCredentials: true }); 