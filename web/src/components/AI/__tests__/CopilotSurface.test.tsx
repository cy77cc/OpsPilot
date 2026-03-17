import React from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CopilotSurface from '../CopilotSurface';

const mockUseXChat = vi.hoisted(() => vi.fn());
const mockUseXConversations = vi.hoisted(() => vi.fn());
const mockXMarkdown = vi.hoisted(() => vi.fn());

vi.mock('@ant-design/x-sdk', async () => {
  const actual = await vi.importActual<typeof import('@ant-design/x-sdk')>('@ant-design/x-sdk');
  return {
    ...actual,
    useXChat: mockUseXChat,
    useXConversations: mockUseXConversations,
  };
});

vi.mock('@ant-design/x-markdown', () => ({
  default: (props: any) => {
    mockXMarkdown(props);
    return <div data-testid="x-markdown">{props.content}</div>;
  },
}));

vi.mock('../../../api/modules/ai', () => ({
  aiApi: {
    getSessions: vi.fn(async () => ({ data: [] })),
    getScenePrompts: vi.fn(async () => ({ data: { prompts: [] } })),
    getSession: vi.fn(async () => ({ data: { messages: [] } })),
    createSession: vi.fn(),
    chatStream: vi.fn(),
  },
}));

describe('CopilotSurface XMarkdown streaming', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal(
      'IntersectionObserver',
      class IntersectionObserver {
        observe() {}
        unobserve() {}
        disconnect() {}
      },
    );
    mockUseXConversations.mockReturnValue({
      conversations: [{ key: 'sess-1', label: 'Session 1' }],
      activeConversationKey: 'sess-1',
      setActiveConversationKey: vi.fn(),
      addConversation: vi.fn(),
      setConversation: vi.fn(),
      setConversations: vi.fn(),
      getConversation: vi.fn(),
    });
  });

  it('passes XMarkdown streaming props while assistant content is updating', () => {
    mockUseXChat.mockReturnValue({
      messages: [
        {
          id: 'assistant-1',
          status: 'updating',
          message: { role: 'assistant', content: 'hello' },
        },
      ],
      onRequest: vi.fn(),
      isRequesting: true,
      queueRequest: vi.fn(),
    });

    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId('x-markdown')).toHaveTextContent('hello');
    expect(mockXMarkdown).toHaveBeenCalledWith(
      expect.objectContaining({
        content: 'hello',
        streaming: expect.objectContaining({
          hasNextChunk: true,
          enableAnimation: true,
        }),
      }),
    );
  });

  it('marks XMarkdown stream as done after assistant message succeeds', () => {
    mockUseXChat.mockReturnValue({
      messages: [
        {
          id: 'assistant-1',
          status: 'success',
          message: { role: 'assistant', content: 'hello' },
        },
      ],
      onRequest: vi.fn(),
      isRequesting: false,
      queueRequest: vi.fn(),
    });

    render(
      <MemoryRouter initialEntries={['/services/payment-api']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    expect(mockXMarkdown).toHaveBeenCalledWith(
      expect.objectContaining({
        streaming: expect.objectContaining({
          hasNextChunk: false,
          enableAnimation: true,
        }),
      }),
    );
  });
});
