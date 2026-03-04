import React, { useMemo } from 'react';
import { Card, Empty, Skeleton } from 'antd';
import { Line } from '@ant-design/charts';
import dayjs from 'dayjs';
import type { MetricPoint } from '../../api/modules/dashboard';

interface TimeseriesChartProps {
  title: string;
  data: MetricPoint[];
  loading?: boolean;
}

const TimeseriesChart: React.FC<TimeseriesChartProps> = ({ title, data, loading }) => {
  const chartData = useMemo(() => data.map((item) => ({
    time: dayjs(item.timestamp).format('HH:mm'),
    value: Number(item.value || 0),
    ts: item.timestamp,
  })), [data]);

  const avg = useMemo(() => {
    if (chartData.length === 0) {
      return 0;
    }
    const total = chartData.reduce((sum, item) => sum + item.value, 0);
    return Number((total / chartData.length).toFixed(2));
  }, [chartData]);

  const config = {
    data: chartData,
    xField: 'time',
    yField: 'value',
    smooth: true,
    height: 260,
    tooltip: {
      title: (_: any, datum: any) => dayjs(datum?.ts).format('YYYY-MM-DD HH:mm:ss'),
      items: (datum: any) => [{
        name: title,
        value: `${datum?.value?.toFixed?.(2) ?? datum?.value}%`,
      }],
    },
    annotations: avg > 0 ? [
      {
        type: 'lineY',
        yField: avg,
        style: { stroke: '#9ca3af', lineDash: [6, 4] },
        text: {
          content: `平均值 ${avg}%`,
          style: { fill: '#6b7280' },
          position: 'right',
        },
      },
    ] : [],
  };

  return (
    <Card title={title} styles={{ body: { minHeight: 300 } }}>
      {loading ? <Skeleton active paragraph={{ rows: 8 }} /> : null}
      {!loading && chartData.length === 0 ? <Empty description="暂无数据" /> : null}
      {!loading && chartData.length > 0 ? <Line {...config} /> : null}
    </Card>
  );
};

export default TimeseriesChart;
