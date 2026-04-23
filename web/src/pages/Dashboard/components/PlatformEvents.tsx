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
    <Card title="平台动态" className="h-full shadow-sm border-none" extra={<a href="#" className="text-blue-500 text-sm">查看全部动态 &gt;</a>}>
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
    </Card>
  );
};
