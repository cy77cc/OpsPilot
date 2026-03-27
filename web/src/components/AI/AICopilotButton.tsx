/**
 * AI Copilot 统一入口按钮
 */
import React, { useEffect } from 'react';
import { Button, Tooltip } from 'antd';
import { RobotOutlined } from '@ant-design/icons';

interface AICopilotButtonProps {
  onOpen?: () => void;
}

export function AICopilotButton({ onOpen }: AICopilotButtonProps) {
  const handleOpen = React.useCallback(() => {
    onOpen?.();
  }, [onOpen]);

  // 快捷键监听: Cmd/Ctrl + /
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === '/' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        handleOpen();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleOpen]);

  return (
    <Tooltip title="AI Assistant (Cmd+/)">
      <Button type="text" icon={<RobotOutlined />} onClick={handleOpen}>
        AI Assistant
      </Button>
    </Tooltip>
  );
}
