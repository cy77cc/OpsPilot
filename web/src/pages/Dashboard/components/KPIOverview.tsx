import React from 'react';
import { Card } from 'antd';
import { 
  Server, 
  Layers, 
  Folder, 
  Component, 
  Rocket, 
  AlertCircle,
  type LucideIcon,
} from 'lucide-react';
import type { HealthOverview } from '../../../api/modules/dashboard';

interface KPIOverviewProps {
  data?: HealthOverview;
  alerts?: number;
}

export const KPIOverview: React.FC<KPIOverviewProps> = ({ data, alerts = 0 }) => {
  const kpiData = [
    { title: '主机总数', total: data?.hosts?.total || 0, sub1: `正常 ${data?.hosts?.healthy || 0}`, sub2: `离线 ${data?.hosts?.offline || 0}`, icon: Server, iconColor: '#1890ff', bgColor: 'bg-[#e6f7ff]' },
    { title: '集群总数', total: data?.clusters?.total || 0, sub1: `正常 ${data?.clusters?.healthy || 0}`, sub2: `离线 ${data?.clusters?.offline || 0}`, icon: Layers, iconColor: '#52c41a', bgColor: 'bg-[#f6ffed]' },
    { title: '项目总数', total: data?.applications?.total || 0, sub1: `运行中 ${data?.applications?.healthy || 0}`, sub2: `异常 ${data?.applications?.unhealthy || 0}`, icon: Folder, iconColor: '#722ed1', bgColor: 'bg-[#f9f0ff]' },
    { title: '服务总数', total: data?.workloads?.services || 0, sub1: '正常', sub2: '活跃', icon: Component, iconColor: '#1890ff', bgColor: 'bg-[#e6f7ff]' },
    { title: '发布中心', total: data?.workloads?.deployments?.total || 0, sub1: `正常 ${data?.workloads?.deployments?.healthy || 0}`, sub2: '待审批', icon: Rocket, iconColor: '#13c2c2', bgColor: 'bg-[#e6fffb]' },
    { title: '告警总数', total: alerts, sub1: '正在发生', sub2: '等待处理', icon: AlertCircle, iconColor: '#ff4d4f', bgColor: 'bg-[#fff1f0]' },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-2">
      {kpiData.map((item, index) => (
        <Card key={index} className="shadow-sm border-none" styles={{ body: { padding: '20px' } }}>
          {(() => {
            const Icon = item.icon as LucideIcon;
            return (
          <div className="flex justify-between items-center">
            <div className="flex-1 min-w-0">
              <div className="text-gray-500 text-sm mb-1 truncate">{item.title}</div>
              <div className="text-3xl font-bold mb-2 text-gray-900">{item.total}</div>
              <div className="flex text-[11px] text-gray-400 gap-2 whitespace-nowrap">
                 <span>{item.sub1}</span>
                 <span className="text-gray-200">|</span>
                 <span>{item.sub2}</span>
              </div>
            </div>
            <div className={`w-12 h-12 rounded-full ${item.bgColor} flex items-center justify-center flex-shrink-0 ml-3`}>
               <Icon size={24} color={item.iconColor} />
            </div>
          </div>
            );
          })()}
        </Card>
      ))}
    </div>
  );
};
