import React from 'react';
import { Button, Dropdown, Input, Select, Space, message } from 'antd';
import {
  DownOutlined,
  FilterOutlined,
  ImportOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';

interface Props {
  onRefresh: () => void;
  onCreate: () => void;
  onSearch: (value: string) => void;
  onTypeChange: (value: string | undefined) => void;
  onStatusChange: (value: string | undefined) => void;
}

export const CredentialToolbar: React.FC<Props> = ({
  onRefresh,
  onCreate,
  onSearch,
  onTypeChange,
  onStatusChange,
}) => {
  const handleComingSoon = () => message.info('该操作将在后续联调中接入');

  return (
    <div className="mb-4 flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
      <Space size={10} wrap>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={onCreate}
          className="!h-9 !rounded-lg !bg-[#2f6bff] !px-4 !shadow-none"
        >
          创建凭证
        </Button>
        <Button icon={<ImportOutlined />} className="!h-9 !rounded-lg !border-[#d8e1ee]" onClick={handleComingSoon}>
          导入
        </Button>
        <Dropdown
          menu={{
            items: [
              { key: 'export', label: '批量导出元数据', onClick: handleComingSoon },
              { key: 'tag', label: '批量打标签', onClick: handleComingSoon },
              { key: 'rotate', label: '批量轮换', onClick: handleComingSoon },
              { key: 'delete', label: '批量删除', danger: true, onClick: handleComingSoon },
            ],
          }}
        >
          <Button className="!h-9 !rounded-lg !border-[#d8e1ee]">
            更多操作 <DownOutlined />
          </Button>
        </Dropdown>
      </Space>

      <Space size={10} wrap className="justify-end">
        <Input.Search
          placeholder="搜索凭证名称、类型、备注等"
          allowClear
          onSearch={onSearch}
          className="w-full xl:!w-[280px]"
        />
        <Select
          placeholder="凭证类型"
          allowClear
          onChange={onTypeChange}
          className="!w-[124px]"
          options={[
            { label: 'SSH 密钥', value: 'ssh_key' },
            { label: '密码', value: 'password' },
            { label: 'Token', value: 'token' },
            { label: '证书', value: 'certificate' },
          ]}
        />
        <Select
          placeholder="状态"
          allowClear
          onChange={onStatusChange}
          className="!w-[112px]"
          options={[
            { label: '可用', value: 'available' },
            { label: '即将过期', value: 'expiring_soon' },
            { label: '已过期', value: 'expired' },
            { label: '禁用', value: 'disabled' },
          ]}
        />
        <Button icon={<ReloadOutlined />} className="!h-9 !w-9 !rounded-lg !border-[#d8e1ee] !p-0" onClick={onRefresh} />
        <Button
          icon={<FilterOutlined />}
          className="!h-9 !w-9 !rounded-lg !border-[#d8e1ee] !p-0"
          onClick={handleComingSoon}
        />
      </Space>
    </div>
  );
};
