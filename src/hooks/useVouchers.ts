import { useState, useCallback } from 'react';
import { api } from '../services/api';

/**
 * Interfaces for voucher management
 */
interface User {
  id: string;
  username: string;
}

export interface TempVoucher {
  code: string;
  isContinuous: boolean;
  createdAt: string;
}

interface Voucher {
  code: string;
  createdBy: User;
  isContinuous: boolean;
  isActive: boolean;
  createdAt: string;
  expiresAt?: string;
  usedAt?: string;
  usedById?: string;
  usedBy?: User;
}

/**
 * Hook for managing agent vouchers
 * 
 * Features:
 * - Create vouchers (single-use or continuous)
 * - Fetch active vouchers
 * - Disable existing vouchers
 * 
 * Error Handling:
 * - API errors are caught and can be handled by the consuming component
 * - Network failures are properly handled
 * - Invalid responses trigger appropriate error messages
 */
export const useVouchers = () => {
  const [tempVoucher, setTempVoucher] = useState<TempVoucher | null>(null);
  const [vouchers, setVouchers] = useState<Voucher[]>([]);
  const [error, setError] = useState<string | null>(null);

  const resetTempVoucher = useCallback(() => {
    setTempVoucher(null);
  }, []);

  const createTempVoucher = useCallback(async (isContinuous: boolean) => {
    try {
      const response = await api.post<TempVoucher>('/api/vouchers/temp', { isContinuous });
      setTempVoucher(response.data);
      setError(null);
      await fetchVouchers(); // Refresh the vouchers list
      return response.data;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create voucher';
      setError(message);
      throw new Error(message);
    }
  }, []);

  const fetchVouchers = useCallback(async () => {
    try {
      const response = await api.get<Voucher[]>('/api/vouchers');
      setVouchers(response.data);
      setError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch vouchers';
      setError(message);
      throw new Error(message);
    }
  }, []);

  const disableVoucher = useCallback(async (code: string) => {
    try {
      await api.post(`/api/vouchers/${code}/disable`);
      setError(null);
      await fetchVouchers();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to disable voucher';
      setError(message);
      throw new Error(message);
    }
  }, []);

  return {
    tempVoucher,
    vouchers,
    error,
    createTempVoucher,
    fetchVouchers,
    disableVoucher,
    resetTempVoucher,
  };
}; 