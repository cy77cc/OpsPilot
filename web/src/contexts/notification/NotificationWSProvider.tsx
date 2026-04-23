import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { useNotificationWebSocket } from '../../hooks/useNotificationWebSocket';
import { playNotificationSound } from '../../hooks/useNotificationSound';
import { notify as sendBrowserNotification } from '../../utils/browserNotification';
import { safeNavigate } from '../../utils/safeNavigate';
import type { WSConnectionStatus, WSMessage } from '../../types/notification';
import {
  useNotificationDataActionsContext,
  useNotificationDataMutationsContext,
} from './NotificationDataProvider';

interface NotificationWSProviderProps {
  children: React.ReactNode;
  userId?: number | string;
  pollInterval?: number;
}

interface NotificationWSContextValue {
  wsStatus: WSConnectionStatus;
}

const NotificationWSContext = createContext<NotificationWSContextValue | null>(null);

export const NotificationWSProvider: React.FC<NotificationWSProviderProps> = ({
  children,
  userId,
  pollInterval = 30000,
}) => {
  const hasAuth = Boolean(userId);
  const [wsStatus, setWsStatus] = useState<WSConnectionStatus>('disconnected');
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const mountedRef = useRef(false);
  const { refresh } = useNotificationDataActionsContext();
  const { prependNotification, applyRemoteUpdate } = useNotificationDataMutationsContext();

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
    };
  }, []);

  const handleWSMessage = React.useCallback((message: WSMessage) => {
    if (!mountedRef.current) {
      return;
    }

    if (message.type === 'new' && message.notification) {
      const notification = message.notification;
      prependNotification(notification);

      const severity = notification.notification?.severity;
      const soundType = severity === 'critical' ? 'error' : severity === 'warning' ? 'warning' : 'default';
      playNotificationSound(soundType as 'default' | 'warning' | 'error');

      if (typeof document !== 'undefined' && (document.hidden || document.visibilityState === 'hidden')) {
        sendBrowserNotification(
          notification.notification?.title || '新通知',
          notification.notification?.content,
          {
            tag: notification.id,
            onClick: () => {
              if (notification.notification?.action_url) {
                safeNavigate(notification.notification.action_url);
              }
            },
          }
        );
      }

      return;
    }

    if (message.type === 'update') {
      applyRemoteUpdate(message);
    }
  }, [applyRemoteUpdate, prependNotification]);

  const { status: wsConnectionStatus } = useNotificationWebSocket({
    userId,
    onMessage: handleWSMessage,
    onConnect: () => {
      if (mountedRef.current) {
        setWsStatus('connected');
      }
    },
    onDisconnect: () => {
      if (mountedRef.current) {
        setWsStatus('disconnected');
      }
    },
    reconnectInterval: 1000,
    maxReconnectInterval: 30000,
  });

  useEffect(() => {
    if (mountedRef.current) {
      setWsStatus(wsConnectionStatus);
    }
  }, [wsConnectionStatus]);

  useEffect(() => {
    if (!hasAuth) {
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
      return;
    }

    if (wsStatus === 'connected') {
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
        console.log('Notification: WebSocket 已连接，停止轮询');
      }
      return;
    }

    if (!pollingRef.current) {
      pollingRef.current = setInterval(() => {
        void refresh();
      }, pollInterval);
      console.log(`Notification: WebSocket 断开，启动轮询 (间隔 ${pollInterval}ms)`);
    }

    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
    };
  }, [hasAuth, pollInterval, refresh, wsStatus]);

  const value = useMemo<NotificationWSContextValue>(() => ({
    wsStatus,
  }), [wsStatus]);

  return (
    <NotificationWSContext.Provider value={value}>
      {children}
    </NotificationWSContext.Provider>
  );
};

export function useNotificationWSContext(): NotificationWSContextValue {
  const context = useContext(NotificationWSContext);
  if (!context) {
    throw new Error('useNotificationWSContext must be used within NotificationWSProvider');
  }
  return context;
}
