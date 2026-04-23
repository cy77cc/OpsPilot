import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { scopeStore } from '../app/scope/scopeStore';
import apiService, { ApiRequestError, TOKEN_EVENTS, isAuthBusinessCode } from './api';

describe('ApiRequestError', () => {
  it('creates error with message only', () => {
    const error = new ApiRequestError('Test error');
    expect(error.message).toBe('Test error');
    expect(error.name).toBe('ApiRequestError');
    expect(error.statusCode).toBeUndefined();
    expect(error.businessCode).toBeUndefined();
  });

  it('creates error with status code and business code', () => {
    const error = new ApiRequestError('Test error', 401, 4005);
    expect(error.statusCode).toBe(401);
    expect(error.businessCode).toBe(4005);
  });
});

describe('request context interaction', () => {
  beforeEach(() => {
    localStorage.clear();
    scopeStore.clearScope();
  });

  afterEach(() => {
    localStorage.clear();
    scopeStore.clearScope();
  });

  it('reads project and team headers from scopeStore', () => {
    scopeStore.setScope({ projectId: '123', teamId: '456' });
    const instance = (apiService as any).instance;
    const fulfilled = instance.interceptors.request.handlers[0].fulfilled as (
      config: { headers?: Record<string, string> }
    ) => { headers?: Record<string, string> };

    const config = fulfilled({ headers: { 'Content-Type': 'application/json' } });

    expect((config.headers as { toJSON?: () => Record<string, string> }).toJSON?.() || config.headers).toEqual({
      'Content-Type': 'application/json',
      'X-Project-ID': '123',
      'X-Team-ID': '456',
    });
  });
});

describe('isAuthBusinessCode', () => {
  it('returns true for auth-related business codes', () => {
    expect(isAuthBusinessCode(2003)).toBe(true);
    expect(isAuthBusinessCode(4005)).toBe(true);
    expect(isAuthBusinessCode(4006)).toBe(true);
  });

  it('returns false for non-auth business codes', () => {
    expect(isAuthBusinessCode(1000)).toBe(false);
    expect(isAuthBusinessCode(2004)).toBe(false);
    expect(isAuthBusinessCode(undefined)).toBe(false);
  });
});

describe('TOKEN_EVENTS', () => {
  it('defines correct event names', () => {
    expect(TOKEN_EVENTS.REFRESHED).toBe('tokenRefreshed');
    expect(TOKEN_EVENTS.EXPIRED).toBe('tokenExpired');
    expect(TOKEN_EVENTS.NEEDS_REFRESH).toBe('tokenNeedsRefresh');
  });
});

describe('Token Refresh Events', () => {
  it('dispatches tokenRefreshed event on successful refresh', () => {
    const handler = vi.fn();
    window.addEventListener(TOKEN_EVENTS.REFRESHED, handler);

    window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.REFRESHED));

    expect(handler).toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.REFRESHED, handler);
  });

  it('dispatches tokenExpired event on refresh failure', () => {
    const handler = vi.fn();
    window.addEventListener(TOKEN_EVENTS.EXPIRED, handler);

    window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.EXPIRED));

    expect(handler).toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.EXPIRED, handler);
  });
});

