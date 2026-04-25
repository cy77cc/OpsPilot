import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Tag, Modal, message } from 'antd';
import { PlusOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { hostApi } from '../../../../api/modules/hosts';
import type { CredentialTemplate } from '../../../../api/modules/hosts';
import type { ColumnsType } from 'antd/es/table';
import { CreateTemplateModal } from './CreateTemplateModal';

export const PresetAuthTable: React.FC = () => {
  const [data, setData] = useState<CredentialTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const handleComingSoon = () => message.info('功能开发中');

  const fetchData = () => {
    setLoading(true);
    hostApi.listCredentialTemplates().then(res => {
      if (res.success) setData(res.data);
    }).finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleDelete = (record: CredentialTemplate) => {
    Modal.confirm({
      title: '确认删除模板',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除认证模板 "${record.name}" 吗？`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          const res = await hostApi.deleteCredentialTemplate(record.id);
          if (res.success) {
            message.success('模板已删除');
            fetchData();
          }
        } catch (err: any) {
          message.error(err.message || '删除失败');
        }
      },
    });
  };

  const columns: ColumnsType<CredentialTemplate> = [
    { title: '模板名称', dataIndex: 'name', key: 'name', render: text => <a className="text-blue-500 hover:text-blue-700 font-medium">{text}</a> },
    { title: '认证方式', dataIndex: 'authType', key: 'authType', render: text => text === 'key' ? <Tag color="blue">SSH 密钥</Tag> : <Tag color="geekblue">用户名密码</Tag> },
    { title: '默认用户名', dataIndex: 'sshUser', key: 'sshUser' },
    { title: '端口', dataIndex: 'port', key: 'port' },
    { title: '状态', key: 'status', render: () => <Tag color="success" variant="filled">启用</Tag> },
    { title: '更新时间', dataIndex: 'updatedAt', key: 'updatedAt' },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="middle">
          <a className="text-gray-600 hover:text-blue-500" onClick={handleComingSoon}>编辑</a>
          <a className="text-gray-600 hover:text-blue-500" onClick={handleComingSoon}>复制</a>
          <a className="text-gray-600 hover:text-red-500" onClick={() => handleDelete(record)}>删除</a>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-4">
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setIsModalOpen(true)}>新建模板</Button>
      </div>
      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small" />
      
      <CreateTemplateModal 
        open={isModalOpen} 
        onCancel={() => setIsModalOpen(false)} 
        onSuccess={() => {
          setIsModalOpen(false);
          fetchData();
        }}
      />
    </div>
  );
};
