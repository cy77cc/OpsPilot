import React from 'react';
import { Card, Progress, Row, Col, Typography, Skeleton } from 'antd';
import type { UsageStats } from '../../../../api/modules/ai';

interface ApprovalChartProps {
  stats?: UsageStats;
  loading: boolean;
}

const ApprovalChart: React.FC<ApprovalChartProps> = ({ stats, loading }) => {
  if (loading) {
    return (
      <Card title="审批统计" style={{ marginTop: 16 }}>
        <Skeleton active paragraph={{ rows: 4 }} />
      </Card>
    );
  }

  const passRate = stats?.approval_pass_rate || 0;
  const triggerRate = stats?.approval_rate || 0;

  return (
    <Card title="审批统计" style={{ marginTop: 16 }}>
      <Row gutter={16}>
        <Col span={12}>
          <div style={{ textAlign: 'center' }}>
            <Typography.Text type="secondary">审批通过率</Typography.Text>
            <div style={{ marginTop: 16 }}>
              <Progress
                type="circle"
                percent={passRate * 100}
                format={(percent) => `${percent?.toFixed(1)}%`}
              />
            </div>
          </div>
        </Col>
        <Col span={12}>
          <div style={{ textAlign: 'center' }}>
            <Typography.Text type="secondary">审批触发率</Typography.Text>
            <div style={{ marginTop: 16 }}>
              <Progress
                type="circle"
                percent={triggerRate * 100}
                format={(percent) => `${percent?.toFixed(1)}%`}
                strokeColor="#faad14"
              />
            </div>
          </div>
        </Col>
      </Row>
    </Card>
  );
};

export default ApprovalChart;
