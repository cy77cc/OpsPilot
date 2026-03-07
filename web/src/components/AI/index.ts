/**
 * AI Copilot 组件导出
 */

// 主组件
export { AIAssistantDrawer } from './AIAssistantDrawer';
export { AIAssistantButton } from './AIAssistantButton'; // 保留旧组件兼容
export { AICopilotButton } from './AICopilotButton'; // 新统一入口
export { Copilot } from './Copilot'; // 新 @ant-design/x 组件

// 子组件
export { ConversationsPanel } from './components/ConversationsPanel';
export { MessageList } from './components/MessageList';
export { MessageBubble } from './components/MessageBubble';
export { ToolCard } from './components/ToolCard';
export { ConfirmationPanel } from './components/ConfirmationPanel';
export { ChatInput } from './components/ChatInput';

// Hooks
export { useResizableDrawer } from './hooks/useResizableDrawer';
export { useSceneDetector, useHasSceneSupport, useSceneConfig } from './hooks/useSceneDetector';
export { useAutoScene } from './hooks/useAutoScene';
export { useSSEAdapter } from './hooks/useSSEAdapter';
export { useAIChat } from './hooks/useAIChat';

// Provider
export { AIChatProvider, useAIChatContext } from './providers/AIChatProvider';

// 常量
export { SCENE_MAPPINGS, getSceneByPath, getSceneLabel, SCENE_LABELS } from './constants/sceneMapping';

// 类型
export type {
  MessageRole,
  ToolStatus,
  RiskLevel,
  ContentPart,
  ToolExecution,
  ConfirmationRequest,
  ChatMessage,
  Conversation,
  SceneInfo,
  AIChatContextValue,
  DrawerWidthConfig,
  SSEEventType,
  SSEEventPayload,
  ErrorType,
  ErrorInfo,
} from './types';

export type { SceneOption } from './hooks/useAutoScene';
