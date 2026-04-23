import React from 'react';
import { Card } from 'antd';
import { Column } from '@ant-design/charts';

export const AlertTrends: React.FC = () => {
  const data = [
    { time: '10:00', type: '严重', value: 5 }, { time: '10:00', type: '警告', value: 12 }, { time: '10:00', type: '信息', value: 20 },
    { time: '10:15', type: '严重', value: 8 }, { time: '10:15', type: '警告', value: 15 }, { time: '10:15', type: '信息', value: 25 },
    { time: '10:30', type: '严重', value: 3 }, { time: '10:30', type: '警告', value: 10 }, { time: '10:30', type: '信息', value: 18 },
    { time: '10:45', type: '严重', value: 6 }, { time: '10:45', type: '警告', value: 14 }, { time: '10:45', type: '信息', value: 22 },
    { time: '11:00', type: '严重', value: 4 }, { time: '11:00', type: '警告', value: 11 }, { time: '11:00', type: '信息', value: 19 },
  ];

  const config = {
    data,
    xField: 'time',
    yField: 'value',
    stack: true,
    colorField: 'type',
    scale: {
        color: {
            range: ['#f5222d', '#faad14', '#1890ff']
        }
    },
    legend: false,
  };

  return (
    <Card 
      title="告警趋势" 
      className="h-full shadow-sm border-none flex flex-col"
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}
    >
      <div className="flex-1 overflow-auto min-h-0">
        <div className="flex justify-between mb-4 px-4">
           <div className="text-center"><div className="text-xl font-bold text-red-500">23</div><div className="text-xs text-gray-500">严重</div></div>
           <div className="text-center"><div className="text-xl font-bold text-orange-400">56</div><div className="text-xs text-gray-500">警告</div></div>
           <div className="text-center"><div className="text-xl font-bold text-blue-500">132</div><div className="text-xs text-gray-500">信息</div></div>
        </div>
        <div className="h-40">
          <Column {...config} />
        </div>
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看告警中心 >
      </div>
    </Card>
  );
};
