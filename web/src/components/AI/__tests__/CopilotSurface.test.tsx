import React from 'react';
import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import CopilotSurface from '../CopilotSurface';
import { aiApi } from '../../../api/modules/ai';

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
  const scrollToMock = vi.fn();

  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
      configurable: true,
      writable: true,
      value: scrollToMock,
    });
    Object.defineProperty(HTMLElement.prototype, 'offsetTop', {
      configurable: true,
      get() {
        return Number((this as HTMLElement).dataset.offsetTop || 0);
      },
    });
    Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {
      configurable: true,
      get() {
        return Number((this as HTMLElement).dataset.offsetHeight || 0);
      },
    });
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

  it('renders assistant runtime details inside the assistant reply surface', () => {
    mockUseXChat.mockReturnValue({
      messages: [
        {
          id: 'assistant-1',
          status: 'updating',
          message: {
            role: 'assistant',
            content: 'hello',
            runtime: {
              phase: 'planning',
              phaseLabel: '正在规划',
              activities: [],
              status: { kind: 'streaming', label: '持续生成中' },
            },
          },
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

    expect(screen.getByText('正在规划')).toBeInTheDocument();
    expect(screen.getByText('持续生成中')).toBeInTheDocument();
    expect(screen.getAllByTestId('x-markdown').some((node) => node.textContent?.includes('hello'))).toBe(true);
  });

  it('preserves persisted runtime in defaultMessages hydration', async () => {
    let capturedDefaultMessages: ((args: { conversationKey?: string }) => Promise<any[]>) | undefined;
    mockUseXChat.mockImplementation((config: any) => {
      capturedDefaultMessages = config.defaultMessages;
      return {
        messages: [],
        onRequest: vi.fn(),
        isRequesting: false,
        queueRequest: vi.fn(),
      };
    });

    vi.mocked(aiApi.getSession).mockResolvedValue({
      data: {
        messages: [
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
          },
        ],
      },
    } as any);

    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    const hydrated = await capturedDefaultMessages?.({ conversationKey: 'sess-1' });
    expect(hydrated?.[0]?.message?.runtime?.phaseLabel).toBe('已完成诊断');
    expect(hydrated?.[0]?.message?.runtime?.status).toEqual({
      kind: 'completed',
      label: '已生成',
    });
  });

});
