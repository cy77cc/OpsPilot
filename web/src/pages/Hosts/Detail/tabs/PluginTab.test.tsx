import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import PluginTab from './PluginTab';

const mockHost = {
  id: '1',
  pluginInstances: [
    {
      pluginKey: 'opsagent',
      installedVersion: 'v1.0.0',
      installStatus: 'succeeded',
      runtimeStatus: 'online',
      healthStatus: 'healthy',
      lastSeenAt: '2026-05-22T10:00:00Z',
    },
  ],
} as any;

const mockHostNoPlugin = {
  id: '2',
  pluginInstances: [],
} as any;

describe('PluginTab', () => {
  it('renders plugin instances', () => {
    render(<PluginTab host={mockHost} />);
    expect(screen.getByText('opsagent')).toBeInTheDocument();
    expect(screen.getByText('v1.0.0')).toBeInTheDocument();
  });

  it('shows install button when no plugin installed', () => {
    render(<PluginTab host={mockHostNoPlugin} />);
    expect(screen.getByText('安装 Agent')).toBeInTheDocument();
  });

  it('shows uninstall button when plugin installed', () => {
    render(<PluginTab host={mockHost} />);
    const uninstallElements = screen.getAllByText('卸载');
    expect(uninstallElements.length).toBeGreaterThanOrEqual(1);
  });
});
