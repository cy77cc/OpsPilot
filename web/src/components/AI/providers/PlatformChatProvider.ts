import { AbstractChatProvider } from '@ant-design/x-sdk';
import type { TransformMessage } from '@ant-design/x-sdk';
import { AbstractXRequestClass } from '@ant-design/x-sdk';
import type { XRequestOptions } from '@ant-design/x-sdk';
import { aiApi } from '../../../api/modules/ai';
import type { AIChatStreamHandlers } from '../../../api/modules/ai';
import type { ChatRequest, PlatformStreamChunk, SceneContext, XChatMessage } from '../types';

type InitHandler = (payload: { session_id: string; run_id: string }) => void;

interface PlatformChatRequestConfig {
  onInit?: InitHandler;
}

export class PlatformChatRequest extends AbstractXRequestClass<
  ChatRequest,
  PlatformStreamChunk,
  XChatMessage
> {
  private readonly onInit?: InitHandler;
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
    this.onInit = config.onInit;
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
    const chunks: PlatformStreamChunk[] = [];
    let terminalError: { error: Error; info?: unknown } | null = null;
    const headers = new Headers({ 'content-type': 'text/event-stream' });

    const handlers: AIChatStreamHandlers = {
      onInit: (payload) => {
        this.onInit?.(payload);
      },
      onDelta: (payload) => {
        const chunk = { content: payload.contentChunk };
        chunks.push(chunk);
        this.options.callbacks?.onUpdate?.(chunk, headers);
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
        this.options.callbacks?.onSuccess?.(chunks, headers);
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
  onInit?: InitHandler;
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
        onInit: config.onInit,
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
      const finalContent = info.chunks.map((chunk) => chunk.content).join('');
      return {
        role: 'assistant',
        content: finalContent || current,
      };
    }
    return {
      role: 'assistant',
      content: `${current}${chunkContent}`,
    };
  }
}
