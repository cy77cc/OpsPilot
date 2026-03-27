import { useNotificationContext } from '../contexts/NotificationContext';
import type { NotificationType, NotificationSeverity } from '../types/notification';

// 便捷 Hook，提供更简洁的访问方式
export const useNotification = () => {
  const context = useNotificationContext();

  return {
    // 状态
    notifications: context.notifications,
    unreadCount: context.unreadCount.total,
    unreadByType: context.unreadCount.by_type,
    unreadBySeverity: context.unreadCount.by_severity,
    loading: context.loading,
    wsStatus: context.wsStatus,

    // 过滤后的通知
    getNotificationsByType: (type: NotificationType) =>
      context.notifications.filter((n) => n.notification.type === type),

    getNotificationsBySeverity: (severity: NotificationSeverity) =>
      context.notifications.filter((n) => n.notification.severity === severity),

    getUnreadNotifications: () =>
      context.notifications.filter((n) => !n.read_at),

    // 操作
    refresh: context.refresh,
    markAsRead: context.markAsRead,
    dismiss: context.dismiss,
    confirm: context.confirm,
    markAllAsRead: context.markAllAsRead,
  };
};

export default useNotification;
