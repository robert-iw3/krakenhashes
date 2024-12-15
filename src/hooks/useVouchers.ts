import { useState, useCallback } from 'react';
import { api } from '../services/api';

/**
 * Interfaces for voucher management
 */
interface TempVoucher {
  code: string;
  expiresAt: string;
}

interface Voucher {
  code: string;
  createdBy: string;
  isContinuous: boolean;
  createdAt: string;
  disabledAt: string | null;
}

/**
 * Hook for managing agent vouchers
 * 
 * Features:
 * - Create temporary vouchers
 * - Confirm and activate vouchers
 * - Fetch active vouchers
 * - Disable existing vouchers
 * 
 * Error Handling:
 * - API errors are caught and can be handled by the consuming component
 * - Network failures are properly handled
 * - Invalid responses trigger appropriate error messages
 * 
 * Usage:
 * ```tsx
 * const { tempVoucher, createTempVoucher, confirmVoucher } = useVouchers();
 * 
 * // Create a new temporary voucher
 * await createTempVoucher();
 * 
 * // Confirm and activate the voucher
 * await confirmVoucher(code, isContinuous);
 * ```
 */
export const useVouchers = () => {
  const [tempVoucher, setTempVoucher] = useState<TempVoucher | null>(null);
  const [vouchers, setVouchers] = useState<Voucher[]>([]);
  const [error, setError] = useState<string | null>(null);

  const createTempVoucher = useCallback(async () => {
    try {
      const response = await api.post<TempVoucher>('/api/vouchers/temp');
      setTempVoucher(response.data);
      setError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create temporary voucher';
      setError(message);
      throw new Error(message);
    }
  }, []);

  const confirmVoucher = useCallback(async (code: string, isContinuous: boolean) => {
    try {
      await api.post('/api/vouchers/confirm', { code, isContinuous });
      setTempVoucher(null);
      setError(null);
      await fetchVouchers();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to confirm voucher';
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
    confirmVoucher,
    fetchVouchers,
    disableVoucher,
  };
}; 