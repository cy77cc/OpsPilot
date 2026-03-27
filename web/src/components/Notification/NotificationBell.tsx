import React from 'react';
import { Badge, Button, Popover } from 'antd';
import { BellOutlined } from '@ant-design/icons';
import NotificationPanel from './NotificationPanel';
import { useNotificationContext } from '../../contexts/NotificationContext';

interface NotificationBellProps {
  onViewAll?: () => void;
  onSettings?: () => void;
}

const NotificationBell: React.FC<NotificationBellProps> = ({
  onViewAll,
  onSettings,
}) => {
  const { unreadCount } = useNotificationContext();
  const [open, setOpen] = React.useState(false);

  const count = unreadCount.total > 99 ? '99+' : unreadCount.total;

  const bellButton = (
    <Button
      type="text"
      icon={<BellOutlined />}
      className="notification-bell-button"
      onClick={() => setOpen(!open)}
    />
  );

  return (
    <Popover
      open={open}
      onOpenChange={setOpen}
      trigger="click"
      placement="bottomRight"
      arrow={false}
      overlayClassName="notification-bell-popover"
      content={
        <NotificationPanel
          onViewAll={() => {
            setOpen(false);
            onViewAll?.();
          }}
          onSettings={onSettings}
        />
      }
    >
      {unreadCount.total > 0 ? (
        <Badge count={count} size="small" offset={[-2, 2]}>
          {bellButton}
        </Badge>
      ) : (
        bellButton
      )}
    </Popover>
  );
};

export default NotificationBell;
