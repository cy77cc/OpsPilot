import React, { useState, useCallback, useEffect, useMemo } from 'react';
import {
  Button,
  Card,
  Col,
  Row,
  Space,
  Tag,
  message,
  Statistic,
  Timeline,
  Empty,
  Select,
  Input,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  RollbackOutlined,
  EyeOutlined,
  SearchOutlined,
  CloudUploadOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { DeployRelease } from '../../api/modules/deployment';
import { StaggerList, StaggerItem } from '../../components/Motion';

const DeploymentListPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [releases, setReleases] = useState<DeployRelease[]>([]);
  const [services, setServices] = useState<any[]>([]);
  const [targets, setTargets] = useState<any[]>([]);
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [runtimeFilter, setRuntimeFilter] = useState<string>('all');
  const [serviceFilter, setServiceFilter] = useState<string>('all');
  const [targetFilter, setTargetFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [releasesRes, servicesRes, targetsRes] = await Promise.all([
        Api.deployment.getReleasesByRuntime({
          runtime_type: runtimeFilter === 'all' ? undefined : (runtimeFilter as any),
          service_id: serviceFilter === 'all' ? undefined : Number(serviceFilter),
          target_id: targetFilter === 'all' ? undefined : Number(targetFilter),
        }),
        Api.services.getList({ page: 1, pageSize: 500 }),
        Api.deployment.listTargets(),
      ]);
      setReleases(releasesRes.data.list || []);
      setServices(servicesRes.data.list || []);
      setTargets(targetsRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载部署记录失败');
    } finally {
      setLoading(false);
    }
  }, [runtimeFilter, serviceFilter, targetFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  // 统计数据
  const stats = useMemo(() => {
    const succeeded = releases.filter((r) => r.status === 'succeeded' || r.status === 'applied').length;
    const failed = releases.filter((r) => r.status === 'failed').length;
    const pending = releases.filter((r) => r.status === 'pending_approval').length;
    const running = releases.filter(
      (r) => r.status !== 'succeeded' && r.status !== 'failed' && r.status !== 'pending_approval'
    ).length;
    const successRate = releases.length > 0 ? Math.round((succeeded / releases.length) * 100) : 0;
    return { succeeded, failed, pending, running, successRate, total: releases.length };
  }, [releases]);

  // 过滤后的列表
  const filteredReleases = useMemo(() => {
    let filtered = releases;

    // 状态筛选
    if (statusFilter !== 'all') {
      if (statusFilter === 'succeeded') {
        filtered = filtered.filter((r) => r.status === 'succeeded' || r.status === 'applied');
      } else if (statusFilter === 'running') {
        filtered = filtered.filter(
          (r) => r.status !== 'succeeded' && r.status !== 'failed' && r.status !== 'pending_approval'
        );
      } else {
        filtered = filtered.filter((r) => r.status === statusFilter);
      }
    }

    // 搜索筛选
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (r) =>
          String(r.id).includes(query) ||
          String(r.service_id).includes(query) ||
          String(r.target_id).includes(query)
      );
    }

    return filtered;
  }, [releases, statusFilter, searchQuery]);

  // 获取状态配置
  const getStatusConfig = (status: string) => {
    const configs: Record<string, { icon: React.ReactNode; color: string; text: string; dotColor: string }> = {
      succeeded: {
        icon: <CheckCircleOutlined />,
        color: 'success',
        text: '成功',
        dotColor: '#10b981',
      },
      applied: {
        icon: <CheckCircleOutlined />,
        color: 'success',
        text: '已应用',
        dotColor: '#10b981',
      },
      failed: {
        icon: <CloseCircleOutlined />,
        color: 'error',
        text: '失败',
        dotColor: '#ef4444',
      },
      pending_approval: {
        icon: <ClockCircleOutlined />,
        color: 'warning',
        text: '待审批',
        dotColor: '#f59e0b',
      },
      rollback: {
        icon: <RollbackOutlined />,
        color: 'default',
        text: '已回滚',
        dotColor: '#6c757d',
      },
      rolled_back: {
        icon: <RollbackOutlined />,
        color: 'default',
        text: '已回滚',
        dotColor: '#6c757d',
      },
    };
    return (
      configs[status] || {
        icon: <ClockCircleOutlined />,
        color: 'processing',
        text: status,
        dotColor: '#6366f1',
      }
    );
  };

  // 回滚操作
  const handleRollback = async (releaseId: number) => {
    try {
      await Api.deployment.rollbackRelease(releaseId);
      message.success(`回滚任务已提交，来源 release #${releaseId}`);
      await load();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '回滚失败');
    }
  };

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">部署管理</h1>
          <p className="text-sm text-gray-500 mt-1">管理和监控所有部署发布</p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/deployment/create')}>
            创建部署
          </Button>
        </Space>
      </div>

      {/* 统计卡片 */}
      <StaggerList staggerDelay={0.05}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card
                className="hover:shadow-lg transition-shadow cursor-pointer"
                onClick={() => setStatusFilter('all')}
              >
                <Statistic
                  title={<span className="text-gray-600">总部署次数</span>}
                  value={stats.total}
                  prefix={<CloudUploadOutlined className="text-primary-500" />}
                  valueStyle={{ color: '#495057', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card
                className="hover:shadow-lg transition-shadow cursor-pointer"
                onClick={() => setStatusFilter('succeeded')}
              >
                <Statistic
                  title={<span className="text-gray-600">成功部署</span>}
                  value={stats.succeeded}
                  prefix={<CheckCircleOutlined className="text-success" />}
                  valueStyle={{ color: '#10b981', fontSize: '28px', fontWeight: 600 }}
                />
                <div className="mt-2 text-sm text-gray-500">成功率 {stats.successRate}%</div>
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card
                className="hover:shadow-lg transition-shadow cursor-pointer"
                onClick={() => setStatusFilter('failed')}
              >
                <Statistic
                  title={<span className="text-gray-600">失败部署</span>}
                  value={stats.failed}
                  prefix={<CloseCircleOutlined className="text-error" />}
                  valueStyle={{ color: '#ef4444', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card
                className="hover:shadow-lg transition-shadow cursor-pointer"
                onClick={() => setStatusFilter('pending_approval')}
              >
                <Statistic
                  title={<span className="text-gray-600">待审批</span>}
                  value={stats.pending}
                  prefix={<ClockCircleOutlined className="text-warning" />}
                  valueStyle={{ color: '#f59e0b', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
        </Row>
      </StaggerList>

      {/* 筛选和搜索 */}
      <Card>
        <div className="flex flex-wrap gap-3">
          <Input
            placeholder="搜索 Release ID、Service ID、Target ID"
            prefix={<SearchOutlined className="text-gray-400" />}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{ width: 300 }}
            allowClear
          />
          <Select
            value={statusFilter}
            style={{ width: 140 }}
            options={[
              { value: 'all', label: '全部状态' },
              { value: 'succeeded', label: '成功' },
              { value: 'failed', label: '失败' },
              { value: 'pending_approval', label: '待审批' },
              { value: 'running', label: '进行中' },
            ]}
            onChange={setStatusFilter}
          />
          <Select
            value={runtimeFilter}
            style={{ width: 140 }}
            options={[
              { value: 'all', label: '全部运行时' },
              { value: 'k8s', label: 'Kubernetes' },
              { value: 'compose', label: 'Compose' },
            ]}
            onChange={setRuntimeFilter}
          />
          <Select
            value={serviceFilter}
            style={{ width: 180 }}
            options={[
              { value: 'all', label: '全部服务' },
              ...services.map((s) => ({ value: String(s.id), label: s.name })),
            ]}
            onChange={setServiceFilter}
          />
          <Select
            value={targetFilter}
            style={{ width: 180 }}
            options={[
              { value: 'all', label: '全部目标' },
              ...targets.map((t) => ({ value: String(t.id), label: t.name })),
            ]}
            onChange={setTargetFilter}
          />
        </div>
      </Card>

      {/* 部署时间线 */}
      {loading ? (
        <Card>
          <div className="text-center py-12">
            <ReloadOutlined spin className="text-4xl text-primary-500 mb-4" />
            <p className="text-gray-500">加载中...</p>
          </div>
        </Card>
      ) : filteredReleases.length === 0 ? (
        <Card>
          <Empty
            description={
              <span className="text-gray-500">
                {searchQuery || statusFilter !== 'all' || runtimeFilter !== 'all'
                  ? '没有找到匹配的部署记录'
                  : '还没有任何部署记录'}
              </span>
            }
          >
            {!searchQuery && statusFilter === 'all' && runtimeFilter === 'all' && (
              <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/deployment/create')}>
                创建第一个部署
              </Button>
            )}
          </Empty>
        </Card>
      ) : (
        <Card title={<span className="text-base font-semibold">部署时间线</span>}>
          <Timeline
            mode="left"
            items={filteredReleases.map((release) => {
              const statusConfig = getStatusConfig(release.status);
              return {
                key: release.id,
                dot: (
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: statusConfig.dotColor }}
                  />
                ),
                color: statusConfig.dotColor,
                children: (
                  <StaggerItem>
                    <Card
                      size="small"
                      className="hover:shadow-md transition-shadow"
                      style={{ marginBottom: 16 }}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-3 mb-2">
                            <span className="text-base font-semibold text-gray-900">
                              Release #{release.id}
                            </span>
                            <Tag color={statusConfig.color} icon={statusConfig.icon}>
                              {statusConfig.text}
                            </Tag>
                            <Tag color="blue">{release.runtime_type}</Tag>
                            {release.trigger_source && (
                              <Tag color={release.trigger_source === 'ci' ? 'purple' : 'geekblue'}>
                                {release.trigger_source === 'ci' ? 'CI' : 'Manual'}
                              </Tag>
                            )}
                            {release.strategy && <Tag>{release.strategy}</Tag>}
                          </div>
                          <div className="flex flex-wrap gap-4 text-sm text-gray-600">
                            <span>服务: #{release.service_id}</span>
                            <span>目标: #{release.target_id}</span>
                            <span className="text-gray-400">
                              {new Date(release.created_at).toLocaleString()}
                            </span>
                          </div>
                          {release.diagnostics_json && (
                            <div className="mt-2 text-sm text-gray-500">
                              {(() => {
                                try {
                                  const parsed = JSON.parse(release.diagnostics_json);
                                  const first = Array.isArray(parsed) ? parsed[0] : parsed;
                                  return first?.summary || '';
                                } catch {
                                  return '';
                                }
                              })()}
                            </div>
                          )}
                        </div>
                        <Space>
                          <Tooltip title="查看详情">
                            <Button
                              type="text"
                              icon={<EyeOutlined />}
                              onClick={() => navigate(`/deployment/${release.id}`)}
                            />
                          </Tooltip>
                          {(release.status === 'succeeded' || release.status === 'applied') && (
                            <Tooltip title="回滚">
                              <Button
                                type="text"
                                icon={<RollbackOutlined />}
                                onClick={() => handleRollback(release.id)}
                              />
                            </Tooltip>
                          )}
                        </Space>
                      </div>
                    </Card>
                  </StaggerItem>
                ),
              };
            })}
          />
        </Card>
      )}
    </div>
  );
};

export default DeploymentListPage;
