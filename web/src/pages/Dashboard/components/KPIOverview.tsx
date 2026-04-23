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
        <Card key={index} className="shadow-sm border-none" styles={{ body: { padding: '16px' } }}>
          <div className="flex justify-between items-start mb-2">
            <span className="text-gray-500 text-sm">{item.title}</span>
            <div className={`p-2 rounded-lg ${item.bgColor} flex items-center justify-center`}>
               {item.icon}
            </div>
          </div>
          <div className="text-3xl font-semibold mb-4">{item.total}</div>
          <div className="flex text-xs text-gray-500 gap-3">
             <span>{item.sub1}</span>
             <span>{item.sub2}</span>
          </div>
        </Card>
      ))}
    </div>
  );
};
