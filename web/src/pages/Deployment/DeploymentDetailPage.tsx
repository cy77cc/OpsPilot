import React, { useState, useCallback, useEffect } from 'react';
import {
  Button,
  Card,
  Descriptions,
  Tag,
  message,
  Timeline,
  Alert,
  Space,
  Statistic,
  Row,
  Col,
  Empty,
  Modal,
  Tabs,
} from 'antd';
import {
  ArrowLeftOutlined,
  ReloadOutlined,
  RollbackOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  CheckOutlined,
  CloseOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { Api } from '../../api';
import type { DeployRelease, DeployReleaseTimelineEvent } from '../../api/modules/deployment';
import ReleaseStateFlow from '../../components/Deployment/ReleaseStateFlow';
import DeploymentProgressBar from '../../components/Deployment/DeploymentProgressBar';
import HealthCheckStatus from '../../components/Deployment/HealthCheckStatus';
import LiveLogViewer from '../../components/Deployment/LiveLogViewer';
import ApprovalWorkflow from '../../components/Deployment/ApprovalWorkflow';
import { usePolling } from '../../hooks/usePolling';
import { useCancelToken } from '../../hooks/useCancelToken';
import { handleApiError } from '../../utils/apiErrorHandler';

const DeploymentDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [loading, setLoading] = useState(false);
  const [release, setRelease] = useState<DeployRelease | null>(null);
  const [timeline, setTimeline] = useState<DeployReleaseTimelineEvent[]>([]);
  const { getSignal, isCancelled } = useCancelToken();

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const [releaseRes, timelineRes] = await Promise.all([
        Api.deployment.getReleasesByRuntime({}),
        Api.deployment.getReleaseTimeline(Number(id)),
      ]);

      const releaseList = releaseRes.data.list || [];
      const foundRelease = releaseList.find((r) => r.id === Number(id));
      setRelease(foundRelease || null);
      setTimeline(timelineRes.data.list || []);
    } catch (err) {
      if (!isCancelled(err)) {
        handleApiError(err, '加载部署详情失败');
      }
    } finally {
      setLoading(false);
    }
  }, [id, isCancelled]);

  useEffect(() => {
    void load();
  }, [load]);

  // Check if release is in terminal state
  const isTerminalState = (state: string) => {
    return ['applied', 'failed', 'rejected', 'rolled_back'].includes(state);
  };

  // Auto-refresh with polling - stop when in terminal state
  usePolling(load, {
    interval: 10000,
    enabled: release ? !isTerminalState(release.state) : false,
    shouldStop: () => release ? isTerminalState(release.state) : false,
  });

  // 获取状态配置
  const getStatusConfig = (status: string) => {
    const configs: Record<string, { icon: React.ReactNode; color: string; text: string }> = {
      succeeded: { icon: <CheckCircleOutlined />, color: 'success', text: '成功' },
      applied: { icon: <CheckCircleOutlined />, color: 'success', text: '已应用' },
      failed: { icon: <CloseCircleOutlined />, color: 'error', text: '失败' },
      pending_approval: { icon: <ClockCircleOutlined />, color: 'warning', text: '待审批' },
      rollback: { icon: <RollbackOutlined />, color: 'default', text: '已回滚' },
      rolled_back: { icon: <RollbackOutlined />, color: 'default', text: '已回滚' },
      rejected: { icon: <CloseOutlined />, color: 'error', text: '已拒绝' },
    };
    return (
      configs[status] || {
        icon: <ClockCircleOutlined />,
        color: 'processing',
        text: status,
      }
    );
  };

  // 审批操作
  const handleApprove = async () => {
    if (!release) return;
    try {
      await Api.deployment.approveRelease(release.id, {});
      message.success(`Release #${release.id} 已审批并执行`);
      await load();
    } catch (err) {
      handleApiError(err, '审批失败');
    }
  };

  const handleReject = async () => {
    if (!release) return;
    Modal.confirm({
      title: '确认拒绝',
      content: `确定要拒绝 Release #${release.id} 吗？`,
      okText: '确认拒绝',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        try {
          await Api.deployment.rejectRelease(release.id, {});
          message.success(`Release #${release.id} 已拒绝`);
          await load();
        } catch (err) {
          handleApiError(err, '拒绝失败');
        }
      },
    });
  };

  // 回滚操作
  const handleRollback = async () => {
    if (!release) return;
    Modal.confirm({
      title: '确认回滚',
      content: `确定要回滚 Release #${release.id} 吗？这将创建一个新的回滚部署。`,
      okText: '确认回滚',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        try {
          await Api.deployment.rollbackRelease(release.id);
          message.success(`回滚任务已提交，来源 Release #${release.id}`);
          await load();
        } catch (err) {
          handleApiError(err, '回滚失败');
        }
      },
    });
  };

  if (!release && !loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment')}>
            返回列表
          </Button>
        </div>
        <Card>
          <Empty description="未找到部署记录" />
        </Card>
      </div>
    );
  }

  const statusConfig = release ? getStatusConfig(release.status) : null;

  // 解析诊断信息
  const diagnostics = (() => {
    if (!release?.diagnostics_json) return null;
    try {
      const parsed = JSON.parse(release.diagnostics_json);
      return Array.isArray(parsed) ? parsed : [parsed];
    } catch {
      return null;
    }
  })();

  // 解析验证信息
  const verification = (() => {
    if (!release?.verification_json) return null;
    try {
      return JSON.parse(release.verification_json);
    } catch {
      return null;
    }
  })();

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment')}>
            返回列表
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-semibold text-gray-900">
                Release #{release?.id || id}
              </h1>
              {statusConfig && (
                <Tag color={statusConfig.color} icon={statusConfig.icon} className="text-sm">
                  {statusConfig.text}
                </Tag>
              )}
            </div>
            <p className="text-sm text-gray-500 mt-1">
              {release?.created_at && new Date(release.created_at).toLocaleString()}
            </p>
          </div>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          {release?.status === 'pending_approval' && (
            <>
              <Button type="primary" icon={<CheckOutlined />} onClick={handleApprove}>
                审批通过
              </Button>
              <Button danger icon={<CloseOutlined />} onClick={handleReject}>
                拒绝
              </Button>
            </>
          )}
          {(release?.status === 'succeeded' || release?.status === 'applied') && (
            <Button danger icon={<RollbackOutlined />} onClick={handleRollback}>
              回滚
            </Button>
          )}
        </Space>
      </div>

      {/* 统计卡片 */}
      {release && (
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Card className="hover:shadow-lg transition-shadow">
              <Statistic
                title={<span className="text-gray-600">服务</span>}
                value={release.service_name || release.service_id}
                valueStyle={{ color: '#495057', fontSize: '20px', fontWeight: 600 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card className="hover:shadow-lg transition-shadow">
              <Statistic
                title={<span className="text-gray-600">目标</span>}
                value={release.target_name || release.target_id}
                valueStyle={{ color: '#495057', fontSize: '20px', fontWeight: 600 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card className="hover:shadow-lg transition-shadow">
              <Statistic
                title={<span className="text-gray-600">运行时</span>}
                value={release.runtime_type}
                valueStyle={{ color: '#6366f1', fontSize: '20px', fontWeight: 600 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card className="hover:shadow-lg transition-shadow">
              <Statistic
                title={<span className="text-gray-600">策略</span>}
                value={release.strategy || 'rolling'}
                valueStyle={{ color: '#495057', fontSize: '20px', fontWeight: 600 }}
              />
            </Card>
          </Col>
        </Row>
      )}

      {/* Release state flow */}
      {release && (
        <ReleaseStateFlow currentState={release.state || release.status} />
      )}

      {/* Real-time progress for in-progress releases */}
      {release && release.state === 'applying' && (
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <DeploymentProgressBar
              phase={release.phase}
              progress={release.progress}
              pods={release.pods as any}
              runtimeType={release.runtime_type as 'k8s' | 'compose'}
            />
          </Col>
          <Col xs={24} lg={12}>
            <HealthCheckStatus probes={release.health_probes as any || []} />
          </Col>
        </Row>
      )}

      {/* Live logs for in-progress releases */}
      {release && release.state === 'applying' && release.logs && release.logs.length > 0 && (
        <LiveLogViewer logs={release.logs} title="部署日志" />
      )}

      {/* Approval workflow for pending releases */}
      {release && release.state === 'pending_approval' && release.approval_info && (
        <ApprovalWorkflow
          approval={release.approval_info}
          status="pending"
        />
      )}

      {/* Approval workflow for approved/rejected releases */}
      {release && (release.state === 'approved' || release.state === 'rejected') && release.approval_info && (
        <ApprovalWorkflow
          approval={release.approval_info}
          status={release.state === 'approved' ? 'approved' : 'rejected'}
        />
      )}

      {/* Rollback source info */}
      {release && release.source_release_id && (
        <Card>
          <Alert
            message="回滚发布"
            description={
              <div>
                此发布是从 Release{' '}
                <a onClick={() => navigate(`/deployment/${release.source_release_id}`)}>
                  #{release.source_release_id}
                </a>{' '}
                回滚而来。
              </div>
            }
            type="info"
            showIcon
          />
        </Card>
      )}

      {/* 基本信息 */}
      {release && (
        <Card title={<span className="text-base font-semibold">基本信息</span>}>
          <Descriptions bordered column={2}>
            <Descriptions.Item label="Release ID">{release.id}</Descriptions.Item>
            <Descriptions.Item label="状态">
              {statusConfig && (
                <Tag color={statusConfig.color} icon={statusConfig.icon}>
                  {statusConfig.text}
                </Tag>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="服务 ID">{release.service_id}</Descriptions.Item>
            <Descriptions.Item label="目标 ID">{release.target_id}</Descriptions.Item>
            <Descriptions.Item label="运行时">
              <Tag color="blue">{release.runtime_type}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="部署策略">
              <Tag>{release.strategy || '-'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="源 Release">
              {release.source_release_id || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="目标 Revision">
              {release.target_revision || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {new Date(release.created_at).toLocaleString()}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {/* 诊断信息 */}
      {diagnostics && diagnostics.length > 0 && (
        <Card title={<span className="text-base font-semibold">诊断信息</span>}>
          <Space direction="vertical" className="w-full">
            {diagnostics.map((diag, idx) => (
              <Alert
                key={idx}
                type={diag.level === 'error' ? 'error' : diag.level === 'warning' ? 'warning' : 'info'}
                showIcon
                message={diag.summary || '诊断信息'}
                description={diag.details || diag.message}
              />
            ))}
          </Space>
        </Card>
      )}

      {/* 验证信息 */}
      {verification && Object.keys(verification).length > 0 && (
        <Card title={<span className="text-base font-semibold">验证信息</span>}>
          <pre className="bg-gray-50 p-4 rounded-lg text-sm overflow-auto">
            {JSON.stringify(verification, null, 2)}
          </pre>
        </Card>
      )}

      {/* 时间线 */}
      <Card title={<span className="text-base font-semibold">部署时间线</span>}>
        {timeline.length > 0 ? (
          <Timeline
            items={timeline.map((item) => ({
              color: item.action.includes('failed') || item.action.includes('rejected') ? 'red' : 'blue',
              children: (
                <div>
                  <div className="font-medium text-gray-900">{item.action}</div>
                  <div className="text-sm text-gray-500">
                    {new Date(item.created_at).toLocaleString()}
                    {item.actor && ` · Actor #${item.actor}`}
                  </div>
                </div>
              ),
            }))}
          />
        ) : (
          <Empty description="暂无时间线记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
      </Card>
    </div>
  );
};

export default DeploymentDetailPage;
