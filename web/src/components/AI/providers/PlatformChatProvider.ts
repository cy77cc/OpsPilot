import { AbstractChatProvider } from '@ant-design/x-sdk';
import type { TransformMessage } from '@ant-design/x-sdk';
import { AbstractXRequestClass } from '@ant-design/x-sdk';
import type { XRequestOptions } from '@ant-design/x-sdk';
import { aiApi } from '../../../api/modules/ai';
import type { A2UIStreamHandlers } from '../../../api/modules/ai';
import type { ChatRequest, PlatformStreamChunk, SceneContext, XChatMessage } from '../types';

type MetaHandler = (payload: { session_id: string; run_id: string; turn: number }) => void;

type StreamStage = 'idle' | 'preparing' | 'identifying' | 'planning' | 'agent';

const STAGE_PRIORITY: Record<StreamStage, number> = {
  idle: 0,
  preparing: 1,
  identifying: 2,
  planning: 3,
  agent: 4,
};

const AGENT_LABELS: Record<string, string> = {
  diagnosisagent: '诊断助手',
  changeagent: '变更助手',
  diagnosis: '诊断助手',
  change: '变更助手',
  planner: '规划助手',
};

function normalizeAgentName(name?: string): string {
  return (name || '').trim().toLowerCase();
}

function resolveAgentLabel(name?: string): string {
  const normalized = normalizeAgentName(name);
  return AGENT_LABELS[normalized] || (name || '助手');
}

function resolveAgentStatus(assistantType?: string): { stage: StreamStage; content: string } {
  const normalized = normalizeAgentName(assistantType);
  if (normalized === 'planner') {
    return { stage: 'planning', content: '[正在规划处理方式]' };
  }
  if (normalized) {
    return { stage: 'agent', content: `[${resolveAgentLabel(assistantType)}开始处理]` };
  }
  return { stage: 'identifying', content: '[识别任务]' };
}

function applyChunkContent(current: string, chunk: PlatformStreamChunk): string {
  if ((chunk.mode || 'append') === 'replace') {
    return chunk.content;
  }
  return `${current}${chunk.content}`;
}

function buildFinalContent(chunks: PlatformStreamChunk[]): string {
  return chunks.reduce((content, chunk) => applyChunkContent(content, chunk), '');
}

interface PlatformChatRequestConfig {
  onMeta?: MetaHandler;
}

export class PlatformChatRequest extends AbstractXRequestClass<
  ChatRequest,
  PlatformStreamChunk,
  XChatMessage
