import React from 'react';
import { Card, Table, Tag } from 'antd';
import type { AlertItem } from '../../../api/modules/dashboard';
import dayjs from 'dayjs';

export const RecentAlerts: React.FC<{ data?: AlertItem[] }> = ({ data }) => {
  const columns = [
    { 
      title: '级别', 
      dataIndex: 'severity', 
      key: 'severity',
      width: 100,
      render: (text: string) => {
        const severity = text?.toLowerCase();
        let color = 'warning';
        let label = '警告';
        if (severity === 'critical' || severity === 'error') {
          color = 'error';
          label = '严重';
        } else if (severity === 'info') {
          color = 'processing';
          label = '信息';
        }
        return <Tag color={color}>{label}</Tag>;
      }
    },
    { 
      title: '告警内容', 
      dataIndex: 'title', 
      key: 'title',
      render: (text: string) => <span className="font-medium text-gray-800">{text}</span>
    },
    { 
      title: '来源', 
      dataIndex: 'source', 
      key: 'source', 
      render: (text: string) => <span className="text-xs text-gray-500 font-mono">{text}</span> 
    },
    { 
      title: '开始时间', 
      dataIndex: 'createdAt', 
      key: 'createdAt', 
      width: 180,
      render: (text: string) => <span className="text-gray-400 text-xs">{text ? dayjs(text).format('YYYY-MM-DD HH:mm:ss') : '-'}</span> 
    },
    { 
      title: '操作', 
      key: 'action', 
      width: 80,
      render: () => <a className="text-blue-500 text-sm hover:underline">处理</a> 
    },
  ];

  return (
    <Card 
      title="最近告警" 
      className="h-full shadow-sm border-none flex flex-col" 
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}
    >
      <div className="flex-1 overflow-auto min-h-0">
        <Table 
          columns={columns} 
          dataSource={data} 
          pagination={false} 
          size="small" 
          rowKey="id" 
          className="border-none"
        />
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看全部告警 &gt;
      </div>
    </Card>
  );
};
