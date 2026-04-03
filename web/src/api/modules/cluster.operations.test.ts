import { describe, expect, it } from 'vitest';
import { normalizeClusterOperationResponse } from './cluster';

describe('cluster operation envelope normalization', () => {
  it('normalizes completed state', () => {
    const normalized = normalizeClusterOperationResponse({
      state: 'completed',
      message: 'ok',
      audit_id: 12,
      data: { changed: true },
    });

    expect(normalized.state).toBe('completed');
    expect(normalized.success).toBe(true);
    expect(normalized.audit_id).toBe(12);
    expect(normalized.result).toEqual({ changed: true });
  });

  it('normalizes approval required state from approval payload', () => {
    const normalized = normalizeClusterOperationResponse({
      approval: {
        required: true,
        ticket: 'k8s-appr-1',
      },
      message: 'needs approval',
    });

    expect(normalized.state).toBe('approval_required');
    expect(normalized.success).toBe(false);
    expect(normalized.approval?.required).toBe(true);
    expect(normalized.approval?.ticket).toBe('k8s-appr-1');
  });

  it('normalizes failed state with error code', () => {
    const normalized = normalizeClusterOperationResponse({
      error_code: 'approval_token_replayed',
      error_message: 'approval token replayed',
      diagnostics: ['replayed token'],
    });

    expect(normalized.state).toBe('failed');
    expect(normalized.success).toBe(false);
    expect(normalized.error_code).toBe('approval_token_replayed');
    expect(normalized.diagnostics).toEqual(['replayed token']);
  });
});
