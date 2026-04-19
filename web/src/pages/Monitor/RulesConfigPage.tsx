import React, { useEffect, useState } from 'react';
import { Card, Table } from 'antd';
import { Api } from '../../api';

type EffectiveRuleRow = {
  id: string;
  name: string;
  severity: string;
  threshold?: number;
  scope?: string;
  inherit_key?: string;
};

const RulesConfigPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<EffectiveRuleRow[]>([]);

  useEffect(() => {
    let mounted = true;
    const load = async () => {
      setLoading(true);
      try {
        const res = await Api.monitoring.getEffectiveRules({ page: 1, pageSize: 50 });
        const list = (res?.data as any)?.list || [];
        if (!mounted) return;
        setRows(
          list.map((item: any) => ({
            id: String(item.id),
            name: item.name || '',
            severity: item.severity || '',
            threshold: item.threshold,
            scope: item.scope,
            inherit_key: item.inherit_key,
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
    <Card title="规则配置">
      <Table
        rowKey="id"
        loading={loading}
        dataSource={rows}
        pagination={false}
        columns={[
          { title: '名称', dataIndex: 'name' },
          { title: '级别', dataIndex: 'severity' },
          { title: '阈值', dataIndex: 'threshold', render: (v: number | undefined) => (v == null ? '-' : v) },
          { title: '作用域', dataIndex: 'scope', render: (v: string | undefined) => v || '-' },
          { title: '继承键', dataIndex: 'inherit_key', render: (v: string | undefined) => v || '-' },
        ]}
      />
    </Card>
  );
};

export default RulesConfigPage;
