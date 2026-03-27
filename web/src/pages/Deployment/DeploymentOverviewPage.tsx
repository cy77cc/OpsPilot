import React, { useState, useEffect, useMemo } from 'react';
import { Card, Row, Col, Statistic, Button, Space, Empty, Tag } from 'antd';
import {
  ReloadOutlined,
  RocketOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  SyncOutlined,
  CloseCircleOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { DeployRelease } from '../../api/modules/deployment';
import { StaggerList, StaggerItem } from '../../components/Motion';

const DeploymentOverviewPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [releases, setReleases] = useState<DeployRelease[]>([]);
  const [targets, setTargets] = useState<any[]>([]);

  const load = async () => {
    setLoading(true);
    try {
      const [releasesRes, targetsRes] = await Promise.all([
        Api.deployment.getReleases(),
        Api.deployment.listTargets(),
      ]);
      setReleases(releasesRes.data.list || []);
      setTargets(targetsRes.data.list || []);
    } catch (err) {
      console.error('Failed to load overview:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // Poll for in-progress deployments
    const interval = setInterval(() => {
      const hasInProgress = releases.some((r) => r.state === 'applying');
      if (hasInProgress) {
        load();
      }
    }, 10000);
    return () => clearInterval(interval);
  }, [releases]);

  // Statistics
  const stats = useMemo(() => {
    const total = releases.length;
    const pendingApproval = releases.filter((r) => r.state === 'pending_approval').length;
    const inProgress = releases.filter((r) => r.state === 'applying').length;
    const succeeded = releases.filter((r) => r.state === 'applied').length;
    const failed = releases.filter((r) => r.state === 'failed').length;
    const successRate = total > 0 ? Math.round((succeeded / total) * 100) : 0;
    return { total, pendingApproval, inProgress, succeeded, failed, successRate };
  }, [releases]);

  // Environment health
  const envHealth = useMemo(() => {
    const envMap: Record<string, { total: number; ready: number; notReady: number }> = {};
    targets.forEach((t) => {
      if (!envMap[t.environment]) {
        envMap[t.environment] = { total: 0, ready: 0, notReady: 0 };
      }
      envMap[t.environment].total++;
      if (t.readiness_status === 'ready') {
        envMap[t.environment].ready++;
      } else {
        envMap[t.environment].notReady++;
      }
    });
    return envMap;
  }, [targets]);

  // Pending approvals
  const pendingApprovals = useMemo(() => {
    return releases.filter((r) => r.state === 'pending_approval').slice(0, 5);
  }, [releases]);

  // In-progress deployments
  const inProgressDeployments = useMemo(() => {
    return releases.filter((r) => r.state === 'applying').slice(0, 5);
  }, [releases]);

  const getStateConfig = (state: string) => {
    const configs: Record<string, { icon: React.ReactNode; color: string; text: string }> = {
      pending_approval: { icon: <ClockCircleOutlined />, color: 'orange', text: '待审批' },
      approved: { icon: <CheckCircleOutlined />, color: 'blue', text: '已批准' },
      applying: { icon: <SyncOutlined spin />, color: 'processing', text: '部署中' },
      applied: { icon: <CheckCircleOutlined />, color: 'success', text: '成功' },
      failed: { icon: <CloseCircleOutlined />, color: 'error', text: '失败' },
      rejected: { icon: <CloseCircleOutlined />, color: 'default', text: '已拒绝' },
    };
    return configs[state] || { icon: null, color: 'default', text: state };
  };

  const EnvironmentStatusCard: React.FC<{ env: string; data: { total: number; ready: number; notReady: number } }> = ({ env, data }) => {
    const healthRate = data.total > 0 ? Math.round((data.ready / data.total) * 100) : 0;
    const color = healthRate >= 80 ? 'success' : healthRate >= 50 ? 'warning' : 'error';

    return (
      <Card className="h-full">
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <Tag color={env === 'production' ? 'red' : env === 'staging' ? 'orange' : 'blue'}>
              {env}
            </Tag>
            <span className={`text-2xl font-semibold text-${color === 'success' ? 'green' : color === 'warning' ? 'yellow' : 'red'}-600`}>
              {healthRate}%
            </span>
          </div>
          <div className="text-sm text-gray-600">
            <div>总计: {data.total} 个目标</div>
            <div>就绪: {data.ready} 个</div>
            <div>未就绪: {data.notReady} 个</div>
          </div>
        </div>
      </Card>
    );
  };

  const PendingApprovalsList: React.FC = () => {
    if (pendingApprovals.length === 0) {
      return (
        <Card title="待审批发布" className="h-full">
          <Empty description="暂无待审批的发布" />
        </Card>
      );
    }

    return (
      <Card title="待审批发布" className="h-full">
        <div className="space-y-3">
          {pendingApprovals.map((release) => (
            <div
              key={release.id}
              className="p-3 border border-gray-200 rounded hover:bg-gray-50 cursor-pointer"
              onClick={() => navigate(`/deployment/${release.id}`)}
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-semibold">{release.service_name}</span>
                <Tag color="orange" icon={<ClockCircleOutlined />}>
                  待审批
                </Tag>
              </div>
              <div className="text-xs text-gray-500">
                <div>目标: {release.target_name}</div>
                <div>创建: {new Date(release.created_at).toLocaleString()}</div>
              </div>
            </div>
          ))}
        </div>
      </Card>
    );
  };

  const InProgressDeploymentsList: React.FC = () => {
    if (inProgressDeployments.length === 0) {
      return (
        <Card title="进行中的部署" className="h-full">
          <Empty description="暂无进行中的部署" />
        </Card>
      );
    }

    return (
      <Card title="进行中的部署" className="h-full">
        <div className="space-y-3">
          {inProgressDeployments.map((release) => (
            <div
              key={release.id}
              className="p-3 border border-gray-200 rounded hover:bg-gray-50 cursor-pointer"
              onClick={() => navigate(`/deployment/${release.id}`)}
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-semibold">{release.service_name}</span>
                <Tag color="processing" icon={<SyncOutlined spin />}>
                  部署中
                </Tag>
              </div>
              <div className="text-xs text-gray-500">
                <div>目标: {release.target_name}</div>
                <div>阶段: {release.phase || 'deploying'}</div>
                {release.progress !== undefined && (
                  <div>进度: {release.progress}%</div>
                )}
              </div>
            </div>
          ))}
        </div>
      </Card>
    );
  };

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">部署概览</h1>
          <p className="text-sm text-gray-500 mt-1">监控所有环境的部署状态</p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<RocketOutlined />} onClick={() => navigate('/deployment/create')}>
            创建发布
          </Button>
        </Space>
      </div>

      {/* Statistics cards */}
      <StaggerList staggerDelay={0.05}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow">
                <Statistic
                  title={<span className="text-gray-600">总发布数</span>}
                  value={stats.total}
                  prefix={<RocketOutlined className="text-primary-500" />}
                  valueStyle={{ color: '#495057', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow">
                <Statistic
                  title={<span className="text-gray-600">待审批</span>}
                  value={stats.pendingApproval}
                  prefix={<ClockCircleOutlined className="text-warning" />}
                  valueStyle={{ color: '#f59e0b', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow">
                <Statistic
                  title={<span className="text-gray-600">进行中</span>}
                  value={stats.inProgress}
                  prefix={<SyncOutlined className="text-processing" />}
                  valueStyle={{ color: '#1890ff', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow">
                <Statistic
                  title={<span className="text-gray-600">成功率</span>}
                  value={stats.successRate}
                  suffix="%"
                  prefix={<CheckCircleOutlined className="text-success" />}
                  valueStyle={{ color: '#10b981', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
        </Row>
      </StaggerList>

      {/* Environment health */}
      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">环境健康状态</h2>
        <StaggerList staggerDelay={0.05}>
          <Row gutter={[16, 16]}>
            {Object.entries(envHealth).map(([env, data]) => (
              <Col xs={24} sm={12} lg={8} key={env}>
                <StaggerItem>
                  <EnvironmentStatusCard env={env} data={data} />
                </StaggerItem>
              </Col>
            ))}
            {Object.keys(envHealth).length === 0 && (
              <Col span={24}>
                <Card>
                  <Empty description="暂无环境数据" />
                </Card>
              </Col>
            )}
          </Row>
        </StaggerList>
      </div>

      {/* Pending approvals and in-progress deployments */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <PendingApprovalsList />
        </Col>
        <Col xs={24} lg={12}>
          <InProgressDeploymentsList />
        </Col>
      </Row>
    </div>
  );
};

export default DeploymentOverviewPage;
