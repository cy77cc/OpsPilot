import React, { useEffect, useMemo, useState } from 'react';
import { Card, Col, Row, Statistic, Table, Tabs, Tag, Progress, Button, Space, message } from 'antd';
import { AlertOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { Api } from '../../api';
import type { Alert, AlertRule, MetricData, AlertChannel } from '../../api/modules/monitoring';
import { PageSkeleton } from '../../components/LoadingSkeleton';

const MonitorPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [cpuMetrics, setCpuMetrics] = useState<MetricData[]>([]);
  const [memMetrics, setMemMetrics] = useState<MetricData[]>([]);
  const [channels, setChannels] = useState<AlertChannel[]>([]);

  const load = async () => {
    setLoading(true);
    try {
      const end = new Date().toISOString();
      const start = dayjs().subtract(24, 'hour').toDate().toISOString();
      const [alertRes, ruleRes, cpuRes, memRes, channelRes] = await Promise.all([
        Api.monitoring.getAlertList({ page: 1, pageSize: 100 }),
        Api.monitoring.getAlertRuleList({ page: 1, pageSize: 100 }),
        Api.monitoring.getMetrics({ metric: 'cpu_usage', startTime: start, endTime: end }),
        Api.monitoring.getMetrics({ metric: 'memory_usage', startTime: start, endTime: end }),
        Api.monitoring.listAlertChannels(),
      ]);
      setAlerts(alertRes.data.list || []);
      setRules(ruleRes.data.list || []);
      setCpuMetrics(cpuRes.data?.series || []);
      setMemMetrics(memRes.data?.series || []);
      setChannels(channelRes.data?.list || []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleSyncRules = async () => {
    try {
      await Api.monitoring.syncAlertRules();
      message.success('规则同步成功');
      await load();
    } catch (error: any) {
      message.error(error?.message || '规则同步失败');
    }
  };

  const firingCount = useMemo(() => alerts.filter((a) => a.status === 'firing').length, [alerts]);
  const criticalCount = useMemo(() => alerts.filter((a) => a.severity === 'critical' && a.status === 'firing').length, [alerts]);
  const cpuAvg = useMemo(() => (cpuMetrics.length ? cpuMetrics.reduce((s, i) => s + Number(i.value), 0) / cpuMetrics.length : 0), [cpuMetrics]);
  const memAvg = useMemo(() => (memMetrics.length ? memMetrics.reduce((s, i) => s + Number(i.value), 0) / memMetrics.length : 0), [memMetrics]);
  const isInitialLoading = loading
    && alerts.length === 0
    && rules.length === 0
    && cpuMetrics.length === 0
    && memMetrics.length === 0
    && channels.length === 0;

  if (isInitialLoading) {
    return <PageSkeleton />;
  }

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <Space>
          <Button size="small" onClick={handleSyncRules}>同步规则</Button>
          <Button size="small" icon={<ReloadOutlined />} loading={loading && !isInitialLoading} onClick={load}>刷新</Button>
        </Space>
      </div>
      <Row gutter={[12, 12]}>
        <Col xs={24} sm={8}>
          <Card size="small">
            <Statistic 
              title="活跃告警" 
              value={firingCount} 
              prefix={<AlertOutlined />} 
              valueStyle={{ fontSize: 20 }} 
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card size="small">
            <Statistic 
              title="严重告警" 
              value={criticalCount} 
              valueStyle={{ fontSize: 20, color: criticalCount > 0 ? '#ff4d4f' : 'inherit' }} 
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card size="small">
            <Statistic 
              title="告警规则" 
              value={rules.length} 
              valueStyle={{ fontSize: 20 }} 
            />
          </Card>
        </Col>
      </Row>
      <Row gutter={[12, 12]}>
        <Col xs={24} md={12}>
          <Card size="small" title="CPU 平均使用率">
            <Progress size="small" percent={Math.round(cpuAvg)} />
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card size="small" title="内存平均使用率">
            <Progress size="small" percent={Math.round(memAvg)} strokeColor="#1677ff" />
          </Card>
        </Col>
      </Row>

      <Tabs
        size="small"
        items={[
          {
            key: 'alerts',
            label: '告警历史',
            children: (
              <Table
                size="small"
                rowKey="id"
                loading={false}
                dataSource={alerts}
                columns={[
                  { title: '消息', dataIndex: 'title', render: (_: string, r: any) => r.message || r.title || '-' },
                  { title: '级别', dataIndex: 'severity', render: (v: string) => <Tag color={v === 'critical' ? 'error' : v === 'warning' ? 'warning' : 'blue'}>{v}</Tag> },
                  { title: '来源', dataIndex: 'source', render: (_: string, r: any) => r.metric || r.source || '-' },
                  { title: '状态', dataIndex: 'status', render: (v: string) => <Tag color={v === 'firing' ? 'error' : 'success'}>{v}</Tag> },
                  { title: '时间', dataIndex: 'createdAt', render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
                ]}
              />
            ),
          },
          {
            key: 'rules',
            label: '告警规则',
            children: (
              <Table
                size="small"
                rowKey="id"
                loading={false}
                dataSource={rules}
                columns={[
                  { title: '名称', dataIndex: 'name' },
                  { title: 'PromQL', dataIndex: 'promqlExpr', render: (_: string, r: any) => r.promqlExpr || `${r.metric} ${r.operator} ${r.threshold}` },
                  { title: '级别', dataIndex: 'severity', render: (v: string) => <Tag>{v}</Tag> },
                  { title: '启用', dataIndex: 'enabled', render: (v: boolean) => <Tag color={v ? 'success' : 'default'}>{v ? '启用' : '禁用'}</Tag> },
                ]}
              />
            ),
          },
          {
            key: 'channels',
            label: '通知渠道',
            children: (
              <Table
                size="small"
                rowKey="id"
                loading={false}
                dataSource={channels}
                columns={[
                  { title: '名称', dataIndex: 'name' },
                  { title: '类型', dataIndex: 'type' },
                  { title: 'Provider', dataIndex: 'provider' },
                  { title: '目标', dataIndex: 'target', render: (v: string) => v || '-' },
                  { title: '状态', dataIndex: 'enabled', render: (v: boolean) => <Tag color={v ? 'success' : 'default'}>{v ? '启用' : '禁用'}</Tag> },
                ]}
              />
            ),
          },
        ]}
      />
    </div>
  );
};

export default MonitorPage;