> {
  private readonly onMeta?: MetaHandler;
  private _asyncHandler: Promise<void> = Promise.resolve();
  private _isRequesting = false;
  private abortController: AbortController | null = null;

  constructor(config: PlatformChatRequestConfig = {}) {
    super('/api/v1/ai/chat', {
      manual: true,
      callbacks: {
        onSuccess: () => undefined,
        onError: () => undefined,
      },
    } as XRequestOptions<ChatRequest, PlatformStreamChunk, XChatMessage>);
    this.onMeta = config.onMeta;
  }

  get asyncHandler() {
    return this._asyncHandler;
  }

  get isTimeout() {
    return false;
  }

  get isStreamTimeout() {
    return false;
  }

  get isRequesting() {
    return this._isRequesting;
  }

  get manual() {
    return true;
  }

  run(params?: ChatRequest) {
    if (!params) {
      return false;
    }
    this.abort();
    this.abortController = new AbortController();
    this._isRequesting = true;
    const visibleChunks: PlatformStreamChunk[] = [];
    let terminalError: { error: Error; info?: unknown } | null = null;
    let hasVisibleContent = false;
    let stage: StreamStage = 'idle';
    let lastStatusContent = '';
    const headers = new Headers({ 'content-type': 'text/event-stream' });

    const emitStatus = (nextStage: StreamStage, content: string) => {
      if (!content || hasVisibleContent) {
        return;
      }
      if (STAGE_PRIORITY[nextStage] < STAGE_PRIORITY[stage]) {
        return;
      }
      if (STAGE_PRIORITY[nextStage] === STAGE_PRIORITY[stage] && content === lastStatusContent) {
        return;
      }
      stage = nextStage;
      lastStatusContent = content;
      this.options.callbacks?.onUpdate?.({ content, mode: 'replace' }, headers);
    };

    const emitVisibleChunk = (content: string) => {
      if (!content) {
        return;
      }
      const chunk: PlatformStreamChunk = {
        content,
        mode: hasVisibleContent ? 'append' : 'replace',
      };
      hasVisibleContent = true;
      visibleChunks.push(chunk);
      this.options.callbacks?.onUpdate?.(chunk, headers);
    };

    const handlers: A2UIStreamHandlers = {
      onMeta: (payload) => {
        this.onMeta?.(payload);
        emitStatus('preparing', '[准备中]');
      },
      onAgentHandoff: (payload) => {
        const status = resolveAgentStatus(payload.to);
        emitStatus(status.stage, status.content);
      },
      onPlan: () => {
        emitStatus('planning', '[正在规划处理方式]');
      },
      onDelta: (payload) => {
        emitVisibleChunk(payload.content);
      },
      onError: (payload) => {
        if (payload.code === 'tool_timeout_soft') {
          return;
        }
        terminalError = {
          error: new Error(payload.message || 'AI chat request failed'),
          info: payload,
        };
      },
    };

    this._asyncHandler = aiApi.chatStream(
      {
        message: params.message,
        sessionId: params.sessionId,
        scene: params.scene,
        context: params.context,
      },
      handlers,
      this.abortController.signal,
    )
      .then(() => {
        if (terminalError) {
          this.options.callbacks?.onError?.(terminalError.error, terminalError.info, headers);
          return;
        }
        this.options.callbacks?.onSuccess?.(visibleChunks, headers);
      })
      .catch((error: unknown) => {
        const normalized = error instanceof Error ? error : new Error('AI chat request failed');
        this.options.callbacks?.onError?.(normalized, undefined, headers);
      })
      .finally(() => {
        this._isRequesting = false;
      });

    return true;
  }

  abort() {
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }
    this._isRequesting = false;
  }
}

interface PlatformChatProviderConfig {
  scene?: string;
  getSceneContext?: () => SceneContext | undefined;
  getSessionId?: () => string | undefined;
  onMeta?: MetaHandler;
}

export class PlatformChatProvider extends AbstractChatProvider<
  XChatMessage,
  ChatRequest,
  PlatformStreamChunk
> {
  private readonly scene?: string;
  private readonly getSceneContext?: () => SceneContext | undefined;
  private readonly getSessionId?: () => string | undefined;

  constructor(config: PlatformChatProviderConfig = {}) {
    super({
      request: new PlatformChatRequest({
        onMeta: config.onMeta,
      }),
    });
    this.scene = config.scene;
    this.getSceneContext = config.getSceneContext;
    this.getSessionId = config.getSessionId;
  }

  transformParams(requestParams: Partial<ChatRequest>, options: XRequestOptions<ChatRequest, PlatformStreamChunk, XChatMessage>): ChatRequest {
    return {
      message: requestParams.message || options?.params?.message || '',
      ...(options?.params || {}),
      ...requestParams,
      sessionId: requestParams.sessionId || options?.params?.sessionId || this.getSessionId?.(),
      scene: requestParams.scene || options?.params?.scene || this.scene,
      context: requestParams.context || options?.params?.context || this.getSceneContext?.(),
    };
  }

  transformLocalMessage(requestParams: Partial<ChatRequest>): XChatMessage {
    return {
      role: 'user',
      content: requestParams.message || '',
    };
  }

  transformMessage(info: TransformMessage<XChatMessage, PlatformStreamChunk>): XChatMessage {
    const current = info.originMessage?.content || '';
    const chunkContent = info.chunk?.content || '';
    if (info.status === 'success') {
      const finalContent = buildFinalContent(info.chunks);
      return {
        role: 'assistant',
        content: finalContent || current,
      };
    }
    if (info.chunk) {
      return {
        role: 'assistant',
        content: applyChunkContent(current, info.chunk),
      };
    }
    return {
      role: 'assistant',
      content: `${current}${chunkContent}`,
    };
  }
}
