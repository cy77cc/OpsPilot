import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
  hydrateAssistantHistoryFromProjection,
  loadRunContent,
  loadRunProjection,
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

    expect(hydrated.content).toBe('回答内容不可恢复');
    expect(hydrated.runtime).toEqual({
      activities: [],
      status: {
        kind: 'error',
        label: 'projection missing summary',
      },
    });
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
});
