import React, { useEffect, useState } from 'react';
import { Card, Space, Table, Tag, Typography, Button } from 'antd';
import type { TablePaginationConfig } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import { Api } from '../../api';
import type { Alert } from '../../api/modules/monitoring';
import type { AlertHealJob } from '../../api/modules/aiAlertHeal';
import { normalizeHealStatus } from './monitorAlertHealStatus';

interface AlertRow extends Alert {
  latestJob?: AlertHealJob;
}

const severityColor: Record<string, string> = {
  critical: 'error',
  warning: 'warning',
  info: 'blue',
};

function pickLatestJob(jobs: AlertHealJob[]): AlertHealJob | undefined {
  if (!jobs.length) {
    return undefined;
  }
  return [...jobs].sort((a, b) => new Date(b.updated_at || 0).getTime() - new Date(a.updated_at || 0).getTime())[0];
}

const AlertsPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<AlertRow[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);

  const load = async (nextPage = page, nextPageSize = pageSize) => {
    setLoading(true);
    try {
      const alertRes = await Api.monitoring.getAlertList({ page: nextPage, pageSize: nextPageSize });
      const list = alertRes.data?.list || [];
      const healEntries = await Promise.all(
        list.map(async (alert) => {
          const jobsRes = await Api.aiAlertHeal.listByAlert(alert.id);
          const latestJob = pickLatestJob(jobsRes.data?.list || []);
          return [alert.id, latestJob] as const;
        }),
      );
      const latestJobMap = new Map(healEntries);
      setRows(
        list.map((alert) => ({
          ...alert,
          latestJob: latestJobMap.get(alert.id),
        })),
      );
      setTotal(alertRes.data?.total || 0);
      setPage(nextPage);
      setPageSize(nextPageSize);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load(1, pageSize);
  }, []);

  const onTableChange = (pagination: TablePaginationConfig) => {
    const nextPage = pagination.current || 1;
    const nextPageSize = pagination.pageSize || pageSize;
    void load(nextPage, nextPageSize);
  };

  return (
    <Card
      title="告警列表"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void load(page, pageSize)}>
            刷新
          </Button>
        </Space>
      }
    >
      <Table<AlertRow>
        rowKey="id"
        loading={loading}
        dataSource={rows}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
        }}
        onChange={onTableChange}
        columns={[
          {
            title: '告警消息',
            dataIndex: 'title',
            render: (_value: string, record) => (
              <Link to={`/monitor/alerts/${record.id}`}>
                <Typography.Text style={{ color: '#1677ff' }}>{record.title || '-'}</Typography.Text>
              </Link>
            ),
          },
          {
            title: '级别',
            dataIndex: 'severity',
            render: (value: string) => <Tag color={severityColor[value] || 'default'}>{value || '-'}</Tag>,
          },
          {
            title: '处理状态',
            key: 'processingStatus',
            render: (_value: unknown, record) => {
              const normalized = normalizeHealStatus(record.latestJob?.status || '');
              return <Tag color={normalized.processingColor}>{normalized.processing}</Tag>;
            },
          },
          {
            title: '自愈状态',
            key: 'healStatus',
            render: (_value: unknown, record) => {
              const normalized = normalizeHealStatus(record.latestJob?.status || '');
              return <Tag color={normalized.healingColor}>{normalized.healing}</Tag>;
            },
          },
          {
            title: '最近处理',
            key: 'latestProcessedAt',
            render: (_value: unknown, record) => {
              const latest = record.latestJob?.updated_at || record.createdAt;
              return latest ? new Date(latest).toLocaleString() : '-';
            },
          },
        ]}
      />
    </Card>
  );
};

export default AlertsPage;
