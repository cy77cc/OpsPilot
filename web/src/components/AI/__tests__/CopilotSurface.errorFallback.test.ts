import { describe, expect, it } from 'vitest';
import { buildAssistantErrorContent } from '../CopilotSurface';

describe('buildAssistantErrorContent', () => {
  it('preserves streamed assistant content when an error arrives', () => {
    const content = buildAssistantErrorContent('partial answer', 'stream failed');

    expect(content).toContain('partial answer');
    expect(content).toContain('stream failed');
  });

  it('falls back to plain error text when there is no streamed content yet', () => {
    expect(buildAssistantErrorContent('', 'stream failed')).toBe('stream failed');
  });
});
