import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ChatInterface from './ChatInterface';

const mockApi = vi.hoisted(() => ({
  ai: {
    getSessionDetail: vi.fn(),
    getCurrentSession: vi.fn(),
    getSessions: vi.fn(),
    branchSession: vi.fn(),
    deleteSession: vi.fn(),
    updateSessionTitle: vi.fn(),
    chatStream: vi.fn(),
    confirmApproval: vi.fn(),
    respondApproval: vi.fn(),
  },
}));

vi.mock('../../api', () => ({ Api: mockApi }));

describe('ChatInterface', () => {
  beforeEach(() => {
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: vi.fn(),
    });
    vi.clearAllMocks();
    mockApi.ai.getCurrentSession.mockResolvedValue({ data: null });
    mockApi.ai.getSessions.mockResolvedValue({ data: [] });
    mockApi.ai.getSessionDetail.mockResolvedValue({ data: { id: 's1', title: 'AI Session', messages: [], createdAt: '', updatedAt: '' } });
    mockApi.ai.branchSession.mockResolvedValue({ data: { id: 's2', title: '分支会话', messages: [], createdAt: '', updatedAt: '' } });
    mockApi.ai.deleteSession.mockResolvedValue({ data: true });
    mockApi.ai.updateSessionTitle.mockResolvedValue({ data: { id: 's1', title: 'AI Session', updatedAt: '' } });
    mockApi.ai.chatStream.mockResolvedValue(undefined);
    mockApi.ai.confirmApproval.mockResolvedValue({ data: { success: true } });
    mockApi.ai.respondApproval.mockResolvedValue({ data: { resumed: true } });
  });

  it('reloads scene data when scene prop changes', async () => {
    const { rerender } = render(<ChatInterface scene="scene:hosts" />);

    await waitFor(() => expect(mockApi.ai.getCurrentSession).toHaveBeenCalledWith('scene:hosts'));
    await waitFor(() => expect(mockApi.ai.getSessions).toHaveBeenCalledWith('scene:hosts'));

    mockApi.ai.getCurrentSession.mockClear();
    mockApi.ai.getSessions.mockClear();

    rerender(<ChatInterface scene="scene:services" />);

    await waitFor(() => expect(mockApi.ai.getCurrentSession).toHaveBeenCalledWith('scene:services'));
    await waitFor(() => expect(mockApi.ai.getSessions).toHaveBeenCalledWith('scene:services'));
  });

  it('shows expert progress while streaming helpers', async () => {
    mockApi.ai.chatStream.mockImplementationOnce(async (_params: any, handlers: any) => {
      handlers.onMeta?.({ sessionId: 's1', createdAt: new Date().toISOString(), turn_id: 't1' });
      handlers.onExpertProgress?.({ expert: 'k8s_expert', status: 'running', task: '检查Pod状态', turn_id: 't1' });
      handlers.onExpertProgress?.({ expert: 'k8s_expert', status: 'done', duration_ms: 120, turn_id: 't1' });
      handlers.onDone?.({
        session: { id: 's1', title: 'AI Session', messages: [], createdAt: '', updatedAt: '' },
        stream_state: 'ok',
        turn_id: 't1',
      });
    });

    render(<ChatInterface scene="scene:services" />);
    const inputs = screen.getAllByPlaceholderText('请输入您的问题...');
    const input = inputs[inputs.length - 1];
    fireEvent.change(input, { target: { value: '分析服务状态' } });
    const sendButtons = screen.getAllByRole('button', { name: /发送/ });
    fireEvent.click(sendButtons[sendButtons.length - 1]);

    await waitFor(() => expect(screen.getByText(/k8s_expert/)).toBeInTheDocument());
  });

  it('uses interrupt approval endpoint when approval trace contains checkpoint target', async () => {
    mockApi.ai.chatStream.mockImplementationOnce(async (_params: any, handlers: any) => {
      handlers.onMeta?.({ sessionId: 's1', createdAt: new Date().toISOString(), turn_id: 't1' });
      handlers.onApprovalRequired?.({
        checkpoint_id: 'sess-1',
        interrupt_targets: ['tool:host_batch_exec_apply:call-1'],
        tool: 'host_batch_exec_apply',
        turn_id: 't1',
      });
      handlers.onDone?.({
        session: { id: 's1', title: 'AI Session', messages: [], createdAt: '', updatedAt: '' },
        stream_state: 'ok',
        turn_id: 't1',
      });
    });

    render(<ChatInterface scene="scene:hosts" />);
    const inputs = screen.getAllByPlaceholderText('请输入您的问题...');
    fireEvent.change(inputs[inputs.length - 1], { target: { value: '执行写入命令' } });
    const sendButtons = screen.getAllByRole('button', { name: /发送/ });
    fireEvent.click(sendButtons[sendButtons.length - 1]);

    const traceHeaders = await screen.findAllByText(/工具调用轨迹/);
    fireEvent.click(traceHeaders[traceHeaders.length - 1]);
    const approveButtons = await screen.findAllByRole('button', { name: '批准' });
    fireEvent.click(approveButtons[approveButtons.length - 1]);

    await waitFor(() => expect(mockApi.ai.respondApproval).toHaveBeenCalledWith({
      checkpoint_id: 'sess-1',
      target: 'tool:host_batch_exec_apply:call-1',
      approved: true,
    }));
  });

});
