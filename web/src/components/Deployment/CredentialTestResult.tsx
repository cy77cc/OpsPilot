import React from 'react';
import { Card, Descriptions, Tag, Space } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, ClockCircleOutlined } from '@ant-design/icons';

interface CredentialTestResultProps {
  credentialId: number;
  connected: boolean;
  message: string;
  latencyMs?: number;
  timestamp?: string;
}

const CredentialTestResult: React.FC<CredentialTestResultProps> = ({
  credentialId,
  connected,
  message,
  latencyMs,
  timestamp,
}) => {
  return (
    <Card
      size="small"
      className={`border-l-4 ${
        connected ? 'border-l-green-500 bg-green-50' : 'border-l-red-500 bg-red-50'
      }`}
    >
      <Space direction="vertical" className="w-full">
        <div className="flex items-center justify-between">
          <Space>
            {connected ? (
              <CheckCircleOutlined className="text-xl text-green-600" />
            ) : (
              <CloseCircleOutlined className="text-xl text-red-600" />
            )}
            <span className="font-semibold">
              {connected ? '连接成功' : '连接失败'}
            </span>
          </Space>
          {timestamp && (
            <Space className="text-xs text-gray-500">
              <ClockCircleOutlined />
              <span>{new Date(timestamp).toLocaleString()}</span>
            </Space>
          )}
        </div>

        <Descriptions size="small" column={1}>
          <Descriptions.Item label="凭证 ID">{credentialId}</Descriptions.Item>
          {latencyMs !== undefined && (
            <Descriptions.Item label="延迟">
              <Tag color={latencyMs < 100 ? 'green' : latencyMs < 500 ? 'orange' : 'red'}>
                {latencyMs}ms
              </Tag>
            </Descriptions.Item>
          )}
          <Descriptions.Item label="详情">
            <span className={connected ? 'text-green-700' : 'text-red-700'}>
              {message}
            </span>
          </Descriptions.Item>
        </Descriptions>
      </Space>
    </Card>
  );
};

export default CredentialTestResult;
