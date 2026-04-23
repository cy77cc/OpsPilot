import React from 'react';
import { Button, Card, Space } from 'antd';
import { Link } from 'react-router-dom';

type ClusterOperationsPanelProps = {
  operationCenterHref: string;
  securityHref: string;
  policyHref: string;
  nodesLoading: boolean;
  onOpenOperationCenter: () => void;
  onSyncNodes: () => void | Promise<void>;
};

const ClusterOperationsPanel: React.FC<ClusterOperationsPanelProps> = ({
  operationCenterHref,
  securityHref,
  policyHref,
  nodesLoading,
  onOpenOperationCenter,
  onSyncNodes,
}) => (
  <div data-testid="cluster-operations-panel">
    <Card size="small" title="关键操作台">
      <Space direction="vertical" size={8} style={{ width: '100%' }}>
        <Button block onClick={onOpenOperationCenter}>
          进入操作中心
        </Button>
        <Button block onClick={() => { void onSyncNodes(); }} loading={nodesLoading}>
          同步节点
        </Button>
        <Link to={securityHref}>进入安全中心</Link>
        <Link to={policyHref}>进入策略中心</Link>
        <Link to={operationCenterHref}>查看全部操作</Link>
      </Space>
    </Card>
  </div>
);

export default ClusterOperationsPanel;
