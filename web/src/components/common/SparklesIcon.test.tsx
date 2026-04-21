import React from 'react';
import { describe, expect, it } from 'vitest';
import { render, screen } from '../../test/utils/render';
import SparklesIcon from './SparklesIcon';

describe('SparklesIcon', () => {
  it('renders the AI assist SVG icon with pulse animation', () => {
    const { container } = render(<SparklesIcon />);

    const icon = screen.getByLabelText('AI 辅助图标');
    expect(icon.tagName.toLowerCase()).toBe('svg');
    expect(icon).toHaveAttribute('width', '16');
    expect(icon).toHaveAttribute('height', '16');
    expect(icon).toHaveAttribute('viewBox', '0 0 24 24');
    expect(icon).toHaveStyle({ animation: 'pulse 2s infinite' });
    expect(container.querySelectorAll('path')).toHaveLength(5);
  });

  it('dims the icon when inactive', () => {
    const { container } = render(<SparklesIcon active={false} />);

    expect(container.querySelector('svg')).toHaveStyle({
      filter: 'grayscale(1)',
      opacity: '0.5',
    });
  });
});
