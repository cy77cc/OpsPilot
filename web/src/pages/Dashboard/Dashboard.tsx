import React, { useEffect, useState } from 'react';
import { KPIOverview } from './components/KPIOverview';
import { ResourceHealth } from './components/ResourceHealth';
import { ClusterUsage } from './components/ClusterUsage';
import { AlertTrends } from './components/AlertTrends';
import { SystemStatus } from './components/SystemStatus';
import { DeliveryOverview } from './components/DeliveryOverview';
import { LLMUsage } from './components/LLMUsage';
import { QuickAccess } from './components/QuickAccess';
import { RecentAlerts } from './components/RecentAlerts';
import { PlatformEvents } from './components/PlatformEvents';
import { Select, Button, Spin, message } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { dashboardApi, type OverviewResponseV2, type TimeRange } from '../../api/modules/dashboard';

import { PageSkeleton } from '../../components/LoadingSkeleton';

const Dashboard: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<OverviewResponseV2 | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>('1h');

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await dashboardApi.getOverviewV2(timeRange);
      if (res.success) {
        setData(res.data);
      } else {
        message.error('获取概览数据失败: ' + res.message);
      }
    } catch (err) {
      console.error(err);
      message.error('网络错误，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [timeRange]);

  if (loading && !data) {
    return <PageSkeleton />;
  }

  return (
    <div className="p-0 bg-gray-50 min-h-screen">
      <div className="flex justify-end items-center mb-6">
        <div className="flex items-center gap-3">
          <Select 
            value={timeRange}
            onChange={(value) => setTimeRange(value as TimeRange)}
            options={[
              { value: '1h', label: '最近 1 小时' },
              { value: '6h', label: '最近 6 小时' },
              { value: '24h', label: '最近 24 小时' },
            ]} 
            className="w-32"
          />
          <Button icon={<ReloadOutlined />} onClick={fetchData} loading={loading} />
          <Button type="primary">自定义</Button>
        </div>
      </div>
      
      <div className="flex flex-col gap-3">
        <KPIOverview data={data?.health} alerts={data?.alerts?.firing} />
        
        {/* Main Grid Layout - Charts Row */}
        <div className="grid grid-cols-12 gap-3 xl:auto-rows-[360px]">
          <div className="col-span-12 xl:col-span-3">
            <ResourceHealth data={data?.health} />
          </div>
          <div className="col-span-12 xl:col-span-4">
            <ClusterUsage data={data?.resources} />
          </div>
          <div className="col-span-12 xl:col-span-3">
            <AlertTrends data={data?.alerts} />
          </div>
          <div className="col-span-12 xl:col-span-2">
            <SystemStatus data={data?.health} />
          </div>
        </div>

        {/* AI & Delivery Row */}
        <div className="grid grid-cols-12 gap-3">
          <div className="col-span-12 xl:col-span-3">
            <DeliveryOverview data={data?.operations} />
          </div>
          <div className="col-span-12 xl:col-span-6">
            <LLMUsage data={data?.ai} />
          </div>
          <div className="col-span-12 xl:col-span-3">
            <QuickAccess />
          </div>
        </div>

        <div className="grid grid-cols-12 gap-3">
          <div className="col-span-12 xl:col-span-8">
            <RecentAlerts data={data?.alerts?.recent} />
          </div>
          <div className="col-span-12 xl:col-span-4">
            <PlatformEvents data={data?.events} />
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
