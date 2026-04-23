import React from 'react';
import { Card, Badge } from 'antd';

export const SystemStatus: React.FC = () => {
  const services = [
    { name: 'API 服务', status: 'success', text: '运行中' },
    { name: '调度服务', status: 'success', text: '运行中' },
    { name: '监控服务', status: 'success', text: '运行中' },
    { name: '消息队列', status: 'success', text: '运行中' },
    { name: '数据库', status: 'warning', text: '告警' },
    { name: '缓存服务', status: 'success', text: '运行中' },
  ];

  return (
    <Card title="系统状态" className="h-full shadow-sm border-none">
      <div className="flex flex-col gap-4 h-56 justify-between">
        {services.map((svc, idx) => (
          <div key={idx} className="flex justify-between items-center text-sm">
            <span className="text-gray-600">{svc.name}</span>
            <Badge status={svc.status as any} text={svc.text} />
          </div>
        ))}
      </div>
      <div className="text-right mt-6">
         <a href="#" className="text-blue-500 text-sm">全部服务状态 &gt;</a>
      </div>
    </Card>
  );
};
