import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createSessionStore } from './sessionStore';
import type { AuthUser, LoginParams, RegisterParams } from '../../api/modules/auth';
import type { ApiResponse } from '../../api/api';

type AuthApiLike = {
  login: (data: LoginParams) => Promise<ApiResponse<void>>;
  register: (data: RegisterParams) => Promise<ApiResponse<void>>;
  getMe: () => Promise<ApiResponse<AuthUser>>;
  logout: () => Promise<ApiResponse<void>>;
};

describe('createSessionStore', () => {
  const user: AuthUser = {
    id: 1,
    username: 'alice',
    name: 'Alice',
    email: 'alice@example.com',
    status: 'active',
    roles: ['user'],
    permissions: ['svc:view'],
  };

  let authApi: AuthApiLike;

  beforeEach(() => {
    authApi = {
      login: vi.fn().mockResolvedValue({ success: true, data: undefined }),
      register: vi.fn().mockResolvedValue({ success: true, data: undefined }),
      getMe: vi.fn().mockResolvedValue({ success: true, data: user }),
      logout: vi.fn().mockResolvedValue({ success: true, data: undefined }),
    };
    localStorage.clear();
  });

  it('bootstraps from /auth/me and finalizes authenticated state in memory only', async () => {
    const store = createSessionStore(authApi);

    await store.bootstrap();

    expect(authApi.getMe).toHaveBeenCalledTimes(1);
    expect(store.getSnapshot()).toEqual({
      user,
      permissions: ['svc:view'],
      loading: false,
      isAuthenticated: true,
    });
    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('refreshToken')).toBeNull();
    expect(localStorage.getItem('user')).toBeNull();
    expect(localStorage.getItem('permissions')).toBeNull();
  });

  it('finalizes login by calling /auth/me after the login handshake', async () => {
    const store = createSessionStore(authApi);

    await store.login({ username: 'alice', password: 'secret' });

    expect(authApi.login).toHaveBeenCalledWith({ username: 'alice', password: 'secret' });
    expect(authApi.getMe).toHaveBeenCalledTimes(1);
    expect(store.getSnapshot().isAuthenticated).toBe(true);
  });
});
