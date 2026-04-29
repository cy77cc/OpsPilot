import React from 'react';
import { Card, Descriptions, Tag, Tooltip, message } from 'antd';
import { CopyOutlined } from '@ant-design/icons';
import type { Host } from '../../../../api/modules/hosts';

interface HostBasicInfoCardProps {
  host: Host | null;
  loading?: boolean;
}

const HostBasicInfoCard: React.FC<HostBasicInfoCardProps> = ({ host, loading }) => {
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    message.success('已复制到剪贴板');
  };

  const renderValue = (value: string | number | undefined, copyable = false) => {
    if (!value && value !== 0) return '-';
    return (
      <div className="flex items-center gap-1 group">
        <span>{value}</span>
        {copyable && (
          <Tooltip title="复制">
            <CopyOutlined 
              className="text-gray-400 cursor-pointer hover:text-blue-600 opacity-0 group-hover:opacity-100 transition-opacity" 
              onClick={() => copyToClipboard(String(value))}
            />
          </Tooltip>
        )}
      </div>
    );
  };

  return (
    <Card title="基本信息" loading={loading} className="h-full">
      <Descriptions column={2} size="small" layout="horizontal">
        <Descriptions.Item label="主机名称">{renderValue(host?.name, true)}</Descriptions.Item>
        <Descriptions.Item label="IP 地址">{renderValue(host?.ip, true)}</Descriptions.Item>
        <Descriptions.Item label="主机组">Web 服务器</Descriptions.Item>
        <Descriptions.Item label="所属集群">生产集群</Descriptions.Item>
        <Descriptions.Item label="操作系统">{renderValue(host?.os)}</Descriptions.Item>
        <Descriptions.Item label="内核版本">5.15.0-94-generic</Descriptions.Item>
        <Descriptions.Item label="系统架构">x86_64</Descriptions.Item>
        <Descriptions.Item label="运行时间">15 天 3 小时 22 分</Descriptions.Item>
        <Descriptions.Item label="Agent 版本">v1.6.3</Descriptions.Item>
        <Descriptions.Item label="安装目录">/opt/opspilot/agent</Descriptions.Item>
        <Descriptions.Item label="最后上线">
          {host?.lastActive ? new Date(host.lastActive).toLocaleString() : '-'}
        </Descriptions.Item>
        <Descriptions.Item label="插件数">
          {(host?.pluginInstances || []).length}
        </Descriptions.Item>
        <Descriptions.Item label="创建时间">
          {host?.createdAt ? new Date(host.createdAt).toLocaleString() : '-'}
        </Descriptions.Item>
        <Descriptions.Item label="标签" span={2}>
          <div className="flex flex-wrap gap-1">
            {host?.tags?.map(tag => (
              <Tag key={tag} className="bg-gray-50 border-gray-200 text-gray-600 m-0">
                {tag}
              </Tag>
            )) || '-'}
          </div>
        </Descriptions.Item>
        <Descriptions.Item label="描述" span={2}>
          {host?.description || '-'}
        </Descriptions.Item>
      </Descriptions>
    </Card>
  );
};

export default HostBasicInfoCard;
