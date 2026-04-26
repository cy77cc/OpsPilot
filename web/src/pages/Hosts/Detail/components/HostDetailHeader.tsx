import React from 'react';
import { Button, Space, Tag, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { ArrowLeftOutlined, ClockCircleOutlined, DownOutlined, EditOutlined, ConsoleSqlOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { Host } from '../../../../api/modules/hosts';

interface HostDetailHeaderProps {
  host: Host | null;
  onEdit: () => void;
  onTerminal: () => void;
  onHealthCheck: () => void;
  onAction: (action: string) => void;
  loading?: boolean;
}

const HostDetailHeader: React.FC<HostDetailHeaderProps> = ({
  host,
  onEdit,
  onTerminal,
  onHealthCheck,
  onAction,
}) => {
  const navigate = useNavigate();

  const statusMap: Record<string, { color: string; text: string }> = {
    online: { color: 'success', text: '运行中' },
    offline: { color: 'default', text: '已离线' },
    maintenance: { color: 'warning', text: '维护中' },
    unknown: { color: 'error', text: '未知' },
  };

  const status = statusMap[host?.status || 'unknown'] || statusMap.unknown;

  const statusToneMap: Record<string, string> = {
    online: '#22c55e',
    offline: '#94a3b8',
    maintenance: '#f59e0b',
    unknown: '#ef4444',
  };

  const statusDotColor = statusToneMap[host?.status || 'unknown'] || statusToneMap.unknown;

  const metadata = [
    host?.ip,
    host?.os || 'Linux',
    host?.osVersion || '未知版本',
    host?.cpu ? `${host.cpu} 核 ${host.memory}GB` : '配置未知',
    host?.environment ? `${host.environment.toUpperCase()} 环境` : null,
    'Web 服务器',
  ].filter(Boolean) as string[];

  const lastActiveText = host?.lastActive ? new Date(host.lastActive).toLocaleString() : '-';

  const moreActions: MenuProps['items'] = [
    { key: 'restart', label: '重启主机' },
    { key: 'shutdown', label: '关机' },
    { type: 'divider' },
    { key: 'refresh', label: '刷新配置' },
    { key: 're-enroll', label: '重新纳管' },
  ];

  return (
    <div className="w-full">
      <div className="flex flex-col gap-2.5 xl:flex-row xl:items-start xl:justify-between">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2.5">
            <Button
              type="text"
              icon={<ArrowLeftOutlined />}
              onClick={() => navigate('/resources/hosts')}
              aria-label="返回主机列表"
              className="h-8 w-8 shrink-0 rounded-lg border border-slate-200 bg-white p-0 text-slate-500 hover:border-slate-300 hover:text-slate-700"
            />

            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h1 className="m-0 text-[22px] font-semibold leading-tight tracking-tight text-slate-900">
                  {host?.name || '...'}
                </h1>
                <span className="inline-flex items-center gap-2 text-sm font-medium text-slate-600">
                  <span
                    className="h-2.5 w-2.5 rounded-full"
                    style={{ backgroundColor: statusDotColor }}
                  />
                  {status.text}
                </span>
              </div>

              <div className="mt-1.5 flex flex-wrap gap-1.5">
                {metadata.map((item) => (
                  <Tag
                    key={item}
                    className="m-0 rounded-full border-slate-200 bg-slate-50 px-2.5 py-0.5 text-xs font-medium text-slate-600"
                  >
                    {item}
                  </Tag>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-1.5 xl:items-end">
          <div className="inline-flex items-center gap-2 rounded-xl bg-slate-50 px-3 py-1 text-xs font-medium text-slate-500">
            <ClockCircleOutlined />
            <span>最后更新：{lastActiveText}</span>
          </div>

          <Space wrap size={6} className="justify-start xl:justify-end">
            <Dropdown menu={{ items: moreActions, onClick: (info) => onAction(info.key) }}>
              <Button className="h-8 rounded-xl border-slate-200 bg-white px-3 text-slate-700 shadow-none">
                更多操作 <DownOutlined />
              </Button>
            </Dropdown>
            <Button
              onClick={onHealthCheck}
              className="h-8 rounded-xl border-slate-200 bg-white px-3 text-slate-700 shadow-none"
            >
              健康检查
            </Button>
            <Button
              icon={<ConsoleSqlOutlined />}
              onClick={onTerminal}
              className="h-8 rounded-xl border-slate-200 bg-white px-3 text-slate-700 shadow-none"
            >
              连接终端
            </Button>
            <Button
              type="primary"
              icon={<EditOutlined />}
              onClick={onEdit}
              className="h-8 rounded-xl shadow-none"
            >
              编辑主机
            </Button>
          </Space>
        </div>
      </div>
    </div>
  );
};

export default HostDetailHeader;
