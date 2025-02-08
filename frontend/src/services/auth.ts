import { api } from './api';
import { 
  LoginResponse, 
  AuthSettings, 
  MFASettings, 
  PasswordPolicy,
  AccountSecurity,
  AuthSettingsUpdate,
  AuthCheckResponse,
  MFAVerifyResponse
} from '../types/auth';

export const login = async (username: string, password: string): Promise<LoginResponse> => {
  try {
    const response = await api.post<LoginResponse>(
      '/api/login', 
      { username, password }
    );
    return response.data;
  } catch (error: unknown) {
    if (error && typeof error === 'object' && 'response' in error) {
      throw (error as any).response?.data;
    }
    throw new Error('An error occurred during login');
  }
};

export const logout = async (): Promise<void> => {
  try {
    // Let the backend handle cookie cleanup
    await api.post('/api/logout');
  } catch (error) {
    console.error('Logout failed:', error);
    throw error;
  }
};

export const isAuthenticated = async (): Promise<AuthCheckResponse> => {
  try {
    const response = await api.get<AuthCheckResponse>('/api/check-auth');
    return response.data;
  } catch (error) {
    return { authenticated: false };
  }
};

// Admin Auth Settings API
export const getAuthSettings = async (): Promise<AuthSettings> => {
  const response = await api.get<AuthSettings>('/api/admin/auth/settings');
  return response.data;
};

export const updateAuthSettings = async (settings: AuthSettingsUpdate): Promise<void> => {
  const requestData = {
    min_password_length: settings.minPasswordLength,
    require_uppercase: settings.requireUppercase,
    require_lowercase: settings.requireLowercase,
    require_numbers: settings.requireNumbers,
    require_special_chars: settings.requireSpecialChars,
    max_failed_attempts: settings.maxFailedAttempts,
    lockout_duration_minutes: settings.lockoutDuration,
    jwt_expiry_minutes: settings.jwtExpiryMinutes,
    display_timezone: "UTC",
    notification_aggregation_minutes: settings.notificationAggregationMinutes
  };
  
  console.log('Auth Settings Update Request:', requestData);
  
  try {
    await api.put('/api/admin/auth/settings', requestData);
  } catch (error: any) {
    console.error('Auth Settings Update Error:', error.response?.data);
    if (error.response?.data?.message) {
      throw new Error(error.response.data.message);
    }
    throw error;
  }
};

// Get MFA settings for admin configuration
export const getAdminMFASettings = async (): Promise<MFASettings> => {
  try {
    const response = await api.get('/api/admin/auth/settings/mfa');
    return response.data;
  } catch (error: any) {
    console.error('Get Admin MFA Settings Error:', error.response?.data);
    if (error.response?.data?.message) {
      throw new Error(error.response.data.message);
    }
    throw error;
  }
};

// Get MFA settings for the current user
export const getUserMFASettings = async (): Promise<MFASettings> => {
  try {
    const response = await api.get('/api/user/mfa/settings');
    return response.data;
  } catch (error: any) {
    console.error('Get User MFA Settings Error:', error.response?.data);
    if (error.response?.data?.message) {
      throw new Error(error.response.data.message);
    }
    throw error;
  }
};

