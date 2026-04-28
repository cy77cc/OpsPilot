import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Select, DatePicker, Spin } from 'antd';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { hostApi } from '../../../../api/modules/hosts';
import type { HostMetricPoint } from '../../../../api/modules/hosts';

export interface MetricsSnapshot {
  id: number;
  time: string;
  cpu: number;
  memory: number;
  disk: number;
  diskIo: number;
  netIn: number;
  netOut: number;
  latency_ms?: number;
  health_state?: string;
  error_message?: string;
  summary?: Record<string, unknown>;
  created_at?: string;
}

const { RangePicker } = DatePicker;

const ChartCard: React.FC<{ title: string; data: MetricsSnapshot[]; dataKey: string; color: string; unit: string; loading?: boolean }> = ({ title, data, dataKey, color, unit, loading }) => (
  <Card title={<span className="text-sm font-medium">{title}</span>} size="small" bordered={false} className="bg-white border-none shadow-sm h-full">
    <Spin spinning={loading}>
      <div style={{ height: 200, width: '100%' }}>
        <ResponsiveContainer>
          <LineChart data={data} margin={{ top: 5, right: 10, bottom: 5, left: -20 }}>
            <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f0f0f0" />
            <XAxis dataKey="time" axisLine={false} tickLine={false} tick={{ fontSize: 10, fill: '#999' }} />
            <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 10, fill: '#999' }} unit={unit} />
            <Tooltip 
              contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }}
              itemStyle={{ fontSize: 12 }}
              labelStyle={{ fontSize: 12, fontWeight: 'bold' }}
            />
            <Line type="monotone" dataKey={dataKey} stroke={color} strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </Spin>
  </Card>
);

const MonitorTab: React.FC<{ hostId: string }> = ({ hostId }) => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<MetricsSnapshot[]>([]);

  useEffect(() => {
    const fetchMetrics = async () => {
      setLoading(true);
      try {
        const res = await hostApi.getHostMetrics(hostId);
        setData((res.data || []).map((m: HostMetricPoint) => ({
          id: Number(m.id),
          time: m.createdAt,
          cpu: m.cpu,
          memory: m.memory,
          disk: m.disk,
          diskIo: 0,
          netIn: 0,
          netOut: 0,
          latency_ms: m.latencyMs,
          health_state: m.healthState,
          error_message: m.errorMessage,
        })));
      } finally {
        setLoading(false);
      }
    };
    if (hostId) {
      fetchMetrics();
    }
  }, [hostId]);

  return (
    <div className="flex flex-col gap-4 py-4">
      <div className="flex justify-between items-center bg-white px-4 py-3 rounded-lg shadow-sm border border-gray-100">
        <h3 className="text-base font-medium m-0">详细监控指标</h3>
        <div className="flex gap-4">
          <Select defaultValue="1h" style={{ width: 120 }} options={[
            { value: '1h', label: '最近 1 小时' },
            { value: '6h', label: '最近 6 小时' },
            { value: '24h', label: '最近 24 小时' },
            { value: '7d', label: '最近 7 天' },
          ]} />
          <RangePicker showTime size="middle" />
        </div>
      </div>
      
      <Row gutter={[16, 16]}>
        <Col span={12}>
          <ChartCard title="CPU 使用率 (%)" data={data} dataKey="cpu" color="#1890ff" unit="%" loading={loading} />
        </Col>
        <Col span={12}>
          <ChartCard title="内存使用率 (%)" data={data} dataKey="memory" color="#52c41a" unit="%" loading={loading} />
        </Col>
        <Col span={12}>
          <ChartCard title="磁盘 I/O (MB/s)" data={data} dataKey="diskIo" color="#faad14" unit="" loading={loading} />
        </Col>
        <Col span={12}>
          <ChartCard title="网络流量 (Mbps)" data={data} dataKey="netIn" color="#722ed1" unit="" loading={loading} />
        </Col>
      </Row>
    </div>
  );
};

export default MonitorTab;
