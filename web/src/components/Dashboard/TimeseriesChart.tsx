import React, { useMemo } from 'react';
import { Card, Empty, Skeleton } from 'antd';
import { Line } from '@ant-design/charts';
import dayjs from 'dayjs';
import type { MetricSeries } from '../../api/modules/dashboard';

interface TimeseriesChartProps {
  title: string;
  series: MetricSeries[];
  loading?: boolean;
}

// Color palette for multiple hosts
const HOST_COLORS = [
  '#1890ff', '#52c41a', '#faad14', '#eb2f96', '#722ed1',
  '#13c2c2', '#fa8c16', '#a0d911', '#2f54eb', '#f5222d',
];

const TimeseriesChart: React.FC<TimeseriesChartProps> = ({ title, series, loading }) => {
  const chartData = useMemo(() => {
    const result: Array<{
      time: string;
      value: number;
      ts: string;
      host: string;
      color: string;
    }> = [];
    series.forEach((s, idx) => {
      const hostName = s.hostName || `主机 ${s.hostId}`;
      const color = HOST_COLORS[idx % HOST_COLORS.length];
      s.data.forEach((item) => {
        result.push({
          time: dayjs(item.timestamp).format('HH:mm'),
          value: Number(item.value || 0),
          ts: item.timestamp,
          host: hostName,
          color,
        });
      });
    });
    return result;
  }, [series]);

  const hasData = chartData.length > 0;

  const config = {
    data: chartData,
    xField: 'time',
    yField: 'value',
    colorField: 'host',
    seriesField: 'host',
    smooth: true,
    height: 260,
    legend: {
      position: 'top' as const,
      itemSpacing: 16,
    },
    // Shared tooltip shows all hosts at the hovered time point
    tooltip: {
      shared: true,
      showCrosshairs: true,
      crosshairs: {
        type: 'x' as const,
        line: {
          style: {
            stroke: '#9ca3af',
            lineDash: [4, 4],
          },
        },
      },
      title: (title: string, datum: any) => {
        if (datum?.ts) {
          return dayjs(datum.ts).format('YYYY-MM-DD HH:mm:ss');
        }
        return title;
      },
      customItems: (originalItems: any[]) => {
        return originalItems.map((item) => ({
          ...item,
          name: item.data?.host || item.name,
          value: `${(item.data?.value ?? item.value)?.toFixed?.(2) ?? item.value}%`,
          marker: {
            style: { fill: item.data?.color || item.color },
          },
        }));
      },
    },
    scale: {
      color: {
        range: HOST_COLORS,
      },
    },
  };

  return (
    <Card title={title} styles={{ body: { minHeight: 300 } }}>
      {loading ? <Skeleton active paragraph={{ rows: 8 }} /> : null}
      {!loading && !hasData ? <Empty description="暂无数据" /> : null}
      {!loading && hasData ? <Line {...config} /> : null}
    </Card>
  );
};

export default TimeseriesChart;
