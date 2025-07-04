export interface User {
  id: string;
  username: string;
  email: string;
  role: string;
  firstName?: string;
  lastName?: string;
  createdAt: string;
  updatedAt: string;
  
  // Account status
  accountEnabled: boolean;
  accountLocked: boolean;
  accountLockedUntil?: string;
  
  // MFA settings
  mfaEnabled: boolean;
  mfaType: string[];
  preferredMFAMethod?: string;
  
  // Login information
  lastLogin?: string;
  lastPasswordChange?: string;
  failedLoginAttempts: number;
  lastFailedAttempt?: string;
  
  // Disable information
  disabledReason?: string;
  disabledAt?: string;
  disabledBy?: string;
  
  // Teams (if applicable)
  teams?: Team[];
}

export interface Team {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

export interface UserUpdateRequest {
  username?: string;
  email?: string;
  role?: string;
}

export interface DisableUserRequest {
  reason: string;
}

export interface ResetPasswordRequest {
  password?: string;
  temporary?: boolean;
}

export interface UserListResponse {
  data: User[];
}

export interface UserDetailResponse {
  data: User;
}