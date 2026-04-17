import { describe, expect, it } from 'vitest';
import { createPendingRunStore } from '../pendingRunStore';

describe('pendingRunStore', () => {
  it('stores pending runs in memory only', () => {
    const store = createPendingRunStore();
    store.upsert({ runId: 'r1', updatedAt: '2026-04-17T00:00:00Z' });
    expect(store.get('r1')?.runId).toBe('r1');
  });
});
