import React from 'react';
import { Card, Button, Typography } from 'antd';
import type { AIRun } from '../../api/modules/ai';

interface DiagnosisSummaryCardProps {
  run?: AIRun | null;
  onOpenReport: (reportId: string) => void;
}

const DiagnosisSummaryCard: React.FC<DiagnosisSummaryCardProps> = ({ run, onOpenReport }) => {
  if (!run?.report?.report_id) return null;

  return (
    <Card size="small" title="诊断摘要" style={{ borderRadius: 16 }}>
      <Typography.Paragraph>{run.report.summary || '诊断报告已生成。'}</Typography.Paragraph>
      <Button type="link" onClick={() => onOpenReport(run.report!.report_id)}>
        查看完整报告
      </Button>
    </Card>
  );
};

export default DiagnosisSummaryCard;
