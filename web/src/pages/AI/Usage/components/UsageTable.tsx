import React from 'react';
import { Table, Tag } from 'antd';
import type { UsageLog } from '../../../../api/modules/ai';
import dayjs from 'dayjs';

interface UsageTableProps {
  data?: UsageLog[];
  total: number;
  loading: boolean;
  page: number;
  pageSize: number;
  onPageChange: (page: number, pageSize: number) => void;
}

const statusColors: Record<string, string> = {
  completed: 'green',
  failed: 'red',
  rejected: 'orange',
  waiting_approval: 'blue',
};

const UsageTable: React.FC<UsageTableProps> = ({ data, total, loading, page, pageSize, onPageChange }) => {
  const columns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '场景',
      dataIndex: 'scene',
      key: 'scene',
      width: 120,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (v: string) => <Tag color={statusColors[v] || 'default'}>{v}</Tag>,
    },
    {
      title: 'Tokens',
      dataIndex: 'total_tokens',
      key: 'total_tokens',
      width: 100,
      render: (v: number) => v?.toLocaleString?.() || v,
    },
    {
      title: '耗时',
      dataIndex: 'duration_ms',
      key: 'duration_ms',
      width: 100,
      render: (v: number) => `${v}ms`,
    },
  ];

  return (
    <Table
      columns={columns}
      dataSource={data}
      rowKey="id"
      loading={loading}
      pagination={{
        current: page,
        pageSize,
        total,
        onChange: onPageChange,
        showSizeChanger: true,
        showTotal: (total) => `共 ${total} 条`,
      }}
    />
  );
};

export default UsageTable;
