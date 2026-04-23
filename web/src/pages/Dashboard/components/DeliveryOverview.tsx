import React from 'react';
import { Card, Progress } from 'antd';

export const DeliveryOverview: React.FC = () => {
  return (
    <Card title="应用交付概览" className="h-full shadow-sm border-none">
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
      <div className="text-right mt-6">
         <a href="#" className="text-blue-500 text-sm">查看 CICD &gt;</a>
      </div>
    </Card>
  );
};
