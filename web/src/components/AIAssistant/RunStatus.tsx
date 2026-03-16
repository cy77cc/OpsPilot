import React from 'react';
import { Card, Tag, Typography } from 'antd';
import type { AIRun } from '../../api/modules/ai';

interface RunStatusProps {
  run?: AIRun | null;
}

const RunStatus: React.FC<RunStatusProps> = ({ run }) => {
  if (!run) return null;

  return (
    <Card size="small" title="当前运行" style={{ borderRadius: 16 }}>
      <Typography.Paragraph style={{ marginBottom: 8 }}>
        状态: <Tag color={run.status === 'completed' ? 'green' : 'blue'}>{run.status}</Tag>
      </Typography.Paragraph>
      {run.intent_type ? <Typography.Text>意图: {run.intent_type}</Typography.Text> : null}
      {run.progress_summary ? <Typography.Paragraph style={{ marginTop: 8, marginBottom: 0 }}>{run.progress_summary}</Typography.Paragraph> : null}
    </Card>
  );
};

export default RunStatus;
