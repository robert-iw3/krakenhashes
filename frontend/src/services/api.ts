/**
 * API service for making HTTP requests to the backend
 */
import axios from 'axios';
import type { AxiosError } from 'axios';
import { Client } from '../types/client'; // Moved import to top
import {
  PresetJob,
  JobWorkflow,
  PresetJobFormDataResponse,
  CreateWorkflowRequest,
  UpdateWorkflowRequest,
  PresetJobInput,
  PresetJobApiData,
  JobWorkflowFormDataResponse,
} from '../types/adminJobs';

// Use HTTPS API URL for all secure endpoints
const API_URL = process.env.REACT_APP_API_URL || 'https://localhost:31337';

// Function to fetch and store CA certificate
const fetchCACertificate = async (): Promise<void> => {
  try {
    console.debug('[API] Fetching CA certificate...');
    // Use HTTP API URL specifically for CA certificate
    const response = await fetch(`http://${window.location.hostname}:1337/ca.crt`, {
      method: 'GET',
      credentials: 'include',
      mode: 'cors',
      headers: {
        'Accept': 'application/x-x509-ca-cert'
      }
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch CA certificate: ${response.statusText}`);
    }

    const certBlob = await response.blob();
    console.debug('[API] Successfully fetched CA certificate');

    // Create a download link for the user
    const downloadUrl = window.URL.createObjectURL(certBlob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = 'krakenhashes-ca.crt';
    
    // Add instructions for the user
    const instructions = document.createElement('div');
    instructions.style.position = 'fixed';
    instructions.style.top = '20px';
    instructions.style.left = '50%';
    instructions.style.transform = 'translateX(-50%)';
    instructions.style.backgroundColor = '#f8d7da';
    instructions.style.color = '#721c24';
    instructions.style.padding = '20px';
    instructions.style.borderRadius = '5px';
    instructions.style.zIndex = '9999';
    instructions.innerHTML = `
      <h3>Security Certificate Required</h3>
      <p>To use KrakenHashes securely, you need to install our CA certificate:</p>
      <ol>
        <li>Click "Download Certificate" below</li>
        <li>Open the downloaded certificate (krakenhashes-ca.crt)</li>
        <li>When prompted, select "Trust this CA to identify websites"</li>
        <li>Complete the installation</li>
        <li>Restart your browser</li>
        <li>Refresh this page after installation</li>
      </ol>
      <button onclick="this.parentElement.remove()" style="margin-top: 10px; padding: 8px 16px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;">
        Download Certificate
      </button>
    `;

    // Add click handler to the button
    instructions.querySelector('button')?.addEventListener('click', () => {
      link.click();
      window.URL.revokeObjectURL(downloadUrl);
    });

    document.body.appendChild(instructions);
  } catch (error) {
    console.error('[API] Error fetching CA certificate:', error);
    throw error;
  }
};

// Debug logging for API calls
const logApiCall = (method: string, url: string, data?: any) => {
  console.debug(`[API] ${method} ${url}`, data || '');
};

// Debug logging for API responses
const logApiResponse = (method: string, url: string, response: any) => {
  console.debug(`[API] Response from ${method} ${url}:`, response);
};

// Debug logging for API errors
const logApiError = (method: string, url: string, error: any) => {
  console.error(`[API] Error in ${method} ${url}:`, {
    message: error.message,
    code: error.code,
    response: error.response?.data,
    status: error.response?.status,
    headers: error.response?.headers,
    config: error.config
  });

  // Additional TLS/certificate error debugging
  if (error.message?.includes('certificate')) {
    console.error('[API] Certificate Error Details:', {
      url: error.config?.url,
      protocol: error.config?.protocol,
      method: error.config?.method,
      headers: error.config?.headers
    });
  }
};

// Initialize API client with HTTPS API URL
export const api = axios.create({
  baseURL: API_URL,
  withCredentials: true, // Required for cookies/session
  headers: {
  }
});

// Request interceptor
api.interceptors.request.use((config) => {
  // Log the request
  logApiCall(config.method?.toUpperCase() || 'UNKNOWN', config.url || '', config.data);
  
  // Add debug info for auth-related requests
  if (config.url?.includes('auth') || config.url?.includes('login') || config.url?.includes('logout')) {
    console.debug('[API] Auth request cookies:', document.cookie);
  }
  
  // Special handling for multipart/form-data uploads
  if (config.data instanceof FormData) {
    console.debug('[API] Handling FormData upload');
    config.withCredentials = true;
    console.debug('[API] Upload request cookies:', document.cookie);
  } else if (config.headers && config.headers['Content-Type'] === 'multipart/form-data') {
    console.warn('[API] multipart/form-data Content-Type header was set manually?');
    config.withCredentials = true;
    console.debug('[API] Upload request cookies:', document.cookie);
  }
  
  return config;
});

// Response interceptor
api.interceptors.response.use(
  (response) => {
    // Log successful response
    logApiResponse(response.config.method?.toUpperCase() || 'UNKNOWN', response.config.url || '', response.data);
    return response;
  },
  async (error: AxiosError) => {
    // Log detailed error information
    logApiError(
      error.config?.method?.toUpperCase() || 'UNKNOWN',
      error.config?.url || '',
      error
    );

    // If we get a certificate error, try to fetch and prompt for CA certificate installation
    if (error.message?.includes('certificate') || error.code === 'CERT_NOT_TRUSTED') {
      await fetchCACertificate();
      // Redirect to root to trigger certificate check
      window.location.href = '/';
      return Promise.reject(error);
    }

    // Skip logout for network errors (which could be CORS issues)
    if (error.code === 'ERR_NETWORK') {
      console.debug('[API] Network error detected, skipping logout:', error.message);
      return Promise.reject(error);
    }

    // Handle authentication errors
    if (error.response?.status === 401) {
      console.debug('[API] Auth error, current cookies:', document.cookie);
      
      // Don't handle 401s from login/logout endpoints to prevent loops
      // Also skip auto-logout for rule and wordlist update endpoints
      const skipAutoLogoutEndpoints = [
        '/login', 
        '/logout',
        '/api/rules/',
        '/api/wordlists/'
      ];
      
      const shouldSkipAutoLogout = skipAutoLogoutEndpoints.some(endpoint => 
        error.config?.url?.includes(endpoint)
      );
      
      if (!shouldSkipAutoLogout) {
        try {
          // Call logout endpoint to clean up server-side session
          await api.post('/api/logout');
        } catch (logoutError) {
          console.error('[API] Error during logout:', logoutError);
        }
        
        // Only redirect if we're not already on the login page
        if (window.location.pathname !== '/login') {
          window.location.href = '/login';
        }
      } else {
        console.debug('[API] Skipping auto-logout for endpoint:', error.config?.url);
      }
      
      return Promise.reject(error);
    }

    return Promise.reject(error);
  }
);

// Email configuration
export const getEmailConfig = () => api.get('/api/admin/email/config');
export const updateEmailConfig = (config: any) => api.put('/api/admin/email/config', config);
export const testEmailConfig = (config: any) => api.post('/api/admin/email/test', config);

// Email templates
export const getEmailTemplates = () => api.get('/api/admin/email/templates');
export const createEmailTemplate = (template: any) => api.post('/api/admin/email/templates', template);
export const updateEmailTemplate = (id: number, template: any) => api.put(`/api/admin/email/templates/${id}`, template);
export const deleteEmailTemplate = (id: number) => api.delete(`/api/admin/email/templates/${id}`);
export const getEmailTemplate = (id: number) => api.get(`/api/admin/email/templates/${id}`);

// Email usage
export const getEmailUsage = () => api.get('/api/admin/email/usage');

// --- Client Settings (Admin) ---

// Define the actual API response structure
interface ClientSettingResponse { 
  data: ClientSetting;
}

// Define the setting object structure
interface ClientSetting { 
  key: string;
  value?: string | null;
  description?: string | null;
  updatedAt: string;
}

interface UpdateClientSettingPayload {
  value: string; // Value must be sent as string
}

// Get Default Client Data Retention Setting
export const getDefaultClientRetentionSetting = () => api.get<ClientSettingResponse>('/api/admin/settings/retention');

// Update Default Client Data Retention Setting
export const updateDefaultClientRetentionSetting = (payload: UpdateClientSettingPayload) => 
  api.put<any>('/api/admin/settings/retention', payload); // Backend expects { value: "months" }


// --- Client Management (Admin) ---


// List all clients
export const listAdminClients = () => api.get<{data: Client[]}>('/api/admin/clients');

// Get a single client by ID
export const getAdminClient = (id: string) => api.get<{data: Client}>(`/api/admin/clients/${id}`);

// Create a new client
export const createAdminClient = (clientData: Omit<Client, 'id' | 'createdAt' | 'updatedAt'>) => 
  api.post<{data: Client}>('/api/admin/clients', clientData);

// Update an existing client
export const updateAdminClient = (id: string, clientData: Partial<Omit<Client, 'id' | 'createdAt' | 'updatedAt'>>) => 
  api.put<{data: Client}>(`/api/admin/clients/${id}`, clientData);

// Delete a client
export const deleteAdminClient = (id: string) => api.delete<any>(`/api/admin/clients/${id}`); 

// --- Admin: Preset Jobs ---

export const getPresetJobFormData = async (): Promise<PresetJobFormDataResponse> => {
  const response = await api.get<PresetJobFormDataResponse>('/api/admin/preset-jobs/form-data');
  return response.data;
};

export const listPresetJobs = async (): Promise<PresetJob[]> => {
  // TODO: Add pagination params if needed
  const response = await api.get<PresetJob[]>('/api/admin/preset-jobs');
  return response.data;
};

export const getPresetJob = async (id: string): Promise<PresetJob> => {
  const response = await api.get<PresetJob>(`/api/admin/preset-jobs/${id}`);
  return response.data;
};

export const createPresetJob = async (data: PresetJobInput): Promise<PresetJob> => {
  // Prepare the data to match what the backend expects (convert to strings)
  console.log('Original input data:', JSON.stringify(data, null, 2));
  
  const apiData = {
    ...data,
    // Convert priority to number if it's a string
    priority: typeof data.priority === 'string' ? parseInt(data.priority) || 0 : data.priority,
    // Handle both number[] and string[] inputs by ensuring all IDs are strings
    wordlist_ids: Array.isArray(data.wordlist_ids) 
      ? data.wordlist_ids.map(id => id.toString())
      : [],
    rule_ids: Array.isArray(data.rule_ids) 
      ? data.rule_ids.map(id => id.toString())
      : [],
    // Add missing required fields
    status_updates_enabled: true, // Default to true for new jobs
    // Ensure allow_high_priority_override is properly set
    allow_high_priority_override: Boolean(data.allow_high_priority_override)
  };
  
  console.log('Converted API data:', JSON.stringify(apiData, null, 2));
  
  const response = await api.post<PresetJob>('/api/admin/preset-jobs', apiData);
  return response.data;
};

export const updatePresetJob = async (id: string, data: PresetJobInput): Promise<PresetJob> => {
  // Prepare the data to match what the backend expects (convert to strings)
  console.log('Original update data:', JSON.stringify(data, null, 2));
  
  const apiData = {
    ...data,
    // Convert priority to number if it's a string
    priority: typeof data.priority === 'string' ? parseInt(data.priority) || 0 : data.priority,
    // Handle both number[] and string[] inputs by ensuring all IDs are strings
    wordlist_ids: Array.isArray(data.wordlist_ids) 
      ? data.wordlist_ids.map(id => id.toString())
      : [],
    rule_ids: Array.isArray(data.rule_ids) 
      ? data.rule_ids.map(id => id.toString())
      : [],
    // Add missing required fields
    status_updates_enabled: true, // Default to true for updated jobs
    // Ensure allow_high_priority_override is properly set
    allow_high_priority_override: Boolean(data.allow_high_priority_override)
  };
  
  console.log('Converted update API data:', JSON.stringify(apiData, null, 2));
  
  const response = await api.put<PresetJob>(`/api/admin/preset-jobs/${id}`, apiData);
  return response.data;
};

export const deletePresetJob = async (id: string): Promise<void> => {
  await api.delete(`/api/admin/preset-jobs/${id}`);
};

// --- Admin: Job Workflows ---

export const listJobWorkflows = async (): Promise<JobWorkflow[]> => {
    // Returns list without steps populated
  const response = await api.get<JobWorkflow[]>('/api/admin/job-workflows');
  return response.data;
};

export const getJobWorkflow = async (id: string): Promise<JobWorkflow> => {
    // Returns workflow with steps populated
  const response = await api.get<JobWorkflow>(`/api/admin/job-workflows/${id}`);
  return response.data;
};

export const getJobWorkflowFormData = async (): Promise<JobWorkflowFormDataResponse> => {
  // Returns available preset jobs for workflow form
  const response = await api.get<JobWorkflowFormDataResponse>('/api/admin/job-workflows/form-data');
  return response.data;
};

export const createJobWorkflow = async (data: CreateWorkflowRequest): Promise<JobWorkflow> => {
  const response = await api.post<JobWorkflow>('/api/admin/job-workflows', data);
  return response.data;
};

export const updateJobWorkflow = async (id: string, data: UpdateWorkflowRequest): Promise<JobWorkflow> => {
  const response = await api.put<JobWorkflow>(`/api/admin/job-workflows/${id}`, data);
  return response.data;
};

export const deleteJobWorkflow = async (id: string): Promise<void> => {
  await api.delete(`/api/admin/job-workflows/${id}`);
}; 