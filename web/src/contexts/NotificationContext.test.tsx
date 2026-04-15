import React from 'react';
import { act, render, waitFor } from '@testing-library/react';
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { NotificationProvider } from './NotificationContext';
import type { WSMessage } from '../types/notification';

const notifyMock = vi.hoisted(() => vi.fn());
const safeNavigateMock = vi.hoisted(() => vi.fn());
let latestWsOptions: { onMessage?: (message: WSMessage) => void } = {};

vi.mock('../utils/browserNotification', () => ({
  notify: notifyMock,
}));

vi.mock('../utils/safeNavigate', () => ({
  safeNavigate: safeNavigateMock,
}));

vi.mock('../hooks/useNotificationSound', () => ({
  playNotificationSound: vi.fn(),
}));

vi.mock('../hooks/useNotificationWebSocket', () => ({
  useNotificationWebSocket: (options: { onMessage?: (message: WSMessage) => void }) => {
    latestWsOptions = options;
    return { status: 'connected' };
  },
}));

vi.mock('../api/modules/notification', () => ({
  notificationApi: {
    getNotifications: vi.fn().mockResolvedValue({
      data: { list: [], total: 0 },
    }),
    getUnreadCount: vi.fn().mockResolvedValue({
      data: {
        total: 0,
        by_type: { alert: 0, task: 0, system: 0, approval: 0 },
        by_severity: { critical: 0, warning: 0, info: 0 },
      },
    }),
    markAsRead: vi.fn().mockResolvedValue({ success: true }),
    dismiss: vi.fn().mockResolvedValue({ success: true }),
    confirm: vi.fn().mockResolvedValue({ success: true }),
    markAllAsRead: vi.fn().mockResolvedValue({ success: true }),
  },
}));

vi.mock('../api/modules/ai', () => ({
  aiApi: {
    submitApproval: vi.fn(),
    confirmApproval: vi.fn(),
  },
}));

describe('NotificationProvider browser notification click', () => {
  const originalHidden = Object.getOwnPropertyDescriptor(document, 'hidden');
  const originalVisibilityState = Object.getOwnPropertyDescriptor(document, 'visibilityState');

  beforeEach(() => {
    vi.clearAllMocks();
    latestWsOptions = {};
    localStorage.setItem('token', 'token');
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => true });
    Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' });
  });

  afterEach(() => {
    localStorage.clear();
    if (originalHidden) {
      Object.defineProperty(document, 'hidden', originalHidden);
    }
    if (originalVisibilityState) {
      Object.defineProperty(document, 'visibilityState', originalVisibilityState);
    }
  });

  it('uses safeNavigate from browser notification click callback', async () => {
    render(
      <NotificationProvider userId="1">
        <div>child</div>
      </NotificationProvider>
    );

    const message: WSMessage = {
      type: 'new',
      notification: {
        id: '1',
        user_id: '1',
        notification_id: 'n-1',
        notification: {
          id: 'n-1',
          type: 'alert',
          title: '告警',
          content: '测试内容',
          severity: 'critical',
          source: 'source',
          source_id: 's-1',
          action_url: '/monitor?alert_id=s-1',
          action_type: 'confirm',
          created_at: new Date().toISOString(),
        },
      },
    };

    await act(async () => {
      latestWsOptions.onMessage?.(message);
    });

    await waitFor(() => {
      expect(notifyMock).toHaveBeenCalled();
    });

    const options = notifyMock.mock.calls[0][2] as { onClick?: () => void };
    expect(typeof options.onClick).toBe('function');

    options.onClick?.();
    expect(safeNavigateMock).toHaveBeenCalledWith('/monitor?alert_id=s-1');
  });
});
