import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Copilot } from './Copilot';
import { aiApi } from '../../api/modules/ai';

let restoredConversation: any = null;

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
  Prompts: () => null,
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
  useScenePrompts: () => ({ prompts: [] }),
}));

vi.mock('./components/MessageActions', () => ({
  MessageActions: ({ onRegenerate }: { onRegenerate?: () => void }) => (
    <button type="button" onClick={onRegenerate}>重新生成</button>
  ),
}));

describe('Copilot', () => {
  beforeEach(() => {
    restoredConversation = null;
    vi.spyOn(aiApi, 'chatStream').mockResolvedValue();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    restoredConversation = null;
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

  it('renders plan, tool, and approval blocks during regenerate on the turn/block path', async () => {
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
      handlers.onStageDelta?.({ stage: 'plan', status: 'loading', summary: '正在整理执行步骤' } as any);
      handlers.onStepUpdate?.({
        plan_id: 'plan-1',
        step_id: 'step-1',
        tool: 'scale_deployment',
        title: '扩容 nginx',
        status: 'loading',
        user_visible_summary: '准备调用扩容工具',
      } as any);
      handlers.onApprovalRequired?.({
        id: 'approval-1',
        session_id: 'sess-stream',
        plan_id: 'plan-1',
        step_id: 'step-1',
        checkpoint_id: 'cp-1',
        tool: 'scale_deployment',
        status: 'pending',
        risk: 'medium',
        title: '扩容 nginx 需要确认',
        user_visible_summary: '该步骤会修改工作负载副本数',
      } as any);
      handlers.onDone?.({ stream_state: 'ok' } as any);
    });

    render(<Copilot open scene="global" />);

    const regenerateButtons = await screen.findAllByRole('button', { name: '重新生成' });
    fireEvent.click(regenerateButtons[regenerateButtons.length - 1]);

    await waitFor(() => {
      expect(aiApi.chatStream).toHaveBeenCalledTimes(1);
    });

    expect(await screen.findByText('整理执行步骤')).toBeInTheDocument();
    expect(screen.getByText('扩容 nginx 需要确认')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '扩容 nginx 需要确认，确认执行' })).toBeInTheDocument();
  });
});
