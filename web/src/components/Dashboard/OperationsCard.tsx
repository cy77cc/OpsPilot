import React from 'react';
import { Card, Badge, List, Typography } from 'antd';
import {
  RocketOutlined,
  ThunderboltOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import type { OperationsOverview } from '../../api/modules/dashboard';

const { Text } = Typography;

interface DetailItem {
  label: string;
  value: number;
  type?: 'success' | 'error';
}

interface Props {
  data: OperationsOverview;
  loading?: boolean;
}

const OperationsCard: React.FC<Props> = ({ data, loading }) => {
  const items: Array<{
    key: string;
    icon: React.ReactNode;
    label: string;
    badge: number;
    badgeStatus?: 'success' | 'error' | 'processing';
    details: DetailItem[];
  }> = [
    {
      key: 'deployments',
      icon: <RocketOutlined />,
      label: '部署状态',
      badge: data.deployments.running + data.deployments.pendingApproval,
      details: [
        { label: '进行中', value: data.deployments.running },
        { label: '待审批', value: data.deployments.pendingApproval },
        { label: '今日成功', value: data.deployments.todaySuccess, type: 'success' },
        { label: '今日失败', value: data.deployments.todayFailed, type: 'error' },
      ],
    },
    {
      key: 'cicd',
      icon: <ThunderboltOutlined />,
      label: 'CI/CD',
      badge: data.cicd.running + data.cicd.queued,
      details: [
        { label: '运行中', value: data.cicd.running },
        { label: '排队中', value: data.cicd.queued },
        { label: '今日成功', value: data.cicd.success, type: 'success' },
        { label: '今日失败', value: data.cicd.failed, type: 'error' },
      ],
    },
    {
      key: 'issue_pods',
      icon: <WarningOutlined />,
      label: '异常 Pod',
      badge: data.issuePods.total,
      badgeStatus: data.issuePods.total > 0 ? 'error' : 'success',
      details: Object.entries(data.issuePods.byType).map(([type, count]) => ({
        label: type,
        value: count,
      })).slice(0, 4),
    },
  ];

  return (
    <Card title="运行状态" size="small" loading={loading}>
      <List
        dataSource={items}
        renderItem={(item) => (
          <List.Item className="!py-2 !px-0">
            <div className="w-full">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  {item.icon}
                  <Text strong>{item.label}</Text>
                </div>
                <Badge
                  count={item.badge}
                  status={item.badgeStatus || 'processing'}
                  showZero
                />
              </div>
              <div className="grid grid-cols-2 gap-2 pl-6">
                {item.details.slice(0, 4).map((detail, idx) => (
                  <div key={idx} className="flex justify-between">
                    <Text type="secondary" className="text-xs">{detail.label}</Text>
                    <Text
                      strong
                      className="text-xs"
                      style={detail.type === 'success' ? { color: '#52c41a' } : detail.type === 'error' ? { color: '#ff4d4f' } : {}}
                    >
                      {detail.value}
                    </Text>
                  </div>
                ))}
              </div>
            </div>
          </List.Item>
        )}
      />
    </Card>
  );
};

export default OperationsCard;
