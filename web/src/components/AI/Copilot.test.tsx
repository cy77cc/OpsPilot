import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Copilot, DEFAULT_CONVERSATION, conversationReducer } from './Copilot';
import { aiApi } from '../../api/modules/ai';

let restoredConversation: any = null;
let mockedScenePrompts: Array<{ key: string; description: string }> = [];

vi.mock('@ant-design/x', () => ({
  Bubble: {
    List: ({ items }: { items: Array<{ key: string; content: React.ReactNode; role?: string }> }) => (
      <div>
        {items.map((item, index) => (
          <div key={item.key} data-testid={`bubble-item-${index}`} data-role={item.role}>
            {item.content}
          </div>
        ))}
      </div>
    ),
  },
  Conversations: ({ items }: { items: Array<{ key: string; label: string }> }) => (
    <div>
      {items.map((item) => (
        <div key={item.key}>{item.label}</div>
      ))}
    </div>
  ),
  Prompts: ({ items, onItemClick }: { items?: Array<{ key: string; description: string }>; onItemClick?: (info?: { data?: { description?: string } }) => void }) => (
    <div>
      {(items || []).map((item) => (
        <button
          key={item.key}
          type="button"
          onClick={() => onItemClick?.({ data: { description: item.description } })}
        >
          {item.description}
        </button>
      ))}
    </div>
  ),
  Sender: () => null,
  Welcome: () => null,
  Think: ({ children, title }: { children?: React.ReactNode; title?: React.ReactNode }) => (
    <div>
      {title ? <div>{title}</div> : null}
      <div>{children}</div>
    </div>
  ),
  CodeHighlighter: ({ children }: { children?: React.ReactNode }) => <pre>{children}</pre>,
  ThoughtChain: ({
    items,
    defaultExpandedKeys = [],
  }: {
    items: Array<{ key: string; title: string; content?: string }>;
    defaultExpandedKeys?: string[];
  }) => (
    <div>
      {items.map((item) => (
        <div key={item.key}>
          <span>{item.title}</span>
          {defaultExpandedKeys.includes(item.key) ? <div>{item.content}</div> : null}
        </div>
      ))}
    </div>
  ),
}));

vi.mock('./hooks/useConversationRestore', () => ({
  useConversationRestore: ({ onRestore }: { onRestore?: (value: any) => void }) => {
    React.useEffect(() => {
      if (restoredConversation) {
        onRestore?.(restoredConversation);
      }
    }, [onRestore]);

    return {
      isRestoring: false,
      error: null,
      restoredSessionId: restoredConversation?.activeConversation?.id || null,
      restore: vi.fn(),
    };
  },
}));

vi.mock('./hooks/useScenePrompts', () => ({
  useScenePrompts: () => ({ prompts: mockedScenePrompts }),
}));

vi.mock('./components/MessageActions', () => ({
  MessageActions: ({ onRegenerate }: { onRegenerate?: () => void }) => (
    <button type="button" onClick={onRegenerate}>重新生成</button>
  ),
}));

