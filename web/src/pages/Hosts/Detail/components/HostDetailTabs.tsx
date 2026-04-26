import React from 'react';
import { Tabs } from 'antd';
import type { TabsProps } from 'antd';

interface HostDetailTabsProps {
  activeTab: string;
  onChange: (key: string) => void;
  tabContent: Record<string, React.ReactNode>;
}

const HostDetailTabs: React.FC<HostDetailTabsProps> = ({
  activeTab,
  onChange,
  tabContent,
}) => {
  const items: TabsProps['items'] = [
    { key: 'overview', label: '概览' },
    { key: 'monitor', label: '监控' },
    { key: 'process', label: '进程' },
    { key: 'service', label: '服务' },
    { key: 'disk', label: '磁盘' },
    { key: 'network', label: '网络' },
    { key: 'packages', label: '软件包' },
    { key: 'config', label: '配置' },
    { key: 'alarm', label: '告警' },
    { key: 'logs', label: '操作记录' },
  ].map(item => ({
    ...item,
    children: tabContent[item.key] || <div className="p-8 text-center text-gray-400">正在开发中...</div>
  }));

  return (
    <Tabs 
      activeKey={activeTab} 
      onChange={onChange} 
      items={items}
      className="host-detail-tabs"
      destroyInactiveTabPane={false}
    />
  );
};

export default HostDetailTabs;
