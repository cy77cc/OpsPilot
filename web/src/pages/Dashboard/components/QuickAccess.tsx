import React from 'react';
import { Card } from 'antd';
import {
  MessageSquare,
  PlusSquare,
  Server,
  Layers,
  Rocket,
  Bell,
  CheckCircle,
  Network
} from 'lucide-react';

export const QuickAccess: React.FC = () => {
  const items = [
    { icon: <MessageSquare size={20} />, name: 'AI 对话', bg: 'bg-blue-50', color: 'text-blue-500' },
    { icon: <PlusSquare size={20} />, name: '创建项目', bg: 'bg-indigo-50', color: 'text-indigo-500' },
    { icon: <Server size={20} />, name: '主机管理', bg: 'bg-green-50', color: 'text-green-500' },
    { icon: <Layers size={20} />, name: '集群管理', bg: 'bg-purple-50', color: 'text-purple-500' },
    { icon: <Rocket size={20} />, name: '部署应用', bg: 'bg-emerald-50', color: 'text-emerald-500' },
    { icon: <Bell size={20} />, name: '告警中心', bg: 'bg-orange-50', color: 'text-orange-500' },
    { icon: <CheckCircle size={20} />, name: '审批中心', bg: 'bg-blue-50', color: 'text-blue-500' },
    { icon: <Network size={20} />, name: '拓扑视图', bg: 'bg-cyan-50', color: 'text-cyan-500' },
  ];

  return (
    <Card 
      title="快捷入口" 
      className="h-full shadow-sm border-none flex flex-col"
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}
    >
      <div className="flex-1 overflow-auto min-h-0">
        <div className="grid grid-cols-4 gap-4 content-start">
          {items.map((item, idx) => (
            <div key={idx} className="flex flex-col items-center justify-center cursor-pointer group p-2 transition-all">
              <div className={`w-12 h-12 rounded-2xl flex items-center justify-center text-xl mb-2 ${item.bg} ${item.color} shadow-sm group-hover:shadow-md transition-all`}>
                {item.icon}
              </div>
              <span className="text-xs text-gray-500 group-hover:text-blue-600 transition-colors whitespace-nowrap">{item.name}</span>
            </div>
          ))}
        </div>
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        全部应用 &gt;
      </div>
    </Card>
  );
};
