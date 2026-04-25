import React from 'react';
import { Dropdown, Modal, Space, Table, Tag, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DownOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { hostApi } from '../../../../api/modules/hosts';
import type { CredentialListRowViewModel } from '../viewModels';

interface Props {
  data: CredentialListRowViewModel[];
  loading: boolean;
  selectedId?: string;
  onRowClick: (record: CredentialListRowViewModel) => void;
  onRefresh: () => void;
}

const toneClassMap = {
  success: 'bg-[#edf9f1] text-[#16a34a]',
  warning: 'bg-[#fff4e8] text-[#fa8c16]',
  danger: 'bg-[#fff1f0] text-[#ff4d4f]',
  default: 'bg-[#f3f4f6] text-[#6b7280]',
} as const;

export const CredentialTable: React.FC<Props> = ({
  data,
  loading,
  selectedId,
  onRowClick,
  onRefresh,
}) => {
  const handleComingSoon = () => message.info('该操作将在后续联调中接入');

  const handleDelete = (record: CredentialListRowViewModel) => {
    const realId = record.id.replace(/^(key|tpl)-/, '');
    Modal.confirm({
      title: '确认删除凭证',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除凭证 "${record.name}" 吗？此操作不可撤销。`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          const res = record.id.startsWith('key-')
            ? await hostApi.deleteSSHKey(realId)
            : await hostApi.deleteCredentialTemplate(realId);
          if (res.success) {
            message.success('凭证已删除');
            onRefresh();
          }
        } catch (err: any) {
          message.error(err.message || '删除失败');
        }
      },
    });
  };

  const columns: ColumnsType<CredentialListRowViewModel> = [
    {
      title: '凭证名称',
      dataIndex: 'name',
      key: 'name',
      width: 220,
      render: (_, record) => (
        <button type="button" className="text-left" onClick={() => onRowClick(record)}>
          <div className="text-[14px] font-medium text-[#2f6bff]">{record.name}</div>
          <div className="mt-1 text-[12px] text-[#6b7280]">{record.description || '-'}</div>
        </button>
      ),
    },
    {
      title: '类型',
      dataIndex: 'typeLabel',
      key: 'typeLabel',
      width: 100,
      render: (_, record) => <Tag className="!rounded-md !border-[#dce7fb] !bg-[#f5f9ff] !px-2 !py-0.5 !text-[#3b82f6]">{record.typeLabel}</Tag>,
    },
    {
      title: '认证方式',
      dataIndex: 'authMethodLabel',
      key: 'authMethodLabel',
      width: 112,
    },
    {
      title: '关联主机数',
      dataIndex: 'hostCount',
      key: 'hostCount',
      width: 96,
    },
    {
      title: '标签',
      dataIndex: 'tags',
      key: 'tags',
      width: 120,
      render: (_, record) => (
        <Space size={6} wrap>
          {record.tags?.map((tag) => (
            <Tag key={tag} className="!rounded-md !border-[#d8e8ff] !bg-[#f7fbff] !px-2 !py-0 !text-[#4f83ff]">
              {tag}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'statusLabel',
      key: 'statusLabel',
      width: 112,
      render: (_, record) => (
        <span className={`inline-flex items-center gap-1 rounded-full px-2 py-1 text-[12px] ${toneClassMap[record.statusTone]}`}>
          <span className="h-1.5 w-1.5 rounded-full bg-current" />
          {record.statusLabel}
        </span>
      ),
    },
    {
      title: '过期时间',
      dataIndex: 'expireAt',
      key: 'expireAt',
      width: 132,
      render: (_, record) => (
        <div className={record.statusTone === 'danger' ? 'text-[#ff4d4f]' : record.statusTone === 'warning' ? 'text-[#fa8c16]' : 'text-[#111827]'}>
          <div>{record.expireAt || '-'}</div>
          {record.expireHint ? <div className="mt-1 text-[12px]">{record.expireHint}</div> : null}
        </div>
      ),
    },
    {
      title: '更新时间',
      key: 'updatedAt',
      width: 128,
      render: (_, record) => (
        <div>
          <div>{record.updatedAt || '-'}</div>
          <div className="mt-1 text-[12px] text-[#6b7280]">{record.updatedBy || '-'}</div>
        </div>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 110,
      render: (_, record) => (
        <Space size={12}>
          <button type="button" className="text-[#2f6bff]" onClick={() => onRowClick(record)}>
            编辑
          </button>
          <Dropdown
            menu={{
              items: [
                { key: 'detail', label: '查看详情', onClick: () => onRowClick(record) },
                { key: 'copy', label: '复制配置', onClick: handleComingSoon },
                { key: 'rotate', label: '轮换密钥', onClick: handleComingSoon },
                { key: 'usage', label: '查看使用记录', onClick: handleComingSoon },
                { key: 'relation', label: '关联主机', onClick: handleComingSoon },
                { type: 'divider' },
                { key: 'delete', label: '删除', danger: true, onClick: () => handleDelete(record) },
              ],
            }}
          >
            <button type="button" className="text-[#2f6bff]">
              更多 <DownOutlined />
            </button>
          </Dropdown>
        </Space>
      ),
    },
  ];

  return (
    <Table
      columns={columns}
      dataSource={data}
      loading={loading}
      rowKey="id"
      scroll={{ x: 1080 }}
      rowSelection={{
        type: 'checkbox',
        selectedRowKeys: selectedId ? [selectedId] : [],
      }}
      onRow={(record) => ({
        onClick: () => onRowClick(record),
        className: `cursor-pointer ${selectedId === record.id ? 'bg-[#f7fbff]' : ''}`,
      })}
      pagination={{
        total: data.length,
        pageSize: 10,
        showSizeChanger: true,
        showQuickJumper: true,
        showTotal: (total) => `共 ${total} 条`,
      }}
      className="[&_.ant-table]:!bg-transparent [&_.ant-table-container]:!border [&_.ant-table-container]:!border-[#edf2f7] [&_.ant-table-container]:!rounded-xl [&_.ant-table-thead>tr>th]:!bg-[#fbfcfe] [&_.ant-table-thead>tr>th]:!border-b-[#e6edf5] [&_.ant-table-thead>tr>th]:!text-[12px] [&_.ant-table-thead>tr>th]:!font-medium [&_.ant-table-thead>tr>th]:!text-[#6b7280] [&_.ant-table-tbody>tr>td]:!border-b-[#eef2f7] [&_.ant-table-tbody>tr>td]:!py-4 [&_.ant-pagination]:!mt-4 [&_.ant-pagination]:!px-1 [&_.ant-pagination-item]:!border-none [&_.ant-pagination-item]:!rounded-md [&_.ant-pagination-item-active]:!bg-[#eef4ff] [&_.ant-pagination-item-active]:!text-[#2f6bff]"
    />
  );
};
