import React, { useEffect, useState } from 'react';
import { Card, Table } from 'antd';
import { Api } from '../../api';

type RouteRow = {
  id: string;
  scope: string;
  severity: string;
  channel_ids_json?: string;
  enabled: boolean;
};

const RoutingConfigPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<RouteRow[]>([]);

  useEffect(() => {
    let mounted = true;
    const load = async () => {
      setLoading(true);
      try {
        const res = await Api.monitoring.getSeverityRoutes();
        const list = (res?.data as any)?.list || [];
        if (!mounted) return;
        setRows(
          list.map((item: any) => ({
            id: String(item.id),
            scope: item.scope || '',
            severity: item.severity || '',
            channel_ids_json: item.channel_ids_json || '[]',
            enabled: !!item.enabled,
          })),
        );
      } finally {
        if (mounted) setLoading(false);
      }
    };
    load();
    return () => {
      mounted = false;
    };
  }, []);

  return (
    <Card title="路由配置">
      <Table
        rowKey="id"
        loading={loading}
        dataSource={rows}
        pagination={false}
        columns={[
          { title: '作用域', dataIndex: 'scope' },
          { title: '级别', dataIndex: 'severity' },
          { title: '渠道', dataIndex: 'channel_ids_json', render: (v: string | undefined) => v || '[]' },
          { title: '启用', dataIndex: 'enabled', render: (v: boolean) => (v ? '是' : '否') },
        ]}
      />
    </Card>
  );
};

export default RoutingConfigPage;
