/**
 * AI Copilot 抽屉组件
 * 支持场景自动感知与手动切换
 */
import React from 'react';
import { Button, Drawer, Space, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import { AISurfaceBoundary } from './AISurfaceBoundary';
import type { SceneOption } from './hooks/useAutoScene';

interface AIAssistantDrawerProps {
  open: boolean;
  onClose: () => void;
  scene: string;
  selectValue?: string;
  onSceneChange?: (scene: string) => void;
  availableScenes?: SceneOption[];
  isAuto?: boolean;
}

/**
 * AI Copilot 抽屉组件
 */
export function AIAssistantDrawer({
  open,
  onClose,
  scene,
}: AIAssistantDrawerProps) {
  const navigate = useNavigate();

  return (
    <Drawer
      open={open}
      onClose={onClose}
      placement="right"
      width={420}
      closable
      maskClosable
      styles={{ body: { display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 32 } }}
      title="AI Assistant"
    >
      <AISurfaceBoundary>
        <Space direction="vertical" size={16} style={{ width: '100%', textAlign: 'center' }}>
          <Typography.Title level={4} style={{ marginBottom: 0 }}>
            AI 工作台已迁移到独立页面
          </Typography.Title>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            Phase 1 的问答与诊断体验统一在 `/ai`。当前场景为 {scene}，点击下方按钮继续。
          </Typography.Paragraph>
          <Button
            type="primary"
            size="large"
            onClick={() => {
              onClose();
              navigate('/ai');
            }}
          >
            打开 AI Assistant
          </Button>
        </Space>
      </AISurfaceBoundary>
    </Drawer>
  );
}

export default AIAssistantDrawer;
