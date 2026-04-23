import React from 'react';
import { Card } from 'antd';
import { 
  Server, 
  Layers, 
  Folder, 
  Component, 
  Rocket, 
  AlertCircle 
} from 'lucide-react';

const kpiData = [
  { title: '主机总数', total: 128, sub1: '在线 102', sub2: '离线 26', icon: <Server size={22} color="#1890ff" />, bgColor: 'bg-[#e6f7ff]' },
  { title: '集群总数', total: 12, sub1: '正常 10', sub2: '异常 2', icon: <Layers size={22} color="#52c41a" />, bgColor: 'bg-[#f6ffed]' },
  { title: '项目总数', total: 35, sub1: '运行中 28', sub2: '已停用 7', icon: <Folder size={22} color="#722ed1" />, bgColor: 'bg-[#f9f0ff]' },
  { title: '服务总数', total: 256, sub1: '运行中 214', sub2: '异常 42', icon: <Component size={22} color="#1890ff" />, bgColor: 'bg-[#e6f7ff]' },
  { title: '部署总数', total: 512, sub1: '成功 478', sub2: '失败 34', icon: <Rocket size={22} color="#13c2c2" />, bgColor: 'bg-[#e6fffb]' },
  { title: '告警总数', total: 23, sub1: '严重 5', sub2: '警告 18', icon: <AlertCircle size={22} color="#ff4d4f" />, bgColor: 'bg-[#fff1f0]' },
];

export const KPIOverview: React.FC = () => {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-2">
      {kpiData.map((item, index) => (
        <Card key={index} className="shadow-sm border-none" styles={{ body: { padding: '20px' } }}>
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
               {React.cloneElement(item.icon as React.ReactElement, { size: 24 })}
            </div>
          </div>
        </Card>
      ))}
    </div>
  );
};
