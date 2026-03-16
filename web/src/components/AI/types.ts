/**
 * AI 助手组件类型定义（简化版）
 */

// 消息角色
export type MessageRole = 'user' | 'assistant' | 'system';

// 聊天消息
export interface ChatMessage {
  id: string;
  role: MessageRole;
  content: string;
  createdAt: string;
}

// 场景信息
export interface SceneInfo {
  key: string;
  label: string;
  description?: string;
}

// 抽屉宽度设置
export interface DrawerWidthConfig {
  default: number;
  min: number;
  max: number;
}
