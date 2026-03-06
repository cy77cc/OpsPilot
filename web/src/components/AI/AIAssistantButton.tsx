import React, { useState, useEffect } from 'react';
import { Button, Tooltip, Badge } from 'antd';
import { RobotOutlined, ApiOutlined } from '@ant-design/icons';
import { AIAssistantDrawer } from './AIAssistantDrawer';
import { useSceneConfig } from './hooks/useSceneDetector';

/**
 * AI 助手按钮组件
 * 包含全局助手按钮和场景助手按钮
 */
export function AIAssistantButton() {
  const { key: scene, label, hasSceneSupport } = useSceneConfig();

  // 状态
  const [globalOpen, setGlobalOpen] = useState(false);
  const [sceneOpen, setSceneOpen] = useState(false);

  // 快捷键监听
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl + / : 全局助手
      if (e.key === '/' && (e.metaKey || e.ctrlKey) && !e.shiftKey) {
        e.preventDefault();
        setGlobalOpen(true);
      }
      // Cmd/Ctrl + Shift + / : 场景助手
      if (e.key === '/' && (e.metaKey || e.ctrlKey) && e.shiftKey && hasSceneSupport) {
        e.preventDefault();
        setSceneOpen(true);
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [hasSceneSupport]);

  return (
    <>
      {/* 全局助手按钮 */}
      <Tooltip title="AI 助手 (Cmd+/)">
        <Button
          type="text"
          icon={<RobotOutlined />}
          onClick={() => setGlobalOpen(true)}
          style={{ color: globalOpen ? '#1890ff' : undefined }}
        >
          AI 助手
        </Button>
      </Tooltip>

      {/* 场景助手按钮 (条件渲染) */}
      {hasSceneSupport && (
        <Tooltip title={`${label} (Cmd+Shift+/)`}>
          <Button
            type="primary"
            icon={<ApiOutlined />}
            onClick={() => setSceneOpen(true)}
            size="small"
            style={{ marginLeft: 4 }}
          >
            {label}
          </Button>
        </Tooltip>
      )}

      {/* 全局助手抽屉 */}
      <AIAssistantDrawer
        open={globalOpen}
        onClose={() => setGlobalOpen(false)}
        scene="global"
      />

      {/* 场景助手抽屉 */}
      {hasSceneSupport && (
        <AIAssistantDrawer
          open={sceneOpen}
          onClose={() => setSceneOpen(false)}
          scene={scene}
        />
      )}
    </>
  );
}
