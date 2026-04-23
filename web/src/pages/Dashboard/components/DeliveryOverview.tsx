import React from 'react';
import { Card, Progress } from 'antd';

export const DeliveryOverview: React.FC = () => {
  return (
    <Card 
      title="应用交付概览" 
      className="h-full shadow-sm border-none flex flex-col"
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}
    >
      <div className="flex-1 overflow-auto min-h-0">
        <div className="flex justify-between text-center mb-8 border-b pb-4">
          <div><div className="text-lg font-bold">256</div><div className="text-xs text-gray-500">服务</div></div>
          <div className="text-gray-300 mt-2">&rarr;</div>
          <div><div className="text-lg font-bold">512</div><div className="text-xs text-gray-500">部署</div></div>
          <div className="text-gray-300 mt-2">&rarr;</div>
          <div><div className="text-lg font-bold">72</div><div className="text-xs text-gray-500">流水线</div></div>
          <div className="text-gray-300 mt-2">&rarr;</div>
          <div><div className="text-lg font-bold">132</div><div className="text-xs text-gray-500">任务</div></div>
        </div>
        
        <div className="flex flex-col gap-4">
          <div>
            <div className="flex justify-between text-sm mb-1"><span>部署成功率</span><span>93.4%</span></div>
            <Progress percent={93.4} showInfo={false} size="small" strokeColor="#1890ff" />
          </div>
          <div>
            <div className="flex justify-between text-sm mb-1"><span>流水线成功率</span><span>88.1%</span></div>
            <Progress percent={88.1} showInfo={false} size="small" strokeColor="#1890ff" />
          </div>
          <div>
            <div className="flex justify-between text-sm mb-1"><span>任务成功率</span><span>90.2%</span></div>
            <Progress percent={90.2} showInfo={false} size="small" strokeColor="#1890ff" />
          </div>
        </div>
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看 CICD &gt;
      </div>
    </Card>
  );
};
