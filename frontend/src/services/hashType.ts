import { api } from './api';
import { HashType, HashTypeCreateRequest, HashTypeUpdateRequest } from '../types/hashType';

export const getHashTypes = async (enabledOnly: boolean = false): Promise<HashType[]> => {
  const response = await api.get('/api/hashtypes', {
    params: { enabled_only: enabledOnly.toString() }
  });
  return response.data;
};

export const createHashType = async (data: HashTypeCreateRequest): Promise<HashType> => {
  const response = await api.post('/api/hashtypes', data);
  return response.data;
};

export const updateHashType = async (id: number, data: HashTypeUpdateRequest): Promise<HashType> => {
  const response = await api.put(`/api/hashtypes/${id}`, data);
  return response.data;
};

export const deleteHashType = async (id: number): Promise<void> => {
  await api.delete(`/api/hashtypes/${id}`);
};