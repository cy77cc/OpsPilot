import { describe, expect, it, vi } from 'vitest';
import {
  TOKEN_EVENTS,
  dispatchTokenExpired,
  dispatchTokenNeedsRefresh,
  dispatchTokenRefreshed,
  startTokenExpiryCheck,
  stopTokenExpiryCheck,
} from './tokenManager';

describe('tokenManager', () => {
  it('defines the expected event names', () => {
    expect(TOKEN_EVENTS.REFRESHED).toBe('tokenRefreshed');
    expect(TOKEN_EVENTS.EXPIRED).toBe('tokenExpired');
    expect(TOKEN_EVENTS.NEEDS_REFRESH).toBe('tokenNeedsRefresh');
  });

  it('dispatches tokenRefreshed without token payload or storage access', () => {
    const handler = vi.fn();
    const getItemSpy = vi.spyOn(Storage.prototype, 'getItem');

    window.addEventListener(TOKEN_EVENTS.REFRESHED, handler);
    dispatchTokenRefreshed();

    expect(handler).toHaveBeenCalled();
    expect(getItemSpy).not.toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.REFRESHED, handler);
  });

  it('dispatches tokenExpired', () => {
    const handler = vi.fn();

    window.addEventListener(TOKEN_EVENTS.EXPIRED, handler);
    dispatchTokenExpired();

    expect(handler).toHaveBeenCalled();

    window.removeEventListener(TOKEN_EVENTS.EXPIRED, handler);
  });

  it('dispatches tokenNeedsRefresh with explicit source metadata', () => {
    const handler = vi.fn();

    window.addEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
    dispatchTokenNeedsRefresh('manual');

    expect(handler).toHaveBeenCalledWith(
      expect.objectContaining({
        detail: { source: 'manual' },
      })
    );

    window.removeEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
  });

  it('keeps start/stop expiry checks as safe no-ops under cookie-session auth', () => {
    expect(() => startTokenExpiryCheck()).not.toThrow();
    expect(() => stopTokenExpiryCheck()).not.toThrow();
  });
});
