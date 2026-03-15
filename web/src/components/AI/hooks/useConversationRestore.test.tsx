import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { aiApi } from '../../../api/modules/ai';
import { useConversationRestore } from './useConversationRestore';

describe('useConversationRestore', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('restores the current scene session with content and raw evidence', async () => {
    vi.spyOn(aiApi, 'getSessions').mockResolvedValue({
      code: 0,
      data: [
        {
          id: 'sess-older',
          title: 'Older session',
          createdAt: '2026-03-10T00:00:00Z',
          updatedAt: '2026-03-10T00:00:01Z',
          messages: [],
        },
        {
          id: 'sess-1',
          title: 'Current session',
          createdAt: '2026-03-11T00:00:00Z',
          updatedAt: '2026-03-11T00:00:01Z',
          messages: [],
        },
      ],
      msg: 'ok',
    } as any);
    vi.spyOn(aiApi, 'getCurrentSession').mockResolvedValue({
      code: 0,
      data: {
        id: 'sess-1',
        title: 'Current session',
        createdAt: '2026-03-11T00:00:00Z',
        updatedAt: '2026-03-11T00:00:01Z',
        messages: [{
          id: 'msg-1',
          role: 'assistant',
          content: 'final answer',
          thoughtChain: [{ key: 'summary', title: '生成结论', status: 'success', content: 'summary thinking' }],
          rawEvidence: ['tool output'],
          timestamp: '2026-03-11T00:00:01Z',
        }],
      },
      msg: 'ok',
    } as any);

    const onRestore = vi.fn();
    renderHook(() => useConversationRestore({
      scene: 'scene:host',
      onRestore,
    }));

    await waitFor(() => {
      expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({
        conversations: [
          expect.objectContaining({ id: 'sess-1' }),
          expect.objectContaining({ id: 'sess-older' }),
        ],
        activeConversation: expect.objectContaining({
          id: 'sess-1',
          messages: [
            expect.objectContaining({
              content: 'final answer',
              thinking: undefined,
              thoughtChain: [],
              rawEvidence: ['tool output'],
              restored: true,
            }),
          ],
        }),
      }));
    });
  });

  it('falls back to the most recent session detail when no current session exists', async () => {
    vi.spyOn(aiApi, 'getCurrentSession').mockResolvedValue({
      code: 0,
      data: null,
      msg: 'ok',
    } as any);
    vi.spyOn(aiApi, 'getSessions').mockResolvedValue({
      code: 0,
      data: [{
        id: 'sess-2',
        title: 'Recent session',
        createdAt: '2026-03-11T00:00:00Z',
        updatedAt: '2026-03-11T00:00:01Z',
        messages: [],
      }],
      msg: 'ok',
    } as any);
    vi.spyOn(aiApi, 'getSessionDetail').mockResolvedValue({
      code: 0,
      data: {
        id: 'sess-2',
        title: 'Recent session',
        createdAt: '2026-03-11T00:00:00Z',
        updatedAt: '2026-03-11T00:00:01Z',
        messages: [{
          id: 'msg-2',
          role: 'assistant',
          content: 'restored answer',
          timestamp: '2026-03-11T00:00:01Z',
        }],
      },
      msg: 'ok',
    } as any);

    const onRestore = vi.fn();
    renderHook(() => useConversationRestore({
      scene: 'scene:k8s',
      onRestore,
    }));

    await waitFor(() => {
      expect(aiApi.getSessionDetail).toHaveBeenCalledWith('sess-2', 'scene:k8s');
      expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({
        conversations: [expect.objectContaining({ id: 'sess-2' })],
        activeConversation: expect.objectContaining({
          id: 'sess-2',
          messages: [expect.objectContaining({ content: 'restored answer' })],
        }),
      }));
    });
  });

  it('prefers structured turns when replay contract is available', async () => {
    vi.spyOn(aiApi, 'getSessions').mockResolvedValue({ code: 0, data: [], msg: 'ok' } as any);
    vi.spyOn(aiApi, 'getCurrentSession').mockResolvedValue({
      code: 0,
      data: {
        id: 'sess-turn',
        title: 'Turn session',
        createdAt: '2026-03-11T00:00:00Z',
        updatedAt: '2026-03-11T00:00:01Z',
        messages: [],
        turns: [
          {
            id: 'turn-user',
            role: 'user',
            status: 'completed',
            blocks: [
              {
                id: 'user-text',
                blockType: 'text',
                position: 1,
                contentText: 'scale deployment',
                createdAt: '2026-03-11T00:00:00Z',
                updatedAt: '2026-03-11T00:00:00Z',
              },
            ],
            createdAt: '2026-03-11T00:00:00Z',
            updatedAt: '2026-03-11T00:00:00Z',
          },
          {
            id: 'turn-assistant',
            role: 'assistant',
            status: 'completed',
            phase: 'done',
            blocks: [
              {
                id: 'status-1',
                blockType: 'status',
                position: 1,
                title: '执行中',
                contentText: '正在扩容',
                createdAt: '2026-03-11T00:00:01Z',
                updatedAt: '2026-03-11T00:00:01Z',
              },
              {
                id: 'text-1',
                blockType: 'text',
                position: 2,
                contentText: '扩容完成',
                streaming: false,
                createdAt: '2026-03-11T00:00:02Z',
                updatedAt: '2026-03-11T00:00:02Z',
              },
            ],
            createdAt: '2026-03-11T00:00:01Z',
            updatedAt: '2026-03-11T00:00:02Z',
          },
        ],
      },
      msg: 'ok',
    } as any);

    const onRestore = vi.fn();
    renderHook(() => useConversationRestore({
      scene: 'global',
      onRestore,
    }));

    await waitFor(() => {
      expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({
        activeConversation: expect.objectContaining({
          id: 'sess-turn',
          messages: [
            expect.objectContaining({ role: 'user', content: 'scale deployment' }),
            expect.objectContaining({
              role: 'assistant',
              content: '扩容完成',
              restored: true,
              turn: expect.objectContaining({ id: 'turn-assistant' }),
            }),
          ],
        }),
      }));
    });
  });

  it('prefers persisted assistant runtime over empty replay blocks', async () => {
    vi.spyOn(aiApi, 'getSessions').mockResolvedValue({ code: 0, data: [], msg: 'ok' } as any);
    vi.spyOn(aiApi, 'getCurrentSession').mockResolvedValue({
      code: 0,
      data: {
        id: 'sess-runtime',
        title: 'Runtime-first restore',
        createdAt: '2026-03-11T00:00:00Z',
        updatedAt: '2026-03-11T00:00:01Z',
        messages: [
          {
            id: 'assistant-message-1',
            role: 'assistant',
            turnId: 'turn-assistant',
            content: '',
            runtime: {
              turn_id: 'turn-assistant',
              nodes: [
                {
                  node_id: 'tool:step-1',
                  kind: 'tool',
                  title: 'host_list_inventory',
                  status: 'done',
                  headline: '已获取 2 台主机',
                  structured: { resource: 'hosts', rows: [{ id: 1, name: 'test', status: 'online' }] },
                },
              ],
            },
            timestamp: '2026-03-11T00:00:01Z',
          },
        ],
        turns: [
          {
            id: 'turn-assistant',
            role: 'assistant',
            status: 'completed',
            phase: 'done',
            blocks: [],
            createdAt: '2026-03-11T00:00:01Z',
            updatedAt: '2026-03-11T00:00:01Z',
          },
        ],
      },
      msg: 'ok',
    } as any);

    const onRestore = vi.fn();
    renderHook(() => useConversationRestore({ scene: 'global', onRestore }));

    await waitFor(() => {
      expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({
        activeConversation: expect.objectContaining({
          id: 'sess-runtime',
          messages: [
            expect.objectContaining({
              role: 'assistant',
              runtime: expect.objectContaining({
                nodes: [
                  expect.objectContaining({
                    nodeId: 'tool:step-1',
                    title: 'host_list_inventory',
                  }),
                ],
              }),
            }),
          ],
        }),
      }));
    });
  });

  it('keeps persisted legacy user messages when replay turns only contain assistant turns', async () => {
    vi.spyOn(aiApi, 'getSessions').mockResolvedValue({ code: 0, data: [], msg: 'ok' } as any);
    vi.spyOn(aiApi, 'getCurrentSession').mockResolvedValue({
      code: 0,
      data: {
        id: 'sess-mixed',
        title: 'Mixed restore',
        createdAt: '2026-03-11T00:00:00Z',
        updatedAt: '2026-03-11T00:00:02Z',
        messages: [
          {
            id: 'msg-user-1',
            role: 'user',
            content: '把 nginx 扩容到 3 个副本',
            timestamp: '2026-03-11T00:00:00Z',
          },
          {
            id: 'msg-assistant-1',
            role: 'assistant',
            content: '',
            timestamp: '2026-03-11T00:00:02Z',
          },
        ],
        turns: [
          {
            id: 'turn-assistant',
            role: 'assistant',
            status: 'completed',
            phase: 'done',
            blocks: [
              {
                id: 'status-1',
                blockType: 'status',
                position: 1,
                title: '执行中',
                contentText: '正在扩容',
                createdAt: '2026-03-11T00:00:01Z',
                updatedAt: '2026-03-11T00:00:01Z',
              },
              {
                id: 'text-1',
                blockType: 'text',
                position: 2,
                contentText: '扩容完成',
                createdAt: '2026-03-11T00:00:02Z',
                updatedAt: '2026-03-11T00:00:02Z',
              },
            ],
            createdAt: '2026-03-11T00:00:01Z',
            updatedAt: '2026-03-11T00:00:02Z',
          },
        ],
      },
      msg: 'ok',
    } as any);

    const onRestore = vi.fn();
    renderHook(() => useConversationRestore({ scene: 'global', onRestore }));

    await waitFor(() => {
      expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({
        activeConversation: expect.objectContaining({
          id: 'sess-mixed',
          messages: [
            expect.objectContaining({ role: 'user', content: '把 nginx 扩容到 3 个副本' }),
            expect.objectContaining({ role: 'assistant', content: '扩容完成' }),
          ],
        }),
      }));
    });
  });

  it('uses markdown-like status content as assistant fallback when replay text block is missing', async () => {
    vi.spyOn(aiApi, 'getSessions').mockResolvedValue({ code: 0, data: [], msg: 'ok' } as any);
    vi.spyOn(aiApi, 'getCurrentSession').mockResolvedValue({
      code: 0,
      data: {
        id: 'sess-markdown',
        title: 'Markdown restore',
        createdAt: '2026-03-11T00:00:00Z',
        updatedAt: '2026-03-11T00:00:01Z',
        messages: [],
        turns: [
          {
            id: 'turn-assistant',
            role: 'assistant',
            status: 'completed',
            phase: 'done',
            blocks: [
              {
                id: 'status-1',
                blockType: 'status',
                position: 1,
                title: '整理最终结论',
                contentText: '## 结果\n- 项目正常',
                createdAt: '2026-03-11T00:00:01Z',
                updatedAt: '2026-03-11T00:00:01Z',
              },
            ],
            createdAt: '2026-03-11T00:00:01Z',
            updatedAt: '2026-03-11T00:00:01Z',
          },
        ],
      },
      msg: 'ok',
    } as any);

    const onRestore = vi.fn();
    renderHook(() => useConversationRestore({ scene: 'global', onRestore }));

    await waitFor(() => {
      expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({
        activeConversation: expect.objectContaining({
          messages: [expect.objectContaining({ content: '## 结果\n- 项目正常' })],
        }),
      }));
    });
  });

  it('falls back to summary thought stage content when legacy assistant content is empty', async () => {
    vi.spyOn(aiApi, 'getSessions').mockResolvedValue({ code: 0, data: [], msg: 'ok' } as any);
    vi.spyOn(aiApi, 'getCurrentSession').mockResolvedValue({
      code: 0,
      data: {
        id: 'sess-legacy-summary',
        title: 'Legacy summary session',
        createdAt: '2026-03-11T00:00:00Z',
        updatedAt: '2026-03-11T00:00:01Z',
        messages: [{
          id: 'msg-1',
          role: 'assistant',
          content: '',
          thoughtChain: [{ key: 'summary', title: '整理最终结论', status: 'success', content: '## 最终结论\n- 已完成' }],
          timestamp: '2026-03-11T00:00:01Z',
        }],
      },
      msg: 'ok',
    } as any);

    const onRestore = vi.fn();
    renderHook(() => useConversationRestore({ scene: 'global', onRestore }));

    await waitFor(() => {
      expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({
        activeConversation: expect.objectContaining({
          messages: [expect.objectContaining({ content: '## 最终结论\n- 已完成', thinking: undefined })],
        }),
      }));
    });
  });

  it('preserves raw restored assistant markdown content for direct XMarkdown rendering', async () => {
    vi.spyOn(aiApi, 'getSessions').mockResolvedValue({ code: 0, data: [], msg: 'ok' } as any);
    vi.spyOn(aiApi, 'getCurrentSession').mockResolvedValue({
      code: 0,
      data: {
        id: 'sess-bad-md',
        title: 'Bad markdown session',
        createdAt: '2026-03-11T00:00:00Z',
        updatedAt: '2026-03-11T00:00:01Z',
        messages: [{
          id: 'msg-1',
          role: 'assistant',
          content: '##服务器状态查询结果\n---###系统分析结论',
          timestamp: '2026-03-11T00:00:01Z',
        }],
      },
      msg: 'ok',
    } as any);

    const onRestore = vi.fn();
    renderHook(() => useConversationRestore({ scene: 'global', onRestore }));

    await waitFor(() => {
      expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({
        activeConversation: expect.objectContaining({
          messages: [expect.objectContaining({
            content: '##服务器状态查询结果\n---###系统分析结论',
          })],
        }),
      }));
    });
  });

  it('derives native thought chain runtime snapshots from replay turns', async () => {
    vi.spyOn(aiApi, 'getSessions').mockResolvedValue({ code: 0, data: [], msg: 'ok' } as any);
    vi.spyOn(aiApi, 'getCurrentSession').mockResolvedValue({
      code: 0,
      data: {
        id: 'sess-native-runtime',
        title: 'Native runtime session',
        createdAt: '2026-03-11T00:00:00Z',
        updatedAt: '2026-03-11T00:00:01Z',
        messages: [],
        turns: [{
          id: 'turn-native',
          role: 'assistant',
          status: 'completed',
          phase: 'done',
          blocks: [
            {
              id: 'tool-1',
              blockType: 'tool',
              position: 1,
              title: 'get_deployment',
              contentText: 'nginx status',
              createdAt: '2026-03-11T00:00:01Z',
              updatedAt: '2026-03-11T00:00:01Z',
            },
            {
              id: 'answer-1',
              blockType: 'text',
              position: 2,
              contentText: 'nginx 当前状态正常',
              createdAt: '2026-03-11T00:00:01Z',
              updatedAt: '2026-03-11T00:00:01Z',
            },
          ],
          createdAt: '2026-03-11T00:00:01Z',
          updatedAt: '2026-03-11T00:00:01Z',
          completedAt: '2026-03-11T00:00:02Z',
        }],
      },
      msg: 'ok',
    } as any);

    const onRestore = vi.fn();
    renderHook(() => useConversationRestore({ scene: 'global', onRestore }));

    await waitFor(() => {
      expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({
        activeConversation: expect.objectContaining({
          messages: [expect.objectContaining({
            content: 'nginx 当前状态正常',
            turn: expect.objectContaining({
              id: 'turn-native',
            }),
          })],
        }),
      }));
    });
  });
});
