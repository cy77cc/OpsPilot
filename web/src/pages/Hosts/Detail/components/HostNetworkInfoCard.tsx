import React from 'react';
import { Card, Descriptions, Tooltip, message } from 'antd';
import { CopyOutlined } from '@ant-design/icons';
import type { Host } from '../../../../api/modules/hosts';

interface HostNetworkInfoCardProps {
  host: Host | null;
  loading?: boolean;
}

const HostNetworkInfoCard: React.FC<HostNetworkInfoCardProps> = ({ host, loading }) => {
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    message.success('已复制到剪贴板');
  };

  const renderValue = (value: string | undefined, copyable = false) => {
    if (!value) return '-';
    return (
      <div className="flex items-center gap-1 group">
        <span>{value}</span>
        {copyable && (
          <Tooltip title="复制">
            <CopyOutlined 
              className="text-gray-400 cursor-pointer hover:text-blue-600 opacity-0 group-hover:opacity-100 transition-opacity" 
              onClick={() => copyToClipboard(value)}
            />
          </Tooltip>
        )}
      </div>
    );
  };

  return (
    <Card title="网络信息" loading={loading} className="h-full">
      <Descriptions column={1} size="small">
        <Descriptions.Item label="内网 IP">{renderValue(host?.ip, true)}</Descriptions.Item>
        <Descriptions.Item label="公网 IP">{renderValue('8.8.8.8', true)}</Descriptions.Item>
        <Descriptions.Item label="MAC 地址">00:50:56:af:3e:88</Descriptions.Item>
        <Descriptions.Item label="主机名">{renderValue(host?.name)}</Descriptions.Item>
        <Descriptions.Item label="网卡">ens33</Descriptions.Item>
        <Descriptions.Item label="带宽上限">1000 Mbps</Descriptions.Item>
      </Descriptions>
    </Card>
  );
};

export default HostNetworkInfoCard;