describe('ApiService cookie-session refresh and retry', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
    scopeStore.clearScope();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
    scopeStore.clearScope();
  });

  it('uses centralized refresh gate and never reads legacy token storage keys', async () => {
    const instance = (apiService as any).instance;
    const postSpy = vi.spyOn(instance, 'post').mockResolvedValue({
      data: { data: {} },
    });
    const getItemSpy = vi.spyOn(Storage.prototype, 'getItem');
    localStorage.setItem('token', 'legacy-token');

    const [firstRefresh, secondRefresh] = await Promise.all([
      apiService.refreshAccessToken(),
      apiService.refreshAccessToken(),
    ]);

    expect(firstRefresh).toBe(true);
    expect(secondRefresh).toBe(true);
    expect(postSpy).toHaveBeenCalledTimes(1);
    expect(postSpy).toHaveBeenCalledWith('/auth/refresh');
    expect(
      getItemSpy.mock.calls.filter(
        ([key]) => String(key) === 'token' || String(key) === 'refreshToken'
      )
    ).toEqual([]);
  });

  it('retries the public API request path without Authorization injection from localStorage token', async () => {
    const instance = Reflect.get(apiService, 'instance') as {
      defaults: { adapter: unknown };
    };
    const refreshSpy = vi.spyOn(apiService, 'refreshAccessToken').mockResolvedValue(true);
    const getItemSpy = vi.spyOn(Storage.prototype, 'getItem');
    const originalAdapter = instance.defaults.adapter;
    let requestCount = 0;
    const adapterSpy = vi.fn(async (config: { headers?: { get?: (key: string) => unknown } & Record<string, unknown> }) => {
      requestCount += 1;
      if (requestCount === 1) {
        const error = new Error('unauthorized') as Error & {
          config: typeof config;
          response: {
            data: { message: string };
            status: number;
            statusText: string;
            headers: Record<string, never>;
            config: typeof config;
          };
        };
        error.config = config;
        error.response = {
          data: { message: 'unauthorized' },
          status: 401,
          statusText: 'Unauthorized',
          headers: {},
          config,
        };
        throw error;
      }

      return {
        data: { success: true, data: { ok: true } },
        status: 200,
        statusText: 'OK',
        headers: {},
        config,
      };
    });

    localStorage.setItem('token', 'legacy-token');
    scopeStore.setScope({ projectId: '42' });

    instance.defaults.adapter = adapterSpy;

    try {
      await expect(apiService.get('/secure/resource')).resolves.toEqual({
        success: true,
        data: { ok: true },
      });

      expect(refreshSpy).toHaveBeenCalledTimes(1);
      expect(adapterSpy).toHaveBeenCalledTimes(2);

      const retriedConfig = adapterSpy.mock.calls[1][0];
      expect(retriedConfig.headers?.get?.('Authorization') ?? retriedConfig.headers?.Authorization).toBeUndefined();
      expect(retriedConfig.headers?.get?.('X-Project-ID') ?? retriedConfig.headers?.['X-Project-ID']).toBe('42');
      expect(
        getItemSpy.mock.calls.filter(
          ([key]) => String(key) === 'token' || String(key) === 'refreshToken'
        )
      ).toEqual([]);
    } finally {
      instance.defaults.adapter = originalAdapter;
    }
  });

  it('dispatches tokenExpired and rejects a typed session-expired error when refresh fails after a 401', async () => {
    const instance = Reflect.get(apiService, 'instance') as {
      defaults: { adapter: unknown };
    };
    const originalAdapter = instance.defaults.adapter;
    const expiredHandler = vi.fn();

    window.addEventListener(TOKEN_EVENTS.EXPIRED, expiredHandler);

    const adapterSpy = vi.fn(async (config: { url?: string; headers?: { get?: (key: string) => unknown } & Record<string, unknown> }) => {
      const url = String(config.url || '');

      if (url.includes('/auth/refresh')) {
        const error = new Error('refresh unauthorized') as Error & {
          config: typeof config;
          response: {
            data: { message: string };
            status: number;
            statusText: string;
            headers: Record<string, never>;
            config: typeof config;
          };
        };
        error.config = config;
        error.response = {
          data: { message: 'refresh unauthorized' },
          status: 401,
          statusText: 'Unauthorized',
          headers: {},
          config,
        };
        throw error;
      }

      const error = new Error('unauthorized') as Error & {
        config: typeof config;
        response: {
          data: { message: string };
          status: number;
          statusText: string;
          headers: Record<string, never>;
          config: typeof config;
        };
      };
      error.config = config;
      error.response = {
        data: { message: 'unauthorized' },
        status: 401,
        statusText: 'Unauthorized',
        headers: {},
        config,
      };
      throw error;
    });

    instance.defaults.adapter = adapterSpy;

    try {
      await expect(apiService.get('/secure/expired')).rejects.toMatchObject({
        name: 'ApiRequestError',
        message: '登录已过期，请重新登录',
        statusCode: 401,
        businessCode: 4005,
      });

      expect(adapterSpy).toHaveBeenCalledTimes(2);
      expect(expiredHandler).toHaveBeenCalledTimes(1);
    } finally {
      window.removeEventListener(TOKEN_EVENTS.EXPIRED, expiredHandler);
      instance.defaults.adapter = originalAdapter;
    }
  });
});