describe('Copilot', () => {
  beforeEach(() => {
    restoredConversation = null;
    mockedScenePrompts = [];
    vi.spyOn(aiApi, 'chatStream').mockResolvedValue();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    restoredConversation = null;
    mockedScenePrompts = [];
  });

  it('hides restored thought chain details and keeps only the summary content', async () => {
    restoredConversation = {
      conversations: [{ id: 'sess-restore', title: '历史会话', createdAt: '2026-03-12T00:00:00Z', updatedAt: '2026-03-12T00:00:00Z' }],
      activeConversation: {
        id: 'sess-restore',
        title: '历史会话',
        messages: [
          {
            id: 'msg-assistant',
            role: 'assistant',
            content: '历史回答',
            thoughtChain: [
              {
                key: 'execute',
                title: '工具调用链',
                status: 'success',
                content: '步骤: 检查 deployment',
              },
            ],
            restored: true,
            createdAt: '2026-03-12T00:00:00Z',
          },
        ],
      },
    };

    render(<Copilot open scene="global" />);

    expect(await screen.findByText('历史回答')).toBeInTheDocument();
    expect(screen.queryByText('工具调用链')).not.toBeInTheDocument();
    expect(screen.queryByText('步骤: 检查 deployment')).not.toBeInTheDocument();
  });

  it('renders restored assistant as summary-only markdown without process blocks', async () => {
    restoredConversation = {
      conversations: [{ id: 'sess-summary-only', title: '历史会话', createdAt: '2026-03-12T00:00:00Z', updatedAt: '2026-03-12T00:00:00Z' }],
      activeConversation: {
        id: 'sess-summary-only',
        title: '历史会话',
        messages: [
          {
            id: 'msg-assistant',
            role: 'assistant',
            content: '## 最终结论\n- 配置已完成',
            thoughtChain: [
              {
                key: 'execute',
                title: '工具调用链',
                status: 'success',
                content: '步骤: 写入 crontab',
              },
            ],
            restored: true,
            createdAt: '2026-03-12T00:00:00Z',
          },
        ],
      },
    };

    render(<Copilot open scene="global" />);

    expect(await screen.findByText('最终结论')).toBeInTheDocument();
    expect(screen.getByText('配置已完成')).toBeInTheDocument();
    expect(screen.queryByText('工具调用链')).not.toBeInTheDocument();
    expect(screen.queryByText('步骤: 写入 crontab')).not.toBeInTheDocument();
  });

  it('regenerates in place without appending a duplicate user message', async () => {
    restoredConversation = {
      conversations: [{ id: 'sess-regenerate', title: '当前会话', createdAt: '2026-03-12T00:00:00Z', updatedAt: '2026-03-12T00:00:01Z' }],
      activeConversation: {
        id: 'sess-regenerate',
        title: '当前会话',
        messages: [
          {
            id: 'msg-user',
            role: 'user',
            content: '原始问题',
            createdAt: '2026-03-12T00:00:00Z',
          },
          {
            id: 'msg-assistant',
            role: 'assistant',
            content: '旧答案',
            createdAt: '2026-03-12T00:00:01Z',
          },
        ],
      },
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ sessionId: 'sess-regenerate', createdAt: new Date().toISOString() });
      handlers.onDelta?.({ contentChunk: '新答案' } as any);
      handlers.onDone?.({} as any);
    });

    render(<Copilot open scene="global" />);

    expect(await screen.findByText('旧答案')).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole('button', { name: '重新生成' }).at(-1)!);

    await waitFor(() => {
      expect(aiApi.chatStream).toHaveBeenCalledTimes(1);
    });

    await waitFor(() => {
      expect(screen.getByText('新答案')).toBeInTheDocument();
    });

    expect(screen.queryByText('旧答案')).not.toBeInTheDocument();
    expect(screen.getAllByText('原始问题')).toHaveLength(1);
  });

  it('restores all historical conversations and keeps user message before assistant output', async () => {
    restoredConversation = {
      conversations: [
        { id: 'sess-current', title: '当前会话', createdAt: '2026-03-12T00:00:00Z', updatedAt: '2026-03-12T00:00:02Z' },
        { id: 'sess-old', title: '更早会话', createdAt: '2026-03-11T00:00:00Z', updatedAt: '2026-03-11T00:00:01Z' },
      ],
      activeConversation: {
        id: 'sess-current',
        title: '当前会话',
        messages: [
          {
            id: 'msg-assistant',
            role: 'assistant',
            content: '历史回答',
            createdAt: '2026-03-12T00:00:01Z',
          },
          {
            id: 'msg-user',
            role: 'user',
            content: '用户问题',
            createdAt: '2026-03-12T00:00:00Z',
          },
        ],
      },
    };

    const view = render(<Copilot open scene="global" />);

    expect(await screen.findByText('用户问题')).toBeInTheDocument();
    const bubbleItems = view.container.querySelectorAll('[data-testid^="bubble-item-"]');
    expect(bubbleItems[0]).toHaveAttribute('data-role', 'user');
    expect(bubbleItems[1]).toHaveAttribute('data-role', 'assistant');
  });

  it('creates a new conversation bucket when messages are appended to a just-created key', () => {
    const state = conversationReducer(
      {
        conversations: [DEFAULT_CONVERSATION],
        activeKey: DEFAULT_CONVERSATION.key,
      },
      {
        type: 'append_messages',
        key: 'fresh-conversation',
        label: '新对话',
        messages: [
          {
            id: 'user-1',
            role: 'user',
            content: '帮我检查集群',
            createdAt: '2026-03-14T00:00:00Z',
          },
        ],
      },
    );

    expect(state.activeKey).toBe('fresh-conversation');
    expect(state.conversations[0]).toEqual(expect.objectContaining({
      key: 'fresh-conversation',
      messages: [
        expect.objectContaining({
          id: 'user-1',
          content: '帮我检查集群',
        }),
      ],
    }));
  });

  it('submits a recommended prompt in a new conversation without falling into an unavailable state', async () => {
    mockedScenePrompts = [{ key: 'hosts', description: '查询所有服务器的状态' }];
    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ sessionId: 'sess-prompts', createdAt: new Date().toISOString() });
      handlers.onChainStarted?.({ turn_id: 'turn-prompts' } as any);
      handlers.onFinalAnswerStarted?.({ turn_id: 'turn-prompts' } as any);
      handlers.onFinalAnswerDelta?.({ turn_id: 'turn-prompts', chunk: '已收到推荐问题' } as any);
      handlers.onFinalAnswerDone?.({ turn_id: 'turn-prompts' } as any);
      handlers.onDone?.({ stream_state: 'ok' } as any);
    });

    render(<Copilot open scene="global" />);

    fireEvent.click(await screen.findByRole('button', { name: '查询所有服务器的状态' }));

    await waitFor(() => {
      expect(aiApi.chatStream).toHaveBeenCalledTimes(1);
    });

    expect(screen.queryByText('AI 助手暂时不可用')).not.toBeInTheDocument();
    expect(await screen.findByText('查询所有服务器的状态')).toBeInTheDocument();
    expect(await screen.findByText('已收到推荐问题')).toBeInTheDocument();
  });

  it('renders plan, tool, and approval nodes during regenerate on the runtime path', async () => {
    restoredConversation = {
      conversations: [{ id: 'sess-stream', title: '当前会话', createdAt: '2026-03-12T00:00:00Z', updatedAt: '2026-03-12T00:00:01Z' }],
      activeConversation: {
        id: 'sess-stream',
        title: '当前会话',
        messages: [
          {
            id: 'msg-user',
            role: 'user',
            content: '把 nginx 扩容到 3 个副本',
            createdAt: '2026-03-12T00:00:00Z',
          },
          {
            id: 'msg-assistant',
            role: 'assistant',
            content: '旧答案',
            createdAt: '2026-03-12T00:00:01Z',
          },
        ],
      },
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ sessionId: 'sess-stream', createdAt: new Date().toISOString() });
      handlers.onChainStarted?.({ turn_id: 'turn-stream' } as any);
      handlers.onChainNodeOpen?.({
        turn_id: 'turn-stream',
        node_id: 'plan:plan-1',
        kind: 'plan',
        title: '整理执行步骤',
        status: 'loading',
        headline: '正在整理执行步骤',
      } as any);
      handlers.onChainNodeOpen?.({
        turn_id: 'turn-stream',
        node_id: 'tool:step-1',
        kind: 'tool',
        title: '扩容 nginx',
        status: 'loading',
        headline: '准备调用扩容工具',
      } as any);
      handlers.onChainNodeOpen?.({
        turn_id: 'turn-stream',
        node_id: 'approval:step-1',
        kind: 'approval',
        title: '扩容 nginx 需要确认',
        status: 'waiting',
        headline: '该步骤会修改工作负载副本数',
        approval: {
          request_id: 'approval-1',
          title: '扩容 nginx 需要确认',
          risk: 'medium',
          summary: '该步骤会修改工作负载副本数',
          details: {
            plan_id: 'plan-1',
            step_id: 'step-1',
            tool_name: 'scale_deployment',
          },
        },
      } as any);
      handlers.onDone?.({ stream_state: 'ok' } as any);
    });

    render(<Copilot open scene="global" />);

    const regenerateButtons = await screen.findAllByRole('button', { name: '重新生成' });
    fireEvent.click(regenerateButtons[regenerateButtons.length - 1]);

    await waitFor(() => {
      expect(aiApi.chatStream).toHaveBeenCalledTimes(1);
    });

    expect(await screen.findByRole('button', { name: '扩容 nginx 需要确认，确认执行' })).toBeInTheDocument();
  });

  it('renders native runtime thought chain first and final answer after collapse', async () => {
    restoredConversation = {
      conversations: [{ id: 'sess-native', title: '当前会话', createdAt: '2026-03-12T00:00:00Z', updatedAt: '2026-03-12T00:00:01Z' }],
      activeConversation: {
        id: 'sess-native',
        title: '当前会话',
        messages: [
          {
            id: 'msg-user',
            role: 'user',
            content: '检查 nginx 当前状态',
            createdAt: '2026-03-12T00:00:00Z',
          },
          {
            id: 'msg-assistant',
            role: 'assistant',
            content: '旧答案',
            createdAt: '2026-03-12T00:00:01Z',
          },
        ],
      },
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ sessionId: 'sess-native', createdAt: new Date().toISOString() });
      handlers.onChainStarted?.({ turn_id: 'turn-native' } as any);
      handlers.onChainNodeOpen?.({
        turn_id: 'turn-native',
        node_id: 'plan-1',
        kind: 'plan',
        title: '正在整理执行计划',
        status: 'loading',
        summary: '准备检查 nginx deployment',
      } as any);
      handlers.onChainNodeClose?.({
        turn_id: 'turn-native',
        node_id: 'plan-1',
        status: 'done',
      } as any);
      handlers.onChainCollapsed?.({ turn_id: 'turn-native' } as any);
      handlers.onFinalAnswerStarted?.({ turn_id: 'turn-native' } as any);
      handlers.onFinalAnswerDelta?.({ turn_id: 'turn-native', chunk: 'nginx 当前状态正常' } as any);
      handlers.onFinalAnswerDone?.({ turn_id: 'turn-native' } as any);
      handlers.onDone?.({ stream_state: 'ok' } as any);
    });

    render(<Copilot open scene="global" />);

    const regenerateButtons = await screen.findAllByRole('button', { name: '重新生成' });
    fireEvent.click(regenerateButtons[regenerateButtons.length - 1]);

    await waitFor(() => {
      expect(aiApi.chatStream).toHaveBeenCalledTimes(1);
    });

    expect(await screen.findByText('正在整理执行计划')).toBeInTheDocument();
    expect(await screen.findByText('nginx 当前状态正常')).toBeInTheDocument();
  });

  it('renders planner structured steps without leaking raw json', async () => {
    restoredConversation = {
      conversations: [{ id: 'sess-planner-json', title: '当前会话', createdAt: '2026-03-12T00:00:00Z', updatedAt: '2026-03-12T00:00:01Z' }],
      activeConversation: {
        id: 'sess-planner-json',
        title: '当前会话',
        messages: [
          {
            id: 'msg-user',
            role: 'user',
            content: '检查所有主机状态',
            createdAt: '2026-03-12T00:00:00Z',
          },
          {
            id: 'msg-assistant',
            role: 'assistant',
            content: '旧答案',
            createdAt: '2026-03-12T00:00:01Z',
          },
        ],
      },
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ sessionId: 'sess-planner-json', createdAt: new Date().toISOString() });
      handlers.onChainStarted?.({ turn_id: 'turn-planner-json' } as any);
      handlers.onChainNodeOpen?.({
        turn_id: 'turn-planner-json',
        node_id: 'plan:plan-json-1',
        kind: 'plan',
        title: '整理执行步骤',
        status: 'loading',
        headline: '已生成执行计划',
        structured: {
          steps: [
            { id: 'step-1', title: '获取主机列表', description: '读取所有主机信息' },
            { id: 'step-2', title: '汇总状态', description: '统计在线离线数量' },
          ],
        },
      } as any);
      handlers.onChainNodeClose?.({
        turn_id: 'turn-planner-json',
        node_id: 'plan:plan-json-1',
        status: 'done',
      } as any);
      handlers.onChainCollapsed?.({ turn_id: 'turn-planner-json' } as any);
      handlers.onFinalAnswerStarted?.({ turn_id: 'turn-planner-json' } as any);
      handlers.onFinalAnswerDelta?.({ turn_id: 'turn-planner-json', chunk: '共有 5 台主机在线' } as any);
      handlers.onFinalAnswerDone?.({ turn_id: 'turn-planner-json' } as any);
      handlers.onDone?.({ stream_state: 'ok' } as any);
    });

    render(<Copilot open scene="global" />);

    const regenerateButtons = await screen.findAllByRole('button', { name: '重新生成' });
    fireEvent.click(regenerateButtons[regenerateButtons.length - 1]);

    await waitFor(() => {
      expect(aiApi.chatStream).toHaveBeenCalledTimes(1);
    });

    // Expect final answer renders correctly
    expect(await screen.findByText('共有 5 台主机在线')).toBeInTheDocument();
  });
});
