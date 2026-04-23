import React from 'react';
import { Button, Card, Space } from 'antd';
import { Link, useNavigate } from 'react-router-dom';

type ClusterOperationsPanelProps = {
  clusterId: number;
  nodesLoading: boolean;
  onSyncNodes: () => void | Promise<void>;
};

const ClusterOperationsPanel: React.FC<ClusterOperationsPanelProps> = ({
  clusterId,
  nodesLoading,
  onSyncNodes,
}) => {
  const navigate = useNavigate();

  return (
    <div data-testid="cluster-operations-panel">
      <Card size="small" title="关键操作台">
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          <Button block onClick={() => navigate(`/deployment/infrastructure/clusters/${clusterId}/operations`)}>
            进入操作中心
          </Button>
          <Button block onClick={() => { void onSyncNodes(); }} loading={nodesLoading}>
            同步节点
          </Button>
          <Link to={`/deployment/infrastructure/clusters/${clusterId}/security`}>进入安全中心</Link>
          <Link to={`/deployment/infrastructure/clusters/${clusterId}/policies`}>进入策略中心</Link>
          <Link to={`/deployment/infrastructure/clusters/${clusterId}/operations`}>查看全部操作</Link>
        </Space>
      </Card>
    </div>
  );
};

export default ClusterOperationsPanel;
