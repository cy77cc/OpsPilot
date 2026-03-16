/**
 * Copilot 兼容层
 * 保留旧组件出口，但将用户引导到 Phase 1 独立 AI 页面
 */
import React, { useMemo } from 'react';
import { RobotOutlined } from '@ant-design/icons';
import { Button, Space, Typography, theme } from 'antd';
import { useNavigate } from 'react-router-dom';
import { getSceneLabel } from './constants/sceneMapping';
import type { SceneOption } from './hooks/useAutoScene';

const { useToken } = theme;

interface CopilotProps {
  open?: boolean;
  onClose?: () => void;
  scene: string;
  selectValue?: string;
  onSceneChange?: (scene: string) => void;
  availableScenes?: SceneOption[];
  isAuto?: boolean;
}

export const Copilot: React.FC<CopilotProps> = ({
  open = true,
  onClose,
  scene,
  selectValue,
  availableScenes = [{ key: 'global', label: '全局助手' }],
  isAuto = true,
}) => {
  const navigate = useNavigate();
  const { token } = useToken();

  const sceneLabel = useMemo(() => {
    const displayValue = selectValue || scene;
    const matched = availableScenes.find((item) => item.key === displayValue);
    return matched?.label || getSceneLabel(scene);
  }, [availableScenes, scene, selectValue]);

  if (!open) return null;

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100%',
        padding: 24,
        background: token.colorBgContainer,
        color: token.colorText,
      }}
    >
      <Space direction="vertical" size={16} style={{ maxWidth: 420, textAlign: 'center' }}>
        <RobotOutlined style={{ fontSize: 32, color: token.colorPrimary, margin: '0 auto' }} />
        <Typography.Title level={4} style={{ marginBottom: 0 }}>
          AI Assistant 已升级为独立工作台
        </Typography.Title>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
          {isAuto ? '已根据当前页面自动识别场景。' : '当前场景已固定。'}
          当前上下文为 {sceneLabel}，请前往 `/ai` 继续使用统一的问答与诊断体验。
        </Typography.Paragraph>
        <Button
          type="primary"
          size="large"
          onClick={() => {
            onClose?.();
            navigate('/ai');
          }}
        >
          打开 AI Assistant
        </Button>
      </Space>
    </div>
  );
};

export default Copilot;
