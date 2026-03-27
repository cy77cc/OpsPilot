import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
  hydrateAssistantHistoryFromProjection,
  isProjectionHydrationPending,
  loadRunContent,
  loadRunProjection,
  loadStepContent,
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
    vi.resetAllMocks();
    resetHistoryProjectionCache();
  });

  it('loads run projection', async () => {
    (aiApi.getRunProjection as any).mockResolvedValue({
      data: { run_id: 'run-1', session_id: 'sess-1', version: 1, status: 'completed', blocks: [] },
    });
    const result = await loadRunProjection('run-1');
    expect(result?.run_id).toBe('run-1');
  });

  it('loads historical blocks incrementally via cursor', async () => {
    (aiApi.getRunProjection as any)
      .mockResolvedValueOnce({
        data: {
          version: 1,
          run_id: 'run-1',
          session_id: 'sess-1',
          status: 'completed',
          summary: { title: '结论', content_mode: 'inline', content: '已恢复' },
          blocks: [
            {
              id: 'block_plan_1',
              type: 'plan',
              title: '处理计划',
              steps: ['检查节点状态'],
            },
          ],
          has_more: true,
          next_cursor: 'cursor-1',
        },
      })
      .mockResolvedValueOnce({
        data: {
          version: 1,
          run_id: 'run-1',
          session_id: 'sess-1',
          status: 'completed',
          summary: { title: '结论', content_mode: 'inline', content: '已恢复' },
          blocks: [
            {
              id: 'block_executor_2',
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
          has_more: false,
        },
      });

    const result = await loadRunProjection('run-1');

    expect(aiApi.getRunProjection).toHaveBeenCalledTimes(2);
    expect(aiApi.getRunProjection).toHaveBeenNthCalledWith(1, 'run-1', { cursor: '', limit: 20 });
    expect(aiApi.getRunProjection).toHaveBeenNthCalledWith(2, 'run-1', { cursor: 'cursor-1', limit: 20 });
    expect(result?.blocks.map((block) => block.id)).toEqual(['block_plan_1', 'block_executor_2']);
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
    expect(hydrated.runtime).toMatchObject({
      activities: [],
      summary: {
        title: '结论',
      },
      status: {
        kind: 'completed',
        label: 'completed',
      },
    });
    expect(hydrated.runtime?._executorBlocks).toEqual([]);
  });

  it('normalizes escaped line breaks in projection summary content', async () => {
    (aiApi.getRunProjection as any).mockResolvedValue({
      data: {
        version: 1,
        run_id: 'run-1',
        session_id: 'sess-1',
        status: 'completed',
        summary: { title: '结论', content_mode: 'inline', content: '## 标题\\n\\n正文' },
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

    expect(hydrated.content).toBe('## 标题\n\n正文');
  });

  it('falls back to persisted assistant body when projection summary is missing', async () => {
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

    expect(hydrated.content).toBe('历史回答');
    expect(hydrated.runtime).toEqual(expect.objectContaining({
      activities: [],
      status: {
        kind: 'completed',
        label: 'completed',
      },
    }));
  });

  it('falls back to persisted assistant body when projection is missing for failed run', async () => {
    (aiApi.getRunProjection as any).mockRejectedValue(new Error('projection unavailable'));

    const hydrated = await hydrateAssistantHistoryFromProjection({
      id: 'msg-1',
      role: 'assistant',
      content: 'partial answer',
      error_message: '生成中断，请稍后重试。',
      run_id: 'run-1',
      timestamp: '',
    } as any);

    expect(hydrated.content).toBe('partial answer');
    expect(hydrated.runtime).toEqual({
      activities: [],
      status: {
        kind: 'error',
        label: '生成中断，请稍后重试。',
      },
    });
  });

  it.each([
    ['waiting_approval', '等待审批', 'waiting_approval'],
    ['resuming', '恢复中', 'resuming'],
    ['running', '执行中', 'streaming'],
    ['resume_failed_retryable', '恢复失败，可重试', 'resume_failed_retryable'],
  ] as const)(
    'keeps resumable %s runs on the pending label when projection is missing',
    async (status, label, kind) => {
      (aiApi.getRunProjection as any).mockRejectedValue(new Error('projection unavailable'));

      const hydrated = await hydrateAssistantHistoryFromProjection({
        id: 'msg-1',
        role: 'assistant',
        content: '',
        resumable: true,
        status,
        run_id: 'run-1',
        timestamp: '',
      } as any);

      expect(hydrated.content).toBe(label);
      expect(hydrated.content).not.toBe(PROJECTION_UNRECOVERABLE_PLACEHOLDER);
      expect(hydrated.runtime).toEqual({
        activities: [],
        status: {
          kind,
          label,
        },
      });
    },
  );

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

  it('keeps plan and error status lightweight when hydrating historical projection', async () => {
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

    expect(hydrated.runtime?.activities).toEqual([]);
    expect(hydrated.runtime?.plan?.steps.map((step) => step.title)).toEqual(['修正计划']);
    expect(hydrated.runtime?.status).toEqual({
      kind: 'error',
      label: 'failed_runtime',
    });
  });

  it('keeps historical step titles visible before lazy content loads', async () => {
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
    const hydrated = await hydrateAssistantHistoryFromProjection({
      id: 'msg-1',
      role: 'assistant',
      run_id: 'run-1',
      timestamp: '',
    } as any);

    expect(hydrated.runtime?.plan?.steps).toEqual([
      expect.objectContaining({
        title: '检查节点状态',
        loaded: false,
      }),
      expect.objectContaining({
        title: '汇总结果',
        loaded: false,
      }),
    ]);
    expect(hydrated.runtime?.plan?.steps.map((step) => step.title)).not.toContain('执行过程');
    expect(hydrated.runtime?._executorBlocks).toHaveLength(1);
  });

  it('keeps historical replan steps in lazy runtime instead of hydrating content eagerly', async () => {
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
    const hydrated = await hydrateAssistantHistoryFromProjection({
      id: 'msg-1',
      role: 'assistant',
      run_id: 'run-1',
      timestamp: '',
    } as any);

    expect(hydrated.runtime?.plan?.steps).toEqual([
      expect.objectContaining({ title: '收集上下文', sourceBlockIndex: 0 }),
      expect.objectContaining({ title: '执行检查', loaded: false, sourceBlockIndex: undefined }),
      expect.objectContaining({ title: '汇总结果', loaded: false }),
    ]);
  });

  it('renders only the real step titles returned by the latest historical replan block', async () => {
    (aiApi.getRunProjection as any).mockResolvedValue({
      data: {
        version: 1,
        run_id: 'run-1',
        session_id: 'sess-1',
        status: 'completed',
        summary: { title: '结论', content_mode: 'inline', content: '最终结论' },
        blocks: [
          {
            id: 'replan-1',
            type: 'replan',
            title: '重新规划',
            steps: ['执行检查', '汇总结果'],
            data: { completed: 2, iteration: 1, is_final: true },
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

    const hydrated = await hydrateAssistantHistoryFromProjection({
      id: 'msg-1',
      role: 'assistant',
      run_id: 'run-1',
      timestamp: '',
    } as any);

    expect(hydrated.runtime?.plan?.steps.map((step) => step.title)).toEqual(['执行检查', '汇总结果']);
    expect(hydrated.runtime?.plan?.activeStepIndex).toBeUndefined();
  });

  it('keeps real historical steps visible even when failed projection has no summary', async () => {
    (aiApi.getRunProjection as any).mockResolvedValue({
      data: {
        version: 1,
        run_id: 'run-1',
        session_id: 'sess-1',
        status: 'failed_runtime',
        blocks: [
          {
            id: 'replan-1',
            type: 'replan',
            title: '重新规划',
            steps: ['检查内存指标', '分析高占用进程', '输出诊断结论'],
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
          {
            id: 'error-1',
            type: 'error',
            title: '执行错误',
            data: { message: '生成中断，请稍后重试', code: 'EXECUTION_FAILED' },
          },
        ],
      },
    });

    const hydrated = await hydrateAssistantHistoryFromProjection({
      id: 'msg-1',
      role: 'assistant',
      content: '生成中断，请稍后重试。',
      run_id: 'run-1',
      timestamp: '',
    } as any);

    expect(hydrated.content).toBe('生成中断，请稍后重试。');
    expect(hydrated.runtime?.plan?.steps.map((step) => step.title)).toEqual([
      '检查内存指标',
      '分析高占用进程',
      '输出诊断结论',
    ]);
    expect(hydrated.runtime?.status).toEqual({
      kind: 'error',
      label: 'failed_runtime',
    });
  });

  it('loads step content on demand from a slim executor block', async () => {
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

    const loaded = await loadStepContent(
      {
        id: 'executor-1',
        items: [
          { type: 'content', content_id: 'executor-content-1' },
          {
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
      0,
    );

    expect(loaded.content).toBe('正在检查 node-1');
    expect(loaded.segments).toEqual([
      { type: 'text', text: '正在检查 node-1' },
      { type: 'tool_ref', callId: 'call-1' },
    ]);
    expect(loaded.activities).toEqual([
      expect.objectContaining({
        id: 'call-1',
        label: 'kubectl_describe',
        stepIndex: 0,
      }),
    ]);
  });
});
