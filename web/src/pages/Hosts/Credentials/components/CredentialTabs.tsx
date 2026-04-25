import React from 'react';
import { Tabs } from 'antd';

interface Props {
  activeKey: string;
  onChange: (key: string) => void;
}

export const CredentialTabs: React.FC<Props> = ({ activeKey, onChange }) => {
  const items = [
    { key: 'key-management', label: '密钥管理' },
    { key: 'preset-auth', label: '预设认证方式' },
    { key: 'usage-audit', label: '使用记录' },
    { key: 'permission', label: '权限管理' },
  ];

  return (
    <Tabs
      activeKey={activeKey}
      onChange={onChange}
      items={items}
      className="-mb-px [&_.ant-tabs-nav]:!mb-0 [&_.ant-tabs-tab]:!px-4 [&_.ant-tabs-tab]:!py-4 [&_.ant-tabs-tab-active_.ant-tabs-tab-btn]:!text-[#2f6bff] [&_.ant-tabs-ink-bar]:!bg-[#2f6bff]"
    />
  );
};
