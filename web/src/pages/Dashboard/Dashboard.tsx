import React, { useCallback, useEffect, useState } from 'react';
import { Col, Row, message } from 'antd';
import { useInterval } from 'ahooks';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { OverviewResponseV2, TimeRange } from '../../api/modules/dashboard';
import TimeRangeSelector from '../../components/Dashboard/TimeRangeSelector';
import HealthCard from '../../components/Dashboard/HealthCard';
import WorkloadHealthCard from '../../components/Dashboard/WorkloadHealthCard';
import ClusterResourceCard from '../../components/Dashboard/ClusterResourceCard';
import OperationsCard from '../../components/Dashboard/OperationsCard';
import TimeseriesChart from '../../components/Dashboard/TimeseriesChart';
import AlertPanel from '../../components/Dashboard/AlertPanel';
import EventStream from '../../components/Dashboard/EventStream';
import AIActivityCard from '../../components/Dashboard/AIActivityCard';

const emptyOverview: OverviewResponseV2 = {
  health: {
    hosts: { total: 0, healthy: 0, degraded: 0, unhealthy: 0, offline: 0 },
    clusters: { total: 0, healthy: 0, degraded: 0, unhealthy: 0, offline: 0 },
    applications: { total: 0, healthy: 0, degraded: 0, unhealthy: 0, offline: 0 },
    workloads: {
      deployments: { total: 0, healthy: 0 },
      statefulsets: { total: 0, healthy: 0 },
      daemonsets: { total: 0, healthy: 0 },
      services: 0,
      ingresses: 0,
    },
  },
  resources: {
    hosts: [],
    clusters: [],
  },
  operations: {
    deployments: { running: 0, pendingApproval: 0, todayTotal: 0, todaySuccess: 0, todayFailed: 0 },
    cicd: { running: 0, queued: 0, todayTotal: 0, success: 0, failed: 0 },
    issuePods: { total: 0, byType: {} },
  },
  alerts: { firing: 0, recent: [] },
  events: [],
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
  const [overview, setOverview] = useState<OverviewResponseV2>(emptyOverview);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await Api.dashboard.getOverviewV2(timeRange);
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

      {/* 健康概览 - 4 列 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <HealthCard title="主机健康" data={overview.health.hosts} onClick={() => navigate('/hosts')} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <HealthCard title="集群健康" data={overview.health.clusters} onClick={() => navigate('/deployment/infrastructure/clusters')} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <HealthCard title="应用健康" data={overview.health.applications} onClick={() => navigate('/services')} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <WorkloadHealthCard data={overview.health.workloads} loading={loading} />
        </Col>
      </Row>

      {/* 资源使用 - 2 列 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} xl={12}>
          <TimeseriesChart title="CPU 使用率" series={overview.resources.hosts} loading={loading} />
        </Col>
        <Col xs={24} xl={12}>
          <ClusterResourceCard data={overview.resources.clusters} loading={loading} />
        </Col>
      </Row>

      {/* 运行状态 + 告警 + AI - 3 列 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}>
          <OperationsCard data={overview.operations} loading={loading} />
        </Col>
        <Col xs={24} md={8}>
          <AlertPanel alerts={overview.alerts.recent} loading={loading} />
        </Col>
        <Col xs={24} md={8}>
          <AIActivityCard data={overview.ai} loading={loading} />
        </Col>
      </Row>

      {/* 事件流 - 全宽 */}
      <Row gutter={[16, 16]}>
        <Col xs={24}>
          <EventStream events={overview.events} loading={loading} />
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
