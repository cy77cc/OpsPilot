import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import type { UnreadCountResponse, UserNotification, WSMessage } from '../../types/notification';
import { notificationApi } from '../../api/modules/notification';
import { ApiRequestError, isAuthBusinessCode } from '../../api/api';

interface NotificationDataProviderProps {
  children: React.ReactNode;
  userId?: number | string;
}

interface NotificationDataContextValue {
  notifications: UserNotification[];
  unreadCount: UnreadCountResponse;
  loading: boolean;
}

interface NotificationDataActionsContextValue {
  refresh: () => Promise<void>;
  markAsRead: (id: string) => Promise<void>;
  dismiss: (id: string) => Promise<void>;
  markAllAsRead: () => Promise<void>;
}

interface NotificationDataMutationsContextValue {
  prependNotification: (notification: UserNotification) => void;
  applyRemoteUpdate: (message: Pick<WSMessage, 'id' | 'read_at' | 'dismissed_at' | 'confirmed_at'>) => void;
  confirmNotificationLocally: (id: string) => UserNotification | null;
  removeNotificationLocally: (id: string) => UserNotification | null;
}

const NotificationDataContext = createContext<NotificationDataContextValue | null>(null);
const NotificationDataActionsContext = createContext<NotificationDataActionsContextValue | null>(null);
const NotificationDataMutationsContext = createContext<NotificationDataMutationsContextValue | null>(null);

const EMPTY_UNREAD_COUNT: UnreadCountResponse = {
  total: 0,
  by_type: { alert: 0, task: 0, system: 0, approval: 0 },
  by_severity: { critical: 0, warning: 0, info: 0 },
};

