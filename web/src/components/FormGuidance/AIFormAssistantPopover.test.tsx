import React from 'react';
import { describe, it, expect, vi, afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import { fireEvent, screen, renderWithAntd } from '../../test/utils/render';
import AIFormAssistantPopover from './AIFormAssistantPopover';

describe('AIFormAssistantPopover', () => {
  afterEach(() => {
    cleanup();
  });

  const defaultProps = {
    isOpen: true,
    isStreaming: false,
    prompt: '',
    preview: '',
    error: null,
    onCancel: vi.fn(),
    onSubmit: vi.fn(),
    onApply: vi.fn(),
  };

  it('renders correctly when open', () => {
    renderWithAntd(
      <AIFormAssistantPopover {...defaultProps}>
        <button>Trigger</button>
      </AIFormAssistantPopover>
    );

    expect(screen.getByText('AI 辅助生成')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/描述你想要的内容/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /生成建议/ })).toBeInTheDocument();
  });

  it('calls onSubmit when clicking generate button', async () => {
    const onSubmit = vi.fn();
    renderWithAntd(
      <AIFormAssistantPopover {...defaultProps} onSubmit={onSubmit}>
        <button>Trigger</button>
      </AIFormAssistantPopover>
    );

    const textArea = screen.getByPlaceholderText(/描述你想要的内容/);
    fireEvent.change(textArea, { target: { value: 'test prompt' } });

    const submitBtn = screen.getByRole('button', { name: /生成建议/ });
    fireEvent.click(submitBtn);

    expect(onSubmit).toHaveBeenCalledWith('test prompt');
  });

  it('shows preview and apply button when preview is available', () => {
    renderWithAntd(
      <AIFormAssistantPopover {...defaultProps} preview="AI suggested text">
        <button>Trigger</button>
      </AIFormAssistantPopover>
    );

    expect(screen.getByText('建议结果')).toBeInTheDocument();
    expect(screen.getByText('AI suggested text')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /采纳建议/ })).toBeInTheDocument();
  });

  it('calls onApply when clicking apply button', () => {
    const onApply = vi.fn();
    renderWithAntd(
      <AIFormAssistantPopover {...defaultProps} preview="AI suggested text" onApply={onApply}>
        <button>Trigger</button>
      </AIFormAssistantPopover>
    );

    const applyBtn = screen.getByRole('button', { name: /采纳建议/ });
    fireEvent.click(applyBtn);

    expect(onApply).toHaveBeenCalledTimes(1);
  });

  it('shows error message when error is provided', () => {
    renderWithAntd(
      <AIFormAssistantPopover {...defaultProps} error="Something went wrong">
        <button>Trigger</button>
      </AIFormAssistantPopover>
    );

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('shows streaming state', () => {
    renderWithAntd(
      <AIFormAssistantPopover {...defaultProps} isStreaming={true}>
        <button>Trigger</button>
      </AIFormAssistantPopover>
    );

    expect(screen.getByText('生成中...')).toBeInTheDocument();
    expect(screen.getByText('AI 正在构思中...')).toBeInTheDocument();
  });
});
