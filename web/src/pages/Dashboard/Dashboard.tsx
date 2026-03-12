import React, { useCallback, useEffect, useState } from 'react';
import { Col, Row, message } from 'antd';
import { useInterval } from 'ahooks';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { OverviewResponse, TimeRange } from '../../api/modules/dashboard';
import TimeRangeSelector from '../../components/Dashboard/TimeRangeSelector';
import HealthCard from '../../components/Dashboard/HealthCard';
import TimeseriesChart from '../../components/Dashboard/TimeseriesChart';
import AlertPanel from '../../components/Dashboard/AlertPanel';
import EventStream from '../../components/Dashboard/EventStream';
import AIActivityCard from '../../components/Dashboard/AIActivityCard';

const emptyOverview: OverviewResponse = {
  hosts: { total: 0, healthy: 0, degraded: 0, unhealthy: 0, offline: 0 },
  clusters: { total: 0, healthy: 0, degraded: 0, unhealthy: 0, offline: 0 },
  services: { total: 0, healthy: 0, degraded: 0, unhealthy: 0, offline: 0 },
  alerts: { firing: 0, recent: [] },
  events: [],
  metrics: {
    cpu_usage: [],
    memory_usage: [],
  },
  ai: {
    stats: { sessionCount: 0, tokenCount: 0, avgDurationMs: 0, successRate: 0 },
    sessions: [],
    byScene: {},
  },
};

const Dashboard: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [timeRange, setTimeRange] = useState<TimeRange>('1h');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [overview, setOverview] = useState<OverviewResponse>(emptyOverview);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await Api.dashboard.getOverview(timeRange);
      setOverview(response.data || emptyOverview);
    } catch (error) {
      message.error('加载主控台概览失败');
    } finally {
      setLoading(false);
    }
  }, [timeRange]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    const handler = () => {
      load();
    };
    window.addEventListener('project:changed', handler as EventListener);
    return () => window.removeEventListener('project:changed', handler as EventListener);
  }, [load]);

  useInterval(() => {
    load();
  }, autoRefresh ? 60000 : undefined);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">主控台</h1>
          <p className="text-sm text-gray-500 mt-1">实时监控系统运行状态</p>
        </div>
        <TimeRangeSelector
          value={timeRange}
          autoRefresh={autoRefresh}
          loading={loading}
          onChange={setTimeRange}
          onRefresh={load}
          onAutoRefreshChange={setAutoRefresh}
        />
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}>
          <HealthCard title="主机健康" data={overview.hosts} onClick={() => navigate('/hosts')} />
        </Col>
        <Col xs={24} md={8}>
          <HealthCard title="集群健康" data={overview.clusters} onClick={() => navigate('/deployment/infrastructure/clusters')} />
        </Col>
        <Col xs={24} md={8}>
          <HealthCard title="服务健康" data={overview.services} onClick={() => navigate('/services')} />
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} xl={12}>
          <TimeseriesChart title="CPU 使用率" series={overview.metrics.cpu_usage} loading={loading} />
        </Col>
        <Col xs={24} xl={12}>
          <TimeseriesChart title="内存使用率" series={overview.metrics.memory_usage} loading={loading} />
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} xl={12}>
          <AlertPanel alerts={overview.alerts.recent} loading={loading} />
        </Col>
        <Col xs={24} xl={12}>
          <EventStream events={overview.events} loading={loading} />
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24}>
          <AIActivityCard data={overview.ai} loading={loading} />
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
