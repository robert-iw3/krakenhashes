import { useState, useCallback } from 'react';
import { api } from '../services/api';

/**
 * Interfaces for voucher management
 */
interface User {
  id: string;
  username: string;
  email: string;
  role: string;
  createdAt: string;
}

export interface TempVoucher {
  code: string;
  is_continuous: boolean;
  created_at: string;
}

interface Voucher {
  code: string;
  created_by: User;
  created_by_id: string;
  is_continuous: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  used_at?: {
    Time: string;
    Valid: boolean;
  };
  used_by_agent_id?: {
    Int64: number;
    Valid: boolean;
  };
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
      const response = await api.post<TempVoucher>('/api/vouchers/temp', { is_continuous: isContinuous });
      console.log('Created temp voucher:', response.data);
      setTempVoucher(response.data);
      setError(null);
      await fetchVouchers(); // Refresh the vouchers list
      return response.data;
    } catch (err) {
      console.error('Error creating temp voucher:', err);
      const message = err instanceof Error ? err.message : 'Failed to create voucher';
      setError(message);
      throw new Error(message);
    }
  }, []);

  const fetchVouchers = useCallback(async () => {
    try {
      console.log('Fetching vouchers from API...');
      const response = await api.get<Voucher[]>('/api/vouchers');
      console.log('Received vouchers from API:', response.data);
      
      if (!Array.isArray(response.data)) {
        console.error('Expected array of vouchers but got:', response.data);
        setError('Invalid response format from server');
        return;
      }
      
      setVouchers(response.data);
      setError(null);
    } catch (err) {
      console.error('Error fetching vouchers:', err);
      const message = err instanceof Error ? err.message : 'Failed to fetch vouchers';
      setError(message);
      throw new Error(message);
    }
  }, []);

  const disableVoucher = useCallback(async (code: string) => {
    try {
      console.log('Disabling voucher:', code);
      await api.post(`/api/vouchers/${code}/disable`);
      setError(null);
      await fetchVouchers();
    } catch (err) {
      console.error('Error disabling voucher:', err);
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