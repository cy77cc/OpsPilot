export type PendingRunMetadata = {
  runId: string;
  sessionId?: string;
  clientRequestId?: string;
  latestEventId?: string;
  approvalId?: string;
  approvalCallId?: string;
  status?: 'waiting_approval' | 'resuming' | 'running' | 'resume_failed_retryable';
  resumable?: boolean;
  messageId?: string;
  updatedAt?: string;
};

export function createPendingRunStore() {
  const memory = new Map<string, PendingRunMetadata>();

  return {
    upsert(item: PendingRunMetadata) {
      const previous = item.runId ? memory.get(item.runId) : undefined;
      const next: PendingRunMetadata = {
        ...(previous || {}),
        ...item,
        updatedAt: item.updatedAt || new Date().toISOString(),
      };
      memory.set(next.runId, next);
      return next;
    },
    get(runId: string) {
      return memory.get(runId) ?? null;
    },
    list() {
      return Array.from(memory.values()).sort((left, right) =>
        (right.updatedAt || '').localeCompare(left.updatedAt || ''),
      );
    },
    remove(runId: string) {
      memory.delete(runId);
    },
    clear() {
      memory.clear();
    },
  };
}

const store = createPendingRunStore();

export function listPendingRuns(): PendingRunMetadata[] {
  return store.list();
}

export function getPendingRun(runId: string): PendingRunMetadata | null {
  if (!runId) return null;
  return store.get(runId);
}

export function upsertPendingRun(partial: PendingRunMetadata): PendingRunMetadata {
  return store.upsert(partial);
}

export function removePendingRun(runId: string): void {
  if (!runId) return;
  store.remove(runId);
}

export function clearPendingRuns(): void {
  store.clear();
}
