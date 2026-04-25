import React from 'react';
import { AlertOutlined, CloudServerOutlined, DeploymentUnitOutlined, NodeIndexOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Typography } from 'antd';
import dayjs from 'dayjs';
import { useNavigate } from 'react-router-dom';
import type { EventItem } from '../../api/modules/dashboard';

interface EventStreamProps {
  events: EventItem[];
  loading?: boolean;
}

const formatRelativeTime = (time: string): string => {
  const at = dayjs(time);
  const now = dayjs();
  const diffMinutes = now.diff(at, 'minute');
  if (diffMinutes < 60) {
    return `${Math.max(1, diffMinutes)} 分钟前`;
  }
  const diffHours = now.diff(at, 'hour');
  if (diffHours < 24 && now.isSame(at, 'day')) {
    return `${diffHours} 小时前`;
  }
  return at.format('YYYY-MM-DD HH:mm');
};

const iconByType = (type: string) => {
  if (type.includes('host') || type.includes('node')) {
    return <NodeIndexOutlined className="text-blue-500" />;
  }
  if (type.includes('cluster')) {
    return <CloudServerOutlined className="text-purple-500" />;
  }
  if (type.includes('deploy') || type.includes('release')) {
    return <DeploymentUnitOutlined className="text-green-500" />;
  }
  if (type.includes('alert') || type.includes('critical') || type.includes('warning')) {
    return <AlertOutlined className="text-red-500" />;
  }
  return <AlertOutlined className="text-gray-500" />;
};

const EventStream: React.FC<EventStreamProps> = ({ events, loading }) => {
  const navigate = useNavigate();

  return (
    <Card
      title={<span>最近事件</span>}
      extra={(
        <Button type="link" onClick={() => navigate('/delivery/deployments/observability/audit-logs')}>
          查看全部
        </Button>
      )}
      styles={{ body: { minHeight: 240, paddingTop: 8, paddingBottom: 8 } }}
    >
      <div className="flex flex-col">
        {loading && <div className="text-center py-4 text-gray-400">加载中...</div>}
        {!loading && events.length === 0 && (
          <Empty description="暂无事件" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
        {!loading && events.map((item, index) => (
          <div key={index} className="flex gap-3 py-3 border-b border-gray-50 last:border-0 items-start">
            <div className="pt-1">
              {iconByType(item.type)}
            </div>
            <div className="flex-1 flex items-center justify-between gap-3">
              <Typography.Text className="text-sm">{item.message}</Typography.Text>
              <Typography.Text type="secondary" className="shrink-0 text-xs">
                {formatRelativeTime(item.createdAt)}
              </Typography.Text>
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
};

export default EventStream;
