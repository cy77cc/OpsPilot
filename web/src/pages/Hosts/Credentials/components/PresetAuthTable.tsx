import React, { useEffect, useState } from 'react';
import { Button, Space, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined } from '@ant-design/icons';
import { hostApi } from '../../../../api/modules/hosts';
import { buildTemplateRows } from '../viewModels';
import { CreateTemplateModal } from './CreateTemplateModal';

interface TemplateRow {
  id: string;
  name: string;
  authType: 'password' | 'key';
  sshUser: string;
  port: number;
  relatedCredential: string;
  scope: string;
  statusLabel: string;
  updatedAt: string;
}

export const PresetAuthTable: React.FC = () => {
  const [data, setData] = useState<TemplateRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);

  const fetchData = () => {
    setLoading(true);
    hostApi
      .listCredentialTemplates()
      .then((res) => {
        if (res.success) {
          setData(buildTemplateRows(res.data));
        }
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchData();
  }, []);

  const columns: ColumnsType<TemplateRow> = [
    { title: '模板名称', dataIndex: 'name', key: 'name' },
    {
      title: '认证方式',
      dataIndex: 'authType',
      key: 'authType',
      render: (value) =>
        value === 'key' ? (
          <Tag className="!rounded-md !border-[#dce7fb] !bg-[#f5f9ff] !text-[#3b82f6]">SSH Key</Tag>
        ) : (
          <Tag className="!rounded-md !border-[#dce7fb] !bg-[#f8fafc] !text-[#64748b]">用户名密码</Tag>
        ),
    },
    { title: '默认用户名', dataIndex: 'sshUser', key: 'sshUser' },
    { title: '端口', dataIndex: 'port', key: 'port' },
    { title: '关联凭证', dataIndex: 'relatedCredential', key: 'relatedCredential' },
    { title: '适用范围', dataIndex: 'scope', key: 'scope' },
    {
      title: '状态',
      dataIndex: 'statusLabel',
      key: 'statusLabel',
      render: (value) => <Tag className="!rounded-full !border-none !bg-[#edf9f1] !text-[#16a34a]">{value}</Tag>,
    },
    { title: '更新时间', dataIndex: 'updatedAt', key: 'updatedAt' },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space size={12}>
          <button type="button" className="text-[#2f6bff]">编辑</button>
          <button type="button" className="text-[#2f6bff]">复制</button>
          <button type="button" className="text-[#ff4d4f]">删除</button>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <div className="text-[20px] font-semibold text-[#111827]">预设认证方式</div>
          <div className="mt-1 text-[13px] text-[#6b7280]">管理认证模板，降低重复配置成本。</div>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          新建模板
        </Button>
      </div>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={data}
        columns={columns}
        className="[&_.ant-table-thead>tr>th]:!bg-[#f8fafc] [&_.ant-table-thead>tr>th]:!text-[#6b7280]"
      />
      <CreateTemplateModal
        open={open}
        onCancel={() => setOpen(false)}
        onSuccess={() => {
          setOpen(false);
          fetchData();
        }}
      />
    </div>
  );
};
