import React, { useEffect, useRef } from 'react';
import { Card } from 'antd';

interface BootstrapLogViewerProps {
  logs: string[];
  autoScroll?: boolean;
}

const BootstrapLogViewer: React.FC<BootstrapLogViewerProps> = ({ logs, autoScroll = true }) => {
  const logEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (autoScroll && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  return (
    <Card title="实时日志" className="h-full">
      <div className="bg-gray-900 text-gray-100 p-4 rounded font-mono text-xs overflow-auto" style={{ maxHeight: '400px' }}>
        {logs.length === 0 ? (
          <div className="text-gray-500">等待日志输出...</div>
        ) : (
          logs.map((log, index) => (
            <div key={index} className="whitespace-pre-wrap">
              {log}
            </div>
          ))
        )}
        <div ref={logEndRef} />
      </div>
    </Card>
  );
};

export default BootstrapLogViewer;
