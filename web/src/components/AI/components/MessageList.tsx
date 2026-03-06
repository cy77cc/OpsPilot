import React, { useEffect, useRef } from 'react';
import { Empty } from 'antd';
import { MessageBubble } from './MessageBubble';
import type { ChatMessage } from '../types';

interface MessageListProps {
  messages: ChatMessage[];
  isLoading: boolean;
  scene: string;
}

/**
 * 消息列表组件
 */
export function MessageList({ messages, isLoading, scene }: MessageListProps) {
  const listRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  // 自动滚动到底部
  useEffect(() => {
    if (bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);

  if (messages.length === 0 && !isLoading) {
    return (
      <div className="ai-message-list">
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="开始新的对话"
          style={{ marginTop: '40px' }}
        />
      </div>
    );
  }

  return (
    <div className="ai-message-list" ref={listRef}>
      {messages.map((message) => (
        <MessageBubble key={message.id} message={message} />
      ))}

      {/* 加载中指示器 */}
      {isLoading && (
        <div className="ai-message-item assistant">
          <div className="ai-message-avatar assistant">AI</div>
          <div className="ai-typing-indicator">
            <span className="dot" />
            <span className="dot" />
            <span className="dot" />
            <span style={{ marginLeft: 8 }}>思考中...</span>
          </div>
        </div>
      )}

      {/* 底部锚点 */}
      <div ref={bottomRef} />
    </div>
  );
}
