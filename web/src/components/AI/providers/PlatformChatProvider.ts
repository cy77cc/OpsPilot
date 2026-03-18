import { AbstractChatProvider } from '@ant-design/x-sdk';
import type { TransformMessage } from '@ant-design/x-sdk';
import { AbstractXRequestClass } from '@ant-design/x-sdk';
import type { XRequestOptions } from '@ant-design/x-sdk';
import { aiApi } from '../../../api/modules/ai';
import type { A2UIStreamHandlers } from '../../../api/modules/ai';
import {
  applyAgentHandoff,
  applyDelta,
  applyDone,
  applyMeta,
  applyPlan,
  applyRecoverableError,
  applyReplan,
  applyStepDelta,
  applySoftTimeout,
  applyTerminalError,
  applyToolApproval,
  applyToolCall,
  applyToolResult,
  createEmptyAssistantRuntime,
} from '../replyRuntime';
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

function buildFinalRuntime(chunks: PlatformStreamChunk[]) {
  for (let index = chunks.length - 1; index >= 0; index -= 1) {
    if (chunks[index].runtime) {
      return chunks[index].runtime;
    }
  }
  return undefined;
}

function hasPlanContent(runtime: ReturnType<typeof createEmptyAssistantRuntime>): boolean {
  return Boolean(runtime.plan?.steps?.some((step) => (step.content || '').trim()));
}

