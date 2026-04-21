import React, { useEffect, useState } from 'react';
import { Card, Table } from 'antd';
import { Api } from '../../api';
import type { AlertDelivery } from '../../api/modules/monitoring';

const DeliveriesPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<AlertDelivery[]>([]);

  useEffect(() => {
    let mounted = true;
    const load = async () => {
      setLoading(true);
      try {
        const res = await Api.monitoring.listAlertDeliveries({ page: 1, pageSize: 50 });
        const list = (res?.data as any)?.list || [];
        if (!mounted) return;
        setRows(list);
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
    <Card title="投递记录" size="small">
      <Table
        rowKey="id"
        size="small"
        loading={loading}
        dataSource={rows}
        pagination={false}
        columns={[
          { title: '渠道类型', dataIndex: 'channelType' },
          { title: '目标', dataIndex: 'target' },
          { title: '状态', dataIndex: 'status' },
          { title: '错误信息', dataIndex: 'errorMessage', render: (v: string | undefined) => v || '-' },
          { title: '时间', dataIndex: 'deliveredAt' },
        ]}
      />
    </Card>
  );
};

export default DeliveriesPage;
