import { render, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ChatInterface from './ChatInterface';

const mockApi = vi.hoisted(() => ({
  ai: {
    getSessionDetail: vi.fn(),
    getCurrentSession: vi.fn(),
    getSessions: vi.fn(),
    deleteSession: vi.fn(),
    updateSessionTitle: vi.fn(),
    chatStream: vi.fn(),
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
    mockApi.ai.deleteSession.mockResolvedValue({ data: true });
    mockApi.ai.updateSessionTitle.mockResolvedValue({ data: { id: 's1', title: 'AI Session', updatedAt: '' } });
    mockApi.ai.chatStream.mockResolvedValue(undefined);
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
});
