import type { AIMessage } from '../../api/modules/ai';
import { aiApi } from '../../api/modules/ai';
import type { AssistantReplyRuntime, XChatMessage } from './types';

// 缓存配置
const MAX_CACHE_SIZE = 50;

// LRU 缓存
const runtimeCache = new Map<string, AssistantReplyRuntime>();
const cacheOrder: string[] = [];

// 淘汰最旧的缓存
function evictOldest(): void {
  if (cacheOrder.length > MAX_CACHE_SIZE) {
    const oldest = cacheOrder.shift();
    if (oldest) {
      runtimeCache.delete(oldest);
    }
  }
}

// loadMessageRuntime 懒加载消息的 runtime 数据。
//
// 使用 LRU 缓存避免重复请求。
export async function loadMessageRuntime(messageId: string): Promise<AssistantReplyRuntime | null> {
  // 缓存命中
  if (runtimeCache.has(messageId)) {
    // 更新 LRU 顺序
    const index = cacheOrder.indexOf(messageId);
    if (index > -1) {
      cacheOrder.splice(index, 1);
      cacheOrder.push(messageId);
    }
    return runtimeCache.get(messageId)!;
  }

  try {
    const response = await aiApi.getMessageRuntime(messageId);
    if (response.data?.runtime) {
      const runtime = response.data.runtime as unknown as AssistantReplyRuntime;
      runtimeCache.set(messageId, runtime);
      cacheOrder.push(messageId);
      evictOldest();
      return runtime;
    }
  } catch (error) {
    console.error('Failed to load runtime:', error);
  }
  return null;
}

// hydrateAssistantHistoryMessage 将历史消息转换为前端可用的 XChatMessage。
//
// 不再尝试合成 runtime，而是传递 ID 和 hasRuntime 标志，
// 让组件在需要时懒加载。
export function hydrateAssistantHistoryMessage(
  message: AIMessage & { has_runtime?: boolean },
): XChatMessage {
  return {
    id: message.id,
    role: message.role === 'assistant' ? 'assistant' : 'user',
    content: message.content || '',
    runtime: message.runtime as AssistantReplyRuntime | undefined,
    hasRuntime: message.has_runtime ?? false,
  };
}
