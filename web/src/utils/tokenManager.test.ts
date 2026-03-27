import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  parseJwtExpiresAt,
  isTokenExpiringSoon,
  getTokenRemainingTime,
  TOKEN_EVENTS,
  startTokenExpiryCheck,
  stopTokenExpiryCheck,
  dispatchTokenRefreshed,
  dispatchTokenExpired,
  tokenExpiryChecker,
} from './tokenManager';

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

// Helper to create a valid JWT with specific exp
function createMockJwt(expSeconds: number): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(JSON.stringify({ exp: expSeconds, uid: 1 }));
  const signature = btoa('mock-signature');
  // Convert to Base64Url
  return [header, payload, signature]
    .map((s) => s.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, ''))
    .join('.');
}

describe('parseJwtExpiresAt', () => {
  it('parses valid JWT and returns expiration time', () => {
    const expSeconds = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now
    const token = createMockJwt(expSeconds);

    const result = parseJwtExpiresAt(token);

    expect(result).toBe(expSeconds * 1000);
  });

  it('returns null for invalid token format', () => {
    expect(parseJwtExpiresAt('invalid')).toBeNull();
    expect(parseJwtExpiresAt('a.b')).toBeNull();
  });

  it('returns null for token without exp claim', () => {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({ uid: 1 }));
    const signature = btoa('mock-signature');
    const token = [header, payload, signature]
      .map((s) => s.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, ''))
      .join('.');

    expect(parseJwtExpiresAt(token)).toBeNull();
  });
});

describe('isTokenExpiringSoon', () => {
  it('returns true for token expiring within threshold', () => {
    const expSeconds = Math.floor(Date.now() / 1000) + 60; // 1 minute from now
    const token = createMockJwt(expSeconds);

    expect(isTokenExpiringSoon(token)).toBe(true);
  });

  it('returns false for token not expiring soon', () => {
    const expSeconds = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now
    const token = createMockJwt(expSeconds);

    expect(isTokenExpiringSoon(token)).toBe(false);
  });

  it('returns true for already expired token', () => {
    const expSeconds = Math.floor(Date.now() / 1000) - 60; // 1 minute ago
    const token = createMockJwt(expSeconds);

    expect(isTokenExpiringSoon(token)).toBe(true);
  });

  it('returns true for unparseable token', () => {
    expect(isTokenExpiringSoon('invalid-token')).toBe(true);
  });

  it('respects custom threshold', () => {
    const expSeconds = Math.floor(Date.now() / 1000) + 600; // 10 minutes from now
    const token = createMockJwt(expSeconds);

    // Default threshold (5 min) - not expiring soon
    expect(isTokenExpiringSoon(token)).toBe(false);

    // Custom threshold (15 min) - expiring soon
    expect(isTokenExpiringSoon(token, 15 * 60 * 1000)).toBe(true);
  });
});

describe('getTokenRemainingTime', () => {
  it('returns remaining time in milliseconds', () => {
    const expSeconds = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now
    const token = createMockJwt(expSeconds);

    const remaining = getTokenRemainingTime(token);

    // Should be approximately 1 hour (with some tolerance for test execution time)
    expect(remaining).toBeGreaterThan(3599 * 1000);
    expect(remaining).toBeLessThanOrEqual(3600 * 1000);
  });

  it('returns 0 for expired token', () => {
    const expSeconds = Math.floor(Date.now() / 1000) - 60; // 1 minute ago
    const token = createMockJwt(expSeconds);

    expect(getTokenRemainingTime(token)).toBe(0);
  });

  it('returns 0 for unparseable token', () => {
    expect(getTokenRemainingTime('invalid')).toBe(0);
  });
});

describe('TokenExpiryChecker', () => {
  beforeEach(() => {
    localStorageMock.clear();
    vi.useFakeTimers();
  });

  afterEach(() => {
    stopTokenExpiryCheck();
    vi.useRealTimers();
  });

  it('dispatches NEEDS_REFRESH event when token is expiring', () => {
    const expSeconds = Math.floor(Date.now() / 1000) + 60; // 1 minute from now
    const token = createMockJwt(expSeconds);
    const refreshToken = 'mock-refresh-token';

    localStorageMock.setItem('token', token);
    localStorageMock.setItem('refreshToken', refreshToken);

    const handler = vi.fn();
    window.addEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);

    startTokenExpiryCheck();

    // Should check immediately
    expect(handler).toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
  });

  it('does not dispatch event when token is not expiring', () => {
    const expSeconds = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now
    const token = createMockJwt(expSeconds);
    const refreshToken = 'mock-refresh-token';

    localStorageMock.setItem('token', token);
    localStorageMock.setItem('refreshToken', refreshToken);

    const handler = vi.fn();
    window.addEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);

    startTokenExpiryCheck();

    expect(handler).not.toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
  });

  it('stops checking when stop is called', () => {
    const expSeconds = Math.floor(Date.now() / 1000) + 60;
    const token = createMockJwt(expSeconds);
    const refreshToken = 'mock-refresh-token';

    localStorageMock.setItem('token', token);
    localStorageMock.setItem('refreshToken', refreshToken);

    const handler = vi.fn();
    window.addEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);

    startTokenExpiryCheck();
    stopTokenExpiryCheck();

    // Clear previous calls
    handler.mockClear();

    // Advance time - should not trigger because stopped
    vi.advanceTimersByTime(60 * 1000);

    expect(handler).not.toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
  });

  it('checks periodically', () => {
    const expSeconds = Math.floor(Date.now() / 1000) + 60;
    const token = createMockJwt(expSeconds);
    const refreshToken = 'mock-refresh-token';

    localStorageMock.setItem('token', token);
    localStorageMock.setItem('refreshToken', refreshToken);

    const handler = vi.fn();
    window.addEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);

    startTokenExpiryCheck();

    // Clear initial check
    handler.mockClear();
    tokenExpiryChecker.reset();

    // Advance 1 minute
    vi.advanceTimersByTime(60 * 1000);

    // Should have checked again
    expect(handler).toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
  });
});

describe('Event dispatchers', () => {
  it('dispatches tokenRefreshed event with data', () => {
    const handler = vi.fn();
    window.addEventListener(TOKEN_EVENTS.REFRESHED, handler);

    dispatchTokenRefreshed({
      token: 'new-token',
      refreshToken: 'new-refresh-token',
    });

    expect(handler).toHaveBeenCalledWith(
      expect.objectContaining({
        detail: {
          token: 'new-token',
          refreshToken: 'new-refresh-token',
        },
      })
    );

    window.removeEventListener(TOKEN_EVENTS.REFRESHED, handler);
  });

  it('dispatches tokenExpired event', () => {
    const handler = vi.fn();
    window.addEventListener(TOKEN_EVENTS.EXPIRED, handler);

    dispatchTokenExpired();

    expect(handler).toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.EXPIRED, handler);
  });
});
