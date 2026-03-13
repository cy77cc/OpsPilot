import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { aiApi } from '../../../api/modules/ai';
import { useAIChat } from './useAIChat';

const messageFns = vi.hoisted(() => ({
  success: vi.fn(),
  info: vi.fn(),
  error: vi.fn(),
}));

vi.mock('antd', () => ({
  message: messageFns,
}));

describe('useAIChat', () => {
  beforeEach(() => {
    vi.spyOn(aiApi, 'chatStream').mockResolvedValue();
    vi.spyOn(aiApi, 'respondApproval').mockResolvedValue({ code: 0, data: {} as any, msg: 'ok' } as any);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    messageFns.success.mockReset();
    messageFns.info.mockReset();
    messageFns.error.mockReset();
  });

  it('maps ThoughtChain stage and step events into assistant message state', async () => {
    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ sessionId: 'sess-1' } as any);
      handlers.onStageDelta?.({ stage: 'plan', status: 'loading', summary: '正在整理执行步骤' } as any);
      handlers.onStepUpdate?.({
        plan_id: 'plan-1',
        step_id: 'step-1',
        tool: 'scale_deployment',
        title: '扩容 nginx',
        status: 'loading',
        user_visible_summary: '准备调用扩容工具',
      } as any);
      handlers.onDone?.({} as any);
    });

    const { result } = renderHook(() => useAIChat({ scene: 'global' }));

    await act(async () => {
      await result.current.sendMessage('把 nginx 扩容到 3 个副本');
    });

    await waitFor(() => {
      expect(result.current.messages).toHaveLength(2);
    });

    const assistant = result.current.messages[1];
    expect(assistant.role).toBe('assistant');
    expect(assistant.thoughtChain).toEqual(expect.arrayContaining([
      expect.objectContaining({
        key: 'plan',
        title: '整理执行步骤',
        status: 'success',
        description: '正在整理执行步骤',
        content: '正在整理执行步骤',
      }),
      expect.objectContaining({
        key: 'execute',
        title: '工具调用链',
        status: 'success',
        details: [
          expect.objectContaining({
            id: 'step-1',
            label: '扩容 nginx',
            status: 'success',
            content: '准备调用扩容工具',
            plan_id: 'plan-1',
            step_id: 'step-1',
          }),
        ],
      }),
    ]));
  });

  it('confirms approval with checkpoint_id and finalizes the user_action stage', async () => {
    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ sessionId: 'sess-approval' } as any);
      handlers.onApprovalRequired?.({
        id: 'approval-1',
        session_id: 'sess-approval',
        plan_id: 'plan-1',
        step_id: 'step-1',
        checkpoint_id: 'cp-1',
        tool: 'scale_deployment',
        title: '扩容 nginx 需要确认',
        user_visible_summary: '该步骤会修改工作负载副本数',
        risk: 'medium',
      } as any);
    });

    const { result } = renderHook(() => useAIChat({ scene: 'global' }));

    await act(async () => {
      await result.current.sendMessage('把 nginx 扩容到 3 个副本');
    });

    await waitFor(() => {
      expect(result.current.pendingConfirmation?.id).toBe('approval-1');
    });

    await act(async () => {
      await result.current.confirmAction('approval-1', true);
    });

    expect(aiApi.respondApproval).toHaveBeenCalledWith({
      session_id: 'sess-approval',
      plan_id: 'plan-1',
      step_id: 'step-1',
      checkpoint_id: 'cp-1',
      approved: true,
    });
    expect(messageFns.success).toHaveBeenCalledWith('已确认，继续执行');
    expect(result.current.pendingConfirmation).toBeNull();

    const assistant = result.current.messages[1];
    expect(assistant.confirmation).toBeUndefined();
    expect(assistant.thoughtChain).toEqual(expect.arrayContaining([
      expect.objectContaining({
        key: 'user_action',
        status: 'success',
        title: '等待你确认',
        description: '已确认，继续执行',
      }),
    ]));
  });
});
