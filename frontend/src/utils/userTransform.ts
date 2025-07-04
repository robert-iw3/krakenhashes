import { User } from '../types/user';

// Transform snake_case backend response to camelCase frontend type
export const transformUserResponse = (backendUser: any): User => {
  return {
    id: backendUser.id,
    username: backendUser.username,
    email: backendUser.email,
    role: backendUser.role,
    firstName: backendUser.first_name,
    lastName: backendUser.last_name,
    createdAt: backendUser.created_at,
    updatedAt: backendUser.updated_at,
    
    // Account status - these are the fields causing the issue
    accountEnabled: backendUser.account_enabled,
    accountLocked: backendUser.account_locked,
    accountLockedUntil: backendUser.account_locked_until,
    
    // MFA settings
    mfaEnabled: backendUser.mfa_enabled,
    mfaType: backendUser.mfa_type || [],
    preferredMFAMethod: backendUser.preferred_mfa_method,
    
    // Login information
    lastLogin: backendUser.last_login,
    lastPasswordChange: backendUser.last_password_change,
    failedLoginAttempts: backendUser.failed_login_attempts || 0,
    lastFailedAttempt: backendUser.last_failed_attempt,
    
    // Disable information
    disabledReason: backendUser.disabled_reason,
    disabledAt: backendUser.disabled_at,
    disabledBy: backendUser.disabled_by,
    
    // Teams (if applicable)
    teams: backendUser.teams || []
  };
};

export const transformUserListResponse = (response: any): User[] => {
  if (!response.data || !Array.isArray(response.data)) {
    return [];
  }
  return response.data.map(transformUserResponse);
};