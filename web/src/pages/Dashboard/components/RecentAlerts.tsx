import React from 'react';
import { Card, Table, Tag } from 'antd';

export const RecentAlerts: React.FC = () => {
  const columns = [
    { 
      title: '级别', 
      dataIndex: 'level', 
      key: 'level',
      render: (text: string) => (
        <Tag color={text === '严重' ? 'error' : 'warning'}>{text}</Tag>
      )
    },
    { title: '告警内容', dataIndex: 'content', key: 'content' },
    { title: '资源对象', dataIndex: 'resource', key: 'resource', render: (text: string) => <span className="font-mono text-xs text-gray-500">{text}</span> },
    { title: '所属集群', dataIndex: 'cluster', key: 'cluster' },
    { title: '开始时间', dataIndex: 'time', key: 'time' },
    { title: '状态', dataIndex: 'status', key: 'status', render: () => <span className="text-red-500 text-sm">未恢复</span> },
    { title: '操作', key: 'action', render: () => <a href="#" className="text-blue-500 text-sm">处理</a> },
  ];

  const data = [
    { key: '1', level: '严重', content: 'Pod CrashLoopBackOff', resource: 'payment-7c9f8d5c9-abcde', cluster: '集群-北京生产', time: '2024-05-12 10:58:32' },
    { key: '2', level: '警告', content: '磁盘使用率超过 80%', resource: 'node-10-0-1-8', cluster: '集群-北京生产', time: '2024-05-12 10:55:18' },
    { key: '3', level: '警告', content: 'CPU 使用率超过 75%', resource: 'node-10-0-2-15', cluster: '集群-上海生产', time: '2024-05-12 10:52:07' },
  ];

  return (
    <Card 
      title="最近告警" 
      className="h-full shadow-sm border-none flex flex-col"
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}
    >
      <div className="flex-1 overflow-auto min-h-0">
        <Table columns={columns} dataSource={data} pagination={false} size="small" />
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看全部告警 >
      </div>
    </Card>
  );
};
