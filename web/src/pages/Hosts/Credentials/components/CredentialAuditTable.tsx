import React, { useEffect, useState } from 'react';
import { Table, Tag } from 'antd';
import { hostApi } from '../../../../api/modules/hosts';
import type { CredentialUsageRecord } from '../../../../api/modules/hosts';
import type { ColumnsType } from 'antd/es/table';

export const CredentialAuditTable: React.FC = () => {
  const [data, setData] = useState<CredentialUsageRecord[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    hostApi.getCredentialUsageRecords().then(res => {
      if (res.success) setData(res.data.list);
    }).finally(() => setLoading(false));
  }, []);

  const columns: ColumnsType<CredentialUsageRecord> = [
    { title: '时间', dataIndex: 'time', key: 'time' },
    { title: '凭证名称', dataIndex: 'credentialName', key: 'credentialName' },
    { title: '使用人', dataIndex: 'operator', key: 'operator' },
    { title: '目标对象', dataIndex: 'target', key: 'target' },
    { title: '使用方式', dataIndex: 'method', key: 'method' },
    { 
      title: '结果', 
      dataIndex: 'result', 
      key: 'result',
      render: (text) => text === 'success' ? <Tag color="green">成功</Tag> : <Tag color="red">失败</Tag>
    },
    { title: '来源 IP', dataIndex: 'sourceIp', key: 'sourceIp' },
    { title: '备注', dataIndex: 'remark', key: 'remark' },
  ];

  return (
    <Table columns={columns} dataSource={data} rowKey="id" loading={loading} />
  );
};