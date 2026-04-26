import React, { useState, useEffect, useCallback } from 'react';
import { Card, Table, Tag, Input, Space, Button } from 'antd';
import { SearchOutlined, ReloadOutlined, DownloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { hostApi } from '../../../../api/modules/hosts';

interface PackageItem {
  name: string;
  version: string;
  arch: string;
  status: 'installed' | 'upgradable';
  description: string;
}

const PackageTab: React.FC<{ hostId: string }> = ({ hostId }) => {
  const [searchText, setSearchText] = useState('');
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<PackageItem[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await hostApi.getHostPackages(hostId);
      setData(res.data || []);
    } finally {
      setLoading(false);
    }
  }, [hostId]);

  useEffect(() => {
    if (hostId) fetchData();
  }, [hostId, fetchData]);

  const columns: ColumnsType<PackageItem> = [
    { title: '软件包名称', dataIndex: 'name', key: 'name', width: 200, sorter: (a, b) => a.name.localeCompare(b.name) },
    { title: '版本', dataIndex: 'version', key: 'version', width: 220 },
    { title: '架构', dataIndex: 'arch', key: 'arch', width: 100 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status) => (
        <Tag color={status === 'installed' ? 'success' : 'processing'}>
          {status === 'installed' ? '已安装' : '可升级'}
        </Tag>
      ),
    },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Button type="link" size="small" disabled={record.status === 'installed'} icon={<DownloadOutlined />}>
          升级
        </Button>
      ),
    },
  ];

  return (
    <Card className="h-full border-none shadow-sm mt-4">
      <div className="flex justify-between items-center mb-4">
        <Space>
          <Input
            placeholder="搜索软件包..."
            prefix={<SearchOutlined className="text-gray-400" />}
            value={searchText}
            onChange={e => setSearchText(e.target.value)}
            style={{ width: 300 }}
            allowClear
          />
          <Button icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>刷新列表</Button>
        </Space>
        <div className="text-gray-400 text-xs">
          当前共加载 {data.length} 个软件包
        </div>
      </div>

      <Table
        columns={columns}
        dataSource={data.filter(p => 
          p.name.toLowerCase().includes(searchText.toLowerCase()) || 
          p.description.toLowerCase().includes(searchText.toLowerCase())
        )}
        rowKey="name"
        size="small"
        pagination={{ pageSize: 15, showTotal: (total) => `共 ${total} 条` }}
        loading={loading}
      />
    </Card>
  );
};

export default PackageTab;
