/**
 * AI Copilot 组件导出
 */

// 主组件
export { AIAssistantDrawer } from './AIAssistantDrawer';
export { AICopilotButton } from './AICopilotButton';
export { Copilot } from './Copilot';

// Hooks
export { useResizableDrawer } from './hooks/useResizableDrawer';
export { useSceneDetector, useHasSceneSupport, useSceneConfig } from './hooks/useSceneDetector';
export { useAutoScene } from './hooks/useAutoScene';

// 常量
export { SCENE_MAPPINGS, getSceneByPath, getSceneLabel, SCENE_LABELS } from './constants/sceneMapping';

// 类型
export type {
  MessageRole,
  ChatMessage,
  SceneInfo,
  DrawerWidthConfig,
} from './types';

export type { SceneOption } from './hooks/useAutoScene';
