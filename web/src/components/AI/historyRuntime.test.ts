import { describe, expect, it } from 'vitest';
import { hydrateAssistantHistoryMessage } from './historyRuntime';

describe('hydrateAssistantHistoryMessage', () => {
  it('preserves persisted assistant runtime from session history', () => {
    const hydrated = hydrateAssistantHistoryMessage(
      {
        role: 'assistant',
        content: '历史回答',
        status: 'done',
        runtime: {
          phase: 'completed',
          phaseLabel: '已完成诊断',
          activities: [],
          status: { kind: 'completed', label: '已生成' },
        },
      } as any,
      [],
      [],
    );

    expect(hydrated.runtime?.phaseLabel).toBe('已完成诊断');
    expect(hydrated.runtime?.status).toEqual({
      kind: 'completed',
      label: '已生成',
    });
  });

  it('synthesizes runtime from replay turns and blocks when persisted runtime is absent', () => {
    const hydrated = hydrateAssistantHistoryMessage(
      {
        role: 'assistant',
        content: '历史回答',
        status: 'done',
      } as any,
      [
        {
          role: 'assistant',
          blocks: [
            { blockType: 'phase', contentJson: { phase: 'completed', phaseLabel: '已完成诊断' } },
            { blockType: 'summary', contentJson: { title: '巡检摘要', items: [{ label: '高风险', value: '1' }] } },
          ],
        },
      ] as any,
      [],
    );

    expect(hydrated.runtime?.phaseLabel).toBe('已完成诊断');
    expect(hydrated.runtime?.summary?.title).toBe('巡检摘要');
  });

  it('falls back to markdown-only history when runtime and replay blocks are absent', () => {
    const hydrated = hydrateAssistantHistoryMessage(
      { role: 'assistant', content: '纯文本历史', status: 'done' } as any,
      [],
      [],
    );

    expect(hydrated.runtime).toBeUndefined();
    expect(hydrated.content).toBe('纯文本历史');
  });

  it('maps historical done status to completed footer state', () => {
    const hydrated = hydrateAssistantHistoryMessage(
      {
        role: 'assistant',
        content: '历史回答',
        status: 'done',
        runtime: { activities: [] },
      } as any,
      [],
      [],
    );

    expect(hydrated.runtime?.status).toEqual({
      kind: 'completed',
      label: '已生成',
    });
  });
});
