import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useFormAssist } from './useFormAssist';
import { aiApi } from '../../../api/modules/ai';
import type { FormAssistConfig } from '../types/formAssist';

// Mock the AI API
vi.mock('../../../api/modules/ai', () => ({
  aiApi: {
    formAssistStream: vi.fn(),
  },
}));

describe('useFormAssist', () => {
  const mockConfig: FormAssistConfig = {
    scene: 'test-scene',
    fieldMeta: {
      key: 'test-key',
      label: 'Test Label',
      purpose: 'Test Purpose',
    },
    getFormContext: () => ({ some: 'context' }),
  };

  const onApply = vi.fn();

  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows hint after 3 seconds of inactivity when enabled', () => {
    localStorage.setItem('ai-form-assist-enabled', '1');
    const { result } = renderHook(() => useFormAssist(mockConfig, 'some value', onApply));

    expect(result.current.showHint).toBe(false);

    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(result.current.showHint).toBe(true);
  });

  it('does not show hint when feature flag is off', () => {
    localStorage.setItem('ai-form-assist-enabled', '0');
    const { result } = renderHook(() => useFormAssist(mockConfig, 'some value', onApply));

    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(result.current.showHint).toBe(false);
  });

  it('does not show hint when currentFieldValue is empty', () => {
    localStorage.setItem('ai-form-assist-enabled', '1');
    const { result } = renderHook(() => useFormAssist(mockConfig, '', onApply));

    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(result.current.showHint).toBe(false);
  });

  it('accumulates streamed preview chunks', async () => {
    localStorage.setItem('ai-form-assist-enabled', '1');
    const { result } = renderHook(() => useFormAssist(mockConfig, 'initial', onApply));

    (aiApi.formAssistStream as any).mockImplementation((_params: any, handlers: any) => {
      handlers.onDelta({ content: 'Hello ' });
      handlers.onDelta({ content: 'world' });
      handlers.onDone();
      return Promise.resolve();
    });

    await act(async () => {
      await result.current.submit('Fix this');
    });

    expect(result.current.preview).toBe('Hello world');
    expect(result.current.isStreaming).toBe(false);
  });

  it('calls onApply with final preview text when applySuggestion is called', async () => {
    localStorage.setItem('ai-form-assist-enabled', '1');
    const { result } = renderHook(() => useFormAssist(mockConfig, 'initial', onApply));

    (aiApi.formAssistStream as any).mockImplementation((_params: any, handlers: any) => {
      handlers.onDelta({ content: 'Suggested Text' });
      handlers.onDone();
      return Promise.resolve();
    });

    await act(async () => {
      await result.current.submit('test');
    });

    act(() => {
      result.current.applySuggestion();
    });

    expect(onApply).toHaveBeenCalledWith('Suggested Text');
    expect(result.current.isOpen).toBe(false);
  });

  it('clears state on cancel', async () => {
    localStorage.setItem('ai-form-assist-enabled', '1');
    const { result } = renderHook(() => useFormAssist(mockConfig, 'initial', onApply));

    let capturedHandlers: any;
    (aiApi.formAssistStream as any).mockImplementation((_params: any, handlers: any) => {
      capturedHandlers = handlers;
      handlers.onDelta({ content: 'part' });
      // Keep it pending
      return new Promise(() => {});
    });

    await act(async () => {
      result.current.open();
      // start submit but it won't resolve
      result.current.submit('test');
    });

    expect(result.current.isOpen).toBe(true);
    expect(result.current.preview).toBe('part');
    expect(result.current.prompt).toBe('test');

    act(() => {
      result.current.cancel();
    });

    expect(result.current.isOpen).toBe(false);
    expect(result.current.preview).toBe('');
    expect(result.current.prompt).toBe('');
    expect(result.current.isStreaming).toBe(false);
  });
});
