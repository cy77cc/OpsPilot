import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Copilot } from './Copilot';
import { aiApi } from '../../api/modules/ai';

let restoredConversation: any = null;

vi.mock('@ant-design/x', () => ({
  Bubble: {
    List: ({ items }: { items: Array<{ key: string; content: React.ReactNode }> }) => (
      <div>
        {items.map((item) => (
          <div key={item.key}>{item.content}</div>
        ))}
      </div>
    ),
  },
  Conversations: () => null,
  Prompts: () => null,
  Sender: () => null,
  Welcome: () => null,
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
      restoredSessionId: restoredConversation?.id || null,
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

  it('keeps restored thought chain collapsed by default', async () => {
    restoredConversation = {
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
              title: '调用专家执行',
              status: 'success',
              content: '步骤: 检查 deployment',
            },
          ],
          restored: true,
          createdAt: '2026-03-12T00:00:00Z',
        },
      ],
    };

    render(<Copilot open scene="global" />);

    expect(await screen.findByText('调用专家执行')).toBeInTheDocument();
    expect(screen.queryByText('步骤: 检查 deployment')).not.toBeInTheDocument();
  });

  it('regenerates in place without appending a duplicate user message', async () => {
    restoredConversation = {
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
});
