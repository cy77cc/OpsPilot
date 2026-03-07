import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AICopilotButton } from './AICopilotButton';

vi.mock('./CopilotSurface', () => ({
  default: () => {
    throw new Error('surface failed');
  },
}));

vi.mock('./hooks/useAutoScene', () => ({
  useAutoScene: () => ({
    scene: 'global',
    selectValue: 'global',
    setScene: vi.fn(),
    availableScenes: [{ key: 'global', label: '全局助手' }],
    isAuto: true,
  }),
}));

describe('AICopilotButton', () => {
  it('keeps surrounding UI visible when the AI surface fails locally', async () => {
    render(
      <div>
        <span>shell still works</span>
        <AICopilotButton />
      </div>,
    );

    fireEvent.click(screen.getByRole('button', { name: /AI Copilot/ }));

    expect(screen.getByText('shell still works')).toBeInTheDocument();
    expect(await screen.findByText('AI 助手暂时不可用')).toBeInTheDocument();
  });
});
