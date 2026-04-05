import React, { useCallback, useEffect, useState } from 'react';
import { Alert, Button, Empty, Space, Typography } from 'antd';
import { ArrowLeftOutlined, AuditOutlined, ReloadOutlined, SafetyOutlined } from '@ant-design/icons';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { Api } from '../../../api';
import type {
  Cluster,
  ClusterOperationResponse,
  ClusterPolicyRelease,
  ClusterPolicySimulationResult,
} from '../../../api/modules/cluster';
import { DetailSkeleton } from '../../../components/LoadingSkeleton';
import PolicyTopologyPanel from '../../../components/ClusterPolicy/PolicyTopologyPanel';
import PolicySimulationDiffPanel from '../../../components/ClusterPolicy/PolicySimulationDiffPanel';
import PolicyReleasePanel from '../../../components/ClusterPolicy/PolicyReleasePanel';

const { Paragraph, Text, Title } = Typography;

type PolicyDraft = {
  namespace: string;
  name: string;
  scope: string;
  baseVersion: string;
  candidateVersion: string;
};

type FeedbackState = {
  type: 'success' | 'warning' | 'error' | 'info';
  title: string;
  description?: React.ReactNode;
};

const initialDraft: PolicyDraft = {
  namespace: 'prod',
  name: 'allow-api',
  scope: 'api',
  baseVersion: 'stable-v1',
  candidateVersion: 'candidate-v2',
};

const resolveRelease = (
  response: ClusterOperationResponse<{ release?: ClusterPolicyRelease }>,
): ClusterPolicyRelease | null => response.result?.release || null;

const ClusterPolicyCenterPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const clusterId = Number(id);

  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [loading, setLoading] = useState(true);
  const [simulationLoading, setSimulationLoading] = useState(false);
  const [releaseAction, setReleaseAction] = useState<'' | 'create' | 'apply' | 'rollback'>('');
  const [draft, setDraft] = useState<PolicyDraft>(initialDraft);
  const [simulation, setSimulation] = useState<ClusterPolicySimulationResult | null>(null);
  const [release, setRelease] = useState<ClusterPolicyRelease | null>(null);
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);

  const loadCluster = useCallback(async () => {
    if (!clusterId) {
      setCluster(null);
      setLoading(false);
      return;
    }

    setLoading(true);
    try {
      const response = await Api.cluster.getClusterDetail(clusterId);
      setCluster(response.data);
    } catch (error) {
      setCluster(null);
      setFeedback({
        type: 'error',
        title: '加载集群信息失败',
        description: error instanceof Error ? error.message : '请稍后重试',
      });
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  useEffect(() => {
    void loadCluster();
  }, [loadCluster]);

  const updateDraft = useCallback((field: keyof PolicyDraft, value: string) => {
    setDraft((current) => ({
      ...current,
      [field]: value,
    }));
  }, []);

  const handleSimulate = useCallback(async () => {
    if (!clusterId) return;

    setSimulationLoading(true);
    try {
      const response = await Api.cluster.simulatePolicy(clusterId, draft.namespace, draft.name, {
        base_version: draft.baseVersion || undefined,
        candidate_version: draft.candidateVersion,
        cluster: {
          namespaces: [draft.namespace],
        },
      });
      setSimulation(response.data);
      setFeedback(response.data.passed
        ? {
            type: 'success',
            title: '仿真已通过',
            description: `风险等级 ${response.data.risk_level || '-'}，风险分 ${response.data.risk_score ?? '-'}`,
          }
        : {
            type: 'warning',
            title: '仿真检测到阻断项',
            description: response.data.blocking_issues[0]?.message || '请修复阻断项后再发布',
          });
    } catch (error) {
      setFeedback({
        type: 'error',
        title: '仿真执行失败',
        description: error instanceof Error ? error.message : '请稍后重试',
      });
    } finally {
      setSimulationLoading(false);
    }
  }, [clusterId, draft.baseVersion, draft.candidateVersion, draft.name, draft.namespace]);

  const handleCreateRelease = useCallback(async () => {
    if (!clusterId) return;

    setReleaseAction('create');
    try {
      const response = await Api.cluster.createPolicyRelease(clusterId, draft.namespace, draft.name, {
        version: draft.candidateVersion,
        previous_stable_version: draft.baseVersion || undefined,
      });
      const nextRelease = resolveRelease(response.data);
      setRelease(nextRelease);
      setFeedback({
        type: 'success',
        title: '发布单已创建',
        description: nextRelease
          ? `release #${nextRelease.release_id} 已绑定候选版本 ${nextRelease.version}`
          : response.data.message,
      });
    } catch (error) {
      setFeedback({
        type: 'error',
        title: '创建发布单失败',
        description: error instanceof Error ? error.message : '请稍后重试',
      });
    } finally {
      setReleaseAction('');
    }
  }, [clusterId, draft.baseVersion, draft.candidateVersion, draft.name, draft.namespace]);

  const handleApplyRelease = useCallback(async () => {
    if (!release) {
      setFeedback({
        type: 'warning',
        title: '请先创建发布单',
        description: '当前还没有可以执行的策略发布单。',
      });
      return;
    }

    if (!simulation) {
      setFeedback({
        type: 'warning',
        title: '请先运行仿真',
        description: '发布前必须先确认仿真结果。',
      });
      return;
    }

    if (!simulation.passed || simulation.blocking_issues.length > 0) {
      setFeedback({
        type: 'error',
        title: '仿真存在阻断项，禁止发布',
        description: simulation.blocking_issues[0]?.message || '请先修复策略冲突后再发布。',
      });
      return;
    }

    setReleaseAction('apply');
    try {
      const response = await Api.cluster.applyPolicyRelease(clusterId, release.release_id);
      const nextRelease = resolveRelease(response.data) || release;
      setRelease(nextRelease);

      if (response.data.state === 'approval_required') {
        const approvalTicket = response.data.approval?.ticket || nextRelease.approval?.approval_token || '-';
        setFeedback({
          type: 'warning',
          title: '发布进入审批',
          description: approvalTicket,
        });
        return;
      }

      setFeedback({
        type: 'success',
        title: '发布已提交',
        description: response.data.message,
      });
    } catch (error) {
      setFeedback({
        type: 'error',
        title: '执行发布失败',
        description: error instanceof Error ? error.message : '请稍后重试',
      });
    } finally {
      setReleaseAction('');
    }
  }, [clusterId, release, simulation]);

  const handleRollbackRelease = useCallback(async () => {
    if (!release) {
      setFeedback({
        type: 'warning',
        title: '暂无可回滚发布单',
        description: '请先创建或执行一次发布。',
      });
      return;
    }

    const rollbackTarget = release.rollback_target_version || release.previous_stable_version || draft.baseVersion;
    if (!rollbackTarget) {
      setFeedback({
        type: 'warning',
        title: '缺少回滚目标版本',
        description: '当前发布单没有可用的稳定版本作为回滚目标。',
      });
      return;
    }

    setReleaseAction('rollback');
    try {
      const response = await Api.cluster.rollbackPolicyRelease(clusterId, release.release_id, {
        rollback_target_version: rollbackTarget,
      });
      setRelease(resolveRelease(response.data) || release);
      setFeedback({
        type: 'success',
        title: '回滚已提交',
        description: response.data.message,
      });
    } catch (error) {
      setFeedback({
        type: 'error',
        title: '回滚失败',
        description: error instanceof Error ? error.message : '请稍后重试',
      });
    } finally {
      setReleaseAction('');
    }
  }, [clusterId, draft.baseVersion, release]);

  if (loading) {
    return <DetailSkeleton />;
  }

  if (!cluster) {
    return (
      <div className="space-y-6">
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment/infrastructure/clusters')}>
          返回集群列表
        </Button>
        <Empty description="集群不存在" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-2">
          <Space wrap>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/deployment/infrastructure/clusters/${clusterId}`)}>
              返回集群
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void loadCluster()}>
              刷新
            </Button>
            <Link to={`/deployment/infrastructure/clusters/${clusterId}/operations`}>
              <Button icon={<AuditOutlined />}>查看审计</Button>
            </Link>
          </Space>
          <div>
            <Title level={2} className="!mb-1">集群策略中心</Title>
            <Paragraph className="!mb-0 text-gray-500">
              面向 {cluster.name} 的 NetworkPolicy 可视化编排、仿真校验与发布回滚控制面。
            </Paragraph>
          </div>
        </div>

        <Alert
          showIcon
          type="info"
          title="高风险保护"
          description={(
            <Space orientation="vertical" size={0}>
              <Text>仿真阻断项未清零前，发布操作会被直接拦截。</Text>
              <Text>审批流和回滚链路会沿用当前治理审计能力。</Text>
            </Space>
          )}
          icon={<SafetyOutlined />}
          className="lg:max-w-md"
        />
      </div>

      {feedback ? (
        <Alert
          showIcon
          type={feedback.type}
          title={feedback.title}
          description={feedback.description}
        />
      ) : null}

      <PolicyTopologyPanel
        clusterName={cluster.name}
        clusterStatus={cluster.status}
        draft={draft}
        release={release}
        simulation={simulation}
      />

      <PolicySimulationDiffPanel
        draft={draft}
        simulation={simulation}
        loading={simulationLoading}
        onDraftChange={updateDraft}
        onSimulate={handleSimulate}
      />

      <PolicyReleasePanel
        draft={draft}
        release={release}
        simulation={simulation}
        loadingAction={releaseAction}
        onCreateRelease={handleCreateRelease}
        onApplyRelease={handleApplyRelease}
        onRollbackRelease={handleRollbackRelease}
      />
    </div>
  );
};

export default ClusterPolicyCenterPage;
