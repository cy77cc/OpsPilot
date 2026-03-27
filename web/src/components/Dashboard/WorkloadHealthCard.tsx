import React from 'react';
import { Card, Progress, Space, Typography } from 'antd';
import {
  AppstoreOutlined,
  CloudServerOutlined,
  RocketOutlined,
} from '@ant-design/icons';
import type { WorkloadStats } from '../../api/modules/dashboard';

const { Text } = Typography;

interface Props {
  data: WorkloadStats;
  loading?: boolean;
}

const WorkloadHealthCard: React.FC<Props> = ({ data, loading }) => {
  const items = [
    {
      key: 'deployments',
      icon: <RocketOutlined className="text-blue-500" />,
      label: 'Deployment',
      total: data.deployments.total,
      healthy: data.deployments.healthy,
    },
    {
      key: 'statefulsets',
      icon: <CloudServerOutlined className="text-purple-500" />,
      label: 'StatefulSet',
      total: data.statefulsets.total,
      healthy: data.statefulsets.healthy,
    },
    {
      key: 'daemonsets',
      icon: <AppstoreOutlined className="text-cyan-500" />,
      label: 'DaemonSet',
      total: data.daemonsets.total,
      healthy: data.daemonsets.healthy,
    },
  ];

  return (
    <Card title="工作负载健康" loading={loading} size="small">
      <div className="space-y-3">
        {items.map((item) => {
          const percent = item.total > 0 ? Math.round((item.healthy / item.total) * 100) : 100;
          const status = percent === 100 ? 'success' : percent >= 80 ? 'normal' : 'exception';

          return (
            <div key={item.key} className="flex items-center gap-3">
              <div className="w-6">{item.icon}</div>
              <div className="flex-1">
                <div className="flex justify-between mb-1">
                  <Text type="secondary" className="text-xs">{item.label}</Text>
                  <Text className="text-xs">{item.healthy}/{item.total}</Text>
                </div>
                <Progress
                  percent={percent}
                  size="small"
                  showInfo={false}
                  status={status}
                />
              </div>
            </div>
          );
        })}
        <div className="flex justify-between pt-2 border-t border-gray-100">
          <Space>
            <Text type="secondary" className="text-xs">Service</Text>
            <Text strong>{data.services}</Text>
          </Space>
          <Space>
            <Text type="secondary" className="text-xs">Ingress</Text>
            <Text strong>{data.ingresses}</Text>
          </Space>
        </div>
      </div>
    </Card>
  );
};

export default WorkloadHealthCard;
