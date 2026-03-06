import React, { useState, useCallback, useRef } from 'react';
import { Button, Tooltip } from 'antd';
import { SendOutlined, LoadingOutlined } from '@ant-design/icons';

interface ChatInputProps {
  onSend: (content: string) => void;
  isLoading: boolean;
}

/**
 * 聊天输入组件
 * 使用类似 Ant Design X Sender 的风格
 */
export function ChatInput({ onSend, isLoading }: ChatInputProps) {
  const [value, setValue] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // 发送消息
  const handleSend = useCallback(() => {
    const trimmed = value.trim();
    if (trimmed && !isLoading) {
      onSend(trimmed);
      setValue('');
    }
  }, [value, isLoading, onSend]);

  // 键盘事件处理
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      // Enter 发送，Shift+Enter 换行
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend]
  );

  // 自动调整高度
  const handleInput = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setValue(e.target.value);
    // 自动调整高度
    const textarea = e.target;
    textarea.style.height = 'auto';
    textarea.style.height = Math.min(textarea.scrollHeight, 120) + 'px';
  }, []);

  return (
    <div className="ai-chat-input">
      <div className="ai-chat-input-wrapper">
        <textarea
          ref={textareaRef}
          value={value}
          onChange={handleInput}
          onKeyDown={handleKeyDown}
          placeholder="输入消息... (Shift+Enter 换行)"
          disabled={isLoading}
          rows={1}
        />
        <Tooltip title={isLoading ? '发送中...' : '发送 (Enter)'}>
          <Button
            type="primary"
            icon={isLoading ? <LoadingOutlined /> : <SendOutlined />}
            onClick={handleSend}
            disabled={!value.trim() || isLoading}
            className="ai-chat-input-send"
          />
        </Tooltip>
      </div>
      <div className="ai-chat-input-hint">
        <span>按 Enter 发送，Shift+Enter 换行</span>
      </div>
    </div>
  );
}
