import React from 'react';
import { Card, Progress } from 'antd';
import type { Host } from '../../../../api/modules/hosts';

interface HostResourceSummaryCardProps {
  host: Host | null;
  loading?: boolean;
}

const MiniSparkline: React.FC = () => (
  <svg width="52" height="16" viewBox="0 0 52 16" fill="none" aria-hidden="true">
    <path d="M1 13.5L8 13.5L12 9.5L20 13.5L29 13.5L33 7.5L40 13.5L46 13.5L51 10.5" stroke="#7AA7FF" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

const HostResourceSummaryCard: React.FC<HostResourceSummaryCardProps> = ({ host, loading }) => {
  const cpuUsage = host?.cpuUsagePct || 23;
  const memoryUsage = host?.memoryUsagePct || 48;
  const diskUsage = host?.diskUsagePct || 35;

  return (
    <Card
      title="资源使用率"
      loading={loading}
      className="h-full [&_.ant-card-head]:px-4 [&_.ant-card-body]:p-0"
    >
      <div className="grid grid-cols-3">
        <div className="border-r border-slate-100 px-5 py-4 h-[140px]">
          <div className="flex items-center gap-4">
            <Progress
              type="circle"
              percent={cpuUsage}
              size={48}
              strokeWidth={12}
              format={() => ''}
              strokeColor={cpuUsage > 80 ? '#ff4d4f' : '#3b82f6'}
            />
            <div className="min-w-0 flex-1">
              <div className="text-[13px] text-slate-400">CPU 使用率</div>
              <div className="mt-1 text-[30px] font-semibold leading-none text-slate-800">{cpuUsage}%</div>
              <div className="mt-2 flex items-center justify-between gap-2">
                <span className="text-xs text-slate-500">0.92 / {host?.cpu || 4} Core</span>
                <MiniSparkline />
              </div>
            </div>
          </div>
        </div>

        <div className="border-r border-slate-100 px-5 py-4 h-[140px]">
          <div className="flex items-center gap-4">
            <Progress
              type="circle"
              percent={memoryUsage}
              size={48}
              strokeWidth={12}
              format={() => ''}
              strokeColor={memoryUsage > 80 ? '#ff4d4f' : '#3b82f6'}
            />
            <div className="min-w-0 flex-1">
              <div className="text-[13px] text-slate-400">内存使用率</div>
              <div className="mt-1 text-[30px] font-semibold leading-none text-slate-800">{memoryUsage}%</div>
              <div className="mt-2 flex items-center justify-between gap-2">
                <span className="text-xs text-slate-500">1.92 / {host?.memory ? (host.memory / 1024).toFixed(1) : 4} GB</span>
                <MiniSparkline />
              </div>
            </div>
          </div>
        </div>

        <div className="px-5 py-4 h-[140px]">
          <div className="flex items-center gap-4">
            <Progress
              type="circle"
              percent={diskUsage}
              size={48}
              strokeWidth={12}
              format={() => ''}
              strokeColor={diskUsage > 80 ? '#ff4d4f' : '#3b82f6'}
            />
            <div className="min-w-0 flex-1">
              <div className="text-[13px] text-slate-400">磁盘使用率</div>
              <div className="mt-1 text-[30px] font-semibold leading-none text-slate-800">{diskUsage}%</div>
              <div className="mt-2 flex items-center justify-between gap-2">
                <span className="text-xs text-slate-500">80 / {host?.disk || 200} GB</span>
                <MiniSparkline />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-3 border-t border-slate-100">
        <div className="border-r border-slate-100 px-5 py-4 h-[140px]">
          <div className="text-[13px] text-slate-400">负载 (1/5/5min)</div>
          <div className="mt-3 flex items-center justify-between gap-2">
            <div className="text-[24px] font-semibold leading-none text-slate-800 whitespace-nowrap">0.32 / 0.28 / 0.25</div>
            <MiniSparkline />
          </div>
        </div>

        <div className="border-r border-slate-100 px-5 py-4 h-[140px]">
          <div className="text-[13px] text-slate-400">运行进程数</div>
          <div className="mt-3 flex items-center justify-between gap-2">
            <div className="text-[30px] font-semibold leading-none text-slate-800">128</div>
            <MiniSparkline />
          </div>
        </div>

        <div className="px-5 py-4 h-[140px]">
          <div className="text-[13px] text-slate-400">当前连接数</div>
          <div className="mt-3 flex items-center justify-between gap-2">
            <div className="text-[30px] font-semibold leading-none text-slate-800">36</div>
            <MiniSparkline />
          </div>
        </div>
      </div>
    </Card>
  );
};

export default HostResourceSummaryCard;
