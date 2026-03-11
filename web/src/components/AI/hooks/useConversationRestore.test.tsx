import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { aiApi } from '../../../api/modules/ai';
import { useConversationRestore } from './useConversationRestore';

describe('useConversationRestore', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('restores the current scene session with content and raw evidence', async () => {
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
        id: 'sess-1',
        messages: [
          expect.objectContaining({
            content: 'final answer',
            thinking: 'summary thinking',
            thoughtChain: [],
            rawEvidence: ['tool output'],
          }),
        ],
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
        id: 'sess-2',
        messages: [expect.objectContaining({ content: 'restored answer' })],
      }));
    });
  });
});
