import { afterEach, describe, expect, it, vi } from 'vitest';
import apiService from '../api';
import { aiApi } from './ai';

const unsupportedMethods = [
  {
    name: 'getCurrentSession',
    args: ['scene-a'],
    endpoint: '/ai/sessions/current',
    verb: 'get',
  },
  {
    name: 'branchSession',
    args: ['session-a', { messageId: 'msg-a', title: 'Branch' }],
    endpoint: '/ai/sessions/session-a/branch',
    verb: 'post',
  },
  {
    name: 'updateSessionTitle',
    args: ['session-a', 'Renamed'],
    endpoint: '/ai/sessions/session-a',
    verb: 'patch',
  },
  {
    name: 'getCapabilities',
    args: [],
    endpoint: '/ai/capabilities',
    verb: 'get',
  },
  {
    name: 'getToolParamHints',
    args: ['deploy'],
    endpoint: '/ai/tools/deploy/params/hints',
    verb: 'get',
  },
  {
    name: 'previewTool',
    args: [{ tool: 'deploy' }],
    endpoint: '/ai/tools/preview',
    verb: 'post',
  },
  {
    name: 'executeTool',
    args: [{ tool: 'deploy' }],
    endpoint: '/ai/tools/execute',
    verb: 'post',
  },
  {
    name: 'getExecution',
    args: ['execution-a'],
    endpoint: '/ai/executions/execution-a',
    verb: 'get',
  },
  {
    name: 'submitFeedback',
    args: [{ is_effective: true }],
    endpoint: '/ai/feedback',
    verb: 'post',
  },
  {
    name: 'confirmConfirmation',
    args: ['confirm-a', true],
    endpoint: '/ai/confirmations/confirm-a/confirm',
    verb: 'post',
  },
  {
    name: 'getSceneTools',
    args: ['cluster'],
    endpoint: '/ai/scene/cluster/tools',
    verb: 'get',
  },
  {
    name: 'getScenePrompts',
    args: ['cluster'],
    endpoint: '/ai/scene/cluster/prompts',
    verb: 'get',
  },
  {
    name: 'getUsageStats',
    args: [{ scene: 'cluster' }],
    endpoint: '/ai/usage/stats',
    verb: 'get',
  },
  {
    name: 'getUsageLogs',
    args: [{ page: 1 }],
    endpoint: '/ai/usage/logs',
    verb: 'get',
  },
] as const;

describe('aiApi backend contract', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it.each(unsupportedMethods)('$name is blocked by the backend contract', async ({ name, args, endpoint, verb }) => {
    const verbSpy = vi.spyOn(apiService, verb as 'get' | 'post' | 'patch' | 'delete' | 'put');
    verbSpy.mockImplementation(() => {
      throw new Error(`apiService.${verb} should not be called for ${String(name)} (${endpoint})`);
    });

    await expect((aiApi as any)[name](...args)).rejects.toMatchObject({
      name: 'NotImplementedByBackendError',
    });
    await expect((aiApi as any)[name](...args)).rejects.toThrow(endpoint);
  });
});
