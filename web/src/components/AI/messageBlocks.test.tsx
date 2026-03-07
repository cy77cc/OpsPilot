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
});