function parsePlannerEnvelope(raw: string): { steps?: string[]; response?: string } | null {
  const trimmed = raw.trim();
  if (!trimmed) {
    return null;
  }
  try {
    const payload = JSON.parse(trimmed) as { steps?: string[]; response?: string };
    if (Array.isArray(payload.steps) && payload.steps.length > 0) {
      return { steps: payload.steps };
    }
    if (typeof payload.response === 'string' && payload.response.trim()) {
      return { response: payload.response };
    }
  } catch {
    return null;
  }
  return null;
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
    let content = '';
    let runtime = createEmptyAssistantRuntime();
    const plannerBuffers: Partial<Record<'planner' | 'replanner', string>> = {};
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
      this.options.callbacks?.onUpdate?.({ content, mode: 'replace', runtime }, headers);
    };

    const emitVisibleChunk = (content: string) => {
      if (!content) {
        return;
      }
      const chunk: PlatformStreamChunk = {
        content,
        mode: hasVisibleContent ? 'append' : 'replace',
        runtime,
      };
      hasVisibleContent = true;
      visibleChunks.push(chunk);
      this.options.callbacks?.onUpdate?.(chunk, headers);
    };

    const emitRuntimeOnlyUpdate = () => {
      const chunk: PlatformStreamChunk = {
        content: hasVisibleContent ? content : lastStatusContent || '[准备中]',
        mode: 'replace',
        runtime,
      };
      this.options.callbacks?.onUpdate?.(chunk, headers);
    };

    const handlers: A2UIStreamHandlers = {
      onMeta: (payload) => {
        this.onMeta?.(payload);
        runtime = applyMeta(runtime);
        emitStatus('preparing', '[准备中]');
      },
      onAgentHandoff: (payload) => {
        runtime = applyAgentHandoff(runtime, payload);
        const status = resolveAgentStatus(payload.to);
        emitStatus(status.stage, status.content);
      },
      onPlan: (payload) => {
        runtime = applyPlan(runtime, payload);
        emitStatus('planning', '[正在规划处理方式]');
      },
      onReplan: (payload) => {
        runtime = applyReplan(runtime, payload);
        if (hasVisibleContent) {
          emitRuntimeOnlyUpdate();
        } else {
          emitStatus('planning', '[正在规划处理方式]');
        }
      },
      onToolCall: (payload) => {
        runtime = applyToolCall(runtime, payload);
        if (hasVisibleContent) {
          emitRuntimeOnlyUpdate();
        }
      },
      onToolApproval: (payload) => {
        runtime = applyToolApproval(runtime, payload);
        if (hasVisibleContent) {
          emitRuntimeOnlyUpdate();
        }
      },
      onToolResult: (payload) => {
        runtime = applyToolResult(runtime, payload);
        if (hasVisibleContent) {
          emitRuntimeOnlyUpdate();
        }
      },
      onDelta: (payload) => {
        const agent = normalizeAgentName(payload.agent);
        if (agent === 'planner' || agent === 'replanner') {
          const nextBuffered = `${plannerBuffers[agent] || ''}${payload.content || ''}`;
          const envelope = parsePlannerEnvelope(nextBuffered);
          if (!envelope) {
            plannerBuffers[agent] = nextBuffered;
            return;
          }

          plannerBuffers[agent] = '';

          if (envelope.steps) {
            if (agent === 'planner') {
              runtime = applyPlan(runtime, { steps: envelope.steps, iteration: 0 });
            } else {
              runtime = applyReplan(runtime, {
                steps: envelope.steps,
                completed: runtime.plan?.steps.length
                  ? Math.max(0, runtime.plan.steps.length - envelope.steps.length)
                  : 0,
                iteration: 0,
                is_final: false,
              });
            }
            if (hasVisibleContent) {
              emitRuntimeOnlyUpdate();
            } else {
              emitStatus('planning', '[正在规划处理方式]');
            }
            return;
          }

          if (envelope.response) {
            runtime = applyReplan(runtime, {
              steps: [],
              completed: runtime.plan?.steps.length || 0,
              iteration: 0,
              is_final: true,
            });
            const next = applyDelta(
              {
                content,
                runtime,
              },
              { content: envelope.response },
            );
            content = next.content;
            runtime = next.runtime || runtime;
            emitVisibleChunk(envelope.response);
            return;
          }
        }

        if (runtime.plan?.activeStepIndex !== undefined) {
          runtime = applyStepDelta(runtime, payload);
          emitRuntimeOnlyUpdate();
          return;
        }

        const next = applyDelta(
          {
            content,
            runtime,
          },
          payload,
        );
        content = next.content;
        runtime = next.runtime || runtime;
        emitVisibleChunk(payload.content);
      },
      onDone: () => {
        runtime = applyDone(runtime);
        if (hasVisibleContent || hasPlanContent(runtime)) {
          emitRuntimeOnlyUpdate();
        }
      },
      onError: (payload) => {
        if (payload.code === 'tool_timeout_soft') {
          runtime = applySoftTimeout(runtime);
          if (hasVisibleContent || hasPlanContent(runtime)) {
            const next = applyRecoverableError({ content, runtime }, payload);
            content = next.content;
            runtime = next.runtime || runtime;
            emitRuntimeOnlyUpdate();
          }
          return;
        }

        const isToolError = (payload.code || '').startsWith('tool_');
        const hasProgress = hasVisibleContent || Boolean(content.trim()) || hasPlanContent(runtime);
        if (payload.recoverable || isToolError || hasProgress) {
          const next = applyTerminalError({ content, runtime }, payload);
          content = next.content;
          runtime = next.runtime || runtime;
          emitRuntimeOnlyUpdate();
          return;
        }

        const next = applyTerminalError({ content, runtime }, payload);
        content = next.content;
        runtime = next.runtime || runtime;
        terminalError = {
          error: new Error(payload.message || 'AI chat request failed'),
          info: {
            ...payload,
            runtime,
          },
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
    const runtime = info.chunk?.runtime || info.originMessage?.runtime || buildFinalRuntime(info.chunks);
    if (info.status === 'success') {
      const finalContent = buildFinalContent(info.chunks);
      return {
        role: 'assistant',
        content: finalContent || current,
        runtime,
      };
    }
    if (info.chunk) {
      return {
        role: 'assistant',
        content: applyChunkContent(current, info.chunk),
        runtime,
      };
    }
    return {
      role: 'assistant',
      content: `${current}${chunkContent}`,
      runtime,
    };
  }
}
