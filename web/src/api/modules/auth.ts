import apiService from '../api';
import type { ApiResponse } from '../api';

export interface AuthUser {
  id: number;
  username: string;
  name: string;
  email: string;
  status: string;
  roles: string[];
  permissions?: string[];
}

export interface LoginParams {
  username: string;
  password: string;
}

export interface RegisterParams {
  username: string;
  name?: string;
  email: string;
  password: string;
}

export const authApi = {
  async login(data: LoginParams): Promise<ApiResponse<void>> {
    const res = await apiService.post('/auth/login', data);
    return { ...res, data: undefined };
  },

  async register(data: RegisterParams): Promise<ApiResponse<void>> {
    const res = await apiService.post('/auth/register', data);
    return { ...res, data: undefined };
  },

  async getMe(): Promise<ApiResponse<AuthUser>> {
    const res = await apiService.get<any>('/auth/me');
    return {
      ...res,
      data: {
        id: Number(res.data?.id || 0),
        username: res.data?.username || '',
        name: res.data?.name || res.data?.username || '',
        email: res.data?.email || '',
        status: res.data?.status || 'active',
        roles: res.data?.roles || [],
        permissions: res.data?.permissions || [],
      },
    };
  },

  async logout(): Promise<ApiResponse<void>> {
    return apiService.post('/auth/logout', {});
  },
};
