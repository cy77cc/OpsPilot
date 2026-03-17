export interface SceneContext {
  route?: string;
  resourceType?: string;
  resourceId?: string;
  resourceName?: string;
  [key: string]: unknown;
}

export interface ChatRequest {
  message: string;
  sessionId?: string;
  scene?: string;
  context?: SceneContext;
}

export interface XChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

export interface ConversationSummary {
  key: string;
  label: string;
  scene: string;
  updatedAt?: string;
}

export interface PlatformStreamChunk {
  content: string;
  mode?: 'replace' | 'append';
}
