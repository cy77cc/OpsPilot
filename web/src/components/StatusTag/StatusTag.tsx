import React from 'react';
import { theme } from '../../theme/theme.config';

interface StatusTagProps {
  status: 'RUNNING' | 'WARNING' | 'ERROR' | 'OFFLINE';
  size?: 'small' | 'medium' | 'large';
  shape?: 'circle' | 'rectangle';
}

const StatusTag: React.FC<StatusTagProps> = ({
  status,
  size = 'medium',
  shape = 'rectangle'
}) => {
  const statusConfig = {
    RUNNING: {
      label: '运行中',
      color: theme.colors.status.running,
      className: 'status-running'
    },
    WARNING: {
      label: '警告',
      color: theme.colors.status.warning,
      className: 'status-warning'
    },
    ERROR: {
      label: '故障',
      color: theme.colors.status.error,
      className: 'status-error'
    },
    OFFLINE: {
      label: '离线',
      color: theme.colors.status.offline,
      className: 'status-offline'
    }
  }[status];

  const sizeConfig = {
    small: 'w-2 h-2',
    medium: 'w-3 h-3',
    large: 'w-4 h-4'
  }[size];

  if (shape === 'circle') {
    return (
      <span
        className={`inline-block rounded-full ${sizeConfig} animate-status-pulse`}
        style={{ backgroundColor: statusConfig.color }}
        title={statusConfig.label}
      />
    );
  }

  return (
    <span
      className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium`}
      style={{
        backgroundColor: `${statusConfig.color}20`,
        color: statusConfig.color
      }}
    >
      <span
        className={`inline-block rounded-full ${sizeConfig} mr-1 animate-status-pulse`}
        style={{ backgroundColor: statusConfig.color }}
      />
      {statusConfig.label}
    </span>
  );
};

export default StatusTag;