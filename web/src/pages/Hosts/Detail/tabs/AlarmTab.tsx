import React, { useState, useEffect, useCallback } from 'react';
import { Card, Table, Tag, Input, Space, Button, Select, DatePicker } from 'antd';
import { SearchOutlined, FilterOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { hostApi } from '../../../../api/modules/hosts';

const { RangePicker } = DatePicker;

interface AlarmHistoryItem {
  id: string;
  level: 'critical' | 'warning' | 'info';
  title: string;
  status: 'active' | 'resolved';
  startedAt: string;
  resolvedAt?: string;
  duration: string;
  value: string;
}

const AlarmTab: React.FC<{ hostId: string }> = ({ hostId }) => {
  const [levelFilter, setLevelFilter] = useState<string>('all');
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<AlarmHistoryItem[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await hostApi.getHostAlarms(hostId);
      setData(res.data || []);
    } finally {
      setLoading(false);
    }
  }, [hostId]);

  useEffect(() => {
    if (hostId) fetchData();
  }, [hostId, fetchData]);

  const columns: ColumnsType<AlarmHistoryItem> = [
    {
      title: '告警级别',
      dataIndex: 'level',
      key: 'level',
      width: 100,
      render: (level: AlarmHistoryItem['level']) => {
        const config = {
          critical: { color: 'error', text: '严重' },
          warning: { color: 'warning', text: '警告' },
          info: { color: 'processing', text: '信息' },
        };
        const { color, text } = config[level] || { color: 'default', text: level };
        return <Tag color={color}>{text}</Tag>;
      },
    },
    { title: '告警名称', dataIndex: 'title', key: 'title', width: 200 },
    { title: '告警值', dataIndex: 'value', key: 'value', width: 120 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status) => (
        <span className={status === 'active' || status === 'firing' ? 'text-red-500 font-medium' : 'text-gray-400'}>
          {status === 'active' || status === 'firing' ? '进行中' : '已恢复'}
        </span>
      ),
    },
    { title: '发生时间', dataIndex: 'startedAt', key: 'startedAt', width: 160, render: (t) => new Date(t).toLocaleString() },
    { title: '持续时间', dataIndex: 'duration', key: 'duration', width: 100 },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: () => <Button type="link" size="small">详情</Button>,
    },
  ];

  return (
    <Card className="h-full border-none shadow-sm mt-4">
      <div className="flex flex-col gap-4 mb-6">
        <div className="flex justify-between items-center">
          <h3 className="text-base font-medium m-0">历史告警记录</h3>
          <Space>
            <Button icon={<FilterOutlined />}>导出记录</Button>
          </Space>
        </div>
        
        <div className="flex gap-4 items-center bg-gray-50/50 p-3 rounded-lg">
          <Space size="middle">
            <span className="text-xs text-gray-500">告警级别:</span>
            <Select 
              defaultValue="all" 
              style={{ width: 120 }} 
              size="small"
              onChange={setLevelFilter}
              options={[
                { value: 'all', label: '全部级别' },
                { value: 'critical', label: '严重' },
                { value: 'warning', label: '警告' },
                { value: 'info', label: '信息' },
              ]} 
            />
          </Space>
          <Space size="middle">
            <span className="text-xs text-gray-500">时间范围:</span>
            <RangePicker size="small" />
          </Space>
          <Input
            placeholder="搜索告警内容..."
            prefix={<SearchOutlined className="text-gray-400" />}
            size="small"
            style={{ width: 250 }}
            allowClear
          />
        </div>
      </div>

      <Table
        columns={columns}
        dataSource={levelFilter === 'all' ? data : data.filter(a => a.level === levelFilter)}
        rowKey="id"
        size="small"
        pagination={{ pageSize: 15 }}
        loading={loading}
      />
    </Card>
  );
};

export default AlarmTab;
