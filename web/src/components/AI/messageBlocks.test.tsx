import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { normalizeAssistantMessage, normalizeTurnBlocks, type AssistantMessageBlock } from './messageBlocks';
import { AssistantMessageBlocks } from './components/AssistantMessageBlocks';

describe('normalizeAssistantMessage', () => {
  it('normalizes thinking, markdown, and recommendations into render blocks', () => {
    const blocks = normalizeAssistantMessage({
      content: '```ts\nconst value = 1;\n```',
      thinking: 'analyzing',
      recommendations: [
        {
          id: 'rec-1',
          type: 'followup',
          title: 'Next step',
          content: 'Inspect host state',
          followup_prompt: 'inspect host state',
          relevance: 0.9,
        },
      ],
    });

    expect(blocks.map((block) => block.type)).toEqual(['thinking', 'markdown', 'recommendations']);
  });

  it('includes raw evidence as a dedicated render block', () => {
    const blocks = normalizeAssistantMessage({
      content: 'final answer',
      rawEvidence: ['step output 1', 'step output 2'],
    });

    expect(blocks.map((block) => block.type)).toEqual(['markdown', 'raw_evidence']);
  });
});

describe('AssistantMessageBlocks', () => {
  it('normalizes structured turn blocks into renderer blocks', () => {
    const blocks = normalizeTurnBlocks([
      {
        id: 'turn-text',
        type: 'text',
        position: 2,
        content: 'final answer',
        streaming: true,
      },
      {
        id: 'turn-status',
        type: 'status',
        position: 1,
        title: '执行中',
        content: '正在调用工具',
      },
    ]);

    expect(blocks.map((block) => block.type)).toEqual(['status', 'markdown']);
    expect(blocks[1]).toMatchObject({ type: 'markdown', streaming: true });
  });

  it('falls back safely for unsupported block types', () => {
    const blocks = [
      {
        type: 'fallback',
        id: 'fallback-1',
        content: 'raw fallback content',
      } satisfies AssistantMessageBlock,
    ];

    render(<AssistantMessageBlocks blocks={blocks} />);

    expect(screen.getByText('raw fallback content')).toBeInTheDocument();
  });

  it('renders approval and tool interactions as accessible controls', () => {
    render(
      <AssistantMessageBlocks
        blocks={[
          {
            id: 'tool-1',
            type: 'tool',
            status: 'success',
            payload: {
              tool_name: 'kubectl_scale',
              result: { ok: true, latency_ms: 1000 },
            },
          },
          {
            id: 'approval-1',
            type: 'approval',
            payload: {
              title: '需要确认',
              summary: '扩容 deployment',
              risk: 'high',
            },
          },
        ]}
      />,
    );

    expect(screen.getByRole('button', { name: /kubectl scale 详情/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /需要确认，确认执行/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /需要确认，取消/i })).toBeInTheDocument();
  });

  it('renders a lightweight retry state when approval submission fails', () => {
    render(
      <AssistantMessageBlocks
        blocks={[
          {
            id: 'approval-failed',
            type: 'approval',
            payload: {
              title: '需要确认',
              summary: '扩容 deployment',
              risk: 'high',
              status: 'failed',
              error_message: '审批提交失败',
            },
          },
        ]}
      />,
    );

    expect(screen.getAllByText('审批提交失败').length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: '重试' })).toBeInTheDocument();
  });
});
