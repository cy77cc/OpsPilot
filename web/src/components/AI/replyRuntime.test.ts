import { describe, expect, expectTypeOf, it } from 'vitest';
import {
  applyDone,
  applyDelta,
  applyMeta,
  applyPlan,
  applyReplan,
  applyRecoverableError,
  applySoftTimeout,
  applyTerminalError,
  applyToolApproval,
  applyToolCall,
  applyToolResult,
  applyAgentHandoff,
  createEmptyAssistantRuntime,
} from './replyRuntime';

describe('assistant reply runtime shape', () => {
  it('creates an empty runtime with append-ready activity state', () => {
    expect(createEmptyAssistantRuntime()).toEqual({
      activities: [],
      phase: undefined,
      phaseLabel: undefined,
      summary: undefined,
      status: undefined,
    });
  });

  it('defines spec-required phase, activity, summary, and status fields', () => {
    const runtime = createEmptyAssistantRuntime();

    expect(runtime.activities).toEqual([]);
    expectTypeOf(runtime.phase).toEqualTypeOf<
      | 'preparing'
      | 'identifying'
      | 'planning'
      | 'executing'
      | 'summarizing'
      | 'completed'
      | 'interrupted'
      | undefined
    >();
    expectTypeOf(runtime.summary).toEqualTypeOf<
      | {
          title?: string;
          items?: Array<{
            label: string;
            value: string;
            tone?: 'default' | 'success' | 'warning' | 'danger';
          }>;
        }
      | undefined
    >();
    expectTypeOf(runtime.status).toEqualTypeOf<
      | {
          kind: 'streaming' | 'completed' | 'soft-timeout' | 'error' | 'interrupted';
          label: string;
        }
      | undefined
    >();
    expectTypeOf(runtime.activities).toEqualTypeOf<
      Array<{
        id: string;
        kind:
          | 'agent_handoff'
          | 'plan'
          | 'replan'
          | 'tool_call'
          | 'tool_approval'
          | 'tool_result'
          | 'hint'
          | 'error';
        label: string;
        detail?: string;
        status?: 'pending' | 'active' | 'done' | 'error';
        createdAt?: string;
      }>
    >();
  });

  it('replaces placeholder status when the first delta arrives', () => {
    const state = applyDelta(
      {
        content: '[准备中]',
        runtime: createEmptyAssistantRuntime(),
      },
      { content: '第一段' },
    );

    expect(state.content).toBe('第一段');
  });

  it('coalesces duplicate tool activity rows by call id', () => {
    let runtime = applyToolCall(createEmptyAssistantRuntime(), {
      call_id: 'call-1',
      tool_name: 'kubectl_describe',
      arguments: {},
    });
    runtime = applyToolCall(runtime, {
      call_id: 'call-1',
      tool_name: 'kubectl_describe',
      arguments: {},
    });

    expect(runtime.activities).toHaveLength(1);
  });

  it('marks soft timeout as transient footer status', () => {
    const runtime = applySoftTimeout(createEmptyAssistantRuntime());

    expect(runtime.status).toEqual({
      kind: 'soft-timeout',
      label: '工具执行较慢，正在继续等待结果…',
    });
  });

  it('keeps streamed markdown while surfacing recoverable errors as hints', () => {
    const state = applyRecoverableError(
      {
        content: '已经拿到部分结果',
        runtime: createEmptyAssistantRuntime(),
      },
      { message: '工具执行较慢，正在继续等待结果…' },
    );

    expect(state.content).toBe('已经拿到部分结果');
    expect(state.runtime?.status).toEqual({
      kind: 'soft-timeout',
      label: '工具执行较慢，正在继续等待结果…',
    });
  });

  it('updates phase when meta arrives', () => {
    const runtime = applyMeta(createEmptyAssistantRuntime());

    expect(runtime.phase).toBe('preparing');
    expect(runtime.phaseLabel).toBe('准备中');
  });

  it('appends a handoff activity and updates the phase label', () => {
    const runtime = applyAgentHandoff(createEmptyAssistantRuntime(), {
      from: 'OpsPilotAgent',
      to: 'DiagnosisAgent',
      intent: 'diagnosis',
    });

    expect(runtime.phase).toBe('executing');
    expect(runtime.activities[0]).toEqual(
      expect.objectContaining({
        kind: 'agent_handoff',
        label: 'DiagnosisAgent',
      }),
    );
  });

  it('replaces plan activity with replan activity for the same planning lane', () => {
    let runtime = applyPlan(createEmptyAssistantRuntime(), {
      steps: ['检查节点'],
      iteration: 0,
    });
    runtime = applyReplan(runtime, {
      steps: ['检查节点', '汇总异常'],
      completed: 1,
      iteration: 1,
      is_final: false,
    });

    expect(runtime.activities.filter((item) => item.kind === 'plan' || item.kind === 'replan')).toHaveLength(1);
    expect(runtime.activities[0]).toEqual(
      expect.objectContaining({
        kind: 'replan',
        detail: '检查节点 -> 汇总异常',
      }),
    );
  });

  it('marks tool approval and result updates on the same activity row', () => {
    let runtime = applyToolCall(createEmptyAssistantRuntime(), {
      call_id: 'call-1',
      tool_name: 'kubectl_describe',
      arguments: {},
    });
    runtime = applyToolApproval(runtime, {
      approval_id: 'approval-1',
      call_id: 'call-1',
      tool_name: 'kubectl_describe',
      preview: {},
      timeout_seconds: 300,
    });
    runtime = applyToolResult(runtime, {
      call_id: 'call-1',
      tool_name: 'kubectl_describe',
      content: 'node ok',
    });

    expect(runtime.activities).toHaveLength(1);
    expect(runtime.activities[0]).toEqual(
      expect.objectContaining({
        kind: 'tool_result',
        status: 'done',
        detail: 'node ok',
      }),
    );
  });

  it('marks completion on done', () => {
    const runtime = applyDone(createEmptyAssistantRuntime());

    expect(runtime.status).toEqual({
      kind: 'completed',
      label: '已生成',
    });
  });

  it('preserves content and appends terminal error state', () => {
    const state = applyTerminalError(
      {
        content: '已经拿到部分结果',
        runtime: createEmptyAssistantRuntime(),
      },
      { message: 'stream failed' },
    );

    expect(state.content).toBe('已经拿到部分结果');
    expect(state.runtime?.status).toEqual({
      kind: 'error',
      label: 'stream failed',
    });
  });
});
