/**
 * API service for making HTTP requests to the backend
 */
import axios from 'axios';
import type { AxiosError } from 'axios';

// Use HTTPS API URL for all secure endpoints
const API_URL = process.env.REACT_APP_API_URL || 'https://localhost:31337';
// Use HTTP API URL only for CA certificate
const HTTP_API_URL = process.env.REACT_APP_HTTP_API_URL || 'http://localhost:1337';

// Function to fetch and store CA certificate
const fetchCACertificate = async (): Promise<void> => {
  try {
    console.debug('[API] Fetching CA certificate...');
    // Use HTTP API URL specifically for CA certificate
    const response = await fetch(`${HTTP_API_URL}/ca.crt`, {
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
    link.download = 'hashdom-ca.crt';
    
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
      <p>To use HashDom securely, you need to install our CA certificate:</p>
      <ol>
        <li>Click "Download Certificate" below</li>
        <li>Open the downloaded certificate (hashdom-ca.crt)</li>
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
    'Content-Type': 'application/json',
  }
});

// Request interceptor
api.interceptors.request.use((config) => {
  // Log the request
  logApiCall(config.method?.toUpperCase() || 'UNKNOWN', config.url || '', config.data);

  const token = localStorage.getItem('token');
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`;
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

    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
); 