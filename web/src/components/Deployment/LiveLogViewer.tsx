import React, { useEffect, useRef } from 'react';
import { Card } from 'antd';

interface LiveLogViewerProps {
  logs: string[];
  autoScroll?: boolean;
  title?: string;
}

const LiveLogViewer: React.FC<LiveLogViewerProps> = ({ logs, autoScroll = true, title = '实时日志' }) => {
  const logEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (autoScroll && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  return (
    <Card title={title} className="h-full">
      <div className="bg-gray-900 text-gray-100 p-4 rounded font-mono text-xs overflow-auto" style={{ maxHeight: '500px' }}>
        {logs.length === 0 ? (
          <div className="text-gray-500">等待日志输出...</div>
        ) : (
          logs.map((log, index) => (
            <div key={index} className="whitespace-pre-wrap">
              <span className="text-gray-500 mr-2">[{index + 1}]</span>
              {log}
            </div>
          ))
        )}
        <div ref={logEndRef} />
      </div>
    </Card>
  );
};

export default LiveLogViewer;
