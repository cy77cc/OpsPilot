import React from 'react';
import { Card, Col, Row, Statistic } from 'antd';
import { TransactionOutlined, ClockCircleOutlined, ApiOutlined, DollarOutlined } from '@ant-design/icons';
import type { UsageStats } from '../../../../api/modules/ai';

interface StatCardsProps {
  stats?: UsageStats;
  loading: boolean;
}

const StatCards: React.FC<StatCardsProps> = ({ stats, loading }) => {
  return (
    <Row gutter={16}>
      <Col span={6}>
        <Card loading={loading}>
          <Statistic
            title="总请求数"
            value={stats?.total_requests || 0}
            prefix={<ApiOutlined />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card loading={loading}>
          <Statistic
            title="总 Token"
            value={stats?.total_tokens || 0}
            prefix={<TransactionOutlined />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card loading={loading}>
          <Statistic
            title="总费用"
            value={stats?.total_cost_usd || 0}
            precision={2}
            prefix={<DollarOutlined />}
            suffix="USD"
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card loading={loading}>
          <Statistic
            title="平均延迟"
            value={stats?.avg_first_token_ms || 0}
            precision={0}
            prefix={<ClockCircleOutlined />}
            suffix="ms"
          />
        </Card>
      </Col>
    </Row>
  );
};

export default StatCards;
