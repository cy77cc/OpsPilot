import React from 'react';
import { Button, Input, Select, Space, Dropdown } from 'antd';
import { PlusOutlined, ImportOutlined, ReloadOutlined, DownOutlined, SearchOutlined } from '@ant-design/icons';

export const CredentialToolbar: React.FC = () => {
  return (
    <div className="flex justify-between items-center mb-4">
      <Space>
        <Button type="primary" icon={<PlusOutlined />}>创建凭证</Button>
        <Button icon={<ImportOutlined />}>导入</Button>
        <Dropdown menu={{ items: [{ key: 'export', label: '批量导出' }, { key: 'delete', label: '批量删除' }] }}>
          <Button>更多操作 <DownOutlined /></Button>
        </Dropdown>
      </Space>
      <Space size={8}>
        <Select placeholder="凭证类型" style={{ width: 120 }} allowClear>
          <Select.Option value="ssh_key">SSH 密钥</Select.Option>
          <Select.Option value="password">密码</Select.Option>
          <Select.Option value="token">Token</Select.Option>
          <Select.Option value="certificate">证书</Select.Option>
        </Select>
        <Select placeholder="状态" style={{ width: 100 }} allowClear>
          <Select.Option value="available">可用</Select.Option>
          <Select.Option value="expiring_soon">即将过期</Select.Option>
          <Select.Option value="expired">已过期</Select.Option>
        </Select>
        <Input placeholder="搜索凭证名称或备注" style={{ width: 200 }} />
        <Button icon={<SearchOutlined />} />
        <Button icon={<ReloadOutlined />} />
      </Space>
    </div>
  );
};
