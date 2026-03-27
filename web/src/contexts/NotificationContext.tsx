import React, { createContext, useContext, useState, useEffect, useCallback, useRef } from 'react';
import type {
  UserNotification,
  UnreadCountResponse,
  WSConnectionStatus,
  WSMessage,
} from '../types/notification';
import { notificationApi } from '../api/modules/notification';
import { aiApi } from '../api/modules/ai';
import { useNotificationWebSocket } from '../hooks/useNotificationWebSocket';
import { playNotificationSound } from '../hooks/useNotificationSound';
import { notify as sendBrowserNotification } from '../utils/browserNotification';

type ApprovalActionState = {
  state: 'submitting' | 'refresh-needed';
  message?: string;
};

interface NotificationContextValue {
  // 状态
  notifications: UserNotification[];
  unreadCount: UnreadCountResponse;
  loading: boolean;
  wsStatus: WSConnectionStatus;
  approvalActionStates: Record<string, ApprovalActionState>;

  // 操作
  refresh: () => Promise<void>;
  markAsRead: (id: string) => Promise<void>;
  dismiss: (id: string) => Promise<void>;
  confirm: (id: string) => Promise<void>;
  reject: (id: string) => Promise<void>;
  markAllAsRead: () => Promise<void>;
}

const NotificationContext = createContext<NotificationContextValue | null>(null);

function getApprovalActionKey(notification: UserNotification): string {
  return notification.notification.source_id || notification.id;
}

function getApprovalErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  if (typeof error === 'string' && error.trim()) {
    return error.trim();
  }
  return '提交失败，请刷新后重试';
}

function createApprovalIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }

  return `approval-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

interface NotificationProviderProps {
  children: React.ReactNode;
  userId?: number | string;
  pollInterval?: number; // 轮询间隔（WebSocket 断开时使用）
}

export const NotificationProvider: React.FC<NotificationProviderProps> = ({
  children,
  userId,
  pollInterval = 30000,
}) => {
  const [notifications, setNotifications] = useState<UserNotification[]>([]);
  const [unreadCount, setUnreadCount] = useState<UnreadCountResponse>({
    total: 0,
    by_type: { alert: 0, task: 0, system: 0, approval: 0 },
    by_severity: { critical: 0, warning: 0, info: 0 },
  });
  const [loading, setLoading] = useState(false);
  const [wsStatus, setWsStatus] = useState<WSConnectionStatus>('disconnected');
  const [approvalActionStates, setApprovalActionStates] = useState<Record<string, ApprovalActionState>>({});

  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const approvalActionInflightRef = useRef<Record<string, string>>({});

  // 使用 ref 存储加载函数，避免依赖变化
  const loadNotificationsRef = useRef<(() => Promise<void>) | null>(null);

  loadNotificationsRef.current = async () => {
    try {
      setLoading(true);
      const [listRes, countRes] = await Promise.all([
        notificationApi.getNotifications({ pageSize: 20 }),
        notificationApi.getUnreadCount(),
      ]);
      setNotifications(listRes.data.list);
      setUnreadCount(countRes.data);
    } catch (error) {
      console.error('加载通知失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载通知数据
  const loadNotifications = useCallback(async () => {
    await loadNotificationsRef.current?.();
  }, []);

  const clearApprovalActionState = useCallback((key: string) => {
    setApprovalActionStates((prev) => {
      if (!prev[key]) {
        return prev;
      }
      const next = { ...prev };
      delete next[key];
      return next;
    });
  }, []);

  const setApprovalActionSubmitting = useCallback((key: string) => {
    setApprovalActionStates((prev) => ({
      ...prev,
      [key]: {
        state: 'submitting',
        message: '提交中',
      },
    }));
  }, []);

  const setApprovalActionFailure = useCallback((key: string, error: unknown) => {
    setApprovalActionStates((prev) => ({
      ...prev,
      [key]: {
        state: 'refresh-needed',
        message: `提交失败：${getApprovalErrorMessage(error)}`,
      },
    }));
  }, []);

  // 处理 WebSocket 消息
  const handleWSMessage = useCallback((message: WSMessage) => {
    if (message.type === 'new' && message.notification) {
      // 新通知
      const notif = message.notification;
      setNotifications((prev) => [notif, ...prev.slice(0, 19)]);
      setUnreadCount((prev) => ({
        ...prev,
        total: prev.total + 1,
      }));

      // 播放通知声音
      const severity = notif.notification?.severity;
      const soundType = severity === 'critical' ? 'error' : severity === 'warning' ? 'warning' : 'default';
      playNotificationSound(soundType as 'default' | 'warning' | 'error');

      // 发送浏览器通知（仅在页面不可见时）
      if (document.hidden || document.visibilityState === 'hidden') {
        sendBrowserNotification(
          notif.notification?.title || '新通知',
          notif.notification?.content,
          {
            tag: notif.id,
            onClick: () => {
              if (notif.notification?.action_url) {
                window.location.href = notif.notification.action_url;
              }
            },
          }
        );
      }
    } else if (message.type === 'update') {
      // 状态更新
      setNotifications((prev) =>
        prev.map((n) => {
          if (n.id.toString() === message.id) {
            return {
              ...n,
              read_at: message.read_at || n.read_at,
              dismissed_at: message.dismissed_at || n.dismissed_at,
              confirmed_at: message.confirmed_at || n.confirmed_at,
            };
          }
          return n;
        })
      );
    }
  }, []);

  // WebSocket 连接
  const { status: wsConnectionStatus } = useNotificationWebSocket({
    userId,
    onMessage: handleWSMessage,
    onConnect: () => {
      setWsStatus('connected');
    },
    onDisconnect: () => {
      setWsStatus('disconnected');
    },
    reconnectInterval: 1000,
    maxReconnectInterval: 30000,
  });

  // 刷新
  const refresh = useCallback(async () => {
    await loadNotifications();
  }, [loadNotifications]);

  // 标记已读
  const markAsRead = useCallback(async (id: string) => {
    await notificationApi.markAsRead(id);
    setNotifications((prev) =>
      prev.map((n) =>
        n.id === id ? { ...n, read_at: new Date().toISOString() } : n
      )
    );
    setUnreadCount((prev) => ({
      ...prev,
      total: Math.max(0, prev.total - 1),
    }));
  }, []);

  // 忽略通知
  const dismiss = useCallback(async (id: string) => {
    await notificationApi.dismiss(id);
    setNotifications((prev) => prev.filter((n) => n.id !== id));
    const notification = notifications.find((n) => n.id === id);
    if (notification && !notification.read_at) {
      setUnreadCount((prev) => ({
        ...prev,
        total: Math.max(0, prev.total - 1),
      }));
    }
  }, [notifications]);

  // 确认告警
  const confirm = useCallback(async (id: string) => {
    const target = notifications.find((n) => n.id === id);
    if (!target) return;
    if (target.notification.type === 'approval' && target.notification.source_id) {
      const approvalKey = getApprovalActionKey(target);
      if (approvalActionInflightRef.current[approvalKey]) {
        return;
      }
      const idempotencyKey = createApprovalIdempotencyKey();
      approvalActionInflightRef.current[approvalKey] = idempotencyKey;
      setApprovalActionSubmitting(approvalKey);
      try {
        const confirmApproval = (aiApi as unknown as {
          confirmApproval?: (approvalId: string, approved: boolean) => Promise<unknown>;
        }).confirmApproval;
        if (typeof confirmApproval === 'function') {
          await confirmApproval(target.notification.source_id, true);
        } else {
          await (aiApi.submitApproval as unknown as (
            approvalId: string,
            payload: { approved: boolean },
            options?: { idempotencyKey?: string },
          ) => Promise<unknown>)(target.notification.source_id, { approved: true }, {
            idempotencyKey,
          });
        }
      } catch (error) {
        setApprovalActionFailure(approvalKey, error);
        return;
      } finally {
        delete approvalActionInflightRef.current[approvalKey];
      }
    } else {
      await notificationApi.confirm(id);
    }
    setNotifications((prev) =>
      prev.map((n) =>
        n.id === id
          ? { ...n, read_at: new Date().toISOString(), confirmed_at: new Date().toISOString() }
          : n
      )
    );
    if (target.notification.type === 'approval' && target.notification.source_id) {
      clearApprovalActionState(getApprovalActionKey(target));
    }
    setUnreadCount((prev) => ({
      ...prev,
      total: Math.max(0, prev.total - 1),
    }));
    window.dispatchEvent(new CustomEvent('ai-approval-updated', { detail: { token: target.notification.source_id, status: 'approved' } }));
  }, [clearApprovalActionState, notifications, setApprovalActionFailure, setApprovalActionSubmitting]);

  const reject = useCallback(async (id: string) => {
    const target = notifications.find((n) => n.id === id);
    if (!target) return;
    if (target.notification.type === 'approval' && target.notification.source_id) {
      const approvalKey = getApprovalActionKey(target);
      if (approvalActionInflightRef.current[approvalKey]) {
        return;
      }
      const idempotencyKey = createApprovalIdempotencyKey();
      approvalActionInflightRef.current[approvalKey] = idempotencyKey;
      setApprovalActionSubmitting(approvalKey);
      try {
        const confirmApproval = (aiApi as unknown as {
          confirmApproval?: (approvalId: string, approved: boolean) => Promise<unknown>;
        }).confirmApproval;
        if (typeof confirmApproval === 'function') {
          await confirmApproval(target.notification.source_id, false);
        } else {
          await (aiApi.submitApproval as unknown as (
            approvalId: string,
            payload: { approved: boolean },
            options?: { idempotencyKey?: string },
          ) => Promise<unknown>)(target.notification.source_id, { approved: false }, {
            idempotencyKey,
          });
        }
      } catch (error) {
        setApprovalActionFailure(approvalKey, error);
        return;
      } finally {
        delete approvalActionInflightRef.current[approvalKey];
      }
      setNotifications((prev) => prev.filter((n) => n.id !== id));
      clearApprovalActionState(approvalKey);
      setUnreadCount((prev) => ({ ...prev, total: Math.max(0, prev.total - 1) }));
      window.dispatchEvent(new CustomEvent('ai-approval-updated', { detail: { token: target.notification.source_id, status: 'rejected' } }));
      return;
    }
    await dismiss(id);
  }, [clearApprovalActionState, dismiss, notifications, setApprovalActionFailure, setApprovalActionSubmitting]);

  // 全部已读
  const markAllAsRead = useCallback(async () => {
    await notificationApi.markAllAsRead();
    setNotifications((prev) =>
      prev.map((n) => ({ ...n, read_at: n.read_at || new Date().toISOString() }))
    );
    setUnreadCount((prev) => ({ ...prev, total: 0 }));
  }, []);

  // 初始化 - 只加载一次
  useEffect(() => {
    loadNotifications();
    // 注意：轮询由 WebSocket 状态变化时自动处理，不在这里启动
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // 当 WebSocket 状态变化时处理轮询
  useEffect(() => {
    if (wsStatus === 'connected') {
      // WebSocket 连接成功，停止轮询
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
        console.log('Notification: WebSocket 已连接，停止轮询');
      }
    } else {
      // WebSocket 断开，启动轮询作为降级
      if (!pollingRef.current) {
        pollingRef.current = setInterval(loadNotifications, pollInterval);
        console.log(`Notification: WebSocket 断开，启动轮询 (间隔 ${pollInterval}ms)`);
      }
    }

    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
    };
  }, [wsStatus, pollInterval]); // 移除 loadNotifications 依赖

  // 更新 wsStatus 状态
  useEffect(() => {
    setWsStatus(wsConnectionStatus);
  }, [wsConnectionStatus]);

  useEffect(() => {
    const handler = () => {
      void loadNotifications();
    };
    window.addEventListener('ai-approval-updated', handler);
    return () => {
      window.removeEventListener('ai-approval-updated', handler);
    };
  }, [loadNotifications]);

  const value: NotificationContextValue = {
    notifications,
    unreadCount,
    loading,
    wsStatus,
    approvalActionStates,
    refresh,
    markAsRead,
    dismiss,
    confirm,
    reject,
    markAllAsRead,
  };

  return (
    <NotificationContext.Provider value={value}>
      {children}
    </NotificationContext.Provider>
  );
};

export const useNotificationContext = (): NotificationContextValue => {
  const context = useContext(NotificationContext);
  if (!context) {
    throw new Error('useNotificationContext must be used within NotificationProvider');
  }
  return context;
};

export default NotificationContext;
