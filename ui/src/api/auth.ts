import { apiClient } from './client';
import { Identity } from './types';

export const authApi = {
  getIdentity: () => apiClient.get<Identity>('/identity'),
};
