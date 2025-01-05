export const validatePassword = (password: string): boolean => {
    // Minimum 15 characters, at least one uppercase letter, one lowercase letter, one number
    const passwordRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[a-zA-Z\d]{15,}$/;
    return passwordRegex.test(password);
};

export const calculatePasswordStrength = (password: string): number => {
    let strength = 0;
    if (password.length >= 15) strength += 1;
    if (/[A-Z]/.test(password)) strength += 1;
    if (/[a-z]/.test(password)) strength += 1;
    if (/[0-9]/.test(password)) strength += 1;
    if (/[^A-Za-z0-9]/.test(password)) strength += 1;
    return strength * 20;
}; 