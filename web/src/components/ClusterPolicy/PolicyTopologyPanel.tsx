import React from 'react';
import { Card, Col, Descriptions, Row, Space, Tag, Typography } from 'antd';
import type { ClusterPolicyRelease, ClusterPolicySimulationResult } from '../../api/modules/cluster';

const { Text } = Typography;

type PolicyDraft = {
  namespace: string;
  name: string;
  scope: string;
  baseVersion: string;
  candidateVersion: string;
};

type PolicyTopologyPanelProps = {
  clusterName?: string;
  clusterStatus?: string;
  draft: PolicyDraft;
  release: ClusterPolicyRelease | null;
  simulation: ClusterPolicySimulationResult | null;
};

const getStatusTagColor = (status?: string) => {
  if (status === 'active') return 'green';
  if (status === 'provisioning') return 'blue';
  if (status === 'error') return 'red';
  return 'default';
};

const PolicyTopologyPanel: React.FC<PolicyTopologyPanelProps> = ({
  clusterName,
  clusterStatus,
  draft,
  release,
  simulation,
}) => {
  const impactedNamespaces = simulation?.impact_summary?.affected_namespaces.length
    ? simulation.impact_summary.affected_namespaces
    : [draft.namespace];
  const deniedFlows = simulation?.impact_summary?.new_denied_flows.length
    ? simulation.impact_summary.new_denied_flows
    : [`${draft.scope} -> kube-dns`];

  return (
    <Card
      title="策略拓扑"
      extra={clusterStatus ? <Tag color={getStatusTagColor(clusterStatus)}>{clusterStatus}</Tag> : null}
    >
      <Space orientation="vertical" size="large" className="w-full">
        <Descriptions bordered size="small" column={2}>
          <Descriptions.Item label="集群">{clusterName || '-'}</Descriptions.Item>
          <Descriptions.Item label="命名空间">{draft.namespace}</Descriptions.Item>
          <Descriptions.Item label="策略名称">{draft.name}</Descriptions.Item>
          <Descriptions.Item label="工作负载">{draft.scope}</Descriptions.Item>
          <Descriptions.Item label="基线版本">{draft.baseVersion || '-'}</Descriptions.Item>
          <Descriptions.Item label="候选版本">{draft.candidateVersion}</Descriptions.Item>
        </Descriptions>

        <Row gutter={[16, 16]}>
          <Col xs={24} lg={8}>
            <Card size="small" title="命名空间关系">
              <Space wrap>
                {impactedNamespaces.map((namespace) => (
                  <Tag key={namespace} color={namespace === draft.namespace ? 'blue' : 'default'}>
                    {namespace}
                  </Tag>
                ))}
              </Space>
              <div className="mt-3 text-sm text-gray-500">
                当前视图聚焦 `{draft.namespace}` 命名空间及其受影响边界。
              </div>
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card size="small" title="工作负载覆盖">
              <Space orientation="vertical" size="small">
                <Tag color="processing">{draft.scope}</Tag>
                <Text type="secondary">
                  策略 `{draft.name}` 正在覆盖入口工作负载，并联动下游流量访问控制。
                </Text>
              </Space>
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card size="small" title="预估受影响流量">
              <Space wrap>
                {deniedFlows.map((flow) => (
                  <Tag key={flow} color="volcano">
                    {flow}
                  </Tag>
                ))}
              </Space>
              <div className="mt-3 text-sm text-gray-500">
                {release
                  ? `当前发布单 #${release.release_id} 绑定候选版本 ${release.version}。`
                  : '尚未创建发布单，当前为草稿拓扑预览。'}
              </div>
            </Card>
          </Col>
        </Row>
      </Space>
    </Card>
  );
};

export default PolicyTopologyPanel;
