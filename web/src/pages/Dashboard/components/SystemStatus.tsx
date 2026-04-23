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
    <Card 
      title="系统状态" 
      className="h-full shadow-sm border-none flex flex-col"
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}
    >
      <div className="flex-1 overflow-auto min-h-0">
        <div className="flex flex-col gap-2">
          {services.map((svc, idx) => (
            <div key={idx} className="flex justify-between items-center text-sm py-1 border-b border-gray-50 last:border-0">
              <span className="text-gray-600">{svc.name}</span>
              <Badge status={svc.status as any} text={svc.text} />
            </div>
          ))}
        </div>
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        全部服务状态 >
      </div>
    </Card>
  );
};
