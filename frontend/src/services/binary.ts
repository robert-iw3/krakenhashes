import { api } from './api';

export interface BinaryVersion {
  id: number;
  binary_type: 'hashcat' | 'john';
  compression_type: '7z' | 'zip' | 'tar.gz' | 'tar.xz';
  source_url: string;
  file_name: string;
  md5_hash: string;
  file_size: number;
  created_at: string;
  is_active: boolean;
  is_default: boolean;
  last_verified_at: string | null;
  verification_status: 'pending' | 'verified' | 'failed' | 'deleted';
}

export interface AddBinaryRequest {
  binary_type: 'hashcat' | 'john';
  compression_type: '7z' | 'zip' | 'tar.gz' | 'tar.xz';
  source_url: string;
  file_name: string;
  set_as_default?: boolean;
}

export const listBinaries = async () => {
  try {
    const response = await api.get<BinaryVersion[]>('/api/admin/binary');
    console.debug('Binary list response:', response);
    return response;
  } catch (error) {
    console.error('Error in listBinaries:', error);
    throw error;
  }
};

export const addBinary = (binary: AddBinaryRequest) => {
  return api.post<BinaryVersion>('/api/admin/binary', binary);
};

export const verifyBinary = (id: number) => {
  return api.post<void>(`/api/admin/binary/${id}/verify`);
};

export const deleteBinary = (id: number) => {
  return api.delete<void>(`/api/admin/binary/${id}`);
};

export const getBinary = (id: number) => {
  return api.get<BinaryVersion>(`/api/admin/binary/${id}`);
};

export const setDefaultBinary = (id: number) => {
  return api.put<{ message: string }>(`/api/admin/binary/${id}/set-default`);
}; 