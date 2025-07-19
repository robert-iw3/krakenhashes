import { api } from './api';

export interface CrackedHash {
  id: string;
  original_hash: string;
  password: string;
  hash_type_id: number;
  username?: string;
}

export interface PotResponse {
  hashes: CrackedHash[];
  total_count: number;
  limit: number;
  offset: number;
}

export interface PotParams {
  limit: number;
  offset: number;
}

export const potService = {
  // Get all cracked hashes
  getPot: async (params: PotParams): Promise<PotResponse> => {
    const response = await api.get<PotResponse>('/api/pot', { params });
    return response.data;
  },

  // Get cracked hashes by hashlist
  getPotByHashlist: async (hashlistId: string, params: PotParams): Promise<PotResponse> => {
    const response = await api.get<PotResponse>(`/api/pot/hashlist/${hashlistId}`, { params });
    return response.data;
  },

  // Get cracked hashes by client
  getPotByClient: async (clientId: string, params: PotParams): Promise<PotResponse> => {
    const response = await api.get<PotResponse>(`/api/pot/client/${clientId}`, { params });
    return response.data;
  },
};