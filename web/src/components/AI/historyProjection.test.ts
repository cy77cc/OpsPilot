import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
  hydrateAssistantHistoryFromProjection,
  isProjectionHydrationPending,
  loadRunContent,
  loadRunProjection,
  PROJECTION_MISSING_SUMMARY_LABEL,
  PROJECTION_UNRECOVERABLE_PLACEHOLDER,
  resetHistoryProjectionCache,
} from './historyProjection';
import { aiApi } from '../../api/modules/ai';

vi.mock('../../api/modules/ai', () => ({
  aiApi: {
    getRunProjection: vi.fn(),
    getRunContent: vi.fn(),
  },
}));

describe('historyProjection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetHistoryProjectionCache();
  });

  it('loads run projection', async () => {
    (aiApi.getRunProjection as any).mockResolvedValue({
      data: { run_id: 'run-1', session_id: 'sess-1', version: 1, status: 'completed', blocks: [] },
    });
    const result = await loadRunProjection('run-1');
    expect(result?.run_id).toBe('run-1');
  });

  it('loads run content', async () => {
    (aiApi.getRunContent as any).mockResolvedValue({
      data: { id: 'content-1', run_id: 'run-1', session_id: 'sess-1', content_kind: 'executor_content', encoding: 'text', body_text: 'hello' },
    });
    const result = await loadRunContent('content-1');
    expect(result?.body_text).toBe('hello');
  });

  it('hydrates assistant body from projection summary only', async () => {
    (aiApi.getRunProjection as any).mockResolvedValue({
      data: {
        version: 1,
        run_id: 'run-1',
        session_id: 'sess-1',
        status: 'completed',
        summary: { title: '结论', content_mode: 'inline', content: '已恢复' },
        blocks: [],
      },
    });

    const hydrated = await hydrateAssistantHistoryFromProjection({
      id: 'msg-1',
      role: 'assistant',
      content: '历史回答',
      run_id: 'run-1',
      timestamp: '',
    } as any);

    expect(hydrated.content).toBe('已恢复');
    expect(hydrated.runtime).toEqual({
      activities: [],
      summary: {
        title: '结论',
      },
      status: {
        kind: 'completed',
        label: 'completed',
      },
    });
  });

  it('returns an error placeholder when projection summary is missing', async () => {
    (aiApi.getRunProjection as any).mockResolvedValue({
      data: {
        version: 1,
        run_id: 'run-1',
        session_id: 'sess-1',
        status: 'completed',
        blocks: [],
      },
    });

    const hydrated = await hydrateAssistantHistoryFromProjection({
      id: 'msg-1',
      role: 'assistant',
      content: '历史回答',
      run_id: 'run-1',
      timestamp: '',
    } as any);

    expect(hydrated.content).toBe(PROJECTION_UNRECOVERABLE_PLACEHOLDER);
    expect(hydrated.runtime).toEqual({
      activities: [],
      status: {
        kind: 'error',
        label: PROJECTION_MISSING_SUMMARY_LABEL,
      },
    });
  });

  it('recognizes the transient projection-missing hydration state', () => {
    expect(isProjectionHydrationPending({
      id: 'msg-1',
      role: 'assistant',
      content: PROJECTION_UNRECOVERABLE_PLACEHOLDER,
      runtime: {
        activities: [],
        status: {
          kind: 'error',
          label: PROJECTION_MISSING_SUMMARY_LABEL,
        },
      },
    })).toBe(true);

    expect(isProjectionHydrationPending({
      id: 'msg-1',
      role: 'assistant',
      content: '已恢复',
      runtime: {
        activities: [],
        status: {
          kind: 'completed',
          label: 'completed',
        },
      },
    })).toBe(false);
  });

  it('retries projection fetch after a transient failure', async () => {
    (aiApi.getRunProjection as any)
      .mockRejectedValueOnce(new Error('temporary failure'))
      .mockResolvedValueOnce({
        data: {
          version: 1,
          run_id: 'run-1',
          session_id: 'sess-1',
          status: 'completed',
          summary: { title: '结论', content_mode: 'inline', content: '已恢复' },
          blocks: [],
        },
      });

    const first = await loadRunProjection('run-1');
    const second = await loadRunProjection('run-1');

    expect(first).toBeNull();
    expect(second?.summary?.content).toBe('已恢复');
    expect(aiApi.getRunProjection).toHaveBeenCalledTimes(2);
  });

  it('hydrates handoff, replan and error details from projection', async () => {
    (aiApi.getRunProjection as any).mockResolvedValue({
      data: {
        version: 1,
        run_id: 'run-1',
        session_id: 'sess-1',
        status: 'failed_runtime',
        summary: { title: '结论', content_mode: 'inline', content: '已完成诊断' },
        blocks: [
          {
            id: 'handoff-1',
            type: 'agent_handoff',
            title: '任务转交',
            data: { from: 'OpsPilotAgent', to: 'DiagnosisAgent', intent: 'diagnosis' },
          },
          {
            id: 'plan-1',
            type: 'plan',
            title: '处理计划',
            steps: ['初始计划'],
          },
          {
            id: 'replan-1',
            type: 'replan',
            title: '重新规划',
            steps: ['修正计划'],
          },
          {
            id: 'error-1',
            type: 'error',
            title: '执行错误',
            data: { message: 'stream failed', code: 'stream_failed' },
          },
        ],
      },
    });

    const hydrated = await hydrateAssistantHistoryFromProjection({
      id: 'msg-1',
      role: 'assistant',
      content: '历史回答',
      run_id: 'run-1',
      timestamp: '',
    } as any);

    expect(hydrated.runtime?.activities).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ kind: 'agent_handoff', label: 'DiagnosisAgent' }),
        expect.objectContaining({ kind: 'replan', label: '重新规划' }),
        expect.objectContaining({ kind: 'error', detail: 'stream failed' }),
      ]),
    );
    expect(hydrated.runtime?.status).toEqual({
      kind: 'error',
      label: 'failed_runtime',
    });
  });

  it('maps executor blocks onto planned steps instead of appending 执行过程 as a new step', async () => {
    (aiApi.getRunProjection as any).mockResolvedValue({
      data: {
        version: 1,
        run_id: 'run-1',
        session_id: 'sess-1',
        status: 'completed',
        summary: { title: '结论', content_mode: 'inline', content: '最终结论' },
        blocks: [
          {
            id: 'plan-1',
            type: 'plan',
            title: '处理计划',
            steps: ['检查节点状态', '汇总结果'],
          },
          {
            id: 'executor-1',
            type: 'executor',
            title: '执行过程',
            items: [
              {
                id: 'content-1',
                type: 'content',
                content_id: 'executor-content-1',
              },
              {
                id: 'tool-1',
                type: 'tool_call',
                tool_call_id: 'call-1',
                tool_name: 'kubectl_describe',
                arguments: { node: 'node-1' },
                result: {
                  status: 'done',
                  preview: 'ok',
                  result_content_id: 'tool-result-1',
                },
              },
            ],
          },
        ],
      },
    });
    (aiApi.getRunContent as any)
      .mockResolvedValueOnce({
        data: {
          id: 'executor-content-1',
          run_id: 'run-1',
          session_id: 'sess-1',
          content_kind: 'executor_content',
          encoding: 'text',
          body_text: '正在检查 node-1',
        },
      })
      .mockResolvedValueOnce({
        data: {
          id: 'tool-result-1',
          run_id: 'run-1',
          session_id: 'sess-1',
          content_kind: 'tool_result',
          encoding: 'text',
          body_text: 'ok',
        },
      });

    const hydrated = await hydrateAssistantHistoryFromProjection({
      id: 'msg-1',
      role: 'assistant',
      run_id: 'run-1',
      timestamp: '',
    } as any);

    expect(hydrated.runtime?.plan?.steps).toEqual([
      expect.objectContaining({
        title: '检查节点状态',
        content: '正在检查 node-1',
      }),
      expect.objectContaining({
        title: '汇总结果',
      }),
    ]);
    expect(hydrated.runtime?.plan?.steps.map((step) => step.title)).not.toContain('执行过程');
    expect(hydrated.runtime?.activities).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          id: 'call-1',
          label: 'kubectl_describe',
          stepIndex: 0,
        }),
      ]),
    );
  });

  it('preserves completed steps before a replan and attaches executor content to the next remaining step', async () => {
    (aiApi.getRunProjection as any).mockResolvedValue({
      data: {
        version: 1,
        run_id: 'run-1',
        session_id: 'sess-1',
        status: 'completed',
        summary: { title: '结论', content_mode: 'inline', content: '最终结论' },
        blocks: [
          {
            id: 'plan-1',
            type: 'plan',
            title: '处理计划',
            steps: ['收集上下文', '执行检查', '汇总结果'],
          },
          {
            id: 'replan-1',
            type: 'replan',
            title: '重新规划',
            steps: ['执行检查', '汇总结果'],
            data: { completed: 1, iteration: 1, is_final: false },
          },
          {
            id: 'executor-1',
            type: 'executor',
            title: '执行过程',
            items: [
              {
                id: 'content-1',
                type: 'content',
                content_id: 'executor-content-1',
              },
            ],
          },
        ],
      },
    });
    (aiApi.getRunContent as any).mockResolvedValue({
      data: {
        id: 'executor-content-1',
        run_id: 'run-1',
        session_id: 'sess-1',
        content_kind: 'executor_content',
        encoding: 'text',
        body_text: '正在执行检查',
      },
    });

    const hydrated = await hydrateAssistantHistoryFromProjection({
      id: 'msg-1',
      role: 'assistant',
      run_id: 'run-1',
      timestamp: '',
    } as any);

    expect(hydrated.runtime?.plan?.steps).toEqual([
      expect.objectContaining({ title: '收集上下文' }),
      expect.objectContaining({ title: '执行检查', content: '正在执行检查' }),
      expect.objectContaining({ title: '汇总结果' }),
    ]);
  });
});
