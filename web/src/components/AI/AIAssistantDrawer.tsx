import React, { useEffect, useCallback } from 'react';
import { Drawer, Button, Tooltip, Spin } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useResizableDrawer } from './hooks/useResizableDrawer';
import { useAIChat } from './hooks/useAIChat';
import { ConversationsPanel } from './components/ConversationsPanel';
import { MessageList } from './components/MessageList';
import { ChatInput } from './components/ChatInput';
import { getSceneLabel } from './constants/sceneMapping';
import type { ChatMessage } from './types';
import './AIAssistantDrawer.css';

interface AIAssistantDrawerProps {
  open: boolean;
  onClose: () => void;
  scene: string;
}

/**
 * AI 助手抽屉组件
 */
export function AIAssistantDrawer({ open, onClose, scene }: AIAssistantDrawerProps) {
  const { width, isResizing, handleMouseDown } = useResizableDrawer();
  const {
    messages,
    isLoading,
    conversations,
    currentSessionId,
    sendMessage,
    createConversation,
    switchConversation,
    deleteConversation,
    loadConversations,
    setMessages,
  } = useAIChat({ scene });

  const sceneLabel = getSceneLabel(scene);

  // 加载会话列表
  useEffect(() => {
    if (open) {
      void loadConversations();
    }
  }, [open, loadConversations]);

  // 处理发送消息
  const handleSend = useCallback(
    (content: string) => {
      void sendMessage(content);
    },
    [sendMessage]
  );

  // 处理新建会话
  const handleNewConversation = useCallback(() => {
    createConversation();
    setMessages([]);
  }, [createConversation, setMessages]);

  // 拖拽手柄
  const ResizeHandle = (
    <div
      className={`ai-drawer-resize-handle ${isResizing ? 'resizing' : ''}`}
      onMouseDown={handleMouseDown}
    />
  );

  return (
    <Drawer
      open={open}
      onClose={onClose}
      placement="right"
      width={width}
      closable={false}
      maskClosable
      rootClassName="ai-assistant-drawer"
      styles={{
        body: { padding: 0, display: 'flex', flexDirection: 'column', height: '100%' },
        wrapper: { transition: isResizing ? 'none' : undefined },
      }}
      title={null}
    >
      {ResizeHandle}

      {/* 头部 */}
      <div className="ai-drawer-header">
        <div className="ai-drawer-header-title">
          <span className="ai-drawer-scene-badge">{sceneLabel}</span>
          <h3>AI 运维助手</h3>
        </div>
        <div className="ai-drawer-header-actions">
          <Tooltip title="新建会话">
            <Button type="text" icon={<PlusOutlined />} onClick={handleNewConversation} />
          </Tooltip>
          <Tooltip title="刷新会话列表">
            <Button type="text" icon={<ReloadOutlined />} onClick={() => void loadConversations()} />
          </Tooltip>
        </div>
      </div>

      {/* 主内容区 */}
      <div className="ai-drawer-content">
        {/* 会话列表 */}
        <div className="ai-drawer-conversations">
          <ConversationsPanel
            conversations={conversations}
            currentId={currentSessionId}
            onSelect={switchConversation}
            onDelete={deleteConversation}
            onCreate={handleNewConversation}
          />
        </div>

        {/* 消息区域 */}
        <div className="ai-drawer-messages">
          {isLoading && messages.length === 0 ? (
            <div className="ai-drawer-loading">
              <Spin tip="加载中..." />
            </div>
          ) : (
            <MessageList
              messages={messages}
              isLoading={isLoading}
              scene={scene}
            />
          )}
        </div>
      </div>

      {/* 输入区域 */}
      <div className="ai-drawer-input">
        <ChatInput onSend={handleSend} isLoading={isLoading} />
      </div>
    </Drawer>
  );
}
