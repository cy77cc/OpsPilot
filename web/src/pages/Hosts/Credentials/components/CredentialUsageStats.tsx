import React from 'react';
import { Row, Col, Statistic } from 'antd';
import type { CredentialDetail } from '../../../../api/modules/hosts';

export const CredentialUsageStats: React.FC<{ detail: CredentialDetail }> = ({ detail }) => {
  return (
    <div>
      <h3 className="font-semibold mb-2">使用统计 (近30天)</h3>
      <Row gutter={16}>
        <Col span={6}>
          <Statistic title="使用次数" value={detail.usageCount} />
        </Col>
        <Col span={6}>
          <Statistic title="成功次数" value={detail.successCount} valueStyle={{ color: '#3f8600' }} />
        </Col>
        <Col span={6}>
          <Statistic title="失败次数" value={detail.failureCount} valueStyle={{ color: '#cf1322' }} />
        </Col>
        <Col span={6}>
          <Statistic title="成功率" value={detail.successRate} suffix="%" />
        </Col>
      </Row>
    </div>
  );
};