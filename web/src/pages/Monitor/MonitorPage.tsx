import React, { useEffect, useMemo, useState } from 'react';
import { Card, Col, Row, Progress, Button, Space, Table, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { Api } from '../../api';
import type { Alert, MetricData } from '../../api/modules/monitoring';
import { PageSkeleton } from '../../components/LoadingSkeleton';

const MonitorPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [cpuMetrics, setCpuMetrics] = useState<MetricData[]>([]);
  const [memMetrics, setMemMetrics] = useState<MetricData[]>([]);

  const load = async () => {
    setLoading(true);
    try {
      const end = new Date().toISOString();
      const start = dayjs().subtract(24, 'hour').toDate().toISOString();
      const [alertRes, cpuRes, memRes] = await Promise.all([
        Api.monitoring.getAlertList({ page: 1, pageSize: 100, status: 'firing' }), // Only active
        Api.monitoring.getMetrics({ metric: 'cpu_usage', startTime: start, endTime: end }),
        Api.monitoring.getMetrics({ metric: 'memory_usage', startTime: start, endTime: end }),
      ]);
      setAlerts(alertRes.data.list || []);
      setCpuMetrics(cpuRes.data?.series || []);
      setMemMetrics(memRes.data?.series || []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const cpuAvg = useMemo(() => (cpuMetrics.length ? cpuMetrics.reduce((s, i) => s + Number(i.value), 0) / cpuMetrics.length : 0), [cpuMetrics]);
  const memAvg = useMemo(() => (memMetrics.length ? memMetrics.reduce((s, i) => s + Number(i.value), 0) / memMetrics.length : 0), [memMetrics]);
  
  // Format data for recharts
  const chartData = useMemo(() => {
    const mapData = (data: MetricData[]) => data.map(item => ({
      time: dayjs(item.timestamp).format('HH:mm'),
      value: Math.round(Number(item.value) * 100) / 100,
    }));
    return {
      cpu: mapData(cpuMetrics),
      mem: mapData(memMetrics)
    };
  }, [cpuMetrics, memMetrics]);

  const isInitialLoading = loading
    && alerts.length === 0
    && cpuMetrics.length === 0
    && memMetrics.length === 0;

  if (isInitialLoading) {
    return <PageSkeleton />;
  }

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <Space>
          <Button size="small" icon={<ReloadOutlined />} loading={loading && !isInitialLoading} onClick={load}>刷新</Button>
        </Space>
      </div>
      
      <Row gutter={[12, 12]}>
        <Col xs={24} md={12}>
          <Card size="small" title="CPU 资源使用率趋势 (24h)">
             <div style={{ height: 200 }}>
              {chartData.cpu.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={chartData.cpu} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                    <defs>
                      <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#1677ff" stopOpacity={0.8}/>
                        <stop offset="95%" stopColor="#1677ff" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                    <YAxis tick={{ fontSize: 10 }} />
                    <CartesianGrid strokeDasharray="3 3" vertical={false} />
                    <Tooltip />
                    <Area type="monotone" dataKey="value" stroke="#1677ff" fillOpacity={1} fill="url(#colorCpu)" />
                  </AreaChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex items-center justify-center h-full text-gray-400 text-sm">暂无指标数据</div>
              )}
            </div>
            <div className="mt-2 text-xs text-gray-500 text-right">平均使用率: {cpuAvg.toFixed(1)}%</div>
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card size="small" title="内存资源使用率趋势 (24h)">
             <div style={{ height: 200 }}>
              {chartData.mem.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={chartData.mem} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                    <defs>
                      <linearGradient id="colorMem" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#52c41a" stopOpacity={0.8}/>
                        <stop offset="95%" stopColor="#52c41a" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <XAxis dataKey="time" tick={{ fontSize: 10 }} />
                    <YAxis tick={{ fontSize: 10 }} />
                    <CartesianGrid strokeDasharray="3 3" vertical={false} />
                    <Tooltip />
                    <Area type="monotone" dataKey="value" stroke="#52c41a" fillOpacity={1} fill="url(#colorMem)" />
                  </AreaChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex items-center justify-center h-full text-gray-400 text-sm">暂无指标数据</div>
              )}
            </div>
            <div className="mt-2 text-xs text-gray-500 text-right">平均使用率: {memAvg.toFixed(1)}%</div>
          </Card>
        </Col>
      </Row>

      <Card size="small" title="当前活跃告警 (Firing)">
        <Table
          size="small"
          rowKey="id"
          loading={loading && !isInitialLoading}
          dataSource={alerts}
          pagination={false}
          columns={[
            { title: '告警消息', dataIndex: 'title', render: (_: string, r: any) => r.message || r.title || '-' },
            { title: '级别', dataIndex: 'severity', width: 100, render: (v: string) => <Tag color={v === 'critical' ? 'error' : v === 'warning' ? 'warning' : 'blue'}>{v}</Tag> },
            { title: '来源指标', dataIndex: 'source', render: (_: string, r: any) => r.metric || r.source || '-' },
            { title: '触发时间', dataIndex: 'createdAt', width: 180, render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
          ]}
          locale={{ emptyText: '当前系统健康，无活跃告警' }}
        />
      </Card>
    </div>
  );
};

export default MonitorPage;