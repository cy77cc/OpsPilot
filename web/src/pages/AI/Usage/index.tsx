import React, { useState } from 'react';
import { Card, DatePicker, Select, Space, Row, Col } from 'antd';
import dayjs from 'dayjs';
import StatCards from './components/StatCards';
import UsageTrendChart from './components/UsageTrendChart';
import ScenePieChart from './components/ScenePieChart';
import ApprovalChart from './components/ApprovalChart';
import UsageTable from './components/UsageTable';
import { useUsageStats, useUsageLogs } from './hooks/useUsageStats';

const { RangePicker } = DatePicker;

const AIUsagePage: React.FC = () => {
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().startOf('day'),
    dayjs().add(1, 'day').startOf('day'),
  ]);
  const [scene, setScene] = useState<string>();

  const { stats, loading: statsLoading, setParams: setStatsParams } = useUsageStats({
    start_date: dateRange[0].format('YYYY-MM-DD'),
    end_date: dateRange[1].format('YYYY-MM-DD'),
  });

  const { result, loading: logsLoading, params: logsParams, setParams: setLogsParams } = useUsageLogs({
    start_date: dateRange[0].format('YYYY-MM-DD'),
    end_date: dateRange[1].format('YYYY-MM-DD'),
    page: 1,
    page_size: 20,
  });

  const handleDateRangeChange = (dates: [dayjs.Dayjs | null, dayjs.Dayjs | null] | null) => {
    if (dates && dates[0] && dates[1]) {
      setDateRange([dates[0], dates[1]]);
      const params = {
        start_date: dates[0].format('YYYY-MM-DD'),
        end_date: dates[1].format('YYYY-MM-DD'),
      };
      setStatsParams(params);
      setLogsParams({ ...logsParams, ...params, page: 1 });
    }
  };

  const handleSceneChange = (value?: string) => {
    setScene(value);
    setStatsParams({ scene: value });
    setLogsParams({ ...logsParams, scene: value, page: 1 });
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setLogsParams({ ...logsParams, page, page_size: pageSize });
  };

  const sceneOptions = stats?.by_scene?.map((s) => ({ label: s.scene, value: s.scene })) || [];

  return (
    <div style={{ padding: 24 }}>
      <Card>
        <Space style={{ marginBottom: 16 }}>
          <RangePicker
            value={dateRange}
            onChange={handleDateRangeChange}
            format="YYYY-MM-DD"
          />
          <Select
            allowClear
            placeholder="筛选场景"
            style={{ width: 200 }}
            options={sceneOptions}
            value={scene}
            onChange={handleSceneChange}
          />
        </Space>

        <StatCards stats={stats} loading={statsLoading} />
        <UsageTrendChart data={stats?.by_date} loading={statsLoading} />

        <Row gutter={16}>
          <Col span={12}>
            <ScenePieChart data={stats?.by_scene} loading={statsLoading} />
          </Col>
          <Col span={12}>
            <ApprovalChart stats={stats} loading={statsLoading} />
          </Col>
        </Row>

        <Card title="请求列表" style={{ marginTop: 16 }}>
          <UsageTable
            data={result?.items}
            total={result?.total || 0}
            loading={logsLoading}
            page={logsParams.page || 1}
            pageSize={logsParams.page_size || 20}
            onPageChange={handlePageChange}
          />
        </Card>
      </Card>
    </div>
  );
};

export default AIUsagePage;
