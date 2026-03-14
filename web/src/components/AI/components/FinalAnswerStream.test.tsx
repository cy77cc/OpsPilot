import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { FinalAnswerStream } from './FinalAnswerStream';

describe('FinalAnswerStream', () => {
  it('stays hidden until visible is enabled', () => {
    const { rerender } = render(
      <FinalAnswerStream content="最终答案" visible={false} streaming />
    );

    expect(screen.queryByText('最终答案')).not.toBeInTheDocument();

    rerender(<FinalAnswerStream content="最终答案" visible streaming={false} reducedMotion />);

    expect(screen.getByText('最终答案')).toBeInTheDocument();
  });

  it('renders streaming state with a loading indicator', () => {
    render(
      <FinalAnswerStream content="正在生成最终答案" visible streaming reducedMotion />
    );

    expect(screen.getByText('正在生成最终答案')).toBeInTheDocument();
    expect(screen.getByText('正在生成最终答案')).toBeInTheDocument();
    expect(screen.getByLabelText('final-answer-streaming')).toBeInTheDocument();
  });
});
