import React, { useState, useEffect, useMemo } from 'react';
import { Card, Button, Space, Input, Select, Tag, Empty, Row, Col, Badge, Statistic } from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  CloudServerOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../../api';
import type { DeployTarget } from '../../../api/modules/deployment';
import { StaggerList, StaggerItem } from '../../../components/Motion';

const DeploymentTargetListPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [targets, setTargets] = useState<DeployTarget[]>([]);
  const [search, setSearch] = useState('');
  const [envFilter, setEnvFilter] = useState('all');
  const [runtimeFilter, setRuntimeFilter] = useState<'all' | 'k8s' | 'compose'>('all');

  const load = async () => {
    setLoading(true);
    try {
      const params: any = {};
      if (envFilter !== 'all') params.environment = envFilter;
      if (runtimeFilter !== 'all') params.runtime_type = runtimeFilter;
      const res = await Api.deployment.listTargets(params);
      setTargets(res.data.list || []);
    } catch (err) {
      console.error('Failed to load targets:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [envFilter, runtimeFilter]);

  // Statistics
  const stats = useMemo(() => {
    const ready = targets.filter((t) => t.readiness_status === 'ready').length;
    const notReady = targets.filter((t) => t.readiness_status === 'not_ready').length;
    const bootstrapping = targets.filter((t) => t.readiness_status === 'bootstrapping').length;
    return { ready, notReady, bootstrapping, total: targets.length };
  }, [targets]);

  // Filtered targets
  const filtered = useMemo(() => {
    return targets.filter((t) => {
      const hitSearch =
        t.name.toLowerCase().includes(search.toLowerCase()) ||
        t.environment.toLowerCase().includes(search.toLowerCase());
      return hitSearch;
    });
  }, [targets, search]);

  // Group by environment
  const groupedByEnv = useMemo(() => {
    const groups: Record<string, DeployTarget[]> = {};
    filtered.forEach((t) => {
      if (!groups[t.environment]) {
        groups[t.environment] = [];
      }
      groups[t.environment].push(t);
    });
    return groups;
  }, [filtered]);

  const getReadinessConfig = (status: string) => {
    const configs: Record<string, { icon: React.ReactNode; color: string; text: string }> = {
      ready: { icon: <CheckCircleOutlined />, color: 'success', text: '就绪' },
      not_ready: { icon: <CloseCircleOutlined />, color: 'error', text: '未就绪' },
      bootstrapping: { icon: <SyncOutlined spin />, color: 'processing', text: '初始化中' },
    };
    return configs[status] || { icon: <ExclamationCircleOutlined />, color: 'default', text: status };
  };

  const TargetCard: React.FC<{ target: DeployTarget }> = ({ target }) => {
    const readinessConfig = getReadinessConfig(target.readiness_status);

    return (
      <Card
        hoverable
        className="h-full cursor-pointer transition-shadow hover:shadow-lg"
        onClick={() => navigate(`/deployment/targets/${target.id}`)}
      >
        <Space direction="vertical" className="w-full" size="middle">
          <div className="flex items-center justify-between">
            <Space>
              <CloudServerOutlined className="text-2xl text-blue-500" />
              <span className="text-lg font-semibold">{target.name}</span>
            </Space>
            <Tag color={readinessConfig.color} icon={readinessConfig.icon}>
              {readinessConfig.text}
            </Tag>
          </div>

          <div className="space-y-2 text-sm text-gray-600">
            <div className="flex justify-between">
              <span>环境:</span>
              <Tag color={target.environment === 'production' ? 'red' : target.environment === 'staging' ? 'orange' : 'blue'}>
                {target.environment}
              </Tag>
            </div>
            <div className="flex justify-between">
              <span>运行时:</span>
              <Tag color={target.runtime_type === 'k8s' ? 'blue' : 'green'}>
                {target.runtime_type === 'k8s' ? 'Kubernetes' : 'Docker Compose'}
              </Tag>
            </div>
            {target.cluster_name && (
              <div className="flex justify-between">
                <span>集群:</span>
                <span className="font-medium">{target.cluster_name}</span>
              </div>
            )}
            {target.namespace && (
              <div className="flex justify-between">
                <span>命名空间:</span>
                <span className="font-medium">{target.namespace}</span>
              </div>
            )}
          </div>
        </Space>
      </Card>
    );
  };

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">部署目标</h1>
          <p className="text-sm text-gray-500 mt-1">管理应用部署的目标环境</p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/deployment/targets/create')}>
            创建部署目标
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
                  title={<span className="text-gray-600">总数</span>}
                  value={stats.total}
                  prefix={<CloudServerOutlined className="text-primary-500" />}
                  valueStyle={{ color: '#495057', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow">
                <Statistic
                  title={<span className="text-gray-600">就绪</span>}
                  value={stats.ready}
                  prefix={<CheckCircleOutlined className="text-success" />}
                  valueStyle={{ color: '#10b981', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow">
                <Statistic
                  title={<span className="text-gray-600">初始化中</span>}
                  value={stats.bootstrapping}
                  prefix={<SyncOutlined className="text-warning" />}
                  valueStyle={{ color: '#f59e0b', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow">
                <Statistic
                  title={<span className="text-gray-600">未就绪</span>}
                  value={stats.notReady}
                  prefix={<CloseCircleOutlined className="text-error" />}
                  valueStyle={{ color: '#ef4444', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
        </Row>
      </StaggerList>

      {/* Filters */}
      <Card>
        <div className="flex flex-wrap gap-3">
          <Input
            placeholder="搜索目标名称或环境"
            prefix={<SearchOutlined className="text-gray-400" />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ width: 280 }}
            allowClear
          />
          <Select
            value={envFilter}
            style={{ width: 140 }}
            options={[
        { value: 'all', label: '全部环境' },
              { value: 'development', label: '开发' },
              { value: 'staging', label: '预发布' },
              { value: 'production', label: '生产' },
            ]}
            onChange={setEnvFilter}
          />
          <Select
            value={runtimeFilter}
            style={{ width: 140 }}
            options={[
              { value: 'all', label: '全部运行时' },
              { value: 'k8s', label: 'Kubernetes' },
              { value: 'compose', label: 'Docker Compose' },
            ]}
            onChange={setRuntimeFilter}
          />
        </div>
      </Card>

      {/* Target list grouped by environment */}
      {loading ? (
        <Card>
          <div className="text-center py-12">
            <ReloadOutlined spin className="text-4xl text-primary-500 mb-4" />
            <p className="text-gray-500">加载中...</p>
          </div>
        </Card>
      ) : Object.keys(groupedByEnv).length === 0 ? (
        <Card>
          <Empty
            description={
              <span className="text-gray-500">
                {search || envFilter !== 'all' || runtimeFilter !== 'all'
                  ? '没有找到匹配的部署目标'
                  : '还没有创建任何部署目标'}
              </span>
            }
          >
            {!search && envFilter === 'all' && runtimeFilter === 'all' && (
              <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/deployment/targets/create')}>
                创建第一个部署目标
              </Button>
            )}
          </Empty>
        </Card>
      ) : (
        <div className="space-y-6">
          {Object.entries(groupedByEnv).map(([env, envTargets]) => (
            <div key={env}>
              <div className="flex items-center gap-2 mb-4">
                <h2 className="text-lg font-semibold text-gray-900">{env}</h2>
                <Badge count={envTargets.length} showZero style={{ backgroundColor: '#6366f1' }} />
              </div>
              <StaggerList staggerDelay={0.05}>
                <Row gutter={[16, 16]}>
                  {envTargets.map((target) => (
                    <Col xs={24} sm={12} lg={8} xl={6} key={target.id}>
                      <StaggerItem>
                        <TargetCard target={target} />
                      </StaggerItem>
                    </Col>
                  ))}
                </Row>
              </StaggerList>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default DeploymentTargetListPage;
