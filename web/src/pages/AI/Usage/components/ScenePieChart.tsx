import React from 'react';
import { Card, Empty, Skeleton } from 'antd';
import { Pie } from '@ant-design/charts';
import type { SceneStats } from '../../../../api/modules/ai';

interface ScenePieChartProps {
  data?: SceneStats[];
  loading: boolean;
}

const ScenePieChart: React.FC<ScenePieChartProps> = ({ data, loading }) => {
  const hasData = data && data.length > 0;

  const config = {
    data: data || [],
    angleField: 'count',
    colorField: 'scene',
    radius: 0.8,
    height: 260,
    label: {
      type: 'outer',
      content: '{name} {percentage}',
    },
    legend: {
      position: 'bottom' as const,
    },
  };

  return (
    <Card title="场景分布" style={{ marginTop: 16 }}>
      {loading ? <Skeleton active paragraph={{ rows: 8 }} /> : null}
      {!loading && !hasData ? <Empty description="暂无数据" /> : null}
      {!loading && hasData ? <Pie {...config} /> : null}
    </Card>
  );
};

export default ScenePieChart;
