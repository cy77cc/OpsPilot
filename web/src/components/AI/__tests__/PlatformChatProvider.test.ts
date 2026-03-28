import React from 'react';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { PlatformChatProvider, PlatformChatRequest } from '../providers/PlatformChatProvider';
import { aiApi } from '../../../api/modules/ai';
import { AssistantReply } from '../AssistantReply';
import type { AssistantReplyRuntime } from '../types';

vi.mock('../../../api/modules/ai', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../api/modules/ai')>();
  actual.aiApi.chatStream = vi.fn();
  actual.aiApi.submitApproval = vi.fn();
  actual.aiApi.retryResumeApproval = vi.fn();
  actual.aiApi.getApproval = vi.fn();
  actual.aiApi.listPendingApprovals = vi.fn();
  return {
    ...actual,
    aiApi: actual.aiApi,
  };
});

vi.mock('../ToolResultCard', () => ({
  default: () => React.createElement('div', { 'data-testid': 'tool-result-card' }),
}));

afterEach(() => {
  cleanup();
  localStorage.clear();
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

describe('PlatformChatProvider', () => {
  it('passes clientRequestId to aiApi.chatStream', async () => {
    const request = new PlatformChatRequest();
    vi.mocked(aiApi.chatStream).mockImplementation(async () => undefined);
    request.run({ message: 'hi', scene: 'ai', clientRequestId: 'req-1' });
    await request.asyncHandler;

    expect(aiApi.chatStream).toHaveBeenCalledWith(
      expect.objectContaining({ clientRequestId: 'req-1' }),
      expect.anything(),
      expect.anything(),
    );
  });

  it('passes lastEventId to aiApi.chatStream', async () => {
    const request = new PlatformChatRequest();
    vi.mocked(aiApi.chatStream).mockImplementation(async () => undefined);
    request.run({ message: 'hi', scene: 'ai', lastEventId: 'evt-123' });
    await request.asyncHandler;

    expect(aiApi.chatStream).toHaveBeenCalledWith(
      expect.objectContaining({ lastEventId: 'evt-123' }),
      expect.anything(),
      expect.anything(),
    );
  });

  it('replaces runtime snapshots from ops_plan_updated events', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess: vi.fn(),
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onPlan?.({ steps: ['旧步骤'], iteration: 0 });
      handlers.onToolCall?.({ call_id: 'call-1', tool_name: 'kubectl_get', arguments: {} });
      handlers.onOpsPlanUpdated?.({
        run_id: 'run-1',
        session_id: 'sess-1',
        runtime: {
          phase: 'planning',
          phaseLabel: 'Task Board',
          todos: [
            { id: 'todo-1', content: '检查节点', status: 'in_progress' },
            { id: 'todo-2', content: '汇总结果', status: 'pending' },
          ],
        } as any,
      } as any);
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    const snapshotUpdate = onUpdate.mock.calls.find(([chunk]) => Boolean(chunk?.runtime?.todos?.length));

    expect(snapshotUpdate).toBeTruthy();
    expect(snapshotUpdate?.[0]).toEqual(
      expect.objectContaining({
        runtime: expect.objectContaining({
          phase: 'planning',
          phaseLabel: 'Task Board',
          todos: [
            expect.objectContaining({
              id: 'todo-1',
              content: '检查节点',
              status: 'in_progress',
            }),
            expect.objectContaining({
              id: 'todo-2',
              content: '汇总结果',
              status: 'pending',
            }),
          ],
          plan: undefined,
          activities: [],
        }),
      }),
    );
  });

  it('merges scene, session, and context into request params', () => {
    const provider = new PlatformChatProvider({
      scene: 'cluster',
      getSessionId: () => 'sess-1',
      getSceneContext: () => ({ route: '/deployment/infrastructure/clusters/42', resourceId: '42' }),
    });

    const params = provider.transformParams(
      { message: 'check health' },
      { params: undefined } as any,
    );

    expect(params).toEqual(expect.objectContaining({
      message: 'check health',
      sessionId: 'sess-1',
      clientRequestId: expect.any(String),
      scene: 'cluster',
      context: {
        route: '/deployment/infrastructure/clusters/42',
        resourceId: '42',
      },
    }));
  });

  it('streams delta chunks through request callbacks', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onSuccess = vi.fn();
    const onError = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess,
      onError,
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
      handlers.onPlan?.({ steps: ['inspect pods'], iteration: 0 });
      handlers.onDelta?.({ content: 'hello ' });
      handlers.onDelta?.({ content: 'world' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenCalledTimes(5);
    expect(onUpdate).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ content: '[准备中]', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ content: '[正在规划处理方式]', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        content: '[正在规划处理方式]',
        runtime: expect.objectContaining({
          plan: expect.objectContaining({
            steps: [
              expect.objectContaining({ title: 'inspect pods', content: 'hello world' }),
            ],
          }),
        }),
      }),
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith([], expect.any(Headers));
    expect(onError).not.toHaveBeenCalled();
  });

  it('debounces agent labels before emitting status updates', async () => {
    vi.useFakeTimers();
    try {
      const request = new PlatformChatRequest();
      const onUpdate = vi.fn();
      request.options.callbacks = {
        onUpdate,
        onSuccess: vi.fn(),
        onError: vi.fn(),
      };

      vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
        handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
        handlers.onAgentHandoff?.({ from: 'OpsPilotAgent', to: 'DiagnosisAgent', intent: 'diagnosis' });
        handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
      });

      request.run({ message: 'hi', scene: 'ai' });
      await request.asyncHandler;

      expect(onUpdate).toHaveBeenCalledTimes(1);
      await vi.advanceTimersByTimeAsync(399);
      expect(onUpdate).toHaveBeenCalledTimes(1);
      await vi.advanceTimersByTimeAsync(1);
      expect(onUpdate).toHaveBeenCalledTimes(2);
      expect(onUpdate).toHaveBeenLastCalledWith(
        expect.objectContaining({ content: '[诊断助手开始处理]', mode: 'replace' }),
        expect.any(Headers),
      );
    } finally {
      vi.useRealTimers();
    }
  });

  it('shows localized status updates and withholds visible content until intent', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess,
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
      handlers.onAgentHandoff?.({ from: 'OpsPilotAgent', to: 'DiagnosisAgent', intent: 'diagnosis' });
      handlers.onDelta?.({ content: '诊断完成' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ content: '[准备中]', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ content: '[诊断助手开始处理]', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      3,
      expect.objectContaining({ content: '诊断完成', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith(
      [expect.objectContaining({ content: '诊断完成', mode: 'replace' })],
      expect.any(Headers),
    );
  });

  it('appends later visible chunks after replacing the placeholder', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess,
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onPlan?.({ steps: ['step one'], iteration: 0 });
      handlers.onDelta?.({ content: '第一段' });
      handlers.onDelta?.({ content: '，第二段' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ content: '[正在规划处理方式]', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        content: '[正在规划处理方式]',
        runtime: expect.objectContaining({
          plan: expect.objectContaining({
            steps: [
              expect.objectContaining({ title: 'step one', content: '第一段，第二段' }),
            ],
          }),
        }),
      }),
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith([], expect.any(Headers));
  });

  it('preserves runtime activities while streaming visible markdown', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess: vi.fn(),
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onToolCall?.({ call_id: 'call-1', tool_name: 'kubectl_get', arguments: {} });
      handlers.onDelta?.({ content: '巡检完成' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'cluster' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        runtime: expect.objectContaining({
          activities: [expect.objectContaining({ id: 'call-1' })],
        }),
      }),
      expect.any(Headers),
    );
  });

  it('observes unknown stream events without mutating runtime state', async () => {
    const onUnknownEvent = vi.fn();
    const provider = new PlatformChatProvider({ onUnknownEvent });
    const request = provider.request as PlatformChatRequest;
    const onUpdate = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess: vi.fn(),
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
      handlers.onUnknownEvent?.({
        eventType: 'ai.debug.unknown',
        payload: {
          run_id: 'run-1',
          user_id: 7,
          tenant_id: 'tenant-1',
          detail: 'ignored',
        },
        eventId: 'evt-unknown',
        runId: 'run-1',
        userId: 7,
        tenantId: 'tenant-1',
      });
      handlers.onDelta?.({ content: '继续处理' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUnknownEvent).toHaveBeenCalledWith(
      expect.objectContaining({
        eventType: 'ai.debug.unknown',
        eventId: 'evt-unknown',
        runId: 'run-1',
        userId: 7,
        tenantId: 'tenant-1',
        payload: expect.objectContaining({
          detail: 'ignored',
        }),
      }),
    );
    expect(onUpdate).toHaveBeenCalledTimes(3);
    expect(onUpdate).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        content: '继续处理',
        runtime: expect.objectContaining({
          phase: 'preparing',
          phaseLabel: '准备中',
        }),
      }),
      expect.any(Headers),
    );
  });

  it('emits a runtime update for tool approval before any visible text arrives', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess: vi.fn(),
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
      handlers.onToolCall?.({ call_id: 'call-1', tool_name: 'kubectl_apply', arguments: {} });
      handlers.onToolApproval?.({
        approval_id: 'approval-1',
        call_id: 'call-1',
        tool_name: 'kubectl_apply',
        preview: {},
        timeout_seconds: 300,
      });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'cluster' });
    await request.asyncHandler;

    const approvalUpdate = onUpdate.mock.calls.find(([chunk]) => Boolean(chunk?.runtime?.activities?.some((activity: any) => activity.kind === 'tool_approval')));

    expect(approvalUpdate).toBeTruthy();
    expect(approvalUpdate?.[0]).toEqual(
      expect.objectContaining({
        content: '[准备中]',
        runtime: expect.objectContaining({
          activities: [
            expect.objectContaining({
              kind: 'tool_approval',
              approvalId: 'approval-1',
              approvalState: 'waiting-approval',
            }),
          ],
        }),
      }),
    );
  });

  it('projects approval resume events into the runtime stream', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess: vi.fn(),
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onRunResuming?.({ run_id: 'run-1', session_id: 'sess-1', approval_id: 'approval-1' });
      handlers.onRunResumeFailed?.({
        run_id: 'run-1',
        session_id: 'sess-1',
        approval_id: 'approval-1',
        retryable: true,
        message: 'retry later',
      });
      handlers.onRunResumed?.({ run_id: 'run-1', session_id: 'sess-1', approval_id: 'approval-1' });
    });

    request.run({ message: 'hi', scene: 'cluster' });
    await request.asyncHandler;

    const statuses = onUpdate.mock.calls
      .map(([chunk]) => chunk?.runtime?.status?.kind)
      .filter(Boolean);

    expect(statuses).toContain('approved_resuming');
    expect(statuses).toContain('approved_retrying');
    expect(statuses).toContain('approved_done');
  });

  it('stops reconnect attempts when run enters waiting_approval', async () => {
    const request = new PlatformChatRequest();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate: vi.fn(),
      onSuccess,
      onError: vi.fn(),
    };

    const streamMock = vi.mocked(aiApi.chatStream)
      .mockImplementationOnce(async (_params, handlers) => {
        handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
        handlers.onEventId?.('evt-1');
        handlers.onToolApproval?.({
          approval_id: 'approval-1',
          call_id: 'call-1',
          tool_name: 'kubectl_apply',
          preview: {},
          timeout_seconds: 300,
        });
        handlers.onRunState?.({ run_id: 'run-1', status: 'waiting_approval', agent: 'executor' } as any);
      });

    request.run({ message: 'hi', scene: 'cluster' });
    window.dispatchEvent(new CustomEvent('ai-approval-updated', {
      detail: { token: 'approval-1', status: 'approved' },
    }));

    await request.asyncHandler;

    expect(streamMock).toHaveBeenCalledTimes(1);
    expect(onSuccess).toHaveBeenCalledTimes(1);
  });

  it('ignores approval token updates while waiting_approval without reconnecting', async () => {
    const request = new PlatformChatRequest();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate: vi.fn(),
      onSuccess,
      onError: vi.fn(),
    };

    const streamMock = vi.mocked(aiApi.chatStream)
      .mockImplementationOnce(async (_params, handlers) => {
        handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
        handlers.onEventId?.('evt-1');
        handlers.onToolApproval?.({
          approval_id: 'approval-1',
          call_id: 'call-1',
          tool_name: 'kubectl_apply',
          preview: {},
          timeout_seconds: 300,
        });
        handlers.onRunState?.({ run_id: 'run-1', status: 'waiting_approval', agent: 'executor' } as any);
      });

    request.run({ message: 'hi', scene: 'cluster' });
    window.dispatchEvent(new CustomEvent('ai-approval-updated', {
      detail: { token: 'call-1', status: 'approved' },
    }));

    await request.asyncHandler;

    expect(streamMock).toHaveBeenCalledTimes(1);
    expect(onSuccess).toHaveBeenCalledTimes(1);
  });

  it('does not reconnect for resume_failed_retryable after approval updates', async () => {
    const request = new PlatformChatRequest();
    request.options.callbacks = {
      onUpdate: vi.fn(),
      onSuccess: vi.fn(),
      onError: vi.fn(),
    };

    const streamMock = vi.mocked(aiApi.chatStream)
      .mockImplementationOnce(async (_params, handlers) => {
        handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
        handlers.onEventId?.('evt-4');
        handlers.onToolApproval?.({
          approval_id: 'approval-1',
          call_id: 'call-1',
          tool_name: 'kubectl_apply',
          preview: {},
          timeout_seconds: 300,
        });
        handlers.onRunState?.({ run_id: 'run-1', status: 'resume_failed_retryable', agent: 'executor' } as any);
      })
      .mockImplementationOnce(async (_params, handlers) => {
        handlers.onRunState?.({ run_id: 'run-1', status: 'resuming', agent: 'executor' } as any);
        handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
      });

    request.run({ message: 'hi', scene: 'cluster' });
    await vi.waitFor(() => {
      expect(streamMock).toHaveBeenCalledTimes(1);
    });

    window.dispatchEvent(new CustomEvent('ai-approval-updated', {
      detail: { token: 'approval-1', status: 'approved' },
    }));

    await request.asyncHandler;

    expect(aiApi.retryResumeApproval).not.toHaveBeenCalled();
    expect(streamMock).toHaveBeenCalledTimes(1);
  });

  it('handles cursor expired stream errors without request-level failure', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onError = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onError,
      onSuccess: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
      handlers.onError?.({
        code: 'AI_STREAM_CURSOR_EXPIRED',
        message: 'last_event_id is too old; refresh the stream snapshot',
        recoverable: true,
      });
    });

    request.run({ message: 'hi', scene: 'cluster', lastEventId: 'evt-123' });
    await request.asyncHandler;

    expect(onError).not.toHaveBeenCalled();
    expect(onUpdate).toHaveBeenCalled();
  });

  it('treats message-only stale cursor errors as terminal for pending reconnect state', async () => {
    const request = new PlatformChatRequest();
    request.options.callbacks = {
      onUpdate: vi.fn(),
      onError: vi.fn(),
      onSuccess: vi.fn(),
    };

    const streamMock = vi.mocked(aiApi.chatStream);
    streamMock.mockImplementation(async (_params, handlers) => {
      if (streamMock.mock.calls.length > 1) {
        handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
        return;
      }

      handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
      handlers.onRunState?.({ run_id: 'run-1', status: 'resuming', agent: 'executor' } as any);
      handlers.onError?.({
        message: 'last_event_id is too old; refresh the stream snapshot',
        recoverable: true,
      });
    });

    request.run({ message: 'hi', scene: 'cluster', lastEventId: 'evt-stale' });
    await request.asyncHandler;

    expect(streamMock).toHaveBeenCalledTimes(1);
  });

  it('projects replan, approval, recoverable error, handoff, tool result, and terminal error states', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess: vi.fn(),
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
      handlers.onAgentHandoff?.({ from: 'OpsPilotAgent', to: 'DiagnosisAgent', intent: 'diagnosis' });
      handlers.onPlan?.({ steps: ['检查节点'], iteration: 0 });
      handlers.onReplan?.({ steps: ['检查节点', '汇总异常'], completed: 1, iteration: 1, is_final: false });
      handlers.onToolCall?.({ call_id: 'call-1', tool_name: 'kubectl_get', arguments: {} });
      handlers.onToolApproval?.({
        approval_id: 'approval-1',
        call_id: 'call-1',
        tool_name: 'kubectl_get',
        preview: {},
        timeout_seconds: 300,
      });
      handlers.onDelta?.({ content: '已经拿到部分结果' });
      handlers.onToolResult?.({ call_id: 'call-1', tool_name: 'kubectl_get', content: 'node-1 ok' });
      handlers.onError?.({ message: '工具执行较慢，正在继续等待结果…', code: 'tool_timeout_soft', recoverable: true });
      handlers.onError?.({ message: 'stream failed', code: 'stream_failed', recoverable: false });
    });

    request.run({ message: 'hi', scene: 'cluster' });
    await request.asyncHandler;

    const hasExpectedRuntime = onUpdate.mock.calls.some(([chunk]) => {
      const runtime = chunk?.runtime;
      if (!runtime?.plan?.steps) {
        return false;
      }
      const hasStepContent = runtime.plan.steps.some((step: any) => step.content === '已经拿到部分结果');
      const hasReplan = runtime.activities?.some((activity: any) => activity.kind === 'replan');
      return hasStepContent && hasReplan;
    });

    expect(hasExpectedRuntime).toBe(true);
    expect(onUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        runtime: expect.objectContaining({
          status: expect.objectContaining({ kind: 'failed', label: 'stream failed' }),
        }),
      }),
      expect.any(Headers),
    );
  });

  it('does not escalate tool errors to request failure fallback', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onError = vi.fn();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onError,
      onSuccess,
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onPlan?.({ steps: ['获取服务器列表'], iteration: 0 });
      handlers.onDelta?.({ agent: 'executor', content: '开始执行\n' });
      handlers.onError?.({ message: '工具调用失败', code: 'tool_call_failed', recoverable: false });
      handlers.onDelta?.({ agent: 'executor', content: '继续执行' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'host' });
    await request.asyncHandler;

    expect(onError).not.toHaveBeenCalled();
    expect(onSuccess).toHaveBeenCalledWith([], expect.any(Headers));
    expect(onUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        runtime: expect.objectContaining({
          plan: expect.objectContaining({
            steps: [expect.objectContaining({ content: '开始执行\n继续执行' })],
          }),
          status: expect.objectContaining({ kind: 'failed', label: '工具调用失败' }),
        }),
      }),
      expect.any(Headers),
    );
  });

  it('buffers planner and replanner delta envelopes before exposing them', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess,
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onDelta?.({ agent: 'planner', content: '{"steps":["获取服务器列表",' });
      handlers.onDelta?.({ agent: 'planner', content: '"批量执行健康检查"]}' });
      handlers.onToolCall?.({ call_id: 'call-1', tool_name: 'host_list_inventory', arguments: {} });
      handlers.onDelta?.({ agent: 'replanner', content: '{"response":"## 检查完成' });
      handlers.onDelta?.({ agent: 'replanner', content: '\\n\\n全部正常"}' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'host' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        content: '## 检查完成\n\n全部正常',
        runtime: expect.objectContaining({
          plan: expect.objectContaining({
            steps: [
              expect.objectContaining({ title: '获取服务器列表' }),
              expect.objectContaining({ title: '批量执行健康检查' }),
            ],
          }),
        }),
      }),
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith(
      [expect.objectContaining({ content: '## 检查完成\n\n全部正常', mode: 'replace' })],
      expect.any(Headers),
    );
  });

  it('falls back to normalized final markdown when replanner emits escaped content chunks', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess,
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onPlan?.({ steps: ['检查集群'], iteration: 0 });
      handlers.onReplan?.({ steps: [], completed: 1, iteration: 1, is_final: true });
      handlers.onDelta?.({ agent: 'replanner', content: '## Local 集群概览\\n\\n共有 21 个 Pod' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'cluster' });
    await request.asyncHandler;

    expect(onSuccess).toHaveBeenCalledWith(
      [expect.objectContaining({ content: '## Local 集群概览\n\n共有 21 个 Pod', mode: 'replace' })],
      expect.any(Headers),
    );
  });

  it('routes executor delta text into the active step instead of the final markdown body', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess,
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onPlan?.({ steps: ['获取服务器列表', '批量执行健康检查'], iteration: 0 });
      handlers.onDelta?.({ agent: 'executor', content: '正在获取主机列表' });
      handlers.onToolCall?.({ call_id: 'call-1', tool_name: 'host_list_inventory', arguments: {} });
      handlers.onDelta?.({ agent: 'replanner', content: '{"response":"## 检查完成"}' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'host' });
    await request.asyncHandler;

    const hasExecutorStepContent = onUpdate.mock.calls.some(([chunk]) => {
      const steps = chunk?.runtime?.plan?.steps;
      return Array.isArray(steps) && steps.some((step: any) => step.title === '获取服务器列表' && step.content === '正在获取主机列表');
    });

    expect(hasExecutorStepContent).toBe(true);
    expect(onSuccess).toHaveBeenCalledWith(
      [expect.objectContaining({ content: '## 检查完成', mode: 'replace' })],
      expect.any(Headers),
    );
  });

  it.skip('renders approval actions and submits the selected decision', async () => {
    const user = userEvent.setup();
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent');
    vi.mocked(aiApi.submitApproval).mockResolvedValue({
      success: true,
      data: {
        id: 'approval-1',
        status: 'approved',
      },
    } as any);

    const runtime: AssistantReplyRuntime = {
      activities: [
        {
          id: 'call-1',
          kind: 'tool_approval',
          label: 'kubectl_apply',
          detail: '等待审批 300s',
          status: 'pending',
          approvalId: 'approval-1',
        },
      ],
    };

    render(
      React.createElement(AssistantReply, {
        content: '',
        runtime,
        status: 'updating',
      }),
    );

    expect(screen.getByRole('button', { name: /批\s*准/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /拒\s*绝/ })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /批\s*准/ }));

    expect(aiApi.submitApproval).toHaveBeenCalledWith(
      'approval-1',
      { approved: true },
      expect.objectContaining({
        idempotencyKey: expect.any(String),
      }),
    );
    expect(dispatchSpy).toHaveBeenCalledWith(expect.objectContaining({ type: 'ai-approval-updated' }));

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /kubectl_apply/ })).toHaveTextContent('已批准');
    });
  });

  it.skip('shows the refresh-needed failure state when submitApproval fails', async () => {
    const user = userEvent.setup();
    vi.mocked(aiApi.submitApproval).mockRejectedValueOnce(new Error('network down'));

    const runtime: AssistantReplyRuntime = {
      activities: [
        {
          id: 'call-1',
          kind: 'tool_approval',
          label: 'kubectl_apply',
          detail: '等待审批 300s',
          status: 'pending',
          approvalId: 'approval-1',
        },
      ],
    };

    render(
      React.createElement(AssistantReply, {
        content: '',
        runtime,
        status: 'updating',
      }),
    );

    await user.click(screen.getByRole('button', { name: /批\s*准/ }));

    await waitFor(() => {
      expect(screen.getByText(/提交失败：network down/)).toBeInTheDocument();
      expect(screen.getByText(/需刷新/)).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /批\s*准/ })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /拒\s*绝/ })).not.toBeInTheDocument();
    });
  });

  it.skip('refreshes approval state on conflict and shows the latest status', async () => {
    const user = userEvent.setup();
    vi.mocked(aiApi.submitApproval).mockRejectedValueOnce({
      statusCode: 409,
      message: 'already approved',
    });
    vi.mocked(aiApi.getApproval).mockResolvedValueOnce({
      success: true,
      data: {
        id: 'approval-1',
        status: 'approved',
      },
    } as any);

    const runtime: AssistantReplyRuntime = {
      activities: [
        {
          id: 'call-1',
          kind: 'tool_approval',
          label: 'kubectl_apply',
          detail: '等待审批 300s',
          status: 'pending',
          approvalId: 'approval-1',
        },
      ],
    };

    render(
      React.createElement(AssistantReply, {
        content: '',
        runtime,
        status: 'updating',
      }),
    );

    await user.click(screen.getByRole('button', { name: /批\s*准/ }));

    await waitFor(() => {
      expect(aiApi.getApproval).toHaveBeenCalledWith('approval-1');
      expect(screen.getByRole('button', { name: /kubectl_apply/ })).toHaveTextContent('已批准');
    });
  });

  it.skip('keeps conflict refresh fallback non-interactive when refresh is flaky', async () => {
    const user = userEvent.setup();
    vi.mocked(aiApi.submitApproval).mockRejectedValueOnce({
      statusCode: 409,
      message: 'already approved',
    });
    vi.mocked(aiApi.getApproval).mockRejectedValueOnce(new Error('refresh failed'));
    vi.mocked(aiApi.listPendingApprovals).mockResolvedValueOnce({
      success: true,
      data: [
        {
          id: 'approval-1',
          status: 'pending',
        },
      ],
    } as any);

    const runtime: AssistantReplyRuntime = {
      activities: [
        {
          id: 'call-1',
          kind: 'tool_approval',
          label: 'kubectl_apply',
          detail: '等待审批 300s',
          status: 'pending',
          approvalId: 'approval-1',
        },
      ],
    };

    render(
      React.createElement(AssistantReply, {
        content: '',
        runtime,
        status: 'updating',
      }),
    );

    await user.click(screen.getByRole('button', { name: /批\s*准/ }));

    await waitFor(() => {
      expect(aiApi.getApproval).toHaveBeenCalledWith('approval-1');
      expect(aiApi.listPendingApprovals).toHaveBeenCalled();
      expect(screen.getByText(/刷新后查看结果|审批状态可能已变更/)).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /批\s*准/ })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /拒\s*绝/ })).not.toBeInTheDocument();
    });
  });
});
