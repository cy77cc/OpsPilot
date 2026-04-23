import React from 'react';
import { Card } from 'antd';
import { Pie } from '@ant-design/charts';

export const ResourceHealth: React.FC = () => {
  const data = [
    { type: '正常', value: 118 },
    { type: '警告', value: 30 },
    { type: '异常', value: 14 },
    { type: '未知', value: 10 },
  ];

  const config = {
    data,
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
      style: {
        fontWeight: 'bold',
      },
    },
    legend: {
      color: {
        title: false,
        position: 'right',
        rowPadding: 5,
      },
    },
    annotations: [
      {
        type: 'text',
        style: {
          text: '172\n总资源',
          x: '50%',
          y: '50%',
          textAlign: 'center',
          fontSize: 20,
          fontStyle: 'bold',
        },
      },
    ],
  };

  return (
    <Card title="资源健康状态" className="h-full shadow-sm border-none">
      <div className="h-64">
         <Pie {...config} />
      </div>
      <div className="text-right mt-4">
         <a href="#" className="text-blue-500 text-sm">查看资源拓扑 &gt;</a>
      </div>
    </Card>
  );
};
