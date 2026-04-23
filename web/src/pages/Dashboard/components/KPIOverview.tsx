import React from 'react';
import { Card } from 'antd';
import { 
  CloudServerOutlined, 
  ClusterOutlined, 
  ProjectOutlined, 
  AppstoreOutlined, 
  RocketOutlined, 
  AlertOutlined 
} from '@ant-design/icons';

const kpiData = [
  { title: '主机总数', total: 128, sub1: '在线 102', sub2: '离线 26', icon: <CloudServerOutlined style={{ fontSize: '24px', color: '#1890ff' }} />, bgColor: 'bg-blue-50' },
  { title: '集群总数', total: 12, sub1: '正常 10', sub2: '异常 2', icon: <ClusterOutlined style={{ fontSize: '24px', color: '#52c41a' }} />, bgColor: 'bg-green-50' },
  { title: '项目总数', total: 35, sub1: '运行中 28', sub2: '已停用 7', icon: <ProjectOutlined style={{ fontSize: '24px', color: '#722ed1' }} />, bgColor: 'bg-purple-50' },
  { title: '服务总数', total: 256, sub1: '运行中 214', sub2: '异常 42', icon: <AppstoreOutlined style={{ fontSize: '24px', color: '#3f51b5' }} />, bgColor: 'bg-indigo-50' },
  { title: '部署总数', total: 512, sub1: '成功 478', sub2: '失败 34', icon: <RocketOutlined style={{ fontSize: '24px', color: '#13c2c2' }} />, bgColor: 'bg-emerald-50' },
  { title: '告警总数', total: 23, sub1: '严重 5', sub2: '警告 18', icon: <AlertOutlined style={{ fontSize: '24px', color: '#f5222d' }} />, bgColor: 'bg-red-50' },
];

export const KPIOverview: React.FC = () => {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
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
