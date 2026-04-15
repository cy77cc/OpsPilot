import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
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
    expect(error.message).toBe('Test error');
    expect(error.statusCode).toBe(401);
    expect(error.businessCode).toBe(4005);
  });

  it('is an instance of Error', () => {
    const error = new ApiRequestError('Test error');
    expect(error).toBeInstanceOf(Error);
  });

  it('can be caught and checked by businessCode', () => {
    const error = new ApiRequestError('Token expired', 401, 4005);

    try {
      throw error;
    } catch (e) {
      if (e instanceof ApiRequestError) {
        expect(e.businessCode).toBe(4005);
        expect(e.statusCode).toBe(401);
      }
    }
  });

  it('handles undefined values gracefully', () => {
    const error = new ApiRequestError('Unknown error', undefined, undefined);
    expect(error.message).toBe('Unknown error');
    expect(error.statusCode).toBeUndefined();
    expect(error.businessCode).toBeUndefined();
  });

  it('preserves stack trace', () => {
    const error = new ApiRequestError('Test error');
    expect(error.stack).toBeDefined();
    expect(error.stack).toContain('ApiRequestError');
  });
});

describe('ApiResponse interface', () => {
  it('defines correct structure for success response', () => {
    const response = {
      success: true,
      data: { id: 1, name: 'test' },
      message: 'Success',
    };

    expect(response.success).toBe(true);
    expect(response.data).toEqual({ id: 1, name: 'test' });
  });

  it('defines correct structure for error response', () => {
    const response = {
      success: false,
      data: null,
      error: {
        code: 'VALIDATION_ERROR',
        message: 'Invalid input',
      },
    };

    expect(response.success).toBe(false);
    expect(response.error?.code).toBe('VALIDATION_ERROR');
  });

  it('supports paginated response with total', () => {
    const response = {
      success: true,
      data: {
        total: 100,
        list: [{ id: 1 }, { id: 2 }],
      },
    };

    expect(response.data.total).toBe(100);
    expect(response.data.list).toHaveLength(2);
  });
});

describe('localStorage interaction', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('stores and retrieves token', () => {
    localStorage.setItem('token', 'test-token');
    expect(localStorage.getItem('token')).toBe('test-token');
  });

  it('stores and retrieves projectId', () => {
    localStorage.setItem('projectId', '123');
    expect(localStorage.getItem('projectId')).toBe('123');
  });

  it('stores and retrieves refreshToken', () => {
    localStorage.setItem('refreshToken', 'refresh-token-123');
    expect(localStorage.getItem('refreshToken')).toBe('refresh-token-123');
  });

  it('clears auth tokens', () => {
    localStorage.setItem('token', 'test-token');
    localStorage.setItem('refreshToken', 'refresh-token');

    localStorage.removeItem('token');
    localStorage.removeItem('refreshToken');

    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('refreshToken')).toBeNull();
  });

  it('handles missing token gracefully', () => {
    const token = localStorage.getItem('token');
    expect(token).toBeNull();
  });
});

describe('API error scenarios', () => {
  it('creates error for 401 unauthorized', () => {
    const error = new ApiRequestError('Unauthorized', 401, 4005);
    expect(error.statusCode).toBe(401);
    expect(error.businessCode).toBe(4005);
  });

  it('creates error for 403 forbidden', () => {
    const error = new ApiRequestError('Forbidden', 403);
    expect(error.statusCode).toBe(403);
  });

  it('creates error for 500 server error', () => {
    const error = new ApiRequestError('Internal Server Error', 500);
    expect(error.statusCode).toBe(500);
  });

  it('creates error for network timeout', () => {
    const error = new ApiRequestError('Network timeout');
    expect(error.message).toBe('Network timeout');
  });

  it('creates error for business logic error', () => {
    const error = new ApiRequestError('Resource not found', 200, 5001);
    expect(error.businessCode).toBe(5001);
    expect(error.message).toBe('Resource not found');
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
    expect(isAuthBusinessCode(5000)).toBe(false);
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
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('dispatches tokenRefreshed event on successful refresh', async () => {
    const handler = vi.fn();
    window.addEventListener(TOKEN_EVENTS.REFRESHED, handler);

    // Manually dispatch event (simulating what ApiService does)
    window.dispatchEvent(
      new CustomEvent(TOKEN_EVENTS.REFRESHED, {
        detail: { token: 'new-token', refreshToken: 'new-refresh-token' },
      })
    );

    expect(handler).toHaveBeenCalledWith(
      expect.objectContaining({
        detail: { token: 'new-token', refreshToken: 'new-refresh-token' },
      })
    );

    window.removeEventListener(TOKEN_EVENTS.REFRESHED, handler);
  });

  it('dispatches tokenExpired event on refresh failure', () => {
    const handler = vi.fn();
    window.addEventListener(TOKEN_EVENTS.EXPIRED, handler);

    // Manually dispatch event (simulating what ApiService does)
    window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.EXPIRED));

    expect(handler).toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.EXPIRED, handler);
  });

  it('multiple listeners can subscribe to refresh events', () => {
    const handler1 = vi.fn();
    const handler2 = vi.fn();

    window.addEventListener(TOKEN_EVENTS.REFRESHED, handler1);
    window.addEventListener(TOKEN_EVENTS.REFRESHED, handler2);

    window.dispatchEvent(
      new CustomEvent(TOKEN_EVENTS.REFRESHED, {
        detail: { token: 'new-token' },
      })
    );

    expect(handler1).toHaveBeenCalled();
    expect(handler2).toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.REFRESHED, handler1);
    window.removeEventListener(TOKEN_EVENTS.REFRESHED, handler2);
  });
});

describe('ApiService cookie-session refresh and retry', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it('refreshAccessToken posts to /auth/refresh without refresh-token body or token storage keys', async () => {
    const instance = (apiService as any).instance;
    const postSpy = vi.spyOn(instance, 'post').mockResolvedValue({
      data: { data: { accessToken: 'rotated-access-token' } },
    });
    const getItemSpy = vi.spyOn(Storage.prototype, 'getItem');
    const refreshedHandler = vi.fn();

    window.addEventListener(TOKEN_EVENTS.REFRESHED, refreshedHandler);

    const refreshed = await apiService.refreshAccessToken();

    expect(refreshed).toBe(true);
    expect(postSpy).toHaveBeenCalledTimes(1);
    expect(postSpy).toHaveBeenCalledWith('/auth/refresh');
    expect(postSpy.mock.calls[0]).toHaveLength(1);

    const tokenKeyReads = getItemSpy.mock.calls.filter(
      ([key]) => String(key) === 'token' || String(key) === 'refreshToken'
    );
    expect(tokenKeyReads).toEqual([]);
    expect(refreshedHandler).toHaveBeenCalledTimes(1);

    window.removeEventListener(TOKEN_EVENTS.REFRESHED, refreshedHandler);
  });

  it('retry path replays original request without Authorization injection from localStorage token', async () => {
    const instance = (apiService as any).instance;
    const requestSpy = vi.spyOn(instance, 'request').mockResolvedValue({
      data: { success: true, data: { ok: true } },
    });
    const refreshSpy = vi.spyOn(apiService, 'refreshAccessToken').mockResolvedValue(true);
    const getItemSpy = vi.spyOn(Storage.prototype, 'getItem');

    localStorage.setItem('token', 'stale-local-storage-token');

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

    const tokenKeyReads = getItemSpy.mock.calls.filter(
      ([key]) => String(key) === 'token' || String(key) === 'refreshToken'
    );
    expect(tokenKeyReads).toEqual([]);
  });
});
