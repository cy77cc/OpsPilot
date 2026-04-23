import React from 'react';
import { Card, Col, Row, Space, Tag } from 'antd';
import type { Cluster } from '../../../../api/modules/cluster';

type ClusterOverviewPanelProps = {
  cluster: Cluster;
  statusColor: string;
  children?: React.ReactNode;
};

const ClusterOverviewPanel: React.FC<ClusterOverviewPanelProps> = ({
  cluster,
  statusColor,
  children,
}) => (
  <div data-testid="cluster-overview-panel">
    <Card title="集群作战面板">
      <Row gutter={16}>
        <Col xs={24} md={16}>
          <Space size="middle" wrap>
            <Tag color={statusColor}>状态: {cluster.status}</Tag>
            <Tag color="blue">节点: {cluster.node_count}</Tag>
            <Tag color="geekblue">K8s: {cluster.k8s_version || cluster.version || '-'}</Tag>
          </Space>
        </Col>
        <Col xs={24} md={8}>
          {children}
        </Col>
      </Row>
    </Card>
  </div>
);

export default ClusterOverviewPanel;
