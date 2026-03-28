import { describe, expect, expectTypeOf, it } from 'vitest';
import {
  applyApprovalExpired,
  applyDone,
  applyDelta,
  applyMeta,
  applyPlan,
  applyReplan,
  applyRunResuming,
  applyRunResumed,
  applyRunResumeFailed,
  applyRunState,
  applyRuntimeSnapshot,
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
      todos: [],
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
    expectTypeOf(runtime.todos).toEqualTypeOf<
      Array<{
        id: string;
        content: string;
        status: string;
        activeForm?: string;
        cluster?: string;
        namespace?: string;
        resourceType?: string;
        riskLevel?: string;
        requiresApproval?: boolean;
        estimatedDuration?: string;
        dependsOn?: string[];
      }>
      | undefined
    >();
    expectTypeOf(runtime.status).toEqualTypeOf<
      | {
          kind:
            | 'streaming'
            | 'completed'
            | 'soft-timeout'
            | 'error'
            | 'waiting_approval'
            | 'resuming'
            | 'resume_failed_retryable'
            | 'failed'
            | 'interrupted'
            | 'approved_resuming'
            | 'approved_retrying'
            | 'approved_failed_terminal'
            | 'approved_done'
            | 'expired';
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
        approvalId?: string;
        approvalState?:
          | 'waiting-approval'
          | 'submitting'
          | 'approved_resuming'
          | 'approved_retrying'
          | 'approved_failed_terminal'
          | 'approved_done'
          | 'expired'
          | 'approved'
          | 'rejected'
          | 'refresh-needed';
        approvalMessage?: string;
        stepIndex?: number;
        createdAt?: string;
        arguments?: Record<string, unknown>;
        rawContent?: string;
        approvalPreview?: Record<string, unknown>;
        approvalPreviewSummary?: Array<{
          key: string;
          label: string;
          value: string;
        }>;
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

  it('stores approval id and pending state on tool approval activities', () => {
    const runtime = applyToolApproval(createEmptyAssistantRuntime(), {
      approval_id: 'approval-1',
      call_id: 'call-1',
      tool_name: 'kubectl_apply',
      preview: {},
      timeout_seconds: 300,
    });

    expect(runtime.activities[0]).toEqual(
      expect.objectContaining({
        kind: 'tool_approval',
        approvalId: 'approval-1',
        status: 'pending',
      }),
    );
  });

  it('stores tool approval preview and derives ordered summary rows', () => {
    const runtime = applyToolApproval(createEmptyAssistantRuntime(), {
      approval_id: 'approval-1',
      call_id: 'call-1',
      tool_name: 'kubectl_apply',
      timeout_seconds: 300,
      preview: {
        extra9: 'drop-me',
        action: 'apply',
        riskLevel: 'high',
        namespace: 'ops',
        cluster: 'prod',
        name: 'payments-api',
        resourceType: 'deployment',
        kind: 'Deployment',
        dryRun: false,
        manifest: {
          replicas: 3,
        },
      },
    });

    expect(runtime.activities[0]).toEqual(
      expect.objectContaining({
        approvalPreview: expect.objectContaining({
          cluster: 'prod',
          namespace: 'ops',
        }),
        approvalPreviewSummary: [
          { key: 'cluster', label: 'cluster', value: 'prod' },
          { key: 'namespace', label: 'namespace', value: 'ops' },
          { key: 'resource', label: 'resource', value: 'deployment' },
          { key: 'kind', label: 'kind', value: 'Deployment' },
          { key: 'name', label: 'name', value: 'payments-api' },
          { key: 'action', label: 'action', value: 'apply' },
          { key: 'risk', label: 'risk', value: 'high' },
          { key: 'dryRun', label: 'dryRun', value: 'false' },
        ],
      }),
    );
  });

  it('keeps approval preview but falls back gracefully when no structured rows can be derived', () => {
    const runtime = applyToolApproval(createEmptyAssistantRuntime(), {
      approval_id: 'approval-1',
      call_id: 'call-1',
      tool_name: 'kubectl_apply',
      timeout_seconds: 300,
      preview: {},
    });

    expect(runtime.activities[0]).toEqual(
      expect.objectContaining({
        approvalPreview: {},
        approvalPreviewSummary: [],
      }),
    );
  });

  it('replaces runtime snapshots without preserving stale plan or activity state', () => {
    const runtime = applyRuntimeSnapshot(
      {
        activities: [
          {
            id: 'call-1',
            kind: 'tool',
            label: 'kubectl_get',
            status: 'done',
          },
        ],
        plan: {
          activeStepIndex: 0,
          steps: [
            { id: 'plan-step-0', title: '旧步骤', status: 'done' },
          ],
        },
        phase: 'executing',
        phaseLabel: '旧状态',
        summary: { title: '旧摘要' },
        status: { kind: 'streaming', label: 'streaming' },
        todos: [
          { id: 'todo-old', content: '旧任务', status: 'pending' },
        ],
      },
      {
        phase: 'planning',
        phaseLabel: 'Task Board',
        todos: [
          { id: 'todo-1', content: '检查节点', status: 'in_progress', activeForm: '正在检查节点' },
          { id: 'todo-2', content: '汇总结果', status: 'pending' },
        ],
      },
    );

    expect(runtime).toEqual(expect.objectContaining({
      phase: 'planning',
      phaseLabel: 'Task Board',
      activities: [],
      plan: undefined,
      summary: undefined,
      status: undefined,
      todos: [
        expect.objectContaining({
          id: 'todo-1',
          content: '检查节点',
          activeForm: '正在检查节点',
          status: 'in_progress',
        }),
        expect.objectContaining({
          id: 'todo-2',
          content: '汇总结果',
          status: 'pending',
        }),
      ],
    }));
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

  it('maps approval resume states into deterministic runtime statuses', () => {
    const resuming = applyRunResuming(createEmptyAssistantRuntime());
    expect(resuming.status).toEqual({
      kind: 'approved_resuming',
      label: '已批准，恢复中',
    });

    const retrying = applyRunResumeFailed(resuming, { retryable: true, message: 'retry later' });
    expect(retrying.status).toEqual({
      kind: 'approved_retrying',
      label: 'retry later',
    });

    const terminal = applyRunResumeFailed(createEmptyAssistantRuntime(), { retryable: false });
    expect(terminal.status).toEqual({
      kind: 'approved_failed_terminal',
      label: '恢复失败，等待人工处理',
    });
  });

  it('maps run_state resumable statuses into deterministic runtime metadata', () => {
    const waitingApproval = applyRunState(createEmptyAssistantRuntime(), {
      run_id: 'run-1',
      status: 'waiting_approval',
      agent: 'executor',
    });
    expect(waitingApproval.status).toEqual({
      kind: 'waiting_approval',
      label: '等待审批',
    });
    expect(waitingApproval.pendingRun).toEqual(expect.objectContaining({
      runId: 'run-1',
      status: 'waiting_approval',
      resumable: true,
    }));

    const retryableFailure = applyRunState(waitingApproval, {
      run_id: 'run-1',
      status: 'resume_failed_retryable',
      agent: 'executor',
    });
    expect(retryableFailure.status).toEqual({
      kind: 'resume_failed_retryable',
      label: '恢复失败，可重试',
    });
    expect(retryableFailure.pendingRun).toEqual(expect.objectContaining({
      runId: 'run-1',
      status: 'resume_failed_retryable',
      resumable: true,
    }));

    const failed = applyRunState(retryableFailure, {
      run_id: 'run-1',
      status: 'failed_runtime',
      agent: 'executor',
    });
    expect(failed.status).toEqual({
      kind: 'failed',
      label: '运行失败',
    });
    expect(failed.pendingRun).toBeUndefined();
  });

  it('maps run_state completed statuses into terminal runtime metadata', () => {
    const waitingApproval = applyRunState(createEmptyAssistantRuntime(), {
      run_id: 'run-1',
      status: 'waiting_approval',
      agent: 'executor',
    });
    expect(waitingApproval.pendingRun).toEqual(expect.objectContaining({
      runId: 'run-1',
      status: 'waiting_approval',
      resumable: true,
    }));

    const completed = applyRunState(waitingApproval, {
      run_id: 'run-1',
      status: 'completed',
      agent: 'executor',
    });
    expect(completed.status).toEqual({
      kind: 'completed',
      label: '已完成',
    });
    expect(completed.pendingRun).toBeUndefined();

    const completedWithToolErrors = applyRunState(waitingApproval, {
      run_id: 'run-1',
      status: 'completed_with_tool_errors',
      agent: 'executor',
    });
    expect(completedWithToolErrors.status).toEqual({
      kind: 'completed',
      label: '已完成',
    });
    expect(completedWithToolErrors.pendingRun).toBeUndefined();
  });

  it('marks resume completion and expiration with dedicated states', () => {
    const resumed = applyRunResumed(createEmptyAssistantRuntime());
    expect(resumed.status).toEqual({
      kind: 'approved_done',
      label: '已批准，恢复完成',
    });

    const done = applyDone(applyRunResuming(createEmptyAssistantRuntime()));
    expect(done.status).toEqual({
      kind: 'approved_done',
      label: '已批准，恢复完成',
    });

    const expired = applyApprovalExpired(createEmptyAssistantRuntime());
    expect(expired.status).toEqual({
      kind: 'expired',
      label: '审批已过期',
    });
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
      kind: 'failed',
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
