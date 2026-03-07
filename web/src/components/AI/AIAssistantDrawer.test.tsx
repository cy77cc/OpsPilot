import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AIAssistantDrawer } from './AIAssistantDrawer';

vi.mock('./CopilotSurface', () => ({
  default: () => {
    throw new Error('surface failed');
  },
}));

describe('AIAssistantDrawer', () => {
  it('shows a local fallback when the AI surface fails', async () => {
    render(
      <AIAssistantDrawer
        open
        onClose={vi.fn()}
        scene="global"
      />,
    );

    expect(await screen.findByText('AI 助手暂时不可用')).toBeInTheDocument();
    expect(await screen.findByText('你仍然可以继续使用当前页面，其它功能不受影响。')).toBeInTheDocument();
  });
});
