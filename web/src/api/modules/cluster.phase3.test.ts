import { describe, expect, it } from 'vitest';
import { normalizePhase3OperationResponse } from './cluster.phase3';

describe('cluster phase3 response mapping', () => {
  it('maps suggest_only response as warning state', () => {
    const normalized = normalizePhase3OperationResponse({
      state: 'completed',
      code: 'success',
      audit_id: 9,
      data: {
        event_id: 123,
        mode: 'suggest_only',
        status: 'suggested',
      },
    });

    expect(normalized.state).toBe('completed');
    expect(normalized.code).toBe('success');
    expect(normalized.ui_state).toBe('warning');
  });

  it('maps approval_rejected response payload', () => {
    const normalized = normalizePhase3OperationResponse({
      state: 'failed',
      code: 'approval_rejected',
      message: 'request rejected by approver',
      audit_id: 19,
    });

    expect(normalized.state).toBe('rejected');
    expect(normalized.code).toBe('approval_rejected');
    expect(normalized.error_code).toBe('approval_rejected');
    expect(normalized.ui_state).toBe('error');
  });
});
