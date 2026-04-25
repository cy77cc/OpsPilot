import React from 'react';
import { Badge, Button, Card, Empty, Tag, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import type { AlertItem } from '../../api/modules/dashboard';

interface AlertPanelProps {
  alerts: AlertItem[];
  loading?: boolean;
}

const severityColorMap: Record<string, string> = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

const AlertPanel: React.FC<AlertPanelProps> = ({ alerts, loading }) => {
  const navigate = useNavigate();

  return (
    <Card
      title={<span>活跃告警</span>}
      extra={(
        <Button type="link" onClick={() => navigate('/observability/monitor/alerts')}>
          查看全部
        </Button>
      )}
      styles={{ body: { minHeight: 320 } }}
    >
      <div className="mb-3">
        <Badge count={alerts.length} showZero color={alerts.length > 0 ? '#ef4444' : '#22c55e'} />
        <Typography.Text type="secondary" className="ml-2">
          当前活跃告警
        </Typography.Text>
      </div>

      <div className="flex flex-col gap-1">
        {loading && <div className="text-center py-4">加载中...</div>}
        {!loading && alerts.length === 0 && (
          <Empty description="暂无告警" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
        {!loading && alerts.map((item, index) => (
          <div
            key={index}
            className="cursor-pointer rounded-md px-2 py-2 hover:bg-gray-50 flex items-center justify-between border-b border-gray-50 last:border-0"
            onClick={() => navigate('/observability/monitor/alerts')}
          >
            <div className="flex flex-col">
              <span className="text-sm font-medium">{item.title}</span>
              <span className="text-xs text-gray-500">来源: {item.source}</span>
            </div>
            <Tag color={severityColorMap[item.severity] || 'default'}>{item.severity}</Tag>
          </div>
        ))}
      </div>
    </Card>
  );
};

export default AlertPanel;
