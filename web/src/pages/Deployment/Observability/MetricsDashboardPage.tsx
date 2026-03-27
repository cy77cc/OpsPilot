import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Select, Space, Button, message } from 'antd';
import {
  ReloadOutlined,
  RiseOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import { Line } from '@ant-design/charts';
import { Api } from '../../../api';
import type { MetricsSummary, MetricsTrend } from '../../../api/modules/deployment';

const MetricsDashboardPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [timeRange, setTimeRange] = useState<'daily' | 'weekly' | 'monthly'>('daily');
  const [summary, setSummary] = useState<MetricsSummary | null>(null);
  const [trends, setTrends] = useState<MetricsTrend[]>([]);

  const load = async () => {
    setLoading(true);
    try {
      const [summaryRes, trendsRes] = await Promise.all([
        Api.deployment.getMetricsSummary(),
        Api.deployment.getMetricsTrends({ range: timeRange }),
      ]);
      setSummary(summaryRes.data);
      setTrends(trendsRes.data || []);
    } catch (err) {
      message.error('加载指标数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [timeRange]);

  // 部署频率趋势数据
  const deploymentFrequencyData = trends.map((t) => ({
    date: t.date,
    count: t.deployment_count,
  }));

  // 成功率趋势数据
  const successRateData = trends.map((t) => ({
    date: t.date,
    rate: Number(t.success_rate.toFixed(1)),
  }));

  const deploymentFrequencyConfig = {
    data: deploymentFrequencyData,
    xField: 'date',
    yField: 'count',
    point: {
      size: 5,
      shape: 'diamond',
    },
    label: {
      style: {
        fill: '#aaa',
      },
    },
  };

  const successRateConfig = {
    data: successRateData,
    xField: 'date',
    yField: 'rate',
    point: {
      size: 5,
      shape: 'circle',
    },
    yAxis: {
      min: 0,
      max: 100,
    },
    color: '#10b981',
  };

  // 环境数据
  const environmentData = summary?.by_environment || {};
  const envEntries = Object.entries(environmentData);

  const getEnvColor = (env: string) => {
    switch (env) {
      case 'production':
        return '#ef4444';
      case 'staging':
        return '#f59e0b';
      case 'development':
        return '#3b82f6';
      default:
        return '#6366f1';
    }
  };

  const getEnvBgClass = (env: string) => {
    switch (env) {
      case 'production':
        return 'bg-red-50';
      case 'staging':
        return 'bg-orange-50';
      case 'development':
        return 'bg-blue-50';
      default:
        return 'bg-gray-50';
    }
  };

  const getEnvLabel = (env: string) => {
    const labels: Record<string, string> = {
      production: '生产环境',
      staging: '预发布',
      development: '开发环境',
    };
    return labels[env] || env;
  };

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">指标仪表板</h1>
          <p className="text-sm text-gray-500 mt-1">查看部署相关的关键指标和趋势</p>
        </div>
        <Space>
          <Select
            value={timeRange}
            style={{ width: 120 }}
            options={[
              { value: 'daily', label: '每日' },
              { value: 'weekly', label: '每周' },
              { value: 'monthly', label: '每月' },
            ]}
            onChange={setTimeRange}
          />
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
        </Space>
      </div>

      {/* Key metrics */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总发布数"
              value={summary?.total_releases || 0}
              prefix={<RiseOutlined className="text-success" />}
              valueStyle={{ color: '#10b981' }}
            />
            <div className="text-xs text-gray-500 mt-2">
              最近7天: {summary?.recent_releases || 0} 次
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="成功率"
              value={Number((summary?.success_rate || 0).toFixed(1))}
              suffix="%"
              prefix={<CheckCircleOutlined className="text-success" />}
              valueStyle={{ color: '#10b981' }}
            />
            <div className="text-xs text-gray-500 mt-2">
              最近7天失败: {summary?.recent_failures || 0} 次
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="平均部署时长"
              value={Number((summary?.avg_duration_seconds || 0) / 60).toFixed(1)}
              suffix="分钟"
              prefix={<ClockCircleOutlined className="text-primary" />}
              valueStyle={{ color: '#6366f1' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="进行中"
              value={summary?.by_status?.applying || 0}
              suffix="个"
              prefix={<ClockCircleOutlined className="text-warning" />}
              valueStyle={{ color: '#f59e0b' }}
            />
            <div className="text-xs text-gray-500 mt-2">
              待审批: {summary?.by_status?.pending_approval || 0} 个
            </div>
          </Card>
        </Col>
      </Row>

      {/* Deployment frequency chart */}
      <Card title="部署频率趋势">
        {deploymentFrequencyData.length > 0 ? (
          <Line {...deploymentFrequencyConfig} />
        ) : (
          <div className="text-center text-gray-500 py-8">暂无数据</div>
        )}
      </Card>

      {/* Success rate chart */}
      <Card title="成功率趋势">
        {successRateData.length > 0 ? (
          <Line {...successRateConfig} />
        ) : (
          <div className="text-center text-gray-500 py-8">暂无数据</div>
        )}
      </Card>

      {/* Environment comparison */}
      <Card title="环境对比">
        <Row gutter={[16, 16]}>
          {envEntries.length > 0 ? (
            envEntries.map(([env, data]) => (
              <Col xs={24} md={8} key={env}>
                <Card className={getEnvBgClass(env)}>
                  <Statistic
                    title={getEnvLabel(env)}
                    value={data.total}
                    suffix="次部署"
                    valueStyle={{ color: getEnvColor(env) }}
                  />
                  <div className="text-xs text-gray-600 mt-2">
                    成功率: {Number(data.success_rate).toFixed(1)}%
                  </div>
                </Card>
              </Col>
            ))
          ) : (
            <Col span={24}>
              <div className="text-center text-gray-500 py-8">暂无环境数据</div>
            </Col>
          )}
        </Row>
      </Card>
    </div>
  );
};

export default MetricsDashboardPage;
