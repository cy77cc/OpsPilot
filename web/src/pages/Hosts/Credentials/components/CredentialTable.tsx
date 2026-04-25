import React from 'react';
import { Table, Tag, Space, Dropdown, Button } from 'antd';
import { DownOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { CredentialItem } from '../../../../api/modules/hosts';

interface Props {
  data: CredentialItem[];
  loading: boolean;
  onRowClick: (record: CredentialItem) => void;
}

export const CredentialTable: React.FC<Props> = ({ data, loading, onRowClick }) => {
  const columns: ColumnsType<CredentialItem> = [
    {
      title: '凭证名称',
      key: 'name',
      render: (_, record) => (
        <div 
          className="cursor-pointer"
          onClick={() => onRowClick(record)}
        >
          <div className="text-blue-500 hover:text-blue-700">{record.name}</div>
          <div className="text-xs text-gray-400">{record.description}</div>
        </div>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (text) => <Tag color={text === 'ssh_key' ? 'blue' : 'geekblue'} bordered={false}>{text}</Tag>,
    },
    {
      title: '认证方式',
      dataIndex: 'authMethod',
      key: 'authMethod',
    },
    {
      title: '关联主机数',
      dataIndex: 'hostCount',
      key: 'hostCount',
    },
    {
      title: '标签',
      key: 'tags',
      render: (_, record) => (
        <Space size={4} wrap>
          {record.tags?.map(tag => (
            <Tag key={tag} bordered={false}>{tag}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '状态',
      key: 'status',
      render: (_, record) => {
        let color = 'success';
        const status = record.status || 'available';
        if (status === 'expiring_soon') color = 'warning';
        if (status === 'expired') color = 'error';
        if (status === 'disabled') color = 'default';
        
        let text = '可用';
        if (status === 'expiring_soon') text = '即将过期';
        if (status === 'expired') text = '已过期';
        if (status === 'disabled') text = '禁用';
        
        return <Tag color={color} bordered={false}>{text}</Tag>;
      },
    },
    {
      title: '过期时间',
      dataIndex: 'expireAt',
      key: 'expireAt',
      render: (text) => text || '-',
    },
    {
      title: '更新时间',
      key: 'updatedAt',
      render: (_, record) => (
        <div>
          <div>{record.updatedAt}</div>
          <div className="text-xs text-gray-400">{record.updatedBy}</div>
        </div>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="middle">
          <a className="text-gray-600 hover:text-blue-500" onClick={(e) => { e.stopPropagation(); onRowClick(record); }}>编辑</a>
          <Dropdown menu={{ items: [{ key: '1', label: '查看详情' }, { key: '2', label: '复制配置' }, { key: '3', label: '轮换密钥' }, { key: '4', label: '删除', danger: true }] }}>
            <a className="text-gray-600 hover:text-blue-500" onClick={e => e.preventDefault()}>
              更多 <DownOutlined />
            </a>
          </Dropdown>
        </Space>
      ),
    },
  ];

  return (
    <Table
      columns={columns}
      dataSource={data}
      rowKey="id"
      loading={loading}
      rowSelection={{ type: 'checkbox' }}
      pagination={{ showSizeChanger: true, showQuickJumper: true, showTotal: (total) => `共 ${total} 条` }}
    />
  );
};