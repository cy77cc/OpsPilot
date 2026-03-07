import React, { useState, useEffect } from 'react';
import { Button, Tooltip } from 'antd';
import { RobotOutlined } from '@ant-design/icons';
import { AIAssistantDrawer } from './AIAssistantDrawer';
import { useAutoScene } from './hooks/useAutoScene';

/**
 * AI Copilot 统一入口按钮
 * 单一入口，自动感知场景，支持场景切换
 */
export function AICopilotButton() {
  const { scene, selectValue, setScene, availableScenes, isAuto } = useAutoScene();
  const [open, setOpen] = useState(false);

  // 快捷键监听: Cmd/Ctrl + /
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === '/' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen(true);
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  return (
    <>
      <Tooltip title="AI Copilot (Cmd+/)">
        <Button
          type="text"
          icon={<RobotOutlined />}
          onClick={() => setOpen(true)}
          style={{ color: open ? '#1890ff' : undefined }}
        >
          AI Copilot
        </Button>
      </Tooltip>

      <AIAssistantDrawer
        open={open}
        onClose={() => setOpen(false)}
        scene={scene}
        selectValue={selectValue}
        onSceneChange={setScene}
        availableScenes={availableScenes}
        isAuto={isAuto}
      />
    </>
  );
}
