import React, { useState, useEffect, useCallback } from 'react';
import { Card, Table, Tag, Input, Space, Button, DatePicker } from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { hostApi } from '../../../../api/modules/hosts';

const { RangePicker } = DatePicker;

interface OperationLogItem {
  id: string;
  type: string;
  content: string;
  operator: string;
  time: string;
  status: 'success' | 'failed';
  source_ip: string;
}

const OperationLogTab: React.FC<{ hostId: string }> = ({ hostId }) => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<OperationLogItem[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await hostApi.getHostAudits(hostId);
      setData(res.data || []);
    } finally {
      setLoading(false);
    }
  }, [hostId]);

  useEffect(() => {
    if (hostId) fetchData();
  }, [hostId, fetchData]);

  const columns: ColumnsType<OperationLogItem> = [
    { title: '操作时间', dataIndex: 'time', key: 'time', width: 180, render: (t) => new Date(t).toLocaleString(), sorter: (a, b) => a.time.localeCompare(b.time) },
    { title: '操作人', dataIndex: 'operator', key: 'operator', width: 120 },
    { title: '操作类型', dataIndex: 'type', key: 'type', width: 120, render: (type) => <Tag className="m-0">{type}</Tag> },
    { title: '详情内容', dataIndex: 'content', key: 'content', ellipsis: true },
    { title: '来源 IP', dataIndex: 'source_ip', key: 'source_ip', width: 140 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status) => (
        <Tag color={status === 'success' ? 'success' : 'error'}>
          {status === 'success' ? '成功' : '失败'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: () => <Button type="link" size="small">审计</Button>,
    },
  ];

  return (
    <Card className="h-full border-none shadow-sm mt-4">
      <div className="flex flex-col gap-4 mb-6">
        <div className="flex justify-between items-center">
          <h3 className="text-base font-medium m-0">主机操作审计</h3>
          <Button icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>刷新</Button>
        </div>
        
        <div className="flex gap-4 items-center bg-gray-50/50 p-3 rounded-lg">
          <Space size="middle">
            <span className="text-xs text-gray-500">时间范围:</span>
            <RangePicker size="small" showTime />
          </Space>
          <Input
            placeholder="搜索操作内容或人..."
            prefix={<SearchOutlined className="text-gray-400" />}
            size="small"
            style={{ width: 250 }}
            allowClear
          />
        </div>
      </div>

      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        size="small"
        pagination={{ pageSize: 15 }}
        loading={loading}
      />
    </Card>
  );
};

export default OperationLogTab;
