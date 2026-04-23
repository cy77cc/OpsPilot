import React from 'react';
import { Card } from 'antd';
import { Line } from '@ant-design/charts';

export const LLMUsage: React.FC = () => {
  const data = [
    { date: '05-06', type: '对话', value: 800 }, { date: '05-06', type: '工具', value: 400 },
    { date: '05-07', type: '对话', value: 1000 }, { date: '05-07', type: '工具', value: 500 },
    { date: '05-08', type: '对话', value: 1200 }, { date: '05-08', type: '工具', value: 600 },
    { date: '05-09', type: '对话', value: 900 }, { date: '05-09', type: '工具', value: 450 },
    { date: '05-10', type: '对话', value: 1100 }, { date: '05-10', type: '工具', value: 550 },
    { date: '05-11', type: '对话', value: 1300 }, { date: '05-11', type: '工具', value: 650 },
    { date: '05-12', type: '对话', value: 1400 }, { date: '05-12', type: '工具', value: 700 },
  ];

  const config = {
    data,
    xField: 'date',
    yField: 'value',
    colorField: 'type',
    scale: {
        color: {
            range: ['#1890ff', '#722ed1']
        }
    },
    legend: false,
    smooth: true,
  };

  return (
    <Card title="大模型使用统计" className="h-full shadow-sm border-none">
      <div className="grid grid-cols-4 gap-4 mb-4">
        <div><div className="text-xs text-gray-500">总对话数</div><div className="text-xl font-bold">3,285</div><div className="text-xs text-green-500">&uarr; 12.5%</div></div>
        <div><div className="text-xs text-gray-500">Tokens 消耗</div><div className="text-xl font-bold">2.45M</div><div className="text-xs text-green-500">&uarr; 18.7%</div></div>
        <div><div className="text-xs text-gray-500">调用工具数</div><div className="text-xl font-bold">1,268</div><div className="text-xs text-green-500">&uarr; 9.3%</div></div>
        <div><div className="text-xs text-gray-500">审批请求数</div><div className="text-xl font-bold">128</div><div className="text-xs text-red-500">&darr; 4.2%</div></div>
      </div>
      <div className="h-48">
        <Line {...config} />
      </div>
       <div className="text-right mt-2">
         <a href="#" className="text-blue-500 text-sm">查看 AI 使用详情 &gt;</a>
      </div>
    </Card>
  );
};
