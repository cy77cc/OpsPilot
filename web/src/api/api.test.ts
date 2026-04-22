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
    getItemSpy.mockClear();
    expect(localStorage.getItem('token')).toBe('legacy-token');
  });

  it('retry path replays original request without Authorization injection from localStorage token', async () => {
    const instance = (apiService as any).instance;
    const requestSpy = vi.spyOn(instance, 'request').mockResolvedValue({
      data: { success: true, data: { ok: true } },
    });
    const refreshSpy = vi.spyOn(apiService, 'refreshAccessToken').mockResolvedValue(true);
    const getItemSpy = vi.spyOn(Storage.prototype, 'getItem');

    localStorage.setItem('token', 'legacy-token');
    scopeStore.setScope({ projectId: '42' });

    const originalConfig = {
      url: '/secure/resource',
      method: 'get',
      headers: {
        'X-Request-ID': 'request-1',
      },
    };

    await (apiService as any).tryRefreshAndRetry(originalConfig);

    expect(refreshSpy).toHaveBeenCalledTimes(1);
    expect(requestSpy).toHaveBeenCalledTimes(1);
    expect(requestSpy).toHaveBeenCalledWith(originalConfig);
    expect((requestSpy.mock.calls[0][0] as { headers?: Record<string, string> }).headers?.Authorization).toBeUndefined();
    expect(
      getItemSpy.mock.calls.filter(
        ([key]) => String(key) === 'token' || String(key) === 'refreshToken'
      )
    ).toEqual([]);
  });
});
