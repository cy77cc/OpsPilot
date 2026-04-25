import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Tag } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { hostApi } from '../../../../api/modules/hosts';
import type { CredentialTemplate } from '../../../../api/modules/hosts';
import type { ColumnsType } from 'antd/es/table';

export const PresetAuthTable: React.FC = () => {
  const [data, setData] = useState<CredentialTemplate[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    hostApi.listCredentialTemplates().then(res => {
      if (res.success) setData(res.data);
    }).finally(() => setLoading(false));
  }, []);

  const columns: ColumnsType<CredentialTemplate> = [
    { title: '模板名称', dataIndex: 'name', key: 'name', render: text => <a className="text-blue-500 hover:text-blue-700">{text}</a> },
    { title: '认证方式', dataIndex: 'authType', key: 'authType' },
    { title: '默认用户名', dataIndex: 'sshUser', key: 'sshUser' },
    { title: '端口', dataIndex: 'port', key: 'port' },
    { title: '状态', key: 'status', render: () => <Tag color="green">启用</Tag> },
    { title: '更新时间', dataIndex: 'updatedAt', key: 'updatedAt' },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space size="middle">
          <a>编辑</a>
          <a>复制</a>
          <a className="text-red-500">删除</a>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-4">
        <Button type="primary" icon={<PlusOutlined />}>新建模板</Button>
      </div>
      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} />
    </div>
  );
};