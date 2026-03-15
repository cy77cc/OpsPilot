import React from 'react';
import { Card, Empty, Skeleton } from 'antd';
import { Line } from '@ant-design/charts';
import type { DateStats } from '../../../../api/modules/ai';

interface UsageTrendChartProps {
  data?: DateStats[];
  loading: boolean;
}

const UsageTrendChart: React.FC<UsageTrendChartProps> = ({ data, loading }) => {
  const hasData = data && data.length > 0;

  const config = {
    data: data || [],
    xField: 'date',
    yField: 'tokens',
    smooth: true,
    height: 260,
    point: {
      size: 4,
      shape: 'circle',
    },
    tooltip: {
      fields: ['date', 'tokens', 'requests'],
    },
  };

  return (
    <Card title="使用趋势" style={{ marginTop: 16 }}>
      {loading ? <Skeleton active paragraph={{ rows: 8 }} /> : null}
      {!loading && !hasData ? <Empty description="暂无数据" /> : null}
      {!loading && hasData ? <Line {...config} /> : null}
    </Card>
  );
};

export default UsageTrendChart;
