import React, { useState, useEffect } from 'react';
import { Card, Steps, Timeline, Alert, Button, Space, Tag } from 'antd';
import { CheckCircleOutlined, LoadingOutlined, CloseCircleOutlined, SyncOutlined, ClockCircleOutlined } from '@ant-design/icons';

interface BootstrapPhase {
  name: string;
  status: 'pending' | 'running' | 'success' | 'failed';
  startTime?: string;
  endTime?: string;
  logs?: string[];
}

interface BootstrapProgressTrackerProps {
  jobId: string;
  phases: BootstrapPhase[];
  currentPhase?: string;
  overallStatus: 'queued' | 'running' | 'succeeded' | 'failed';
  onRefresh?: () => void;
}

const BootstrapProgressTracker: React.FC<BootstrapProgressTrackerProps> = ({
  jobId,
  phases,
  currentPhase,
  overallStatus,
  onRefresh,
}) => {
  const getPhaseIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
      case 'running':
        return <LoadingOutlined style={{ color: '#1890ff' }} />;
      case 'failed':
        return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
      default:
        return <ClockCircleOutlined style={{ color: '#d9d9d9' }} />;
    }
  };

  const getPhaseStatus = (status: string): 'wait' | 'process' | 'finish' | 'error' => {
    switch (status) {
      case 'success':
        return 'finish';
      case 'running':
        return 'process';
      case 'failed':
        return 'error';
      default:
        return 'wait';
    }
  };

  const getCurrentStep = () => {
    const runningIndex = phases.findIndex((p) => p.status === 'running');
    if (runningIndex !== -1) return runningIndex;
    const successIndex = phases.findIndex((p) => p.status === 'success');
    if (successIndex !== -1) return successIndex + 1;
    return 0;
  };

  const items = phases.map((phase) => ({
    title: phase.name,
    status: getPhaseStatus(phase.status),
    icon: getPhaseIcon(phase.status),
  }));

  return (
    <div className="space-y-6">
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold">Bootstrap 进度</h3>
            <p className="text-sm text-gray-500">Job ID: {jobId}</p>
          </div>
          <Space>
            <Tag color={overallStatus === 'succeeded' ? 'success' : overallStatus === 'failed' ? 'error' : 'processing'}>
              {overallStatus}
            </Tag>
            {onRefresh && (
              <Button icon={<SyncOutlined />} onClick={onRefresh} size="small">
                刷新
              </Button>
            )}
          </Space>
        </div>

        <Steps current={getCurrentStep()} items={items} />
      </Card>

      {phases.map((phase, index) => (
        phase.status !== 'pending' && (
          <Card
            key={index}
            title={
              <Space>
                {getPhaseIcon(phase.status)}
                <span>{phase.name}</span>
              </Space>
            }
            size="small"
          >
            {phase.startTime && (
              <div className="text-xs text-gray-500 mb-2">
                开始时间: {new Date(phase.startTime).toLocaleString()}
                {phase.endTime && ` | 结束时间: ${new Date(phase.endTime).toLocaleString()}`}
              </div>
            )}
            {phase.logs && phase.logs.length > 0 && (
              <div className="bg-gray-900 text-gray-100 p-4 rounded font-mono text-xs overflow-auto max-h-60">
                {phase.logs.map((log, i) => (
                  <div key={i}>{log}</div>
                ))}
              </div>
            )}
          </Card>
        )
      ))}

      {overallStatus === 'failed' && (
        <Alert
          message="Bootstrap 失败"
          description="环境初始化过程中遇到错误，请检查日志并重试。"
          type="error"
          showIcon
        />
      )}

      {overallStatus === 'succeeded' && (
        <Alert
          message="Bootstrap 成功"
          description="环境初始化已完成，部署目标现在可以使用。"
          type="success"
          showIcon
        />
      )}
    </div>
  );
};

export default BootstrapProgressTracker;
