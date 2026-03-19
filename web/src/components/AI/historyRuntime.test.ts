import { describe, expect, it, vi, beforeEach } from 'vitest';
import { hydrateAssistantHistoryMessage, loadMessageRuntime } from './historyRuntime';
import { aiApi } from '../../api/modules/ai';

// Mock aiApi
vi.mock('../../api/modules/ai', () => ({
  aiApi: {
    getMessageRuntime: vi.fn(),
  },
}));

describe('hydrateAssistantHistoryMessage', () => {
  it('returns message with id and hasRuntime flag', () => {
    const hydrated = hydrateAssistantHistoryMessage({
      id: 'msg-123',
      role: 'assistant',
      content: '历史回答',
      status: 'done',
      has_runtime: true,
    } as any);

    expect(hydrated.id).toBe('msg-123');
    expect(hydrated.hasRuntime).toBe(true);
    expect(hydrated.content).toBe('历史回答');
    expect(hydrated.role).toBe('assistant');
  });

  it('sets hasRuntime to false when has_runtime is absent', () => {
    const hydrated = hydrateAssistantHistoryMessage({
      id: 'msg-456',
      role: 'assistant',
      content: '历史回答',
      status: 'done',
    } as any);

    expect(hydrated.hasRuntime).toBe(false);
  });

  it('does not include runtime in initial hydration', () => {
    const hydrated = hydrateAssistantHistoryMessage({
      id: 'msg-789',
      role: 'assistant',
      content: '历史回答',
      status: 'done',
      has_runtime: true,
    } as any);

    expect(hydrated.runtime).toBeUndefined();
  });
});

describe('loadMessageRuntime', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads runtime from API and caches it', async () => {
    const mockRuntime = {
      phase: 'completed',
      phaseLabel: '已完成诊断',
      activities: [],
      status: { kind: 'completed', label: '已生成' },
    };

    (aiApi.getMessageRuntime as any).mockResolvedValue({
      data: {
        message_id: 'msg-123',
        runtime: mockRuntime,
      },
    });

    const result = await loadMessageRuntime('msg-123');

    expect(result).toEqual(mockRuntime);
    expect(aiApi.getMessageRuntime).toHaveBeenCalledWith('msg-123');
  });

  it('returns null when API returns no runtime', async () => {
    (aiApi.getMessageRuntime as any).mockResolvedValue({
      data: {
        message_id: 'msg-456',
        runtime: null,
      },
    });

    const result = await loadMessageRuntime('msg-456');

    expect(result).toBeNull();
  });

  it('returns null on API error', async () => {
    (aiApi.getMessageRuntime as any).mockRejectedValue(new Error('Network error'));

    const result = await loadMessageRuntime('msg-789');

    expect(result).toBeNull();
  });

  it('uses cached runtime on subsequent calls', async () => {
    const mockRuntime = {
      phase: 'completed',
      activities: [],
    };

    (aiApi.getMessageRuntime as any).mockResolvedValue({
      data: {
        message_id: 'msg-cached',
        runtime: mockRuntime,
      },
    });

    // First call
    await loadMessageRuntime('msg-cached');

    // Second call should use cache
    const result = await loadMessageRuntime('msg-cached');

    expect(result).toEqual(mockRuntime);
    // API should only be called once
    expect(aiApi.getMessageRuntime).toHaveBeenCalledTimes(1);
  });
});
