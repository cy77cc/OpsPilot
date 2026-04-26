import React, { useState, useEffect, useCallback } from 'react';
import { Card, Table, Tag, Input, Space, Button, message, Popconfirm } from 'antd';
import { SearchOutlined, PlayCircleOutlined, PauseCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { hostApi } from '../../../../api/modules/hosts';

interface ServiceItem {
  name: string;
  status: 'active' | 'inactive' | 'failed';
  startup: 'enabled' | 'disabled' | 'static';
  description: string;
}

const ServiceTab: React.FC<{ hostId: string }> = ({ hostId }) => {
  const [searchText, setSearchText] = useState('');
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<ServiceItem[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await hostApi.getHostServices(hostId);
      setData(res.data || []);
    } finally {
      setLoading(false);
    }
  }, [hostId]);

  useEffect(() => {
    if (hostId) fetchData();
  }, [hostId, fetchData]);

  const handleAction = async (name: string, action: string) => {
    try {
      await hostApi.serviceAction(hostId, name, action);
      message.success(`已发送 ${action} 指令到服务 ${name}`);
      fetchData();
    } catch (err) {
      message.error('操作失败');
    }
  };

  const columns: ColumnsType<ServiceItem> = [
    { title: '服务名称', dataIndex: 'name', key: 'name', width: 250, sorter: (a, b) => a.name.localeCompare(b.name) },
    {
      title: '运行状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: ServiceItem['status']) => {
        const config = {
          active: { color: 'success', text: '运行中' },
          inactive: { color: 'default', text: '已停止' },
          failed: { color: 'error', text: '异常' },
        };
        const { color, text } = config[status] || { color: 'default', text: status };
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: '自启动',
      dataIndex: 'startup',
      key: 'startup',
      width: 100,
      render: (startup) => <span className="text-gray-600 text-xs uppercase">{startup}</span>,
    },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, record) => (
        <Space size="middle">
          {record.status !== 'active' ? (
            <Button 
              type="link" 
              size="small" 
              icon={<PlayCircleOutlined />} 
              onClick={() => handleAction(record.name, 'start')}
            >
              启动
            </Button>
          ) : (
            <Popconfirm title="确认停止服务？" onConfirm={() => handleAction(record.name, 'stop')}>
              <Button type="link" danger size="small" icon={<PauseCircleOutlined />}>停止</Button>
            </Popconfirm>
          )}
          <Button 
            type="link" 
            size="small" 
            icon={<ReloadOutlined />} 
            onClick={() => handleAction(record.name, 'restart')}
          >
            重启
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Card className="h-full border-none shadow-sm mt-4">
      <div className="flex justify-between items-center mb-4">
        <Input
          placeholder="搜索服务名称或描述..."
          prefix={<SearchOutlined className="text-gray-400" />}
          value={searchText}
          onChange={e => setSearchText(e.target.value)}
          style={{ width: 350 }}
          allowClear
        />
        <Button icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>重新加载列表</Button>
      </div>

      <Table
        columns={columns}
        dataSource={data.filter(s => 
          s.name.toLowerCase().includes(searchText.toLowerCase()) || 
          s.description.toLowerCase().includes(searchText.toLowerCase())
        )}
        rowKey="name"
        size="small"
        pagination={{ pageSize: 15 }}
        loading={loading}
      />
    </Card>
  );
};

export default ServiceTab;
