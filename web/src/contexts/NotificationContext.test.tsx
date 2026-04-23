import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { NotificationProvider } from './NotificationContext';
import { useNotificationDataContext } from './notification/NotificationDataProvider';
import { useApprovalStateContext } from './notification/ApprovalStateProvider';
import type { WSMessage } from '../types/notification';

const notifyMock = vi.hoisted(() => vi.fn());
const safeNavigateMock = vi.hoisted(() => vi.fn());
const getNotificationsMock = vi.hoisted(() => vi.fn());
const getUnreadCountMock = vi.hoisted(() => vi.fn());
const markAsReadMock = vi.hoisted(() => vi.fn());
const dismissMock = vi.hoisted(() => vi.fn());
const confirmMock = vi.hoisted(() => vi.fn());
const markAllAsReadMock = vi.hoisted(() => vi.fn());
const submitApprovalMock = vi.hoisted(() => vi.fn());
const confirmApprovalMock = vi.hoisted(() => vi.fn());
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
    getNotifications: getNotificationsMock,
    getUnreadCount: getUnreadCountMock,
    markAsRead: markAsReadMock,
    dismiss: dismissMock,
    confirm: confirmMock,
    markAllAsRead: markAllAsReadMock,
  },
}));

vi.mock('../api/modules/ai', () => ({
  aiApi: {
    submitApproval: submitApprovalMock,
    confirmApproval: confirmApprovalMock,
  },
}));

function createUnreadCount(total = 0) {
  return {
    total,
    by_type: { alert: 0, task: 0, system: 0, approval: total },
    by_severity: { critical: 0, warning: 0, info: total },
  };
}

function createApprovalNotification() {
  return {
    id: 'approval-user-1',
    user_id: '1',
    notification_id: 'approval-1',
    notification: {
      id: 'approval-1',
      type: 'approval' as const,
      title: '审批请求',
      content: '需要确认',
      severity: 'info' as const,
      source: 'copilot',
      source_id: 'approval-token-1',
      action_type: 'approve' as const,
      created_at: new Date().toISOString(),
    },
  };
}

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function DataOnlyProbe({ onRender }: { onRender: () => void }) {
  const { unreadCount } = useNotificationDataContext();
  onRender();
  return <div data-testid="data-total">{unreadCount.total}</div>;
}

function ApprovalStateProbe({ onRender }: { onRender: () => void }) {
  const { approvalActionStates } = useApprovalStateContext();
  onRender();
  return <div data-testid="approval-total">{Object.keys(approvalActionStates).length}</div>;
}

function ApprovalTrigger() {
  const { confirm } = useApprovalStateContext();

  return (
    <button type="button" onClick={() => void confirm('approval-user-1')}>
      trigger approval
    </button>
  );
}

describe('NotificationProvider browser notification click', () => {
  const originalHidden = Object.getOwnPropertyDescriptor(document, 'hidden');
  const originalVisibilityState = Object.getOwnPropertyDescriptor(document, 'visibilityState');

  beforeEach(() => {
    vi.clearAllMocks();
    latestWsOptions = {};
    getNotificationsMock.mockResolvedValue({
      data: { list: [], total: 0 },
    });
    getUnreadCountMock.mockResolvedValue({
      data: createUnreadCount(0),
    });
    markAsReadMock.mockResolvedValue({ success: true });
    dismissMock.mockResolvedValue({ success: true });
    confirmMock.mockResolvedValue({ success: true });
    markAllAsReadMock.mockResolvedValue({ success: true });
    submitApprovalMock.mockResolvedValue({ success: true });
    confirmApprovalMock.mockResolvedValue({ success: true });
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

  it('keeps approval updates from re-rendering data-only consumers', async () => {
    const approvalNotification = createApprovalNotification();
    const deferredConfirm = createDeferred<{ success: boolean }>();
    const dataRenders: number[] = [];
    const approvalRenders: number[] = [];

    getNotificationsMock.mockResolvedValue({
      data: { list: [approvalNotification], total: 1 },
    });
    getUnreadCountMock.mockResolvedValue({
      data: createUnreadCount(1),
    });
    confirmApprovalMock.mockReturnValue(deferredConfirm.promise);

    render(
      <NotificationProvider userId="1">
        <DataOnlyProbe onRender={() => dataRenders.push(Date.now())} />
        <ApprovalStateProbe onRender={() => approvalRenders.push(Date.now())} />
        <ApprovalTrigger />
      </NotificationProvider>
    );

    await waitFor(() => {
      expect(getNotificationsMock).toHaveBeenCalled();
      expect(getUnreadCountMock).toHaveBeenCalled();
      expect(dataRenders.length).toBeGreaterThan(1);
    });

    const dataRenderBaseline = dataRenders.length;
    const approvalRenderBaseline = approvalRenders.length;

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'trigger approval' }));
      await Promise.resolve();
    });

    await waitFor(() => {
      expect(confirmApprovalMock).toHaveBeenCalledWith('approval-token-1', true);
      expect(approvalRenders.length).toBeGreaterThan(approvalRenderBaseline);
    });

    expect(dataRenders).toHaveLength(dataRenderBaseline);

    deferredConfirm.resolve({ success: true });
  });
});
