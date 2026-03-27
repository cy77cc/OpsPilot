import type {
  A2UIErrorEvent,
  A2UIMetaEvent,
  A2UIRunStateEvent,
  A2UIToolApprovalEvent,
  AIMessage,
  AIChatParams,
} from '../../../api/modules/ai';
import { aiApi } from '../../../api/modules/ai';
import {
  getPendingRun,
  listPendingRuns,
  removePendingRun,
  upsertPendingRun,
} from '../pendingRunStore';
import type { PendingRunMetadata } from '../types';

const REATTACH_DELAY_MS = 300;
const APPROVAL_SIGNAL_WAIT_TIMEOUT_MS = 5000;

function createTriggerId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `retry-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function isReconnectableStatus(status?: string): status is PendingRunMetadata['status'] {
  return status === 'waiting_approval' || status === 'resuming' || status === 'running' || status === 'resume_failed_retryable';
}

export function extractPendingRunFromMessage(message?: Partial<AIMessage> | null): PendingRunMetadata | null {
  if (!message?.run_id || !message.resumable || !isReconnectableStatus(message.status)) {
    return null;
  }

  return {
    runId: message.run_id,
    sessionId: undefined,
    clientRequestId: message.client_request_id,
    latestEventId: message.latest_event_id,
    approvalId: message.approval_id,
    approvalCallId: undefined,
    status: message.status,
    resumable: true,
    messageId: message.id,
  };
}

interface RunReconnectControllerConfig {
  onPendingRunChange?: (pendingRun: PendingRunMetadata | null) => void;
}

export class RunReconnectController {
  private pendingRun: PendingRunMetadata | null = null;
  private lastApprovalUpdate: { token?: string; status?: string } | null = null;
  private readonly onPendingRunChange?: (pendingRun: PendingRunMetadata | null) => void;
  private readonly handleApprovalUpdated = (event: Event) => {
    this.lastApprovalUpdate = (event as CustomEvent<{ token?: string; status?: string }>).detail || {};
  };

  constructor(config: RunReconnectControllerConfig = {}) {
    this.onPendingRunChange = config.onPendingRunChange;
    if (typeof window !== 'undefined') {
      window.addEventListener('ai-approval-updated', this.handleApprovalUpdated as EventListener);
    }
  }

  seedHistoricalMessages(messages: Array<Partial<AIMessage>>): PendingRunMetadata[] {
    const pendingRuns = messages
      .map((message) => extractPendingRunFromMessage(message))
      .filter((message): message is PendingRunMetadata => Boolean(message));

    pendingRuns.forEach((pendingRun) => {
      this.pendingRun = upsertPendingRun(pendingRun);
    });

    return pendingRuns;
  }

  begin(params: AIChatParams): void {
    if (params.clientRequestId) {
      const persisted = this.findPendingRunByClientRequestId(params.clientRequestId);
      if (persisted) {
        this.pendingRun = persisted;
        this.onPendingRunChange?.(persisted);
      }
    }
  }

  dispose(): void {
    if (typeof window !== 'undefined') {
      window.removeEventListener('ai-approval-updated', this.handleApprovalUpdated as EventListener);
    }
  }

  handleMeta(payload: A2UIMetaEvent, params: AIChatParams): void {
    const existing = payload.run_id ? getPendingRun(payload.run_id) : null;
    if (existing) {
      this.pendingRun = existing;
      this.onPendingRunChange?.(existing);
      return;
    }

    this.pendingRun = {
      runId: payload.run_id,
      sessionId: payload.session_id,
      clientRequestId: params.clientRequestId,
      latestEventId: params.lastEventId,
      approvalId: undefined,
      approvalCallId: undefined,
      status: 'running',
      resumable: false,
    };
    this.onPendingRunChange?.(this.pendingRun);
  }

  handleEventId(eventId: string): void {
    if (!this.pendingRun) {
      return;
    }
    this.pendingRun = upsertPendingRun({
      ...this.pendingRun,
      latestEventId: eventId,
      resumable: true,
    });
    this.onPendingRunChange?.(this.pendingRun);
  }

  handleToolApproval(payload: A2UIToolApprovalEvent): void {
    if (!this.pendingRun?.runId) {
      return;
    }

    this.pendingRun = upsertPendingRun({
      ...this.pendingRun,
      approvalId: payload.approval_id,
      approvalCallId: payload.call_id,
    });
    this.onPendingRunChange?.(this.pendingRun);
  }

  handleRunState(payload: A2UIRunStateEvent): void {
    if (!isReconnectableStatus(payload.status)) {
      this.clear(payload.run_id);
      return;
    }

    this.pendingRun = upsertPendingRun({
      runId: payload.run_id,
      sessionId: this.pendingRun?.sessionId,
      clientRequestId: this.pendingRun?.clientRequestId,
      latestEventId: this.pendingRun?.latestEventId,
      approvalId: this.pendingRun?.approvalId,
      approvalCallId: this.pendingRun?.approvalCallId,
      status: payload.status,
      resumable: true,
      messageId: this.pendingRun?.messageId,
    });
    this.onPendingRunChange?.(this.pendingRun);
  }

  handleTerminalError(_payload: A2UIErrorEvent): void {
    if (this.pendingRun?.runId) {
      this.clear(this.pendingRun.runId);
    }
  }

  handleDone(runId?: string): void {
    if (runId) {
      this.clear(runId);
      return;
    }
    if (this.pendingRun?.runId) {
      this.clear(this.pendingRun.runId);
    }
  }

  async nextAttempt(signal?: AbortSignal): Promise<AIChatParams | null> {
    if (!this.pendingRun || !this.pendingRun.resumable) {
      return null;
    }

    const run = this.pendingRun;
    if (run.status === 'resuming' || run.status === 'running') {
      await this.waitForDelay(signal);
      return this.toRetryParams(run);
    }

    const detail = this.consumeApprovalUpdate(run) || await this.waitForApprovalUpdate(run, signal);
    if (!detail) {
      return null;
    }

    if (run.status === 'resume_failed_retryable' && run.approvalId && detail.status === 'approved') {
      await aiApi.retryResumeApproval(run.approvalId, {
        trigger_id: createTriggerId(),
      });
    }

    return this.toRetryParams(run);
  }

  private toRetryParams(run: PendingRunMetadata): AIChatParams {
    return {
      message: '',
      sessionId: run.sessionId,
      clientRequestId: run.clientRequestId,
      lastEventId: run.latestEventId,
    };
  }

  private clear(runId: string): void {
    removePendingRun(runId);
    if (this.pendingRun?.runId === runId) {
      this.pendingRun = null;
      this.onPendingRunChange?.(null);
    }
  }

  private async waitForDelay(signal?: AbortSignal): Promise<void> {
    await new Promise<void>((resolve) => {
      const timer = window.setTimeout(() => resolve(), REATTACH_DELAY_MS);
      signal?.addEventListener('abort', () => {
        window.clearTimeout(timer);
        resolve();
      }, { once: true });
    });
  }

  private async waitForApprovalUpdate(
    run: PendingRunMetadata,
    signal?: AbortSignal,
  ): Promise<{ token?: string; status?: string } | null> {
    const buffered = this.consumeApprovalUpdate(run);
    if (buffered) {
      return buffered;
    }

    return new Promise((resolve) => {
      const timeout = window.setTimeout(() => {
        cleanup();
        resolve({});
      }, APPROVAL_SIGNAL_WAIT_TIMEOUT_MS);

      const cleanup = () => {
        window.clearTimeout(timeout);
        window.removeEventListener('ai-approval-updated', handleEvent as EventListener);
        signal?.removeEventListener('abort', handleAbort);
      };

      const handleAbort = () => {
        cleanup();
        resolve(null);
      };

      const handleEvent = (event: Event) => {
        const detail = (event as CustomEvent<{ token?: string; status?: string }>).detail || {};
        if (!matchesRunApprovalSignal(run, detail.token)) {
          return;
        }
        cleanup();
        resolve(detail);
      };

      window.addEventListener('ai-approval-updated', handleEvent as EventListener);
      signal?.addEventListener('abort', handleAbort, { once: true });
    });
  }

  private consumeApprovalUpdate(
    run: PendingRunMetadata,
  ): { token?: string; status?: string } | null {
    if (!this.lastApprovalUpdate) {
      return null;
    }
    if (!matchesRunApprovalSignal(run, this.lastApprovalUpdate.token)) {
      return null;
    }
    const detail = this.lastApprovalUpdate;
    this.lastApprovalUpdate = null;
    return detail;
  }

  private findPendingRunByClientRequestId(clientRequestId: string): PendingRunMetadata | null {
    if (!clientRequestId) {
      return null;
    }

    const candidates = [this.pendingRun, ...listPendingRuns()]
      .filter((item): item is PendingRunMetadata => Boolean(item));
    return candidates.find((item) => item.clientRequestId === clientRequestId) || null;
  }
}

function matchesRunApprovalSignal(run: PendingRunMetadata, token?: string): boolean {
  const normalized = String(token || '').trim();
  if (!normalized) {
    return false;
  }
  return normalized === run.approvalId || normalized === run.approvalCallId;
}
