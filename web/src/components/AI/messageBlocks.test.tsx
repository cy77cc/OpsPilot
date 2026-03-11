import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { normalizeAssistantMessage, type AssistantMessageBlock } from './messageBlocks';
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

  it('includes structured summary output as a dedicated render block', () => {
    const blocks = normalizeAssistantMessage({
      content: 'final answer',
      summaryOutput: {
        headline: 'payment-api 当前运行正常。',
        conclusion: '未观察到明显异常。',
        key_findings: ['最近一次检查返回状态正常。'],
      },
    });

    expect(blocks.map((block) => block.type)).toEqual(['markdown', 'summary_output']);
  });
});

describe('AssistantMessageBlocks', () => {
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

  it('renders structured summary output without flattening it into markdown', () => {
    const blocks = normalizeAssistantMessage({
      summaryOutput: {
        headline: 'payment-api 当前运行正常。',
        conclusion: '未观察到明显异常。',
        narrative: '从最近执行结果看，服务响应稳定。',
        key_findings: ['最近一次检查返回状态正常。'],
      },
    });

    render(<AssistantMessageBlocks blocks={blocks} />);

    expect(screen.getByText('结构化结论')).toBeInTheDocument();
    expect(screen.getByText('payment-api 当前运行正常。')).toBeInTheDocument();
    expect(screen.getByText('从最近执行结果看，服务响应稳定。')).toBeInTheDocument();
    expect(screen.getByText('最近一次检查返回状态正常。')).toBeInTheDocument();
  });
});
