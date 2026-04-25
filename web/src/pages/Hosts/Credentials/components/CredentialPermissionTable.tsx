import React, { useEffect, useState } from 'react';
import { Table, Tag, Button, Space } from 'antd';
import { hostApi } from '../../../../api/modules/hosts';
import type { CredentialPermissionItem } from '../../../../api/modules/hosts';
import type { ColumnsType } from 'antd/es/table';

export const CredentialPermissionTable: React.FC = () => {
  const [data, setData] = useState<CredentialPermissionItem[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    hostApi.getCredentialPermissions().then(res => {
      if (res.success) setData(res.data.list);
    }).finally(() => setLoading(false));
  }, []);

  const columns: ColumnsType<CredentialPermissionItem> = [
    { title: '凭证名称', dataIndex: 'credentialName', key: 'credentialName' },
    { title: '用户 / 角色', dataIndex: 'targetUserOrRole', key: 'targetUserOrRole' },
    { 
      title: '权限项', 
      dataIndex: 'permissions', 
      key: 'permissions',
      render: (perms: string[]) => (
        <>
          {perms.map(p => <Tag key={p}>{p}</Tag>)}
        </>
      )
    },
    { title: '授权范围', dataIndex: 'scope', key: 'scope' },
    { title: '生效时间', dataIndex: 'effectiveTime', key: 'effectiveTime' },
    { title: '到期时间', dataIndex: 'expireTime', key: 'expireTime' },
    { 
      title: '状态', 
      dataIndex: 'status', 
      key: 'status',
      render: (status) => status === 'active' ? <Tag color="green">生效中</Tag> : <Tag color="red">已过期</Tag>
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space size="middle">
          <a>编辑</a>
          <a className="text-red-500">撤销</a>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-4">
        <Button type="primary">新增授权</Button>
      </div>
      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} />
    </div>
  );
};