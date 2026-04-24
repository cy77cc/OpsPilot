import React, { useState } from 'react';
import { Card, Tabs, Empty } from 'antd';
import { Line } from '@ant-design/charts';
import type { ResourcesOverview } from '../../../api/modules/dashboard';

export const ClusterUsage: React.FC<{ data?: ResourcesOverview }> = ({ data }) => {
  const [activeTab, setActiveTab] = useState('cpu');

  const chartData = activeTab === 'cpu' 
    ? data?.cpuUsage?.flatMap(s => s.data.map(p => ({ ...p, host: s.hostName }))) || []
    : data?.memoryUsage?.flatMap(s => s.data.map(p => ({ ...p, host: s.hostName }))) || [];

  const config = {
    data: chartData,
    xField: (d: any) => new Date(d.timestamp),
    yField: 'value',
    colorField: 'host',
    smooth: true,
    yAxis: { 
      label: { formatter: (v: any) => `${v}%` },
      min: 0,
      max: 100
    },
    legend: {
      color: { position: 'bottom' }
    },
  };

  const items = [
    { key: 'cpu', label: 'CPU' },
    { key: 'memory', label: '内存' },
  ];

  return (
    <Card 
      title="核心资源使用率" 
      className="h-full shadow-sm border-none flex flex-col" 
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, padding: '0 24px 24px' } }}
    >
      <div className="flex-1 overflow-auto min-h-0">
        <Tabs items={items} activeKey={activeTab} onChange={setActiveTab} />
        <div className="h-48 mt-2">
          {chartData.length > 0 ? <Line {...config} /> : <Empty description="暂无指标数据" className="mt-8" />}
        </div>
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看详情 &gt;
      </div>
    </Card>
  );
};
