import React from 'react';
import { Button, Space } from 'antd';
import {
  AlertOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  ClockCircleOutlined,
  CheckOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import type { UserNotification } from '../../types/notification';
import { formatRelativeTime } from './utils';
import { useNotificationContext } from '../../contexts/NotificationContext';

interface NotificationItemProps {
  notification: UserNotification;
  onMarkAsRead?: (id: string) => void;
  onDismiss?: (id: string) => void;
  onConfirm?: (id: string) => void;
  onReject?: (id: string) => void;
  onClick?: (notification: UserNotification) => void;
}

const severityConfig = {
  critical: { icon: <CloseCircleOutlined />, color: '#ff4d4f', bgColor: '#fff1f0' },
  warning: { icon: <AlertOutlined />, color: '#faad14', bgColor: '#fffbe6' },
  info: { icon: <InfoCircleOutlined />, color: '#1677ff', bgColor: '#e6f4ff' },
};

const NotificationItem: React.FC<NotificationItemProps> = ({
  notification,
  onMarkAsRead,
  onDismiss,
  onConfirm,
  onReject,
  onClick,
}) => {
  const { notification: n, read_at, confirmed_at } = notification;
  const { approvalActionStates } = useNotificationContext();
  const config = severityConfig[n.severity];
  const isUnread = !read_at;
  const approvalKey = n.source_id || notification.id;
  const approvalActionState = approvalActionStates[approvalKey];
  const isApprovalSubmitting = approvalActionState?.state === 'submitting';
  const approvalActionMessage = approvalActionState?.message;
  const canConfirm = (
    ((n.action_type === 'confirm' && n.type === 'alert') ||
      (n.action_type === 'approve' && n.type === 'approval')) &&
    !confirmed_at
  );
  const confirmLabel = n.type === 'approval' ? '批准请求' : '确认告警';

  const handleClick = () => {
    onClick?.(notification);
  };

  const handleActionClick = (e: React.MouseEvent, action: () => void) => {
    e.stopPropagation();
    action();
  };

  return (
    <div
      className={`notification-item ${isUnread ? 'notification-item-unread' : ''}`}
      onClick={handleClick}
    >
      <div className="notification-item-icon" style={{ color: config.color, backgroundColor: config.bgColor }}>
        {config.icon}
      </div>

      <div className="notification-item-content">
        <div className="notification-item-header">
          <span className="notification-item-title">{n.title}</span>
          {isUnread && <span className="notification-item-dot" />}
        </div>

        {n.content && (
          <div className="notification-item-text">{n.content}</div>
        )}

        <div className="notification-item-meta">
          <Space size={8}>
            <span className="notification-item-source">{n.source}</span>
            <span className="notification-item-time">
              <ClockCircleOutlined /> {formatRelativeTime(n.created_at)}
            </span>
          </Space>
        </div>

        {(canConfirm || !read_at) && (
          <div className="notification-item-actions">
            {canConfirm && (
              <>
                <Button
                  type="primary"
                  size="small"
                  icon={<CheckOutlined />}
                  loading={isApprovalSubmitting}
                  disabled={isApprovalSubmitting}
                  onClick={(e) => handleActionClick(e, () => onConfirm?.(notification.id))}
                >
                  {confirmLabel}
                </Button>
                {n.type === 'approval' && (
                  <Button
                    size="small"
                    danger
                    loading={isApprovalSubmitting}
                    disabled={isApprovalSubmitting}
                    onClick={(e) => handleActionClick(e, () => onReject?.(notification.id))}
                  >
                    驳回请求
                  </Button>
                )}
              </>
            )}
            {!read_at && (
              <>
                <Button
                  size="small"
                  onClick={(e) => handleActionClick(e, () => onMarkAsRead?.(notification.id))}
                >
                  标记已读
                </Button>
                <Button
                  size="small"
                  icon={<CloseOutlined />}
                  onClick={(e) => handleActionClick(e, () => onDismiss?.(notification.id))}
                />
              </>
            )}
          </div>
        )}

        {approvalActionMessage && n.type === 'approval' && (
          <div
            className="notification-item-action-message"
            style={{
              marginTop: 6,
              fontSize: 12,
              lineHeight: '18px',
              color: approvalActionState?.state === 'refresh-needed' ? '#ff4d4f' : undefined,
            }}
          >
            {approvalActionMessage}
          </div>
        )}

        {confirmed_at && (
          <div className="notification-item-confirmed">
            <CheckCircleOutlined style={{ color: '#52c41a' }} /> 已确认
          </div>
        )}
      </div>
    </div>
  );
};

export default NotificationItem;
