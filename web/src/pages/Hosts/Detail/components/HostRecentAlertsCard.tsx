import React from 'react';
import { Card, Table, Tag, Button } from 'antd';
import { ArrowRightOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';

interface AlertItem {
  id: string;
  level: 'critical' | 'warning' | 'info';
  title: string;
  status: 'active' | 'resolved';
  startedAt: string;
  duration: string;
}

const mockAlerts: AlertItem[] = [
  {
    id: '1',
    level: 'warning',
    title: '磁盘使用率过高',
    status: 'resolved',
    startedAt: '2024-05-12 10:15:22',
    duration: '45 分钟',
  },
  {
    id: '2',
    level: 'critical',
    title: '内存使用率超过 90%',
    status: 'resolved',
    startedAt: '2024-05-12 09:32:10',
    duration: '18 分钟',
  },
  {
    id: '3',
    level: 'info',
    title: '主机连接恢复',
    status: 'resolved',
    startedAt: '2024-05-12 08:20:05',
    duration: '2 分钟',
  },
];

interface HostRecentAlertsCardProps {
  loading?: boolean;
  onViewAll?: () => void;
}

const HostRecentAlertsCard: React.FC<HostRecentAlertsCardProps> = ({ loading, onViewAll }) => {
  const columns: ColumnsType<AlertItem> = [
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 80,
      render: (level: AlertItem['level']) => {
        const config = {
          critical: { color: 'error', text: '严重' },
          warning: { color: 'warning', text: '警告' },
          info: { color: 'processing', text: '信息' },
        };
        const { color, text } = config[level];
        return <Tag color={color} className="m-0">{text}</Tag>;
      },
    },
    {
      title: '告警名称',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => (
        <span className={status === 'active' ? 'text-red-500' : 'text-gray-400'}>
          {status === 'active' ? '进行中' : '已恢复'}
        </span>
      ),
    },
    {
      title: '开始时间',
      dataIndex: 'startedAt',
      key: 'startedAt',
      width: 150,
      render: (text: string) => <span className="text-gray-500 text-xs">{text}</span>,
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: () => <Button type="link" size="small" className="p-0">详情</Button>,
    },
  ];

  return (
    <Card 
      title="最近告警" 
      loading={loading} 
      className="h-full"
      extra={
        <a onClick={onViewAll} className="text-xs text-blue-600 hover:text-blue-800 flex items-center gap-1">
          查看全部告警 <ArrowRightOutlined />
        </a>
      }
    >
      <Table 
        columns={columns} 
        dataSource={mockAlerts} 
        rowKey="id" 
        pagination={false} 
        size="small"
        className="host-recent-table"
      />
    </Card>
  );
};

export default HostRecentAlertsCard;
