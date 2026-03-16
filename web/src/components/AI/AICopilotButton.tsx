import React, { useEffect } from 'react';
import { Button, Tooltip } from 'antd';
import { RobotOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

/**
 * AI Copilot 统一入口按钮
 * Phase 1 统一跳转到独立 AI 页面
 */
export function AICopilotButton() {
  const navigate = useNavigate();

  // 快捷键监听: Cmd/Ctrl + /
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === '/' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        navigate('/ai');
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [navigate]);

  return (
    <Tooltip title="AI Assistant (Cmd+/)">
      <Button type="text" icon={<RobotOutlined />} onClick={() => navigate('/ai')}>
        AI Assistant
      </Button>
    </Tooltip>
  );
}
