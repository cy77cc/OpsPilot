import React from 'react';
import { Card, Tag, Space, Descriptions } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, SyncOutlined } from '@ant-design/icons';

interface HealthProbe {
  type: 'liveness' | 'readiness' | 'startup';
  status: 'passing' | 'failing' | 'unknown';
  lastCheck?: string;
  message?: string;
}

interface HealthCheckStatusProps {
  probes: HealthProbe[];
}

const HealthCheckStatus: React.FC<HealthCheckStatusProps> = ({ probes }) => {
  const getProbeIcon = (status: string) => {
    switch (status) {
      case 'passing':
        return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
      case 'failing':
        return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
      default:
        return <SyncOutlined style={{ color: '#d9d9d9' }} />;
    }
  };

  const getProbeColor = (status: string) => {
    switch (status) {
      case 'passing':
        return 'success';
      case 'failing':
        return 'error';
      default:
        return 'default';
    }
  };

  const getProbeLabel = (type: string) => {
    switch (type) {
      case 'liveness':
        return '存活探针';
      case 'readiness':
        return '就绪探针';
      case 'startup':
        return '启动探针';
      default:
        return type;
    }
  };

  if (probes.length === 0) {
    return (
      <Card title="健康检查">
        <div className="text-sm text-gray-500">暂无健康检查数据</div>
      </Card>
    );
  }

  return (
    <Card title="健康检查">
      <div className="space-y-3">
        {probes.map((probe, index) => (
          <div key={index} className="p-3 border border-gray-200 rounded">
            <div className="flex items-center justify-between mb-2">
              <Space>
                {getProbeIcon(probe.status)}
                <span className="font-semibold">{getProbeLabel(probe.type)}</span>
              </Space>
              <Tag color={getProbeColor(probe.status)}>{probe.status}</Tag>
            </div>
            {probe.lastCheck && (
              <div className="text-xs text-gray-500">
                最后检查: {new Date(probe.lastCheck).toLocaleString()}
              </div>
            )}
            {probe.message && (
              <div className="text-xs text-gray-600 mt-1">{probe.message}</div>
            )}
          </div>
        ))}
      </div>
    </Card>
  );
};

export default HealthCheckStatus;
