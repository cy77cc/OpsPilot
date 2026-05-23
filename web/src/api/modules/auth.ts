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
    const res = await apiService.get<Record<string, unknown>>('/auth/me');
    const raw = (res.data || {}) as Record<string, unknown>;
    // Backend authPublicResp wraps user inside a "user" field:
    // { user: { id, username, ... }, roles: [...], permissions: [...] }
    const userObj = (raw.user && typeof raw.user === 'object' ? raw.user : raw) as Partial<AuthUser> & Record<string, unknown>;
    return {
      ...res,
      data: {
        id: Number(userObj.id || 0),
        username: String(userObj.username || ''),
        name: String(userObj.name || userObj.username || ''),
        email: String(userObj.email || ''),
        status: String(userObj.status || 'active'),
        roles: Array.isArray(userObj.roles) ? userObj.roles.map((role) => String(role)) : [],
        permissions: Array.isArray(userObj.permissions) ? userObj.permissions.map((permission) => String(permission)) : [],
      },
    };
  },

  async logout(): Promise<ApiResponse<void>> {
    return apiService.post('/auth/logout', {});
  },
};
