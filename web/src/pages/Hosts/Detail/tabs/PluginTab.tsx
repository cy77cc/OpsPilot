import React from 'react';
import { Card, Table, Tag } from 'antd';
import type { Host } from '../../../../api/modules/hosts';

interface PluginTabProps {
  host: Host | null;
}

const PluginTab: React.FC<PluginTabProps> = ({ host }) => {
  const rows = host?.pluginInstances || [];

  return (
    <Card title="插件实例">
      <Table
        rowKey={(row) => `${row.pluginKey}-${row.installedVersion}-${row.lastSeenAt || ''}`}
        dataSource={rows}
        pagination={false}
        columns={[
          { title: '插件', dataIndex: 'pluginKey' },
          { title: '版本', dataIndex: 'installedVersion', render: (value: string) => value || '-' },
          { title: '安装状态', dataIndex: 'installStatus', render: (value: string) => <Tag>{value || '-'}</Tag> },
          { title: '运行状态', dataIndex: 'runtimeStatus', render: (value: string) => <Tag>{value || '-'}</Tag> },
          { title: '健康状态', dataIndex: 'healthStatus', render: (value: string) => <Tag>{value || '-'}</Tag> },
          { title: '最近心跳', dataIndex: 'lastSeenAt', render: (value: string) => value ? new Date(value).toLocaleString() : '-' },
        ]}
        locale={{ emptyText: '暂无插件实例' }}
      />
    </Card>
  );
};

export default PluginTab;
