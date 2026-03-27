import React, { useMemo, useCallback, memo } from 'react';
import { Empty, Spin } from 'antd';
import NotificationItem from './NotificationItem';
import type { UserNotification, NotificationType } from '../../types/notification';

interface NotificationListProps {
  notifications: UserNotification[];
  loading?: boolean;
  filterType?: NotificationType | 'all';
  onMarkAsRead?: (id: string) => void;
  onDismiss?: (id: string) => void;
  onConfirm?: (id: string) => void;
  onReject?: (id: string) => void;
  onClick?: (notification: UserNotification) => void;
}

// 使用 memo 优化列表项渲染
const MemoNotificationItem = memo(NotificationItem);

const NotificationList: React.FC<NotificationListProps> = ({
  notifications,
  loading = false,
  filterType = 'all',
  onMarkAsRead,
  onDismiss,
  onConfirm,
  onReject,
  onClick,
}) => {
  // 过滤通知 - 使用 useMemo 缓存结果
  const filteredNotifications = useMemo(() => {
    if (filterType === 'all') {
      return notifications;
    }
    return notifications.filter((n) => n.notification.type === filterType);
  }, [notifications, filterType]);

  // 使用 useCallback 缓存回调函数
  const handleMarkAsRead = useCallback((id: string) => {
    onMarkAsRead?.(id);
  }, [onMarkAsRead]);

  const handleDismiss = useCallback((id: string) => {
    onDismiss?.(id);
  }, [onDismiss]);

  const handleConfirm = useCallback((id: string) => {
    onConfirm?.(id);
  }, [onConfirm]);

  const handleReject = useCallback((id: string) => {
    onReject?.(id);
  }, [onReject]);

  const handleClick = useCallback((notification: UserNotification) => {
    onClick?.(notification);
  }, [onClick]);

  if (loading) {
    return (
      <div className="notification-list-loading">
        <Spin />
      </div>
    );
  }

  if (filteredNotifications.length === 0) {
    return (
      <div className="notification-list-empty">
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="暂无通知"
        />
      </div>
    );
  }

  return (
    <div className="notification-list" role="list" aria-label="通知列表">
      {filteredNotifications.map((notification) => (
        <MemoNotificationItem
          key={notification.id}
          notification={notification}
          onMarkAsRead={handleMarkAsRead}
          onDismiss={handleDismiss}
          onConfirm={handleConfirm}
          onReject={handleReject}
          onClick={handleClick}
        />
      ))}
    </div>
  );
};

// 使用 memo 导出
export default memo(NotificationList);
