import React from 'react';
import { Card, Table, Typography } from 'antd';
import { ArrowRightOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';

const { Text } = Typography;

interface OperationItem {
  id: string;
  type: string;
  content: string;
  operator: string;
  time: string;
}

const mockOperations: OperationItem[] = [
  {
    id: '1',
    type: '登录主机',
    content: '通过 Web 终端登录主机',
    operator: 'admin',
    time: '2024-05-12 14:20:30',
  },
  {
    id: '2',
    type: '执行命令',
    content: '执行命令 `df -h`',
    operator: 'admin',
    time: '2024-05-12 14:19:45',
  },
  {
    id: '3',
    type: '重启服务',
    content: '重启服务 `nginx`',
    operator: 'admin',
    time: '2024-05-12 14:15:12',
  },
  {
    id: '4',
    type: '修改配置',
    content: '修改主机标签',
    operator: 'admin',
    time: '2024-05-12 14:10:05',
  },
];

interface HostRecentOperationCardProps {
  loading?: boolean;
  onViewAll?: () => void;
}

const HostRecentOperationCard: React.FC<HostRecentOperationCardProps> = ({ loading, onViewAll }) => {
  const columns: ColumnsType<OperationItem> = [
    {
      title: '操作类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (text: string) => <Text strong className="text-xs">{text}</Text>,
    },
    {
      title: '操作内容',
      dataIndex: 'content',
      key: 'content',
      ellipsis: true,
      render: (text: string) => <Text className="text-xs">{text}</Text>,
    },
    {
      title: '操作人',
      dataIndex: 'operator',
      key: 'operator',
      width: 80,
      render: (text: string) => <Text className="text-xs">{text}</Text>,
    },
    {
      title: '操作时间',
      dataIndex: 'time',
      key: 'time',
      width: 150,
      render: (text: string) => <span className="text-gray-400 text-xs">{text}</span>,
    },
  ];

  return (
    <Card 
      title="最近操作记录" 
      loading={loading} 
      className="h-full"
      extra={
        <a onClick={onViewAll} className="text-xs text-blue-600 hover:text-blue-800 flex items-center gap-1">
          查看全部记录 <ArrowRightOutlined />
        </a>
      }
    >
      <Table 
        columns={columns} 
        dataSource={mockOperations} 
        rowKey="id" 
        pagination={false} 
        size="small"
        className="host-recent-table"
      />
    </Card>
  );
};

export default HostRecentOperationCard;
