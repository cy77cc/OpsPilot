import React from 'react';
import { Card } from 'antd';
import { useNavigate } from 'react-router-dom';
import {
  MessageSquare,
  PlusSquare,
  HardDrive,
  Layers,
  Rocket,
  Bell,
  CheckCircle,
  Network,
  Users
} from 'lucide-react';

export const QuickAccess: React.FC = () => {
  const navigate = useNavigate();
  const items = [
    { icon: <MessageSquare size={20} />, name: 'AI 对话', bg: 'bg-blue-50', color: 'text-blue-500', onClick: () => (window as any).openCopilot?.() },
    { icon: <PlusSquare size={20} />, name: '创建项目', bg: 'bg-indigo-50', color: 'text-indigo-500', path: '/projects' },
    { icon: <HardDrive size={20} />, name: '主机管理', bg: 'bg-emerald-50', color: 'text-emerald-600', path: '/resources/hosts' },
    { icon: <Layers size={20} />, name: '集群管理', bg: 'bg-purple-50', color: 'text-purple-500', path: '/resources/clusters' },
    { icon: <Rocket size={20} />, name: '部署应用', bg: 'bg-rose-50', color: 'text-rose-500', path: '/delivery/deployments' },
    { icon: <Bell size={20} />, name: '告警中心', bg: 'bg-orange-50', color: 'text-orange-500', path: '/observability/monitor' },
    { icon: <CheckCircle size={20} />, name: '审批中心', bg: 'bg-blue-50', color: 'text-blue-500', path: '/approvals' },
    { icon: <Users size={20} />, name: '部门管理', bg: 'bg-cyan-50', color: 'text-cyan-600', path: '/governance/org' },
  ];

  return (
    <Card title="快捷入口" className="h-full shadow-sm border-none flex flex-col" styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}>
      <div className="flex-1 overflow-auto min-h-0">
        <div className="grid grid-cols-4 gap-2 content-start">
          {items.map((item, idx) => (
            <div 
              key={idx} 
              onClick={() => item.onClick ? item.onClick() : navigate(item.path || '/')} 
              className="flex flex-col items-center justify-center cursor-pointer group p-1.5 transition-all"
            >
              <div className={`w-11 h-11 rounded-2xl flex items-center justify-center text-xl mb-2 ${item.bg} ${item.color} shadow-sm group-hover:shadow-md transition-all`}>
                {item.icon}
              </div>
              <span className="text-[11px] text-gray-500 group-hover:text-blue-600 transition-colors whitespace-nowrap">{item.name}</span>
            </div>
          ))}
        </div>
      </div>
      <div className="text-right pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        全部应用 &gt;
      </div>
    </Card>
  );
};
