import React, { useEffect, useState } from 'react';
import { DatePicker, Select, Space, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { hostApi } from '../../../../api/modules/hosts';
import { credentialUsageFixtures } from '../viewModels';

type AuditRow = (typeof credentialUsageFixtures)[number];

export const CredentialAuditTable: React.FC = () => {
  const [data, setData] = useState<AuditRow[]>(credentialUsageFixtures);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    hostApi
      .getCredentialUsageRecords()
      .then((res) => {
        if (res.success && res.data.list.length > 0) {
          setData(res.data.list);
        }
      })
      .finally(() => setLoading(false));
  }, []);

  const columns: ColumnsType<AuditRow> = [
    { title: '时间', dataIndex: 'time', key: 'time' },
    { title: '凭证名称', dataIndex: 'credentialName', key: 'credentialName' },
    { title: '使用人', dataIndex: 'operator', key: 'operator' },
    { title: '目标对象', dataIndex: 'target', key: 'target' },
    { title: '使用方式', dataIndex: 'method', key: 'method' },
    {
      title: '结果',
      dataIndex: 'result',
      key: 'result',
      render: (value) =>
        value === 'success' ? (
          <Tag className="!rounded-full !border-none !bg-[#edf9f1] !text-[#16a34a]">成功</Tag>
        ) : (
          <Tag className="!rounded-full !border-none !bg-[#fff1f0] !text-[#ff4d4f]">失败</Tag>
        ),
    },
    { title: '来源 IP', dataIndex: 'sourceIp', key: 'sourceIp' },
    { title: '备注', dataIndex: 'remark', key: 'remark' },
  ];

  return (
    <div className="space-y-4">
      <div>
        <div className="text-[20px] font-semibold text-[#111827]">使用记录</div>
        <div className="mt-1 text-[13px] text-[#6b7280]">审计凭证被谁在何时用于哪些对象。</div>
      </div>
      <Space wrap>
        <DatePicker.RangePicker />
        <Select placeholder="凭证" className="!w-[160px]" />
        <Select placeholder="用户" className="!w-[140px]" />
        <Select placeholder="结果" className="!w-[120px]" />
      </Space>
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
