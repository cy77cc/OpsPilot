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

export interface AuthPayload {
  token: string;
  refreshToken?: string;
  user: AuthUser;
  permissions: string[];
}

export const authApi = {
  async login(data: LoginParams): Promise<ApiResponse<AuthPayload>> {
    const res = await apiService.post<any>('/auth/login', data);
    const token = res.data?.token || res.data?.accessToken || '';
    const refreshToken = res.data?.refreshToken || '';
    const fallbackUser: AuthUser = {
      id: Number(res.data?.uid || 0),
      username: data.username,
      name: data.username,
      email: '',
      status: 'active',
      roles: res.data?.roles || [],
      permissions: [],
    };
    const user = res.data?.user ? ({
      ...res.data.user,
      id: Number(res.data.user.id || 0),
      roles: res.data.user.roles || [],
      permissions: res.data.user.permissions || [],
    } as AuthUser) : fallbackUser;
    return {
      ...res,
      data: {
        token,
        refreshToken,
        user,
        permissions: res.data?.permissions || user.permissions || [],
      },
    };
  },

  async register(data: RegisterParams): Promise<ApiResponse<AuthPayload>> {
    const res = await apiService.post<any>('/auth/register', data);
    const token = res.data?.token || res.data?.accessToken || '';
    const refreshToken = res.data?.refreshToken || '';
    const user: AuthUser = {
      id: Number(res.data?.uid || 0),
      username: data.username,
      name: data.name || data.username,
      email: data.email,
      status: 'active',
      roles: res.data?.roles || [],
      permissions: [],
    };
    return {
      ...res,
      data: {
        token,
        refreshToken,
        user,
        permissions: res.data?.permissions || [],
      },
    };
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

  async logout(refreshToken?: string): Promise<ApiResponse<void>> {
    return apiService.post('/auth/logout', refreshToken ? { refreshToken } : {});
  },
};
