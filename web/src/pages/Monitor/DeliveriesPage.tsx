import React, { useEffect, useState, useCallback } from 'react';
import { Card, Table } from 'antd';
import { Api } from '../../api';
import type { AlertDelivery } from '../../api/modules/monitoring';
import { useRegisterMonitorRefresh } from './MonitorRefreshContext';

const DeliveriesPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<AlertDelivery[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await Api.monitoring.listAlertDeliveries({ page: 1, pageSize: 50 });
      const list = (res?.data as any)?.list || [];
      setRows(list);
    } finally {
      setLoading(false);
    }
  }, []);

  useRegisterMonitorRefresh(load, loading);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <Card title="投递记录">
      <Table
        rowKey="id"
       
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
