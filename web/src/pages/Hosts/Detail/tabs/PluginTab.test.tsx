import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import PluginTab from './PluginTab';

describe('PluginTab', () => {
  it('renders plugin instances', () => {
    render(
      <PluginTab
        host={{
          id: '1',
          name: 'host-a',
          ip: '10.0.0.8',
          status: 'online',
          cpu: 4,
          memory: 8192,
          disk: 200,
          network: 0,
          tags: [],
          region: '',
          createdAt: new Date().toISOString(),
          lastActive: new Date().toISOString(),
          pluginInstances: [{
            pluginKey: 'opsagent',
            installedVersion: 'nodeagentx-dc57fbc-dirty',
            installStatus: 'succeeded',
            runtimeStatus: 'online',
            healthStatus: 'healthy',
            lastSeenAt: new Date().toISOString(),
          }],
        }}
      />
    );

    expect(screen.getByText('opsagent')).toBeInTheDocument();
    expect(screen.getByText('nodeagentx-dc57fbc-dirty')).toBeInTheDocument();
    expect(screen.getByText('succeeded')).toBeInTheDocument();
  });
});
