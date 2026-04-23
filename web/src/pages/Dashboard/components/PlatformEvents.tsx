import React from 'react';
import { Card } from 'antd';

export const PlatformEvents: React.FC = () => {
  const events = [
    { text: '用户 admin 执行了部署操作', time: '2 分钟前', dotColor: 'bg-blue-500' },
    { text: '集群-北京生产 节点 node-10-0-1-8 上线', time: '8 分钟前', dotColor: 'bg-green-500' },
    { text: '审批单 #AP202405121000 已通过', time: '15 分钟前', dotColor: 'bg-purple-500' },
    { text: 'AI 任务执行完成：重启异常 Pod', time: '21 分钟前', dotColor: 'bg-blue-500' },
  ];

  return (
    <Card 
      title="平台动态" 
      className="h-full shadow-sm border-none flex flex-col"
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}
    >
      <div className="flex-1 overflow-auto min-h-0">
        <div className="flex flex-col gap-6">
          {events.map((evt, idx) => (
            <div key={idx} className="flex justify-between items-center text-sm">
              <div className="flex items-center gap-3">
                <div className={`w-2 h-2 rounded-full ${evt.dotColor}`}></div>
                <span className="text-gray-700">{evt.text}</span>
              </div>
              <span className="text-gray-400 text-xs">{evt.time}</span>
            </div>
          ))}
        </div>
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看全部动态 >
      </div>
    </Card>
  );
};
