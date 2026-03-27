import React, { useState } from 'react';
import { Tabs, Button, Space } from 'antd';
import { CheckOutlined, SettingOutlined } from '@ant-design/icons';
import NotificationList from './NotificationList';
import { useNotificationContext } from '../../contexts/NotificationContext';
import type { UserNotification, NotificationType } from '../../types/notification';

interface NotificationPanelProps {
  onViewAll?: () => void;
  onSettings?: () => void;
}

const NotificationPanel: React.FC<NotificationPanelProps> = ({
  onViewAll,
  onSettings,
}) => {
  const {
    notifications,
    loading,
    unreadCount,
    markAsRead,
    dismiss,
    confirm,
    reject,
    markAllAsRead,
  } = useNotificationContext();

  const [activeTab, setActiveTab] = useState<NotificationType | 'all'>('all');

  const handleClick = (notification: UserNotification) => {
    // 标记已读
    if (!notification.read_at) {
      markAsRead(notification.id);
    }
    // 跳转到详情页
    if (notification.notification.action_url) {
      window.location.href = notification.notification.action_url;
    }
  };

  const handleViewAll = () => {
    onViewAll?.();
  };

  const tabItems = [
    {
      key: 'all',
      label: '全部',
    },
    {
      key: 'alert',
      label: '告警',
    },
    {
      key: 'task',
      label: '任务',
    },
    {
      key: 'system',
      label: '系统',
    },
    {
      key: 'approval',
      label: '审批',
    },
  ];

  return (
    <div className="notification-panel">
      <div className="notification-panel-header">
        <span className="notification-panel-title">通知中心</span>
        <Space>
          {unreadCount.total > 0 && (
            <Button
              type="link"
              size="small"
              icon={<CheckOutlined />}
              onClick={markAllAsRead}
            >
              全部已读
            </Button>
          )}
          <Button
            type="text"
            size="small"
            icon={<SettingOutlined />}
            onClick={onSettings}
          />
        </Space>
      </div>

      <Tabs
        className="notification-panel-tabs"
        activeKey={activeTab}
        onChange={(key) => setActiveTab(key as NotificationType | 'all')}
        items={tabItems}
        size="small"
      />

      <div className="notification-panel-content">
        <NotificationList
          notifications={notifications}
          loading={loading}
          filterType={activeTab}
          onMarkAsRead={markAsRead}
          onDismiss={dismiss}
          onConfirm={confirm}
          onReject={reject}
          onClick={handleClick}
        />
      </div>

      <div className="notification-panel-footer">
        <Button type="link" block onClick={handleViewAll}>
          查看全部通知
        </Button>
      </div>
    </div>
  );
};

export default NotificationPanel;
