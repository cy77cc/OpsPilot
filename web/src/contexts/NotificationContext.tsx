import React, { useMemo } from 'react';
import type {
  UnreadCountResponse,
  UserNotification,
  WSConnectionStatus,
} from '../types/notification';
import { NotificationDataProvider, useNotificationDataActionsContext, useNotificationDataContext } from './notification/NotificationDataProvider';
import { NotificationWSProvider, useNotificationWSContext } from './notification/NotificationWSProvider';
import { ApprovalStateProvider, type ApprovalActionState, useApprovalStateContext } from './notification/ApprovalStateProvider';

export interface NotificationContextValue {
  notifications: UserNotification[];
  unreadCount: UnreadCountResponse;
  loading: boolean;
  wsStatus: WSConnectionStatus;
  approvalActionStates: Record<string, ApprovalActionState>;
  refresh: () => Promise<void>;
  markAsRead: (id: string) => Promise<void>;
  dismiss: (id: string) => Promise<void>;
  confirm: (id: string) => Promise<void>;
  reject: (id: string) => Promise<void>;
  markAllAsRead: () => Promise<void>;
}

interface NotificationProviderProps {
  children: React.ReactNode;
  userId?: number | string;
  pollInterval?: number;
}

export const NotificationProvider: React.FC<NotificationProviderProps> = ({
  children,
  userId,
  pollInterval = 30000,
}) => {
  return (
    <NotificationDataProvider userId={userId}>
      <NotificationWSProvider userId={userId} pollInterval={pollInterval}>
        <ApprovalStateProvider>{children}</ApprovalStateProvider>
      </NotificationWSProvider>
    </NotificationDataProvider>
  );
};

export const useNotificationContext = (): NotificationContextValue => {
  const { notifications, unreadCount, loading } = useNotificationDataContext();
  const { refresh, markAsRead, dismiss, markAllAsRead } = useNotificationDataActionsContext();
  const { wsStatus } = useNotificationWSContext();
  const { approvalActionStates, confirm, reject } = useApprovalStateContext();

  return useMemo(() => ({
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
  }), [
    approvalActionStates,
    confirm,
    dismiss,
    loading,
    markAllAsRead,
    markAsRead,
    notifications,
    refresh,
    reject,
    unreadCount,
    wsStatus,
  ]);
};

export default useNotificationContext;
