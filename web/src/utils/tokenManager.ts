export const TOKEN_EVENTS = {
  REFRESHED: 'tokenRefreshed',
  EXPIRED: 'tokenExpired',
  NEEDS_REFRESH: 'tokenNeedsRefresh',
} as const;

export function dispatchTokenRefreshed(): void {
  window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.REFRESHED));
}

export function dispatchTokenExpired(): void {
  window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.EXPIRED));
}

export function dispatchTokenNeedsRefresh(source: 'response' | 'manual' = 'response'): void {
  window.dispatchEvent(
    new CustomEvent(TOKEN_EVENTS.NEEDS_REFRESH, {
      detail: { source },
    })
  );
}

export function startTokenExpiryCheck(): void {}

export function stopTokenExpiryCheck(): void {}
