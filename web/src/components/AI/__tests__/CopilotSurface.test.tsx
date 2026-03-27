import React from 'react';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import CopilotSurface from '../CopilotSurface';
import { aiApi } from '../../../api/modules/ai';

const mockUseXChat = vi.hoisted(() => vi.fn());
const mockUseXConversations = vi.hoisted(() => vi.fn());
const mockXMarkdown = vi.hoisted(() => vi.fn());
const senderValueRef = vi.hoisted(() => ({ current: '' }));

vi.mock('@ant-design/x', async () => {
  const actual = await vi.importActual<typeof import('@ant-design/x')>('@ant-design/x');
  return {
    ...actual,
    Sender: (props: any) => {
      senderValueRef.current = props.value || '';
      return (
        <button type="button" aria-label="mock-send" onClick={() => props.onSubmit?.(senderValueRef.current)}>
          send
        </button>
      );
    },
  };
});

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
    getRunProjection: vi.fn(async () => ({ data: null })),
    getRunContent: vi.fn(async () => ({ data: null })),
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
    senderValueRef.current = '';
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
    vi.stubGlobal(
      'ResizeObserver',
      class ResizeObserver {
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

  it('renders fallback prompts without calling getScenePrompts', async () => {
    vi.mocked(aiApi.getScenePrompts).mockImplementation(() => {
      throw new Error('getScenePrompts should not be required by CopilotSurface');
    });

    mockUseXChat.mockReturnValue({
      messages: [],
      onRequest: vi.fn(),
      isRequesting: false,
      queueRequest: vi.fn(),
    });

    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    expect(await screen.findByText('诊断集群健康')).toBeInTheDocument();
    expect(aiApi.getScenePrompts).not.toHaveBeenCalled();
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

  it('hydrates assistant history from run projection', async () => {
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
            id: 'msg-1',
            role: 'assistant',
            content: '历史回答',
            status: 'done',
            run_id: 'run-1',
          },
        ],
      },
    } as any);
    vi.mocked(aiApi.getRunProjection).mockResolvedValue({
      data: {
        version: 1,
        run_id: 'run-1',
        session_id: 'sess-1',
        status: 'completed',
        summary: { title: '结论', content_mode: 'inline', content: '已完成诊断' },
        blocks: [],
      },
    } as any);

    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    const hydrated = await capturedDefaultMessages?.({ conversationKey: 'sess-1' });
    expect(hydrated?.[0]?.message?.runtime?.summary?.title).toBe('结论');
    expect(hydrated?.[0]?.message?.runtime?.status).toEqual({
      kind: 'completed',
      label: 'completed',
    });
  });

  it('scrolls to the latest turn when opening a conversation with existing messages', async () => {
    mockUseXChat.mockReturnValue({
      messages: [
        { id: 'u1', status: 'success', message: { role: 'user', content: 'q1' } },
        { id: 'a1', status: 'success', message: { role: 'assistant', content: 'a1' } },
        { id: 'u2', status: 'success', message: { role: 'user', content: 'q2' } },
        { id: 'a2', status: 'success', message: { role: 'assistant', content: 'a2' } },
      ],
      onRequest: vi.fn(),
      isRequesting: false,
      queueRequest: vi.fn(),
    });

    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    const scrollContainer = screen.getByTestId('copilot-scroll-container');
    Object.defineProperty(scrollContainer, 'scrollHeight', { configurable: true, value: 1280 });

    await waitFor(() => {
      expect(scrollToMock).toHaveBeenCalledWith(
        expect.objectContaining({ top: 1280, behavior: 'auto' }),
      );
    });
  });

  it('forces bottom alignment when sending from detached mode', async () => {
    const onRequest = vi.fn();
    mockUseXChat.mockReturnValue({
      messages: [
        { id: 'u1', status: 'success', message: { role: 'user', content: 'q1' } },
        { id: 'a1', status: 'success', message: { role: 'assistant', content: 'a1' } },
      ],
      onRequest,
      isRequesting: false,
      queueRequest: vi.fn(),
    });

    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    const scrollContainer = screen.getByTestId('copilot-scroll-container');
    Object.defineProperty(scrollContainer, 'scrollHeight', { configurable: true, value: 1600 });
    Object.defineProperty(scrollContainer, 'clientHeight', { configurable: true, value: 400 });
    Object.defineProperty(scrollContainer, 'scrollTop', { configurable: true, writable: true, value: 0 });

    scrollContainer.dispatchEvent(new Event('scroll'));
    senderValueRef.current = '检查集群';
    screen.getByRole('button', { name: 'mock-send' }).click();

    await waitFor(() => {
      expect(onRequest).toHaveBeenCalledWith(
        expect.objectContaining({ message: '检查集群' }),
      );
    });

    await waitFor(() => {
      expect(scrollToMock).toHaveBeenCalledWith(
        expect.objectContaining({ top: 1600, behavior: 'auto' }),
      );
    });
  });

  it('loads step content from the owning assistant message runtime', async () => {
    vi.mocked(aiApi.getRunContent).mockResolvedValue({
      data: {
        id: 'content-first',
        run_id: 'run-1',
        session_id: 'sess-1',
        content_kind: 'executor_content',
        encoding: 'text',
        body_text: 'first body',
      },
    } as any);

    mockUseXChat.mockReturnValue({
      messages: [
        {
          id: 'a1',
          status: 'success',
          message: {
            role: 'assistant',
            content: 'first reply',
            runtime: {
              activities: [],
              plan: {
                steps: [
                  { id: 'step-1', title: 'first-step', status: 'done', loaded: false, sourceBlockIndex: 0 },
                ],
              },
              _executorBlocks: [
                {
                  id: 'block-1',
                  items: [{ type: 'content', content_id: 'content-first' }],
                },
              ],
            },
          },
        },
        {
          id: 'a2',
          status: 'success',
          message: {
            role: 'assistant',
            content: 'second reply',
            runtime: {
              activities: [],
              plan: {
                steps: [
                  { id: 'step-2', title: 'second-step', status: 'done', loaded: false, sourceBlockIndex: 0 },
                ],
              },
              _executorBlocks: [
                {
                  id: 'block-2',
                  items: [{ type: 'content', content_id: 'content-second' }],
                },
              ],
            },
          },
        },
      ],
      onRequest: vi.fn(),
      isRequesting: false,
      queueRequest: vi.fn(),
    });

    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
        <CopilotSurface open onClose={() => undefined} />
      </MemoryRouter>,
    );

    screen.getByRole('button', { name: /first-step/ }).click();

    await waitFor(() => {
      expect(aiApi.getRunContent).toHaveBeenCalledWith('content-first');
    });
  });

});
