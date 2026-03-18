import type { AIMessage, AIReplayBlock, AIReplayTurn } from '../../api/modules/ai';
import type { AssistantReplyRuntime, XChatMessage } from './types';

function mapHistoricalStatus(
  status?: string,
  runtime?: AssistantReplyRuntime,
): AssistantReplyRuntime | undefined {
  if (!runtime) {
    return undefined;
  }

  if (status === 'done') {
    return {
      ...runtime,
      status: {
        kind: 'completed',
        label: '已生成',
      },
    };
  }

  return runtime;
}

function synthesizeRuntimeFromBlocks(blocks: AIReplayBlock[]): AssistantReplyRuntime | undefined {
  if (!blocks.length) {
    return undefined;
  }

  const runtime: AssistantReplyRuntime = {
    activities: [],
  };

  blocks.forEach((block) => {
    if (block.blockType === 'phase' && block.contentJson) {
      runtime.phase = block.contentJson.phase as AssistantReplyRuntime['phase'];
      runtime.phaseLabel = String(block.contentJson.phaseLabel || '');
      return;
    }

    if (block.blockType === 'summary' && block.contentJson) {
      runtime.summary = {
        title: typeof block.contentJson.title === 'string' ? block.contentJson.title : undefined,
        items: Array.isArray(block.contentJson.items)
          ? block.contentJson.items.map((item: any) => ({
              label: String(item.label || ''),
              value: String(item.value || ''),
              tone: item.tone,
            }))
          : undefined,
      };
    }
  });

  if (!runtime.phase && !runtime.summary) {
    return undefined;
  }

  return runtime;
}

export function hydrateAssistantHistoryMessage(
  message: AIMessage,
  turns: AIReplayTurn[] = [],
  blocks: AIReplayBlock[] = [],
): XChatMessage {
  const persistedRuntime = message.runtime as unknown as AssistantReplyRuntime | undefined;
  const turnBlocks = turns.flatMap((turn) => turn.blocks || []);
  const synthesizedRuntime = synthesizeRuntimeFromBlocks([...turnBlocks, ...blocks]);
  const runtime = mapHistoricalStatus(message.status, persistedRuntime || synthesizedRuntime);

  return {
    role: message.role === 'assistant' ? 'assistant' : 'user',
    content: message.content || '',
    ...(runtime ? { runtime } : {}),
  };
}
