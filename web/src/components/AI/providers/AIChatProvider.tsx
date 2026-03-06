import React, { createContext, useContext, useState, useCallback, useMemo } from 'react';
import type { AIChatContextValue, ChatMessage, Conversation, ErrorInfo } from '../types';
import { useAIChat } from '../hooks/useAIChat';

interface AIChatProviderProps {
  children: React.ReactNode;
  defaultScene?: string;
}

// 上下文类型
interface AIChatContext extends AIChatContextValue {
  // 全局状态
  globalOpen: boolean;
  sceneOpen: boolean;
  currentScene: string;

  // 控制
  setGlobalOpen: (open: boolean) => void;
  setSceneOpen: (open: boolean) => void;
  setCurrentScene: (scene: string) => void;

  // 错误状态
  error: ErrorInfo | null;
  clearError: () => void;
}

const AIChatContext = createContext<AIChatContext | null>(null);

/**
 * AI 聊天 Provider
 * 提供全局和场景两种模式的 AI 助手状态管理
 */
export function AIChatProvider({ children, defaultScene = 'global' }: AIChatProviderProps) {
  // 全局助手状态
  const [globalOpen, setGlobalOpen] = useState(false);
  const globalChat = useAIChat({ scene: 'global' });

  // 场景助手状态
  const [sceneOpen, setSceneOpen] = useState(false);
  const [currentScene, setCurrentScene] = useState(defaultScene);
  const sceneChat = useAIChat({ scene: currentScene });

  // 错误状态
  const [error, setError] = useState<ErrorInfo | null>(null);

  // 清除错误
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  // 获取当前活动的聊天状态
  const getActiveChat = useCallback(() => {
    if (sceneOpen && currentScene !== 'global') {
      return sceneChat;
    }
    return globalChat;
  }, [sceneOpen, currentScene, sceneChat, globalChat]);

  // 上下文值
  const contextValue = useMemo<AIChatContext>(() => {
    const activeChat = getActiveChat();

    return {
      // 从活动聊天继承
      messages: activeChat.messages,
      isLoading: activeChat.isLoading,
      currentConversation: activeChat.currentConversation,
      conversations: activeChat.conversations,

      // 操作方法
      sendMessage: activeChat.sendMessage,
      createConversation: activeChat.createConversation,
      switchConversation: activeChat.switchConversation,
      deleteConversation: activeChat.deleteConversation,
      clearMessages: activeChat.createConversation, // 复用创建会话逻辑
      confirmAction: activeChat.confirmAction,

      // 全局状态
      globalOpen,
      sceneOpen,
      currentScene,

      // 控制
      setGlobalOpen,
      setSceneOpen,
      setCurrentScene,

      // 错误状态
      error,
      clearError,
    };
  }, [
    getActiveChat,
    globalOpen,
    sceneOpen,
    currentScene,
    error,
    clearError,
  ]);

  return (
    <AIChatContext.Provider value={contextValue}>
      {children}
    </AIChatContext.Provider>
  );
}

/**
 * 获取 AI 聊天上下文
 */
export function useAIChatContext(): AIChatContext {
  const context = useContext(AIChatContext);
  if (!context) {
    throw new Error('useAIChatContext must be used within AIChatProvider');
  }
  return context;
}

/**
 * 获取全局助手状态
 */
export function useGlobalAIAssistant() {
  const { globalOpen, setGlobalOpen } = useAIChatContext();
  return { open: globalOpen, setOpen: setGlobalOpen };
}

/**
 * 获取场景助手状态
 */
export function useSceneAIAssistant() {
  const { sceneOpen, setSceneOpen, currentScene } = useAIChatContext();
  return { open: sceneOpen, setOpen: setSceneOpen, scene: currentScene };
}
