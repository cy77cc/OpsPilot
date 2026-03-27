import React from 'react';
import { Card, Steps, Tag } from 'antd';
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  SyncOutlined,
  CloseCircleOutlined,
  WarningOutlined,
} from '@ant-design/icons';

interface ReleaseStateFlowProps {
  currentState: string;
  states?: string[];
}

const ReleaseStateFlow: React.FC<ReleaseStateFlowProps> = ({
  currentState,
  states = ['pending_approval', 'approved', 'applying', 'applied'],
}) => {
  const stateConfig: Record<string, { icon: React.ReactNode; color: string; text: string }> = {
    pending_approval: { icon: <ClockCircleOutlined />, color: 'orange', text: '待审批' },
    approved: { icon: <CheckCircleOutlined />, color: 'blue', text: '已批准' },
    applying: { icon: <SyncOutlined spin />, color: 'processing', text: '部署中' },
    applied: { icon: <CheckCircleOutlined />, color: 'success', text: '已完成' },
    failed: { icon: <CloseCircleOutlined />, color: 'error', text: '失败' },
    rejected: { icon: <CloseCircleOutlined />, color: 'default', text: '已拒绝' },
  };

  const getCurrentStep = () => {
    const index = states.indexOf(currentState);
    return index >= 0 ? index : 0;
  };

  const getStepStatus = (state: string): 'wait' | 'process' | 'finish' | 'error' => {
    if (currentState === 'failed' && state === currentState) return 'error';
    if (currentState === 'rejected' && state === currentState) return 'error';
    if (state === currentState) return 'process';
    const currentIndex = states.indexOf(currentState);
    const stateIndex = states.indexOf(state);
    return stateIndex < currentIndex ? 'finish' : 'wait';
  };

  const items = states.map((state) => {
    const config = stateConfig[state] || { icon: null, color: 'default', text: state };
    return {
      title: config.text,
      status: getStepStatus(state),
      icon: state === currentState ? config.icon : undefined,
    };
  });

  return (
    <Card title="发布状态流程">
      <Steps current={getCurrentStep()} items={items} />
    </Card>
  );
};

export default ReleaseStateFlow;
