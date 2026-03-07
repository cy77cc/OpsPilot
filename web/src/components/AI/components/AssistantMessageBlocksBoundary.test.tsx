import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('@ant-design/x', async () => {
  const actual = await vi.importActual<typeof import('@ant-design/x')>('@ant-design/x');

  return {
    ...actual,
    CodeHighlighter: () => {
      throw new Error('code renderer failed');
    },
  };
});

import { AssistantMessageBlocks } from './AssistantMessageBlocks';

describe('AssistantMessageBlocks', () => {
  it('falls back to plain code content when the rich code renderer fails', () => {
    render(
      <AssistantMessageBlocks
        blocks={[
          {
            id: 'markdown-1',
            type: 'markdown',
            content: '```ts\nconst value = 1;\n```',
          },
        ]}
      />,
    );

    expect(screen.getByText(/const value = 1;/)).toBeInTheDocument();
  });
});
