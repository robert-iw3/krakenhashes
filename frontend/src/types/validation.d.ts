declare module 'utils/validation' {
    export function validatePassword(password: string): boolean;
    export function calculatePasswordStrength(password: string): number;
} 