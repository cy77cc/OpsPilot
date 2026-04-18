import type { ApiResponse } from '../../../api/api';
// Re-use types from shared.ts to avoid duplicate exports
import type { ApprovalTicket, SubmitApprovalPayload, SubmitApprovalResult } from './shared';

// Re-export types needed by tests
export type { ApprovalTicket, SubmitApprovalPayload, SubmitApprovalResult };

/**
 * NotImplementedByBackendError is thrown when calling an API method
 * that the backend has not implemented yet.
 */
export class NotImplementedByBackendError extends Error {
  constructor(operation: string, endpoint: string) {
    super(`Backend has not implemented ${operation} (${endpoint})`);
    this.name = 'NotImplementedByBackendError';
  }
}

function notImplementedByBackend(operation: string, endpoint: string): never {
  throw new NotImplementedByBackendError(operation, endpoint);
}

// Type definitions for stub methods (used by tests)
export interface AIScenePromptsPayload {
  scene: string;
  prompts: Array<{ key: string; label: string; content: string }>;
}

export interface AISceneToolsPayload {
  scene: string;
  tools: string[];
}

export interface AICapability {
  name: string;
  description: string;
  enabled: boolean;
}

export interface AIToolParamHints {
  name: string;
  hints: Record<string, unknown>;
}

export interface AIToolExecution {
  id: string;
  status: string;
  result?: Record<string, unknown>;
}

export interface ConfirmationTicket {
  id: string;
  status: string;
}

export interface KnowledgeEntry {
  id: string;
  content: string;
}

export interface AIKnowledgeFeedbackPayload {
  is_effective: boolean;
  content?: string;
}

export interface UsageStatsParams {
  scene?: string;
  start?: string;
  end?: string;
}

export interface UsageStats {
  total_sessions: number;
  total_messages: number;
}

export interface UsageLogsParams {
  page?: number;
  limit?: number;
}

export interface UsageLogsResult {
  logs: Array<Record<string, unknown>>;
  total: number;
}

export interface AISessionBranchParams {
  messageId: string;
  title?: string;
}

/**
 * Stub API methods that throw NotImplementedByBackendError.
 * These are methods the backend has not implemented yet but
 * tests may reference them.
 */
export const stubApi = {
  async getCurrentSession(_scene?: string): Promise<ApiResponse<unknown>> {
    notImplementedByBackend('getCurrentSession', '/ai/sessions/current');
  },

  async branchSession(id: string, _params?: AISessionBranchParams): Promise<ApiResponse<unknown>> {
    notImplementedByBackend('branchSession', `/ai/sessions/${id}/branch`);
  },

  async updateSessionTitle(id: string, _title: string): Promise<ApiResponse<unknown>> {
    notImplementedByBackend('updateSessionTitle', `/ai/sessions/${id}`);
  },

  async getCapabilities(): Promise<ApiResponse<AICapability[]>> {
    notImplementedByBackend('getCapabilities', '/ai/capabilities');
  },

  async getToolParamHints(name: string): Promise<ApiResponse<AIToolParamHints>> {
    notImplementedByBackend('getToolParamHints', `/ai/tools/${name}/params/hints`);
  },

  async previewTool(_params: { tool: string; params?: Record<string, unknown> }): Promise<ApiResponse<Record<string, unknown>>> {
    notImplementedByBackend('previewTool', '/ai/tools/preview');
  },

  async executeTool(_params: { tool: string; params?: Record<string, unknown>; approval_token?: string; checkpoint_id?: string }): Promise<ApiResponse<AIToolExecution>> {
    notImplementedByBackend('executeTool', '/ai/tools/execute');
  },

  async getExecution(id: string): Promise<ApiResponse<AIToolExecution>> {
    notImplementedByBackend('getExecution', `/ai/executions/${id}`);
  },

  async submitFeedback(_payload: AIKnowledgeFeedbackPayload): Promise<ApiResponse<KnowledgeEntry | null>> {
    notImplementedByBackend('submitFeedback', '/ai/feedback');
  },

  async confirmConfirmation(id: string, _approve: boolean): Promise<ApiResponse<ConfirmationTicket>> {
    notImplementedByBackend('confirmConfirmation', `/ai/confirmations/${id}/confirm`);
  },

  async getSceneTools(scene: string): Promise<ApiResponse<AISceneToolsPayload>> {
    notImplementedByBackend('getSceneTools', `/ai/scene/${scene}/tools`);
  },

  async getScenePrompts(scene: string): Promise<ApiResponse<AIScenePromptsPayload>> {
    notImplementedByBackend('getScenePrompts', `/ai/scene/${scene}/prompts`);
  },

  async getUsageStats(_params?: UsageStatsParams): Promise<ApiResponse<UsageStats>> {
    notImplementedByBackend('getUsageStats', '/ai/usage/stats');
  },

  async getUsageLogs(_params?: UsageLogsParams): Promise<ApiResponse<UsageLogsResult>> {
    notImplementedByBackend('getUsageLogs', '/ai/usage/logs');
  },

  /**
   * List unsupported methods for contract testing.
   * Returns empty array as all stubs are now properly defined.
   */
  listUnsupportedMethods(): string[] {
    return [];
  },
};