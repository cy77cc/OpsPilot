import { vi } from 'vitest';

/**
 * Creates a mock API object with all methods mocked.
 * Use this to mock the Api import in tests.
 *
 * @example
 * ```tsx
 * const mockApi = createMockApi();
 * vi.mock('../../api', () => ({ Api: mockApi }));
 *
 * mockApi.deployment.getTargets.mockResolvedValue({ data: { list: [] } });
 * ```
 */
export function createMockApi() {
  return {
    // Auth
    auth: {
      login: vi.fn(),
      logout: vi.fn(),
      refresh: vi.fn(),
      getCurrentUser: vi.fn(),
    },

    // Deployment
    deployment: {
      getTargets: vi.fn(),
      getTarget: vi.fn(),
      createTarget: vi.fn(),
      updateTarget: vi.fn(),
      deleteTarget: vi.fn(),
      getReleases: vi.fn(),
      getReleasesByRuntime: vi.fn(),
      getRelease: vi.fn(),
      previewRelease: vi.fn(),
      applyRelease: vi.fn(),
      approveRelease: vi.fn(),
      rejectRelease: vi.fn(),
      rollbackRelease: vi.fn(),
      getReleaseTimeline: vi.fn(),
      listInspections: vi.fn(),
      runInspection: vi.fn(),
      getGovernance: vi.fn(),
      putGovernance: vi.fn(),
      getClusterBootstrapTask: vi.fn(),
      previewClusterBootstrap: vi.fn(),
      applyClusterBootstrap: vi.fn(),
      importCredential: vi.fn(),
      testCredential: vi.fn(),
      listCredentials: vi.fn(),
      deleteCredential: vi.fn(),
    },

    // Kubernetes
    kubernetes: {
      getClusterList: vi.fn(),
      getCluster: vi.fn(),
      createCluster: vi.fn(),
      updateCluster: vi.fn(),
      deleteCluster: vi.fn(),
      getNodes: vi.fn(),
      getPods: vi.fn(),
      getServices: vi.fn(),
      getDeployments: vi.fn(),
      getNamespaces: vi.fn(),
    },

    // Hosts
    hosts: {
      getHostList: vi.fn(),
      getHost: vi.fn(),
      createHost: vi.fn(),
      updateHost: vi.fn(),
      deleteHost: vi.fn(),
      probeHost: vi.fn(),
    },

    // Services
    services: {
      getList: vi.fn(),
      get: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
    },

    // CICD
    cicd: {
      getPipelines: vi.fn(),
      getPipeline: vi.fn(),
      createPipeline: vi.fn(),
      updatePipeline: vi.fn(),
      deletePipeline: vi.fn(),
      triggerPipeline: vi.fn(),
      getRuns: vi.fn(),
      getRun: vi.fn(),
      cancelRun: vi.fn(),
    },

    // Monitoring
    monitoring: {
      getAlertRules: vi.fn(),
      createAlertRule: vi.fn(),
      updateAlertRule: vi.fn(),
      deleteAlertRule: vi.fn(),
      getAlertEvents: vi.fn(),
      getMetrics: vi.fn(),
    },

    // RBAC
    rbac: {
      getRoles: vi.fn(),
      createRole: vi.fn(),
      updateRole: vi.fn(),
      deleteRole: vi.fn(),
      getPermissions: vi.fn(),
      assignRole: vi.fn(),
      removeRole: vi.fn(),
    },

    // AI
    ai: {
      chat: vi.fn(),
      streamChat: vi.fn(),
      getSessions: vi.fn(),
      getSession: vi.fn(),
      deleteSession: vi.fn(),
    },

    // Notification
    notification: {
      getList: vi.fn(),
      markRead: vi.fn(),
      markAllRead: vi.fn(),
      dismiss: vi.fn(),
    },

    // Topology
    topology: {
      get: vi.fn(),
      refresh: vi.fn(),
    },

    // CMDB
    cmdb: {
      getCIs: vi.fn(),
      getCI: vi.fn(),
      createCI: vi.fn(),
      updateCI: vi.fn(),
      deleteCI: vi.fn(),
      getRelations: vi.fn(),
      createRelation: vi.fn(),
      deleteRelation: vi.fn(),
    },

    // Projects
    projects: {
      getList: vi.fn(),
      get: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
    },

    // Automation
    automation: {
      getPlaybooks: vi.fn(),
      createPlaybook: vi.fn(),
      updatePlaybook: vi.fn(),
      deletePlaybook: vi.fn(),
      executePlaybook: vi.fn(),
      getExecutions: vi.fn(),
    },

    // Tasks
    tasks: {
      getList: vi.fn(),
      get: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
    },

    // Tools
    tools: {
      getList: vi.fn(),
      execute: vi.fn(),
    },

    // Configs
    configs: {
      get: vi.fn(),
      update: vi.fn(),
    },
  };
}

/**
 * Creates a mock localStorage for testing.
 */
export function createMockLocalStorage() {
  let store: Record<string, string> = {};

  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
    get store() {
      return { ...store };
    },
  };
}

/**
 * Sets up mock localStorage for tests.
 */
export function setupMockLocalStorage() {
  const mockStorage = createMockLocalStorage();

  Object.defineProperty(window, 'localStorage', {
    value: mockStorage,
    writable: true,
  });

  return mockStorage;
}

/**
 * Creates mock fetch response.
 */
export function createMockResponse<T>(data: T, status = 200) {
  return {
    data: {
      success: true,
      data,
    },
    status,
    statusText: 'OK',
    headers: {},
    config: {},
  };
}

/**
 * Creates mock API error response.
 */
export function createMockError(message: string, statusCode = 500, businessCode?: number) {
  const error = new Error(message);
  (error as any).statusCode = statusCode;
  (error as any).businessCode = businessCode;
  return error;
}
