import React from 'react';
import { Card, Progress } from 'antd';
import type { OperationsOverview } from '../../../api/modules/dashboard';

export const DeliveryOverview: React.FC<{ data?: OperationsOverview }> = ({ data }) => {
  return (
    <Card title="应用交付概览" className="h-full shadow-sm border-none flex flex-col" styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}>
      <div className="flex-1 overflow-auto min-h-0">
        <div className="flex justify-between text-center mb-8 border-b pb-4 text-[13px]">
          <div><div className="text-lg font-bold">256</div><div className="text-xs text-gray-400 uppercase tracking-wider">服务</div></div>
          <div className="text-gray-200 mt-2">&rarr;</div>
          <div><div className="text-lg font-bold">{data?.deployments?.todayTotal || 0}</div><div className="text-xs text-gray-400 uppercase tracking-wider">今日部署</div></div>
          <div className="text-gray-200 mt-2">&rarr;</div>
          <div><div className="text-lg font-bold">{data?.cicd?.todayTotal || 0}</div><div className="text-xs text-gray-400 uppercase tracking-wider">流水线</div></div>
        </div>
        
        <div className="flex flex-col gap-3">
          <div>
            <div className="flex justify-between text-sm mb-1 text-gray-600"><span>部署成功率</span><span className="font-semibold text-gray-900">{data?.deployments?.todayTotal ? ((data.deployments.todaySuccess / data.deployments.todayTotal) * 100).toFixed(1) : 0}%</span></div>
            <Progress percent={data?.deployments?.todayTotal ? (data.deployments.todaySuccess / data.deployments.todayTotal) * 100 : 0} showInfo={false} size="small" strokeColor="#1890ff" />
          </div>
          <div>
            <div className="flex justify-between text-sm mb-1 text-gray-600"><span>流水线成功率</span><span className="font-semibold text-gray-900">{data?.cicd?.todayTotal ? ((data.cicd.success / data.cicd.todayTotal) * 100).toFixed(1) : 0}%</span></div>
            <Progress percent={data?.cicd?.todayTotal ? (data.cicd.success / data.cicd.todayTotal) * 100 : 0} showInfo={false} size="small" strokeColor="#1890ff" />
          </div>
        </div>
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看 CICD &gt;
      </div>
    </Card>
  );
};
