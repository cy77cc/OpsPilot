import React from 'react';
import { Card, List, Typography } from 'antd';
import type { AIDiagnosisReport } from '../../api/modules/ai';

interface DiagnosisReportViewProps {
  report: AIDiagnosisReport;
}

const Section: React.FC<{ title: string; items?: string[] }> = ({ title, items }) => (
  <Card size="small" title={title} style={{ borderRadius: 16 }}>
    <List
      dataSource={items || []}
      locale={{ emptyText: '暂无内容' }}
      renderItem={(item) => <List.Item>{item}</List.Item>}
    />
  </Card>
);

const DiagnosisReportView: React.FC<DiagnosisReportViewProps> = ({ report }) => {
  return (
    <div style={{ display: 'grid', gap: 16 }}>
      <Card style={{ borderRadius: 20 }}>
        <Typography.Title level={3}>诊断报告</Typography.Title>
        <Typography.Paragraph>{report.summary || '暂无摘要'}</Typography.Paragraph>
      </Card>
      <Section title="Evidence" items={report.evidence} />
      <Section title="Root Causes" items={report.root_causes} />
      <Section title="Recommendations" items={report.recommendations} />
    </div>
  );
};

export default DiagnosisReportView;
