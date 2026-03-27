import React from 'react';
import { Card, Col, Row, Statistic, Tag } from 'antd';

interface Props {
  nodes: any[];
  deployments: any[];
  pods: any[];
  services: any[];
  ingresses: any[];
  dataSourceHint?: string;
}

const ClusterOverview: React.FC<Props> = ({ nodes, deployments, pods, services, ingresses, dataSourceHint }) => {
  const runningPods = pods.filter((p) => String(p.status || p.phase).toLowerCase() === 'running').length;
  return (
    <div>
      {dataSourceHint ? <Tag color={dataSourceHint.includes('live') ? 'success' : 'warning'}>data_source: {dataSourceHint}</Tag> : null}
      <Row gutter={[12, 12]} style={{ marginTop: 12 }}>
        <Col span={8}><Card><Statistic title="Nodes" value={nodes.length} /></Card></Col>
        <Col span={8}><Card><Statistic title="Deployments" value={deployments.length} /></Card></Col>
        <Col span={8}><Card><Statistic title="Pods" value={pods.length} suffix={`/ running ${runningPods}`} /></Card></Col>
        <Col span={12}><Card><Statistic title="Services" value={services.length} /></Card></Col>
        <Col span={12}><Card><Statistic title="Ingresses" value={ingresses.length} /></Card></Col>
      </Row>
    </div>
  );
};

export default ClusterOverview;
