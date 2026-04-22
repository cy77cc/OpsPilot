import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, act, cleanup } from '@testing-library/react';
import React from 'react';

const mockGetMe = vi.fn();
const mockLogin = vi.fn();
const mockRegister = vi.fn();
const mockLogout = vi.fn();

vi.mock('../../api/modules/auth', () => ({
  authApi: {
    login: (payload: unknown) => mockLogin(payload),
    register: (payload: unknown) => mockRegister(payload),
    logout: (payload?: unknown) => mockLogout(payload),
    getMe: () => mockGetMe(),
  },
}));

vi.mock('../../api/api', () => ({
  TOKEN_EVENTS: {
    REFRESHED: 'tokenRefreshed',
    EXPIRED: 'tokenExpired',
    NEEDS_REFRESH: 'tokenNeedsRefresh',
  },
}));

import { AuthProvider, useAuth } from './AuthContext';

const AuthStatusDisplay = () => {
  const { isAuthenticated } = useAuth();
  return <div data-testid="auth-status">{isAuthenticated ? 'authenticated' : 'not-authenticated'}</div>;
};

const TOKEN_STORAGE_KEYS = new Set(['token', 'refreshToken']);

const createTokenStorageSpies = () => {
  const getItemSpy = vi.spyOn(Storage.prototype, 'getItem');
  const setItemSpy = vi.spyOn(Storage.prototype, 'setItem');
  const removeItemSpy = vi.spyOn(Storage.prototype, 'removeItem');

  const assertNoTokenStorageDependency = () => {
    const tokenKeyCalls = [...getItemSpy.mock.calls, ...setItemSpy.mock.calls, ...removeItemSpy.mock.calls]
      .filter(([key]) => TOKEN_STORAGE_KEYS.has(String(key)));

    expect(tokenKeyCalls).toEqual([]);
  };

  return {
    assertNoTokenStorageDependency,
  };
};

describe('AuthContext Cookie Session Flow', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    sessionStorage.clear();

    mockGetMe.mockResolvedValue({
      data: {
        id: 1,
        username: 'testuser',
        name: 'Test User',
        email: 'test@example.com',
        status: 'active',
        roles: ['user'],
        permissions: [],
      },
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    localStorage.clear();
    sessionStorage.clear();
  });

  it('bootstraps authenticated state from cookie session without localStorage token keys', async () => {
    const { assertNoTokenStorageDependency } = createTokenStorageSpies();

    await act(async () => {
      render(
        <AuthProvider>
          <AuthStatusDisplay />
        </AuthProvider>
      );
    });

    await waitFor(() => {
      expect(mockGetMe).toHaveBeenCalledTimes(1);
    });

    await waitFor(() => {
      expect(screen.getByTestId('auth-status').textContent).toBe('authenticated');
    });

    assertNoTokenStorageDependency();
  });

  it('shows not authenticated when cookie session bootstrap fails', async () => {
    const { assertNoTokenStorageDependency } = createTokenStorageSpies();
    mockGetMe.mockRejectedValueOnce(new Error('unauthorized'));

    await act(async () => {
      render(
        <AuthProvider>
          <AuthStatusDisplay />
        </AuthProvider>
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('auth-status').textContent).toBe('not-authenticated');
    });

    assertNoTokenStorageDependency();
  });

  it('refreshes user on tokenRefreshed event without token storage access', async () => {
    const { assertNoTokenStorageDependency } = createTokenStorageSpies();

    await act(async () => {
      render(
        <AuthProvider>
          <AuthStatusDisplay />
        </AuthProvider>
      );
    });

    await waitFor(() => {
      expect(mockGetMe).toHaveBeenCalledTimes(1);
    });

    window.dispatchEvent(new CustomEvent('tokenRefreshed', { detail: { token: 'rotated-token' } }));

    await waitFor(() => {
      expect(mockGetMe).toHaveBeenCalledTimes(2);
    });

    assertNoTokenStorageDependency();
  });

  it('handles tokenExpired by clearing auth state and setting redirect path', async () => {
    await act(async () => {
      render(
        <AuthProvider>
          <AuthStatusDisplay />
        </AuthProvider>
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('auth-status').textContent).toBe('authenticated');
    });

    window.dispatchEvent(new CustomEvent('tokenExpired'));

    await waitFor(() => {
      expect(screen.getByTestId('auth-status').textContent).toBe('not-authenticated');
    });

    expect(sessionStorage.getItem('redirectAfterLogin')).toBe(window.location.pathname + window.location.search);
  });

  it('login finalizes session by refetching /auth/me instead of storing returned tokens', async () => {
    mockLogin.mockResolvedValueOnce({ data: undefined });
    mockGetMe
      .mockResolvedValueOnce({
        data: {
          id: 0,
          username: '',
          name: '',
          email: '',
          status: 'inactive',
          roles: [],
          permissions: [],
        },
      })
      .mockResolvedValueOnce({
        data: {
          id: 2,
          username: 'alice',
          name: 'Alice',
          email: 'alice@example.com',
          status: 'active',
          roles: ['user'],
          permissions: ['svc:view'],
        },
      });

    const Consumer = () => {
      const { login } = useAuth();
      return (
        <button
          type="button"
          onClick={() => {
            void login({ username: 'alice', password: 'secret' });
          }}
        >
          login
        </button>
      );
    };

    render(
      <AuthProvider>
        <Consumer />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(mockGetMe).toHaveBeenCalledTimes(1);
    });

    await act(async () => {
      screen.getByRole('button', { name: 'login' }).click();
    });

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({ username: 'alice', password: 'secret' });
      expect(mockGetMe).toHaveBeenCalledTimes(2);
    });

    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('refreshToken')).toBeNull();
    expect(localStorage.getItem('user')).toBeNull();
    expect(localStorage.getItem('permissions')).toBeNull();
  });
});