export const updateMFASettings = async (settings: MFASettings): Promise<void> => {
  // Validate settings
  const emailValidity = typeof settings.emailCodeValidity === 'number' ? settings.emailCodeValidity : 0;
  const backupCount = typeof settings.backupCodesCount === 'number' ? settings.backupCodesCount : 0;
  const cooldown = typeof settings.mfaCodeCooldownMinutes === 'number' ? settings.mfaCodeCooldownMinutes : 0;
  const expiry = typeof settings.mfaCodeExpiryMinutes === 'number' ? settings.mfaCodeExpiryMinutes : 0;
  const maxAttempts = typeof settings.mfaMaxAttempts === 'number' ? settings.mfaMaxAttempts : 0;

  if (emailValidity < 1) {
    throw new Error('Email code validity must be at least 1 minute');
  }
  if (backupCount < 1) {
    throw new Error('Number of backup codes must be at least 1');
  }
  if (cooldown < 1) {
    throw new Error('Code cooldown must be at least 1 minute');
  }
  if (expiry < 1) {
    throw new Error('Code expiry must be at least 1 minute');
  }
  if (maxAttempts < 1) {
    throw new Error('Maximum attempts must be at least 1');
  }
  if (!settings.allowedMfaMethods || settings.allowedMfaMethods.length === 0) {
    throw new Error('At least one MFA method must be selected');
  }

  console.log('MFA Settings Update Request:', settings);

  try {
    await api.put('/api/admin/auth/settings/mfa', settings);
  } catch (error: any) {
    console.error('MFA Settings Update Error:', error.response?.data);
    if (error.response?.data?.message) {
      throw new Error(error.response.data.message);
    }
    throw error;
  }
};

export const getPasswordPolicy = async (): Promise<PasswordPolicy> => {
  const response = await api.get('/api/password/policy');
  return response.data;
};

export const getAccountSecurity = async (): Promise<AccountSecurity> => {
  const response = await api.get('/api/admin/auth/settings/security');
  return response.data;
};

// User MFA API
export const enableMFA = async (method: string): Promise<{ secret?: string; qrCode?: string }> => {
  try {
    const response = await api.post('/api/user/mfa/enable', { method });
    return response.data;
  } catch (error: any) {
    console.error('Enable MFA Error:', error.response?.data);
    if (error.response?.data?.message) {
      throw new Error(error.response.data.message);
    }
    throw error;
  }
};

export const verifyMFASetup = async (code: string): Promise<{ backupCodes?: string[] }> => {
  try {
    const response = await api.post('/api/user/mfa/verify-setup', { code });
    return response.data;
  } catch (error: any) {
    console.error('Verify MFA Setup Error:', error.response?.data);
    if (error.response?.data?.message) {
      throw new Error(error.response.data.message);
    }
    throw error;
  }
};

export const disableMFA = async (): Promise<void> => {
  try {
    await api.post('/api/user/mfa/disable');
  } catch (error: any) {
    console.error('Disable MFA Error:', error.response?.data);
    if (error.response?.data?.message) {
      throw new Error(error.response.data.message);
    }
    throw error;
  }
};

export const generateBackupCodes = async (): Promise<string[]> => {
  try {
    const response = await api.post('/api/user/mfa/backup-codes');
    return response.data.codes;
  } catch (error: any) {
    console.error('Generate Backup Codes Error:', error.response?.data);
    if (error.response?.data?.message) {
      throw new Error(error.response.data.message);
    }
    throw error;
  }
};

export const verifyMFACode = async (code: string, method: string): Promise<void> => {
  try {
    await api.post('/api/user/mfa/verify', { code, method });
  } catch (error: any) {
    console.error('Verify MFA Code Error:', error.response?.data);
    if (error.response?.data?.message) {
      throw new Error(error.response.data.message);
    }
    throw error;
  }
};

export const verifyMFA = async (
  sessionToken: string,
  code: string,
  method: string
): Promise<MFAVerifyResponse> => {
  try {
    const response = await api.post<MFAVerifyResponse>('/api/verify-mfa', {
      sessionToken,
      code,
      method,
    });
    return {
      ...response.data,
      token: response.data.token || '',
      remainingAttempts: response.data.remainingAttempts || 0,
    };
  } catch (error: unknown) {
    if (error && typeof error === 'object' && 'response' in error) {
      const responseData = (error as any).response?.data;
      return {
        success: false,
        token: '',
        message: responseData?.message || 'Verification failed',
        remainingAttempts: responseData?.remainingAttempts || 0,
      };
    }
    throw new Error('An error occurred during MFA verification');
  }
}; 