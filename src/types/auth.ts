export interface AuthState {
  isAuth: boolean;
  authChecked: boolean;
}

export interface LoginResponse {
  success: boolean;
  message?: string;
  token?: string;
}

export interface User {
  id: string;
  username: string;
  email: string;
}

export interface LoginCredentials {
  username: string;
  password: string;
}

export interface LoginProps {
  setIsAuth: (isAuth: boolean) => void;
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}