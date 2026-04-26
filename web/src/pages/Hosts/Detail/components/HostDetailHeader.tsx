import React from 'react';
import { Breadcrumb, Button, Space, Tag, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { ArrowLeftOutlined, DownOutlined, EditOutlined, ConsoleSqlOutlined } from '@ant-design/icons';
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

  const moreActions: MenuProps['items'] = [
    { key: 'restart', label: '重启主机' },
    { key: 'shutdown', label: '关机' },
    { type: 'divider' },
    { key: 'refresh', label: '刷新配置' },
    { key: 're-enroll', label: '重新纳管' },
  ];

  return (
    <div>
      
      <div className="flex justify-between items-start">
        <Space direction="vertical" size={0}>
          <Button 
            type="link" 
            icon={<ArrowLeftOutlined />} 
            onClick={() => navigate('/resources/hosts')}
            className="p-0 h-auto mb-1 text-gray-500 hover:text-blue-600"
          >
            返回主机列表
          </Button>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold m-0 leading-none">{host?.name || '...'}</h1>
            <Tag color={status.color} className="m-0 font-medium">
              {status.text}
            </Tag>
          </div>
          <div className="flex flex-wrap gap-2 mt-2">
            <Tag className="bg-gray-100 border-none px-2 py-0.5">{host?.ip}</Tag>
            <Tag className="bg-gray-100 border-none px-2 py-0.5">{host?.os || 'Linux'}</Tag>
            <Tag className="bg-gray-100 border-none px-2 py-0.5">{host?.osVersion || '...'}</Tag>
            <Tag className="bg-gray-100 border-none px-2 py-0.5">
              {host?.cpu ? `${host.cpu} 核 ${host.memory}GB` : '...'}
            </Tag>
            {host?.environment && (
              <Tag className="bg-blue-50 text-blue-600 border-none px-2 py-0.5">
                {host.environment.toUpperCase()} 环境
              </Tag>
            )}
            <Tag className="bg-gray-100 border-none px-2 py-0.5">Web 服务器</Tag>
          </div>
        </Space>

        <Space align="start">
          <div className="text-right mr-4 hidden md:block">
            <div className="text-gray-400 text-xs">最后更新</div>
            <div className="text-gray-600 text-sm">
              {host?.lastActive ? new Date(host.lastActive).toLocaleString() : '-'}
            </div>
          </div>
          <Space>
            <Dropdown menu={{ items: moreActions, onClick: (info) => onAction(info.key) }}>
              <Button>
                更多操作 <DownOutlined />
              </Button>
            </Dropdown>
            <Button onClick={onHealthCheck}>
              健康检查
            </Button>
            <Button icon={<ConsoleSqlOutlined />} onClick={onTerminal}>
              连接终端
            </Button>
            <Button type="primary" icon={<EditOutlined />} onClick={onEdit}>
              编辑主机
            </Button>
          </Space>
        </Space>
      </div>
    </div>
  );
};

export default HostDetailHeader;
