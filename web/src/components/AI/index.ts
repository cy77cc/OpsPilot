/**
 * AI 助手组件导出
 */

// 主组件
export { AIAssistantDrawer } from './AIAssistantDrawer';
export { AIAssistantButton } from './AIAssistantButton';

// 子组件
export { ConversationsPanel } from './components/ConversationsPanel';
export { MessageList } from './components/MessageList';
export { MessageBubble } from './components/MessageBubble';
export { ToolCard } from './components/ToolCard';
export { ConfirmationPanel } from './components/ConfirmationPanel';
export { ChatInput } from './components/ChatInput';

// Hooks
export { useResizableDrawer } from './hooks/useResizableDrawer';
export { useSceneDetector } from './hooks/useSceneDetector';
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
