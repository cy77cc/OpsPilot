import React, { useEffect, useState } from 'react';
import { Button, Space, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { hostApi } from '../../../../api/modules/hosts';
import { credentialPermissionFixtures } from '../viewModels';

type PermissionRow = (typeof credentialPermissionFixtures)[number];

export const CredentialPermissionTable: React.FC = () => {
  const [data, setData] = useState<PermissionRow[]>(credentialPermissionFixtures);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    hostApi
      .getCredentialPermissions()
      .then((res) => {
        if (res.success && res.data.list.length > 0) {
          setData(res.data.list);
        }
      })
      .finally(() => setLoading(false));
  }, []);

  const columns: ColumnsType<PermissionRow> = [
    { title: '凭证名称', dataIndex: 'credentialName', key: 'credentialName' },
    { title: '用户 / 角色', dataIndex: 'targetUserOrRole', key: 'targetUserOrRole' },
    {
      title: '权限项',
      dataIndex: 'permissions',
      key: 'permissions',
      render: (values: string[]) => (
        <Space size={6} wrap>
          {values.map((value) => (
            <Tag key={value} className="!rounded-md !border-[#dce7fb] !bg-[#f5f9ff] !text-[#3b82f6]">
              {value}
            </Tag>
          ))}
        </Space>
      ),
    },
    { title: '授权范围', dataIndex: 'scope', key: 'scope' },
    { title: '生效时间', dataIndex: 'effectiveTime', key: 'effectiveTime' },
    { title: '到期时间', dataIndex: 'expireTime', key: 'expireTime' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (value) =>
        value === 'active' ? (
          <Tag className="!rounded-full !border-none !bg-[#edf9f1] !text-[#16a34a]">生效中</Tag>
        ) : (
          <Tag className="!rounded-full !border-none !bg-[#fff1f0] !text-[#ff4d4f]">已过期</Tag>
        ),
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space size={12}>
          <button type="button" className="text-[#2f6bff]">编辑</button>
          <button type="button" className="text-[#ff4d4f]">撤销</button>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <div className="text-[20px] font-semibold text-[#111827]">权限管理</div>
          <div className="mt-1 text-[13px] text-[#6b7280]">控制谁可以查看、编辑、使用、轮换或删除凭证。</div>
        </div>
        <Space>
          <Button>批量授权</Button>
          <Button type="primary">新增授权</Button>
        </Space>
      </div>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={data}
        columns={columns}
        className="[&_.ant-table-thead>tr>th]:!bg-[#f8fafc] [&_.ant-table-thead>tr>th]:!text-[#6b7280]"
      />
    </div>
  );
};
