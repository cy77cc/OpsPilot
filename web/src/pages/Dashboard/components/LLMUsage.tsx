import React from 'react';
import { Card, Empty } from 'antd';
import { Line } from '@ant-design/charts';
import type { AIActivity } from '../../../api/modules/dashboard';

export const LLMUsage: React.FC<{ data?: AIActivity }> = ({ data }) => {
  // Mock trend data as it's not fully provided in V2 yet
  const chartData = [
    { date: '05-06', type: '对话', value: 800 },
    { date: '05-07', type: '对话', value: 1000 },
    { date: '05-08', type: '对话', value: 1200 },
    { date: '05-09', type: '对话', value: 900 },
    { date: '05-10', type: '对话', value: 1100 },
    { date: '05-11', type: '对话', value: 1300 },
    { date: '05-12', type: '对话', value: data?.stats?.sessionCount || 1400 },
  ];

  const config = {
    data: chartData,
    xField: 'date',
    yField: 'value',
    colorField: 'type',
    smooth: true,
    scale: { color: { range: ['#1890ff'] } },
    legend: false,
  };

  return (
    <Card title="大模型使用统计" className="h-full shadow-sm border-none flex flex-col" styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}>
      <div className="flex-1 overflow-auto min-h-0">
        <div className="grid grid-cols-4 gap-4 mb-6">
          <div><div className="text-xs text-gray-400 mb-1">会话总数</div><div className="text-xl font-bold text-gray-900">{data?.stats?.sessionCount || 0}</div></div>
          <div><div className="text-xs text-gray-400 mb-1">Tokens 消耗</div><div className="text-xl font-bold text-gray-900">{((data?.stats?.tokenCount || 0) / 1000000).toFixed(2)}M</div></div>
          <div><div className="text-xs text-gray-400 mb-1">成功率</div><div className="text-xl font-bold text-gray-900">{data?.stats?.successRate || 0}%</div></div>
          <div><div className="text-xs text-gray-400 mb-1">平均耗时</div><div className="text-xl font-bold text-gray-900">{data?.stats?.avgDurationMs || 0}ms</div></div>
        </div>
        <div className="h-32 mt-2">
          {chartData.length > 0 ? <Line {...config} /> : <Empty description="暂无 AI 指标" />}
        </div>
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看 AI 详情 &gt;
      </div>
    </Card>
  );
};
