import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ChannelsConfigPage from './ChannelsConfigPage';

const mockApi = vi.hoisted(() => ({
  monitoring: {
    testAlertChannel: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('ChannelsConfigPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.monitoring.testAlertChannel.mockResolvedValue({
      data: { status: 'sent' },
    });
  });

  it('calls test channel API on test send', async () => {
    render(<ChannelsConfigPage />);

    fireEvent.click(await screen.findByRole('button', { name: '测试发送' }));

    await waitFor(() => {
      expect(mockApi.monitoring.testAlertChannel).toHaveBeenCalled();
    });
  });
});
