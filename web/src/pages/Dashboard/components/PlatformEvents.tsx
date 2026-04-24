import React from 'react';
import { Card, Empty } from 'antd';
import type { EventItem } from '../../../api/modules/dashboard';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

export const PlatformEvents: React.FC<{ data?: EventItem[] }> = ({ data }) => {
  return (
    <Card 
      title="平台动态" 
      className="h-full shadow-sm border-none flex flex-col" 
      styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 } }}
    >
      <div className="flex-1 overflow-auto min-h-0 px-1">
        {data && data.length > 0 ? (
          <div className="flex flex-col gap-6">
            {data.map((evt) => (
              <div key={evt.id} className="flex justify-between items-start text-sm">
                <div className="flex items-start gap-3">
                  <div className="w-2 h-2 rounded-full bg-blue-500 mt-1.5 flex-shrink-0"></div>
                  <div className="flex flex-col gap-0.5">
                    <span className="text-gray-700 leading-snug">{evt.message}</span>
                    <span className="text-[11px] text-gray-300 uppercase tracking-tight">{evt.type}</span>
                  </div>
                </div>
                <span className="text-gray-400 text-[11px] whitespace-nowrap ml-4 pt-0.5">{dayjs(evt.createdAt).fromNow()}</span>
              </div>
            ))}
          </div>
        ) : (
          <Empty description="暂无平台动态" className="mt-10" />
        )}
      </div>
      <div className="text-right mt-4 pt-4 border-t border-gray-50 flex-shrink-0 text-blue-500 text-xs cursor-pointer hover:text-blue-600 transition-colors">
        查看全部动态 &gt;
      </div>
    </Card>
  );
};
