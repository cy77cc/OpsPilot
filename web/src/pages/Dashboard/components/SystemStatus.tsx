import React from 'react';
import { Card, Badge } from 'antd';
import type { HealthOverview } from '../../../api/modules/dashboard';

export const SystemStatus: React.FC<{ data?: HealthOverview }> = ({ data }) => {
  const services = [
    { name: 'API 服务', status: 'success', text: '运行中' },
    { name: '调度服务', status: 'success', text: '运行中' },
    { name: '监控服务', status: 'success', text: '运行中' },
    { name: '消息队列', status: 'success', text: '运行中' },
    { name: '数据库', status: data?.hosts?.unhealthy ? 'warning' : 'success', text: data?.hosts?.unhealthy ? '告警' : '运行中' },
    { name: '缓存服务', status: 'success', text: '运行中' },
  ];

  return (
    <Card 
      title="系统状态" 
      className="h-full shadow-sm border-none flex flex-col"
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, padding: 12 } }}
    >
      <div className="flex-1 min-h-0 overflow-hidden">
        <div className="h-full flex flex-col justify-between">
          {services.map((svc, idx) => (
            <div key={idx} className="flex justify-between items-center text-sm">
              <span className="text-gray-600">{svc.name}</span>
              <Badge status={svc.status as any} text={svc.text} />
            </div>
          ))}
        </div>
      </div>
      <div className="text-right pt-2 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        全部服务状态 &gt;
      </div>
    </Card>
  );
};
