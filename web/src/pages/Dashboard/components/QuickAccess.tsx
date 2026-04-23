import React from 'react';
import { Card } from 'antd';
import {
  MessageOutlined,
  FolderAddOutlined,
  CloudServerOutlined,
  ClusterOutlined,
  RocketOutlined,
  BellOutlined,
  CheckCircleOutlined,
  DeploymentUnitOutlined
} from '@ant-design/icons';

export const QuickAccess: React.FC = () => {
  const items = [
    { icon: <MessageOutlined />, name: 'AI 对话', bg: 'bg-blue-50', color: 'text-blue-500' },
    { icon: <FolderAddOutlined />, name: '创建项目', bg: 'bg-indigo-50', color: 'text-indigo-500' },
    { icon: <CloudServerOutlined />, name: '主机管理', bg: 'bg-green-50', color: 'text-green-500' },
    { icon: <ClusterOutlined />, name: '集群管理', bg: 'bg-purple-50', color: 'text-purple-500' },
    { icon: <RocketOutlined />, name: '部署应用', bg: 'bg-emerald-50', color: 'text-emerald-500' },
    { icon: <BellOutlined />, name: '告警中心', bg: 'bg-orange-50', color: 'text-orange-500' },
    { icon: <CheckCircleOutlined />, name: '审批中心', bg: 'bg-blue-50', color: 'text-blue-500' },
    { icon: <DeploymentUnitOutlined />, name: '拓扑视图', bg: 'bg-cyan-50', color: 'text-cyan-500' },
  ];

  return (
    <Card title="快捷入口" className="h-full shadow-sm border-none">
      <div className="grid grid-cols-4 gap-4 h-56 content-start">
        {items.map((item, idx) => (
          <div key={idx} className="flex flex-col items-center justify-center cursor-pointer hover:bg-gray-50 p-2 rounded transition-colors">
            <div className={`w-10 h-10 rounded-lg flex items-center justify-center text-xl mb-2 ${item.bg} ${item.color}`}>
              {item.icon}
            </div>
            <span className="text-xs text-gray-600 whitespace-nowrap">{item.name}</span>
          </div>
        ))}
      </div>
      <div className="text-right mt-6">
         <a href="#" className="text-blue-500 text-sm">全部应用 &gt;</a>
      </div>
    </Card>
  );
};
