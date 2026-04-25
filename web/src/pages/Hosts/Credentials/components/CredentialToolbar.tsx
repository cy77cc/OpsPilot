import React from 'react';
import { Button, Input, Select, Space, Dropdown, message } from 'antd';
import { PlusOutlined, ImportOutlined, ReloadOutlined, DownOutlined } from '@ant-design/icons';

interface Props {
  onRefresh: () => void;
  onCreate: () => void;
  onSearch: (value: string) => void;
  onTypeChange: (value: string) => void;
  onStatusChange: (value: string) => void;
}

export const CredentialToolbar: React.FC<Props> = ({ onRefresh, onCreate, onSearch, onTypeChange, onStatusChange }) => {
  const handleComingSoon = () => message.info('功能开发中');

  return (
    <div className="flex justify-between items-center mb-4">
      <Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={onCreate}>创建凭证</Button>
        <Button icon={<ImportOutlined />} onClick={handleComingSoon}>导入</Button>
        <Dropdown menu={{ 
          items: [
            { key: 'export', label: '批量导出', onClick: handleComingSoon }, 
            { key: 'delete', label: '批量删除', onClick: handleComingSoon }
          ] 
        }}>
          <Button>更多操作 <DownOutlined /></Button>
        </Dropdown>
      </Space>
      <Space size={8}>
        <Select placeholder="凭证类型" style={{ width: 120 }} allowClear onChange={onTypeChange}>
          <Select.Option value="ssh_key">SSH 密钥</Select.Option>
          <Select.Option value="password">密码</Select.Option>
          <Select.Option value="token">Token</Select.Option>
          <Select.Option value="certificate">证书</Select.Option>
        </Select>
        <Select placeholder="状态" style={{ width: 100 }} allowClear onChange={onStatusChange}>
          <Select.Option value="available">可用</Select.Option>
          <Select.Option value="expiring_soon">即将过期</Select.Option>
          <Select.Option value="expired">已过期</Select.Option>
        </Select>
        <Input.Search 
          placeholder="搜索凭证名称或备注" 
          style={{ width: 200 }} 
          onSearch={onSearch}
          allowClear
        />
        <Button icon={<ReloadOutlined />} onClick={onRefresh} />
      </Space>
    </div>
  );
};
