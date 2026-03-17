import { render, screen } from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { AISurfaceBoundary } from '../AISurfaceBoundary';

function Thrower(): React.JSX.Element {
  throw new Error('boom');
}

describe('AISurfaceBoundary', () => {
  it('renders a local fallback when the AI surface crashes', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);

    render(
      <AISurfaceBoundary>
        <Thrower />
      </AISurfaceBoundary>,
    );

    expect(screen.getByTestId('ai-surface-fallback')).toBeInTheDocument();
    consoleError.mockRestore();
  });
});
