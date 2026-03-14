/**
 * AI Copilot 组件导出
 */

// 主路径
export { AIAssistantDrawer } from './AIAssistantDrawer';
export { AICopilotButton } from './AICopilotButton';
export { Copilot } from './Copilot';

// 主路径内部复用组件
export { ToolCard } from './components/ToolCard';
export { ConfirmationPanel } from './components/ConfirmationPanel';

// Hooks
export { useResizableDrawer } from './hooks/useResizableDrawer';
export { useSceneDetector, useHasSceneSupport, useSceneConfig } from './hooks/useSceneDetector';
export { useAutoScene } from './hooks/useAutoScene';

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
  SceneInfo,
  DrawerWidthConfig,
  ChatTurn,
  ChatTurnStatus,
  ChatTurnPhase,
  TurnBlock,
  TurnBlockType,
  SSEEventType,
  SSEEventPayload,
  ThoughtStageItem,
  ThoughtStageDetailItem,
  ThoughtStageKey,
  ThoughtStageStatus,
  ErrorType,
  ErrorInfo,
} from './types';

export type { SceneOption } from './hooks/useAutoScene';
