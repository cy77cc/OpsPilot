import React, { useState } from 'react';
import { Steps, Button, Typography, Card, Space, Avatar } from 'antd';
import { UserOutlined, LockOutlined, SaveOutlined, MessageOutlined, RocketOutlined } from '@ant-design/icons';

const { Text, Paragraph } = Typography;

// 新手引导步骤数据
export interface OnboardingStep {
  id: string;
  title: string;
  description: string;
  icon: React.ReactNode;
  component?: React.ReactNode;
}

// 新手引导属性
interface OnboardingStepsProps {
  steps?: OnboardingStep[];
  onComplete?: () => void;
  className?: string;
}

// 新手引导步骤组件
const OnboardingSteps: React.FC<OnboardingStepsProps> = ({ 
  steps, 
  onComplete, 
  className 
}) => {
  const [current, setCurrent] = useState(0);

  // 默认引导步骤
  const defaultSteps: OnboardingStep[] = [
    {
      id: 'step-1',
      title: '欢迎使用 DevOps 平台',
      description: '这是一个功能强大的 DevOps 平台，帮助您轻松管理和监控您的基础设施。',
      icon: <UserOutlined />
    },
    {
      id: 'step-2',
      title: '了解权限系统',
      description: '平台采用 RBAC 权限模型，确保您的数据和操作安全。',
      icon: <LockOutlined />
    },
    {
      id: 'step-3',
      title: '管理您的主机',
      description: '轻松查看和管理您的所有主机，监控其状态和性能。',
      icon: <SaveOutlined />
    },
    {
      id: 'step-4',
      title: '使用 AI 助手',
      description: '遇到问题？随时向 AI 助手寻求帮助和建议。',
      icon: <MessageOutlined />
    },
    {
      id: 'step-5',
      title: '开始您的 DevOps 之旅',
      description: '现在您已经了解了平台的核心功能，开始使用吧！',
      icon: <RocketOutlined />
    }
  ];

  const onboardingSteps = steps || defaultSteps;

  // 处理下一步
  const handleNext = () => {
    if (current < onboardingSteps.length - 1) {
      setCurrent(current + 1);
    } else {
      onComplete?.();
    }
  };

  // 处理上一步
  const handlePrevious = () => {
    if (current > 0) {
      setCurrent(current - 1);
    }
  };

  return (
    <Card 
      title={
        <Space>
          <UserOutlined />
          <Text strong>新手引导</Text>
        </Space>
      }
      className={`onboarding-steps ${className || ''}`}
      style={{ maxWidth: 800, margin: '0 auto' }}
    >
      <Steps 
        current={current} 
        style={{ marginBottom: 32 }} 
        items={onboardingSteps.map((step) => ({
          key: step.id,
          title: step.title,
          icon: <Avatar size={24} icon={step.icon} style={{ backgroundColor: '#1890ff' }} />
        }))}
      />

      <div style={{ marginBottom: 32, textAlign: 'center' }}>
        <Space size="large" direction="vertical" style={{ width: '100%' }}>
          <Avatar 
            size={80} 
            icon={onboardingSteps[current].icon} 
            style={{ backgroundColor: '#1890ff', margin: '0 auto' }} 
          />
          <Text strong style={{ fontSize: 18 }}>{onboardingSteps[current].title}</Text>
          <Paragraph style={{ textAlign: 'center', maxWidth: 600, margin: '0 auto' }}>
            {onboardingSteps[current].description}
          </Paragraph>
          {onboardingSteps[current].component}
        </Space>
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <Button 
          onClick={handlePrevious} 
          disabled={current === 0}
        >
          上一步
        </Button>
        <Button 
          type="primary" 
          onClick={handleNext}
        >
          {current === onboardingSteps.length - 1 ? '完成' : '下一步'}
        </Button>
      </div>
    </Card>
  );
};

export default OnboardingSteps;