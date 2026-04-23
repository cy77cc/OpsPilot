import React from 'react';
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
import { Input, Select, Button } from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';

const Dashboard: React.FC = () => {
  return (
    <div className="p-0 bg-gray-50 min-h-screen">
      <div className="flex justify-end items-center mb-6">
        <div className="flex items-center gap-3">
          <Select 
            defaultValue="1h" 
            options={[
              { value: '1h', label: '最近 1 小时' },
              { value: '24h', label: '最近 24 小时' },
              { value: '7d', label: '最近 7 天' },
            ]} 
            className="w-32"
          />
          <Button icon={<ReloadOutlined />} />
          <Button type="primary">自定义</Button>
        </div>
      </div>
      
      <div className="flex flex-col gap-3">
        <KPIOverview />
        
        {/* Main Grid Layout */}
        <div className="grid grid-cols-12 gap-3">
          <div className="col-span-12 xl:col-span-3">
            <ResourceHealth />
          </div>
          <div className="col-span-12 xl:col-span-4">
            <ClusterUsage />
          </div>
          <div className="col-span-12 xl:col-span-3">
            <AlertTrends />
          </div>
          <div className="col-span-12 xl:col-span-2">
            <SystemStatus />
          </div>
        </div>

        <div className="grid grid-cols-12 gap-3">
          <div className="col-span-12 xl:col-span-3">
            <DeliveryOverview />
          </div>
          <div className="col-span-12 xl:col-span-6">
            <LLMUsage />
          </div>
          <div className="col-span-12 xl:col-span-3">
            <QuickAccess />
          </div>
        </div>

        <div className="grid grid-cols-12 gap-3">
          <div className="col-span-12 xl:col-span-8">
            <RecentAlerts />
          </div>
          <div className="col-span-12 xl:col-span-4">
            <PlatformEvents />
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
