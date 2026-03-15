import React, { useMemo } from 'react';
import { Card, Empty, Skeleton, Segmented } from 'antd';
import { Line } from '@ant-design/charts';
import dayjs from 'dayjs';
import type { MetricSeries } from '../../api/modules/dashboard';

interface HostMetricsCardProps {
  cpuSeries: MetricSeries[];
  memorySeries: MetricSeries[];
  loading?: boolean;
}

// Color palette for multiple hosts
const HOST_COLORS = [
  '#1890ff', '#52c41a', '#faad14', '#eb2f96', '#722ed1',
  '#13c2c2', '#fa8c16', '#a0d911', '#2f54eb', '#f5222d',
];

type MetricType = 'cpu' | 'memory';

const HostMetricsCard: React.FC<HostMetricsCardProps> = ({ cpuSeries, memorySeries, loading }) => {
  const [metricType, setMetricType] = React.useState<MetricType>('cpu');

  const series = metricType === 'cpu' ? cpuSeries : memorySeries;

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
      title: (_title: string, datum: any) => {
        if (datum?.ts) {
          return dayjs(datum.ts).format('YYYY-MM-DD HH:mm:ss');
        }
        return _title;
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
    <Card
      title="主机资源使用率"
      styles={{ body: { minHeight: 300 } }}
      extra={
        <Segmented
          value={metricType}
          onChange={(value) => setMetricType(value as MetricType)}
          options={[
            { label: 'CPU', value: 'cpu' },
            { label: '内存', value: 'memory' },
          ]}
          size="small"
        />
      }
    >
      {loading ? <Skeleton active paragraph={{ rows: 8 }} /> : null}
      {!loading && !hasData ? <Empty description="暂无数据" /> : null}
      {!loading && hasData ? <Line {...config} /> : null}
    </Card>
  );
};

export default HostMetricsCard;
