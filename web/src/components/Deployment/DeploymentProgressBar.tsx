import React from 'react';
import { Card, Progress, Tag, Space } from 'antd';
import { CheckCircleOutlined, SyncOutlined, CloseCircleOutlined } from '@ant-design/icons';

interface PodStatus {
  name: string;
  status: 'running' | 'pending' | 'failed' | 'succeeded';
  ready: boolean;
}

interface DeploymentProgressBarProps {
  phase?: string;
  progress?: number;
  pods?: PodStatus[];
  runtimeType: 'k8s' | 'compose';
}

const DeploymentProgressBar: React.FC<DeploymentProgressBarProps> = ({
  phase,
  progress = 0,
  pods = [],
  runtimeType,
}) => {
  const getPodStatusIcon = (status: string, ready: boolean) => {
    if (status === 'running' && ready) {
      return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
    }
    if (status === 'running' && !ready) {
      return <SyncOutlined spin style={{ color: '#1890ff' }} />;
    }
    if (status === 'failed') {
      return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
    }
    return <SyncOutlined spin style={{ color: '#d9d9d9' }} />;
  };

  const getProgressColor = () => {
    if (progress >= 100) return '#52c41a';
    if (progress >= 50) return '#1890ff';
    return '#f59e0b';
  };

  return (
    <Card title="部署进度">
      <div className="space-y-4">
        {phase && (
          <div>
            <span className="text-sm text-gray-600">当前阶段: </span>
            <Tag color="processing">{phase}</Tag>
          </div>
        )}

        <div>
          <div className="flex justify-between text-sm mb-2">
            <span className="text-gray-600">总体进度</span>
            <span className="font-semibold">{progress}%</span>
          </div>
          <Progress percent={progress} strokeColor={getProgressColor()} />
        </div>

        {runtimeType === 'k8s' && pods.length > 0 && (
          <div>
            <div className="text-sm font-semibold mb-2">Pod 状态:</div>
            <div className="space-y-2">
              {pods.map((pod) => (
                <div key={pod.name} className="flex items-center justify-between p-2 bg-gray-50 rounded">
                  <Space>
                    {getPodStatusIcon(pod.status, pod.ready)}
                    <span className="text-sm">{pod.name}</span>
                  </Space>
                  <Tag color={pod.ready ? 'success' : pod.status === 'failed' ? 'error' : 'processing'}>
                    {pod.status}
                  </Tag>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </Card>
  );
};

export default DeploymentProgressBar;
