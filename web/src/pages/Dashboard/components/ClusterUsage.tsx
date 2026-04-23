import React, { useState } from 'react';
import { Card, Tabs } from 'antd';
import { Line } from '@ant-design/charts';

export const ClusterUsage: React.FC = () => {
  const [activeTab, setActiveTab] = useState('cpu');

  const data = [
    { time: '10:00', value: 45, cluster: '北京生产' },
    { time: '10:15', value: 50, cluster: '北京生产' },
    { time: '10:30', value: 48, cluster: '北京生产' },
    { time: '10:45', value: 55, cluster: '北京生产' },
    { time: '11:00', value: 52, cluster: '北京生产' },
    { time: '10:00', value: 65, cluster: '上海生产' },
    { time: '10:15', value: 62, cluster: '上海生产' },
    { time: '10:30', value: 70, cluster: '上海生产' },
    { time: '10:45', value: 68, cluster: '上海生产' },
    { time: '11:00', value: 72, cluster: '上海生产' },
    { time: '10:00', value: 20, cluster: '测试环境' },
    { time: '10:15', value: 22, cluster: '测试环境' },
    { time: '10:30', value: 25, cluster: '测试环境' },
    { time: '10:45', value: 24, cluster: '测试环境' },
    { time: '11:00', value: 28, cluster: '测试环境' },
  ];

  const config = {
    data,
    xField: 'time',
    yField: 'value',
    colorField: 'cluster',
    scale: {
        color: {
            range: ['#1890ff', '#52c41a', '#722ed1']
        }
    },
    yAxis: {
      min: 0,
      max: 100,
      label: { formatter: (v: any) => `${v}%` }
    },
    legend: {
      color: { position: 'bottom' }
    },
    smooth: true,
  };

  const items = [
    { key: 'cpu', label: 'CPU' },
    { key: 'memory', label: '内存' },
    { key: 'disk', label: '磁盘' },
    { key: 'network', label: '网络' },
  ];

  return (
    <Card 
      title="集群资源使用率" 
      className="h-full shadow-sm border-none flex flex-col"
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, padding: '0 24px 24px' } }}
    >
      <div className="flex-1 overflow-auto min-h-0">
        <Tabs items={items} activeKey={activeTab} onChange={setActiveTab} />
        <div className="h-48 mt-2">
          <Line {...config} />
        </div>
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看详情 &gt;
      </div>
    </Card>
  );
};
