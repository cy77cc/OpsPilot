import React from 'react';
import { Badge, Button, Card, Empty, List, Tag, Typography } from 'antd';
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
        <Button type="link" onClick={() => navigate('/monitoring/alerts')}>
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

      <List
        loading={loading}
        dataSource={alerts}
        locale={{ emptyText: <Empty description="暂无告警" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
        renderItem={(item) => (
          <List.Item
            className="cursor-pointer rounded-md px-2"
            onClick={() => navigate('/monitoring/alerts')}
          >
            <List.Item.Meta
              title={<span className="text-sm font-medium">{item.title}</span>}
              description={<span className="text-xs text-gray-500">来源: {item.source}</span>}
            />
            <Tag color={severityColorMap[item.severity] || 'default'}>{item.severity}</Tag>
          </List.Item>
        )}
      />
    </Card>
  );
};

export default AlertPanel;
