import React, { useState } from 'react';
import { Card, Radio, Space } from 'antd';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';

const mockData = Array.from({ length: 20 }).map((_, i) => ({
  time: `${13 + Math.floor(i / 4)}:${(i % 4) * 15}`,
  cpu: 20 + Math.random() * 30,
  memory: 40 + Math.random() * 20,
  diskIo: 5 + Math.random() * 15,
}));

interface HostTrendChartCardProps {
  loading?: boolean;
}

const HostTrendChartCard: React.FC<HostTrendChartCardProps> = ({ loading }) => {
  const [range, setRange] = useState('1h');

  return (
    <Card 
      title={`监控趋势 (${range})`} 
      loading={loading} 
      className="h-full"
      extra={
        <Radio.Group size="small" value={range} onChange={e => setRange(e.target.value)}>
          <Radio.Button value="1h">1小时</Radio.Button>
          <Radio.Button value="6h">6小时</Radio.Button>
          <Radio.Button value="24h">24小时</Radio.Button>
          <Radio.Button value="7d">7天</Radio.Button>
        </Radio.Group>
      }
    >
      <div style={{ width: '100%', height: 300 }}>
        <ResponsiveContainer>
          <LineChart data={mockData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f0f0f0" />
            <XAxis 
              dataKey="time" 
              axisLine={false} 
              tickLine={false} 
              tick={{ fontSize: 12, fill: '#999' }} 
            />
            <YAxis 
              yId="left" 
              axisLine={false} 
              tickLine={false} 
              tick={{ fontSize: 12, fill: '#999' }}
              unit="%" 
            />
            <YAxis 
              yId="right" 
              orientation="right" 
              axisLine={false} 
              tickLine={false} 
              tick={{ fontSize: 12, fill: '#999' }}
              unit="MB/s" 
            />
            <Tooltip 
              contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }}
            />
            <Legend verticalAlign="top" align="right" iconType="circle" />
            <Line
              yId="left"
              type="monotone"
              dataKey="cpu"
              name="CPU 使用率"
              stroke="#1890ff"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
            <Line
              yId="left"
              type="monotone"
              dataKey="memory"
              name="内存使用率"
              stroke="#52c41a"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
            <Line
              yId="right"
              type="monotone"
              dataKey="diskIo"
              name="磁盘 I/O"
              stroke="#faad14"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </Card>
  );
};

export default HostTrendChartCard;
