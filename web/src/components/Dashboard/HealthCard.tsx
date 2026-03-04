import React, { useMemo } from 'react';
import { Card, Progress, Space, Typography } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, WarningOutlined } from '@ant-design/icons';
import type { HealthStats } from '../../api/modules/dashboard';

interface HealthCardProps {
  title: string;
  data: HealthStats;
  onClick?: () => void;
}

const HealthCard: React.FC<HealthCardProps> = ({ title, data, onClick }) => {
  const percent = useMemo(() => {
    if (!data?.total) {
      return 0;
    }
    return Math.round((Number(data.healthy || 0) / Number(data.total || 1)) * 100);
  }, [data]);

  const color = percent >= 90 ? '#22c55e' : percent >= 70 ? '#f59e0b' : '#ef4444';
  const icon = percent >= 90 ? <CheckCircleOutlined /> : percent >= 70 ? <WarningOutlined /> : <CloseCircleOutlined />;

  return (
    <Card
      className="transition-all duration-200 hover:shadow-lg"
      style={{ cursor: onClick ? 'pointer' : 'default' }}
      onClick={onClick}
      styles={{ body: { padding: 16 } }}
    >
      <Space orientation="vertical" size={8} style={{ width: '100%' }}>
        <Typography.Text type="secondary">{title}</Typography.Text>
        <Space align="center" size={8}>
          <Typography.Title level={3} style={{ margin: 0 }}>
            {data.total}
          </Typography.Title>
          <Typography.Text type="secondary">总量</Typography.Text>
        </Space>
        <Space align="center" size={6}>
          <span style={{ color }}>{icon}</span>
          <Typography.Text style={{ color }}>{percent}% 健康</Typography.Text>
          <Typography.Text type="secondary">({data.healthy})</Typography.Text>
        </Space>
        <Progress percent={percent} showInfo={false} strokeColor={color} />
      </Space>
    </Card>
  );
};

export default HealthCard;
