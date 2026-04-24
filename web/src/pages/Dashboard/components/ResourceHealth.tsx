import React from 'react';
import { Card } from 'antd';
import { Pie } from '@ant-design/charts';
import type { HealthOverview } from '../../../api/modules/dashboard';

export const ResourceHealth: React.FC<{ data?: HealthOverview }> = ({ data }) => {
  const chartData = [
    { type: '正常', value: data?.hosts?.healthy || 0 },
    { type: '降级', value: data?.hosts?.degraded || 0 },
    { type: '异常', value: data?.hosts?.unhealthy || 0 },
    { type: '离线', value: data?.hosts?.offline || 0 },
  ];

  const config = {
    data: chartData,
    angleField: 'value',
    colorField: 'type',
    innerRadius: 0.75,
    scale: {
      color: {
        range: ['#52c41a', '#faad14', '#f5222d', '#8c8c8c'],
      },
    },
    label: {
      text: 'value',
      style: { fontWeight: 'bold' },
    },
    legend: {
      color: { position: 'right', rowPadding: 5 },
    },
    annotations: [
      {
        type: 'text',
        style: {
          text: `${data?.hosts?.total || 0}\n总主机`,
          x: '50%', y: '50%', textAlign: 'center', fontSize: 20, fontStyle: 'bold',
        },
      },
    ],
  };

  return (
    <Card 
      title="主机健康状态" 
      className="h-full shadow-sm border-none flex flex-col"
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}
    >
      <div className="flex-1 overflow-auto min-h-0">
        <div className="h-48">
           <Pie {...config} />
        </div>
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看拓扑 &gt;
      </div>
    </Card>
  );
};