export const NotificationDataProvider: React.FC<NotificationDataProviderProps> = ({
  children,
  userId,
}) => {
  const hasAuth = Boolean(userId);
  const [notifications, setNotifications] = useState<UserNotification[]>([]);
  const [unreadCount, setUnreadCount] = useState<UnreadCountResponse>(EMPTY_UNREAD_COUNT);
  const [loading, setLoading] = useState(false);
  const mountedRef = useRef(false);
  const notificationsRef = useRef<UserNotification[]>([]);
  const loadNotificationsRef = useRef<(() => Promise<void>) | null>(null);

  useEffect(() => {
    notificationsRef.current = notifications;
  }, [notifications]);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  loadNotificationsRef.current = async () => {
    if (!hasAuth || !mountedRef.current) {
      return;
    }

    try {
      setLoading(true);
      const [listRes, countRes] = await Promise.all([
        notificationApi.getNotifications({ pageSize: 20 }),
        notificationApi.getUnreadCount(),
      ]);

      if (!mountedRef.current) {
        return;
      }

      setNotifications(listRes.data.list);
      setUnreadCount(countRes.data);
    } catch (error) {
      if (
        error instanceof ApiRequestError
        && (
          error.statusCode === 401
          || isAuthBusinessCode(error.businessCode)
          || /未授权|未认证|登录已过期/.test(error.message)
        )
      ) {
        return;
      }

      console.error('加载通知失败:', error);
    } finally {
      if (mountedRef.current) {
        setLoading(false);
      }
    }
  };

  const refresh = useCallback(async () => {
    await loadNotificationsRef.current?.();
  }, []);

  const markAsRead = useCallback(async (id: string) => {
    await notificationApi.markAsRead(id);
    if (!mountedRef.current) {
      return;
    }

    const target = notificationsRef.current.find((notification) => notification.id === id);
    setNotifications((prev) =>
      prev.map((notification) =>
        notification.id === id
          ? { ...notification, read_at: notification.read_at || new Date().toISOString() }
          : notification
      )
    );

    if (target && !target.read_at) {
      setUnreadCount((prev) => ({
        ...prev,
        total: Math.max(0, prev.total - 1),
      }));
    }
  }, []);

  const dismiss = useCallback(async (id: string) => {
    await notificationApi.dismiss(id);
    if (!mountedRef.current) {
      return;
    }

    const target = notificationsRef.current.find((notification) => notification.id === id);
    setNotifications((prev) => prev.filter((notification) => notification.id !== id));

    if (target && !target.read_at) {
      setUnreadCount((prev) => ({
        ...prev,
        total: Math.max(0, prev.total - 1),
      }));
    }
  }, []);

  const markAllAsRead = useCallback(async () => {
    await notificationApi.markAllAsRead();
    if (!mountedRef.current) {
      return;
    }

    setNotifications((prev) =>
      prev.map((notification) => ({
        ...notification,
        read_at: notification.read_at || new Date().toISOString(),
      }))
    );
    setUnreadCount((prev) => ({ ...prev, total: 0 }));
  }, []);

  const prependNotification = useCallback((notification: UserNotification) => {
    if (!mountedRef.current) {
      return;
    }

    setNotifications((prev) => [notification, ...prev.slice(0, 19)]);
    setUnreadCount((prev) => ({
      ...prev,
      total: prev.total + 1,
    }));
  }, []);

  const applyRemoteUpdate = useCallback((message: Pick<WSMessage, 'id' | 'read_at' | 'dismissed_at' | 'confirmed_at'>) => {
    if (!mountedRef.current || !message.id) {
      return;
    }

    setNotifications((prev) =>
      prev.map((notification) => {
        if (notification.id.toString() !== message.id) {
          return notification;
        }

        return {
          ...notification,
          read_at: message.read_at || notification.read_at,
          dismissed_at: message.dismissed_at || notification.dismissed_at,
          confirmed_at: message.confirmed_at || notification.confirmed_at,
        };
      })
    );
  }, []);

  const confirmNotificationLocally = useCallback((id: string) => {
    if (!mountedRef.current) {
      return null;
    }

    const target = notificationsRef.current.find((notification) => notification.id === id) || null;
    if (!target) {
      return null;
    }

    const timestamp = new Date().toISOString();
    setNotifications((prev) =>
      prev.map((notification) =>
        notification.id === id
          ? {
            ...notification,
            read_at: notification.read_at || timestamp,
            confirmed_at: timestamp,
          }
          : notification
      )
    );

    if (!target.read_at) {
      setUnreadCount((prev) => ({
        ...prev,
        total: Math.max(0, prev.total - 1),
      }));
    }

    return target;
  }, []);

  const removeNotificationLocally = useCallback((id: string) => {
    if (!mountedRef.current) {
      return null;
    }

    const target = notificationsRef.current.find((notification) => notification.id === id) || null;
    if (!target) {
      return null;
    }

    setNotifications((prev) => prev.filter((notification) => notification.id !== id));
    if (!target.read_at) {
      setUnreadCount((prev) => ({
        ...prev,
        total: Math.max(0, prev.total - 1),
      }));
    }

    return target;
  }, []);

  useEffect(() => {
    if (!hasAuth) {
      return;
    }

    void refresh();
  }, [hasAuth, refresh]);

  const dataValue = useMemo<NotificationDataContextValue>(() => ({
    notifications,
    unreadCount,
    loading,
  }), [loading, notifications, unreadCount]);

  const actionsValue = useMemo<NotificationDataActionsContextValue>(() => ({
    refresh,
    markAsRead,
    dismiss,
    markAllAsRead,
  }), [dismiss, markAllAsRead, markAsRead, refresh]);

  const mutationsValue = useMemo<NotificationDataMutationsContextValue>(() => ({
    prependNotification,
    applyRemoteUpdate,
    confirmNotificationLocally,
    removeNotificationLocally,
  }), [applyRemoteUpdate, confirmNotificationLocally, prependNotification, removeNotificationLocally]);

  return (
    <NotificationDataContext.Provider value={dataValue}>
      <NotificationDataActionsContext.Provider value={actionsValue}>
        <NotificationDataMutationsContext.Provider value={mutationsValue}>
          {children}
        </NotificationDataMutationsContext.Provider>
      </NotificationDataActionsContext.Provider>
    </NotificationDataContext.Provider>
  );
};

export function useNotificationDataContext(): NotificationDataContextValue {
  const context = useContext(NotificationDataContext);
  if (!context) {
    throw new Error('useNotificationDataContext must be used within NotificationDataProvider');
  }
  return context;
}

export function useNotificationDataActionsContext(): NotificationDataActionsContextValue {
  const context = useContext(NotificationDataActionsContext);
  if (!context) {
    throw new Error('useNotificationDataActionsContext must be used within NotificationDataProvider');
  }
  return context;
}

export function useNotificationDataMutationsContext(): NotificationDataMutationsContextValue {
  const context = useContext(NotificationDataMutationsContext);
  if (!context) {
    throw new Error('useNotificationDataMutationsContext must be used within NotificationDataProvider');
  }
  return context;
}
