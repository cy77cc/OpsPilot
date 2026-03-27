import { afterEach, describe, expect, it } from 'vitest';
import {
  clearPendingRuns,
  getPendingRun,
  listPendingRuns,
  removePendingRun,
  upsertPendingRun,
} from '../pendingRunStore';

describe('pendingRunStore', () => {
  afterEach(() => {
    clearPendingRuns();
    localStorage.clear();
  });

  it('persists and merges pending run metadata by run id', () => {
    upsertPendingRun({
      runId: 'run-1',
      sessionId: 'sess-1',
      clientRequestId: 'req-1',
      latestEventId: 'evt-1',
      status: 'waiting_approval',
      resumable: true,
    });

    upsertPendingRun({
      runId: 'run-1',
      latestEventId: 'evt-2',
      approvalId: 'approval-1',
      status: 'resume_failed_retryable',
      resumable: true,
    });

    expect(getPendingRun('run-1')).toEqual(expect.objectContaining({
      runId: 'run-1',
      sessionId: 'sess-1',
      clientRequestId: 'req-1',
      latestEventId: 'evt-2',
      approvalId: 'approval-1',
      status: 'resume_failed_retryable',
      resumable: true,
    }));
    expect(listPendingRuns()).toHaveLength(1);
  });

  it('removes pending runs and clears the persisted snapshot', () => {
    upsertPendingRun({
      runId: 'run-1',
      sessionId: 'sess-1',
      status: 'waiting_approval',
      resumable: true,
    });

    removePendingRun('run-1');

    expect(getPendingRun('run-1')).toBeNull();
    expect(listPendingRuns()).toEqual([]);
  });
});
