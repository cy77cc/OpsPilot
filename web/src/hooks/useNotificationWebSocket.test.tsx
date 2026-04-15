import { renderHook } from '@testing-library/react';
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { useNotificationWebSocket } from './useNotificationWebSocket';

class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  static instances: MockWebSocket[] = [];

  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  readyState = MockWebSocket.CONNECTING;
  readonly url: string;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  send() {}

  close(code = 1000, reason = '') {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.({ code, reason } as CloseEvent);
  }

  static reset() {
    MockWebSocket.instances = [];
  }
}

describe('useNotificationWebSocket', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockWebSocket.reset();
    localStorage.clear();
    vi.stubGlobal('WebSocket', MockWebSocket);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('connects without token or user_id query parameters', () => {
    renderHook(() => useNotificationWebSocket({ userId: 1 }));
    vi.runAllTimers();

    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0].url).toContain('/ws/notifications');
    expect(MockWebSocket.instances[0].url).not.toContain('token=');
    expect(MockWebSocket.instances[0].url).not.toContain('user_id=');
  });

  it('connects even when userId is not provided', () => {
    renderHook(() => useNotificationWebSocket());
    vi.runAllTimers();

    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0].url).toContain('/ws/notifications');
    expect(MockWebSocket.instances[0].url).not.toContain('token=');
    expect(MockWebSocket.instances[0].url).not.toContain('user_id=');
  });

  it('does not append token query when local token exists', () => {
    localStorage.setItem('token', 'test-token');

    renderHook(() => useNotificationWebSocket({ userId: 1 }));
    vi.runAllTimers();

    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0].url).toContain('/ws/notifications');
    expect(MockWebSocket.instances[0].url).not.toContain('user_id=');
    expect(MockWebSocket.instances[0].url).not.toContain('token=');
  });

  it('does not reconnect after unmount disconnect', () => {
    const { unmount } = renderHook(() => useNotificationWebSocket({ userId: 1 }));
    vi.runAllTimers();
    expect(MockWebSocket.instances).toHaveLength(1);

    unmount();
    vi.runAllTimers();

    expect(MockWebSocket.instances).toHaveLength(1);
  });
});
