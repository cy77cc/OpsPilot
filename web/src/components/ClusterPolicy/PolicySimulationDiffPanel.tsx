import React from 'react';
import { Alert, Button, Card, Descriptions, Form, Input, Space, Tag, Typography } from 'antd';
import type { ClusterPolicySimulationResult } from '../../api/modules/cluster';

const { Text } = Typography;

type PolicyDraft = {
  namespace: string;
  name: string;
  scope: string;
  baseVersion: string;
  candidateVersion: string;
};

type PolicySimulationDiffPanelProps = {
  draft: PolicyDraft;
  simulation: ClusterPolicySimulationResult | null;
  loading: boolean;
  onDraftChange: (field: keyof PolicyDraft, value: string) => void;
  onSimulate: () => Promise<void>;
};

const getRiskColor = (riskLevel?: string) => {
  if (riskLevel === 'CRITICAL') {return 'red';}
  if (riskLevel === 'HIGH') {return 'volcano';}
  if (riskLevel === 'MEDIUM') {return 'gold';}
  if (riskLevel === 'LOW') {return 'green';}
  return 'default';
};

const PolicySimulationDiffPanel: React.FC<PolicySimulationDiffPanelProps> = ({
  draft,
  simulation,
  loading,
  onDraftChange,
  onSimulate,
}) => (
  <Card
    title="仿真 Diff"
    extra={(
      <Button type="primary" onClick={() => void onSimulate()} loading={loading}>
        运行仿真
      </Button>
    )}
  >
      <Space orientation="vertical" size="large" className="w-full">
      <Form layout="vertical">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <Form.Item label="命名空间" className="mb-0">
            <Input
              aria-label="namespace"
              value={draft.namespace}
              onChange={(event) => onDraftChange('namespace', event.target.value)}
            />
          </Form.Item>
          <Form.Item label="策略名称" className="mb-0">
            <Input
              aria-label="policy_name"
              value={draft.name}
              onChange={(event) => onDraftChange('name', event.target.value)}
            />
          </Form.Item>
          <Form.Item label="基线版本" className="mb-0">
            <Input
              aria-label="base_version"
              value={draft.baseVersion}
              onChange={(event) => onDraftChange('baseVersion', event.target.value)}
            />
          </Form.Item>
          <Form.Item label="候选版本" className="mb-0">
            <Input
              aria-label="candidate_version"
              value={draft.candidateVersion}
              onChange={(event) => onDraftChange('candidateVersion', event.target.value)}
            />
          </Form.Item>
        </div>
      </Form>

      {simulation ? (
        <Space orientation="vertical" size="middle" className="w-full">
          <Alert
            showIcon
            type={simulation.passed ? 'success' : 'error'}
            title={simulation.passed ? '仿真已通过' : '检测到阻断项'}
            description={(
              <Space wrap>
                <Text>风险分: {simulation.risk_score ?? '-'}</Text>
                <Tag color={getRiskColor(simulation.risk_level)}>{simulation.risk_level || 'UNKNOWN'}</Tag>
              </Space>
            )}
          />

          <Descriptions bordered size="small" column={3}>
            <Descriptions.Item label="影响 Pod">{simulation.impact_summary?.affected_pods ?? 0}</Descriptions.Item>
            <Descriptions.Item label="影响命名空间">
              {simulation.impact_summary?.affected_namespaces.join(', ') || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="新增拒绝流量">
              {simulation.impact_summary?.new_denied_flows.join(', ') || '-'}
            </Descriptions.Item>
          </Descriptions>

          <div>
            <div className="mb-2 font-medium text-gray-900">阻断项</div>
            {simulation.blocking_issues.length ? (
              <div className="space-y-2 rounded-lg border border-gray-200 p-3">
                {simulation.blocking_issues.map((issue) => (
                  <div key={`${issue.code || issue.message}-${issue.suggestion || ''}`} className="rounded border border-red-100 bg-red-50 px-3 py-2">
                    <Space orientation="vertical" size={0}>
                      <Text strong>{issue.message}</Text>
                      <Text type="secondary">
                        {issue.code || 'UNSPECIFIED'}
                        {issue.suggestion ? ` · ${issue.suggestion}` : ''}
                      </Text>
                    </Space>
                  </div>
                ))}
              </div>
            ) : (
              <Text type="secondary">无阻断项。</Text>
            )}
          </div>

          <div>
            <div className="mb-2 font-medium text-gray-900">兼容性提示</div>
            {simulation.warnings.length ? (
              <div className="space-y-2 rounded-lg border border-gray-200 p-3">
                {simulation.warnings.map((warning) => (
                  <div key={`${warning.code || warning.message}`} className="rounded border border-amber-100 bg-amber-50 px-3 py-2">
                    <Text>{warning.message}</Text>
                  </div>
                ))}
              </div>
            ) : (
              <Text type="secondary">无兼容性提示。</Text>
            )}
          </div>
        </Space>
      ) : (
        <Alert
          showIcon
          type="info"
          title="尚未运行仿真"
          description="运行仿真后，这里会展示阻断项、风险分和影响面 Diff。"
        />
      )}
    </Space>
  </Card>
);

export default PolicySimulationDiffPanel;
