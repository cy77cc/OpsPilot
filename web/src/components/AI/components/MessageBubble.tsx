import React from 'react';
import { XMarkdown } from "@ant-design/x-markdown";
import { ToolCard } from './ToolCard';
import { ConfirmationPanel } from './ConfirmationPanel';
import type { ChatMessage } from '../types';

interface MessageBubbleProps {
  message: ChatMessage;
}

/**
 * 消息气泡组件
 * 使用 XMarkdown 渲染消息内容
 */
export function MessageBubble({ message }: MessageBubbleProps) {
  const isUser = message.role === 'user';

  return (
    <div className={`ai-message-item ${message.role}`}>
      {/* 头像 */}
      <div className={`ai-message-avatar ${message.role}`}>
        {isUser ? 'U' : 'AI'}
      </div>

      {/* 内容 */}
      <div className="ai-message-content">
        {/* 工具执行卡片 */}
        {message.tools && message.tools.length > 0 && (
          <div className="ai-message-tools">
            {message.tools.map((tool) => (
              <ToolCard key={tool.id} tool={tool} />
            ))}
          </div>
        )}

        {/* 消息内容 - 使用 XMarkdown 渲染 */}
        {message.content && (
          <XMarkdown>{message.content}</XMarkdown>
        )}

        {/* 确认面板 */}
        {message.confirmation && (
          <ConfirmationPanel confirmation={message.confirmation} />
        )}

        {/* 思考过程 */}
        {message.thinking && (
          <details className="ai-message-thinking">
            <summary>思考过程</summary>
            <div className="thinking-content">
              <XMarkdown>{message.thinking}</XMarkdown>
            </div>
          </details>
        )}
      </div>
    </div>
  );
}
