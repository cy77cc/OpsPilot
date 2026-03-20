import { describe, expect, expectTypeOf, it } from 'vitest';
import {
  applyDone,
  applyDelta,
  applyMeta,
  applyPlan,
  applyReplan,
  applyStepDelta,
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
      plan: undefined,
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
          | 'tool'
          | 'tool_approval'
          | 'hint'
          | 'error';
        label: string;
        detail?: string;
        status?: 'pending' | 'active' | 'done' | 'error';
        stepIndex?: number;
        createdAt?: string;
        arguments?: Record<string, unknown>;
        rawContent?: string;
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

  it('tracks active plan step and scopes tool activity to that step', () => {
    let runtime = applyPlan(createEmptyAssistantRuntime(), {
      steps: ['获取服务器列表', '批量执行健康检查'],
      iteration: 0,
    });
    runtime = applyToolCall(runtime, {
      call_id: 'call-1',
      tool_name: 'host_list_inventory',
      arguments: {},
    });
    runtime = applyReplan(runtime, {
      steps: ['批量执行健康检查', '汇总结果'],
      completed: 1,
      iteration: 1,
      is_final: false,
    });

    expect(runtime.plan).toEqual({
      activeStepIndex: 1,
      steps: [
        { id: 'plan-step-0', title: '获取服务器列表', status: 'done' },
        { id: 'plan-step-1', title: '批量执行健康检查', status: 'active' },
        { id: 'plan-step-2', title: '汇总结果', status: 'pending' },
      ],
    });
    expect(runtime.activities).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          id: 'call-1',
          label: 'host_list_inventory',
          stepIndex: 0,
        }),
      ]),
    );
  });

  it('appends executor markdown to the active step instead of the final body', () => {
    let runtime = applyPlan(createEmptyAssistantRuntime(), {
      steps: ['获取服务器列表', '批量执行健康检查'],
      iteration: 0,
    });
    runtime = applyStepDelta(runtime, { content: '正在收集服务器清单\n' });
    runtime = applyStepDelta(runtime, { content: '已找到 5 台服务器' });

    expect(runtime.plan?.steps[0].content).toBe('正在收集服务器清单\n已找到 5 台服务器');
  });

  it('inserts tool references into active step segments in stream order', () => {
    let runtime = applyPlan(createEmptyAssistantRuntime(), {
      steps: ['采集主机网络指标'],
      iteration: 0,
    });
    runtime = applyStepDelta(runtime, { content: 'Let me start by gathering network statistics ' });
    runtime = applyToolCall(runtime, {
      call_id: 'call-1',
      tool_name: 'os_get_net_stat',
      arguments: { target: '2' },
    });
    runtime = applyStepDelta(runtime, { content: 'from the host.' });

    expect(runtime.plan?.steps[0].segments).toEqual([
      { type: 'text', text: 'Let me start by gathering network statistics ' },
      { type: 'tool_ref', callId: 'call-1' },
      { type: 'text', text: 'from the host.' },
    ]);
    expect(runtime.activities).toEqual(expect.arrayContaining([
      expect.objectContaining({
        id: 'call-1',
        kind: 'tool',
        label: 'os_get_net_stat',
      }),
    ]));
  });

  it('coalesces tool call and result into one done tool activity with raw content', () => {
    let runtime = applyToolCall(createEmptyAssistantRuntime(), {
      call_id: 'call-1',
      tool_name: 'kubectl_describe',
      arguments: {},
    });
    runtime = applyToolResult(runtime, {
      call_id: 'call-1',
      tool_name: 'kubectl_describe',
      content: 'node ok',
    });

    expect(runtime.activities).toHaveLength(1);
    expect(runtime.activities[0]).toEqual(
      expect.objectContaining({
        id: 'call-1',
        kind: 'tool',
        status: 'done',
        detail: 'node ok',
        rawContent: 'node ok',
      }),
    );
  });

  it('marks orphaned tool refs as interrupted terminal errors when done arrives without a result', () => {
    let runtime = applyToolCall(createEmptyAssistantRuntime(), {
      call_id: 'call-1',
      tool_name: 'kubectl_describe',
      arguments: { node: 'node-1' },
    });

    runtime = applyDone(runtime);

    expect(runtime.activities).toHaveLength(1);
    expect(runtime.activities[0]).toEqual(
      expect.objectContaining({
        id: 'call-1',
        kind: 'tool',
        status: 'error',
        detail: '执行未完成',
        arguments: { node: 'node-1' },
      }),
    );
  });

  it('marks completion on done', () => {
    const runtime = applyDone(createEmptyAssistantRuntime());

    expect(runtime.status?.kind).toBe('completed');
    expect(runtime.status?.label).toContain('已生成');
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

  it('stores arguments on tool call', () => {
    const runtime = applyToolCall(createEmptyAssistantRuntime(), {
      call_id: 'call-1',
      tool_name: 'host_get_metrics',
      arguments: { host_id: '123', metric: 'cpu' },
    });

    expect(runtime.activities[0]).toEqual(
      expect.objectContaining({
        id: 'call-1',
        kind: 'tool',
        label: 'host_get_metrics',
        arguments: { host_id: '123', metric: 'cpu' },
      }),
    );
  });

  it('propagates error status from tool result', () => {
    let runtime = applyToolCall(createEmptyAssistantRuntime(), {
      call_id: 'call-1',
      tool_name: 'host_get_metrics',
      arguments: {},
    });
    runtime = applyToolResult(runtime, {
      call_id: 'call-1',
      tool_name: 'host_get_metrics',
      content: '{"error": "timeout"}',
      status: 'error',
    });

    expect(runtime.activities[0]).toEqual(
      expect.objectContaining({
        kind: 'tool',
        status: 'error',
        rawContent: '{"error": "timeout"}',
      }),
    );
  });

  it('truncates long content in detail but stores full content in rawContent', () => {
    const longContent = 'x'.repeat(300);
    let runtime = applyToolCall(createEmptyAssistantRuntime(), {
      call_id: 'call-1',
      tool_name: 'host_get_metrics',
      arguments: {},
    });
    runtime = applyToolResult(runtime, {
      call_id: 'call-1',
      tool_name: 'host_get_metrics',
      content: longContent,
    });

    expect(runtime.activities[0].detail).toBe('x'.repeat(200));
    expect(runtime.activities[0].rawContent).toBe(longContent);
  });
});
