import React from 'react';
import { Alert, Button, Card, Descriptions, Space, Tag, Typography } from 'antd';
import type { ClusterPolicyRelease, ClusterPolicySimulationResult } from '../../api/modules/cluster';

const { Text } = Typography;

type PolicyDraft = {
  namespace: string;
  name: string;
  scope: string;
  baseVersion: string;
  candidateVersion: string;
};

type PolicyReleasePanelProps = {
  draft: PolicyDraft;
  release: ClusterPolicyRelease | null;
  simulation: ClusterPolicySimulationResult | null;
  loadingAction: '' | 'create' | 'apply' | 'rollback';
  onCreateRelease: () => Promise<void>;
  onApplyRelease: () => Promise<void>;
  onRollbackRelease: () => Promise<void>;
};

const getPhaseColor = (phase?: string) => {
  if (phase === 'applied' || phase === 'rollback_applied') return 'green';
  if (phase === 'approval_required') return 'orange';
  if (phase === 'draft') return 'blue';
  return 'default';
};

const PolicyReleasePanel: React.FC<PolicyReleasePanelProps> = ({
  draft,
  release,
  simulation,
  loadingAction,
  onCreateRelease,
  onApplyRelease,
  onRollbackRelease,
}) => {
  const rollbackTarget = release?.rollback_target_version || release?.previous_stable_version || draft.baseVersion;

  return (
    <Card
      title="发布操作"
      extra={(
        <Button onClick={() => void onCreateRelease()} loading={loadingAction === 'create'}>
          创建发布单
        </Button>
      )}
    >
      <Space orientation="vertical" size="large" className="w-full">
        <Descriptions bordered size="small" column={2}>
          <Descriptions.Item label="目标命名空间">{draft.namespace}</Descriptions.Item>
          <Descriptions.Item label="策略名称">{draft.name}</Descriptions.Item>
          <Descriptions.Item label="候选版本">{draft.candidateVersion}</Descriptions.Item>
          <Descriptions.Item label="稳定版本">{draft.baseVersion || '-'}</Descriptions.Item>
          <Descriptions.Item label="仿真状态">
            {simulation ? (
              <Tag color={simulation.passed ? 'green' : 'red'}>
                {simulation.passed ? 'passed' : 'blocked'}
              </Tag>
            ) : (
              <Text type="secondary">未执行</Text>
            )}
          </Descriptions.Item>
          <Descriptions.Item label="发布状态">
            {release?.status?.phase ? (
              <Tag color={getPhaseColor(release.status.phase)}>{release.status.phase}</Tag>
            ) : (
              <Text type="secondary">未创建</Text>
            )}
          </Descriptions.Item>
        </Descriptions>

        {release ? (
          <Alert
            showIcon
            type={release.approval?.required ? 'warning' : 'info'}
            title={`发布单 #${release.release_id}`}
            description={release.approval?.required
              ? `当前处于审批流，ticket: ${release.approval.approval_token || '-'}`
              : `已生成候选版本 ${release.version} 的发布单。`}
          />
        ) : (
          <Alert
            showIcon
            type="info"
            title="尚未创建发布单"
            description="先固化候选版本，再执行发布或回滚操作。"
          />
        )}

        <Space wrap>
          <Button
            type="primary"
            onClick={() => void onApplyRelease()}
            loading={loadingAction === 'apply'}
            disabled={!release}
          >
            执行发布
          </Button>
          <Button
            danger
            onClick={() => void onRollbackRelease()}
            loading={loadingAction === 'rollback'}
            disabled={!release || !rollbackTarget}
          >
            {rollbackTarget ? `回滚到 ${rollbackTarget}` : '回滚'}
          </Button>
        </Space>
      </Space>
    </Card>
  );
};

export default PolicyReleasePanel;
