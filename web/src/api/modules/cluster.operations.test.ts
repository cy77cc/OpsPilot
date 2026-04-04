import { describe, expect, it } from 'vitest';
import { normalizeClusterOperationResponse } from './cluster';

describe('cluster operation envelope normalization', () => {
  it('normalizes completed state', () => {
    const normalized = normalizeClusterOperationResponse({
      state: 'completed',
      code: 'success',
      message: 'ok',
      audit_id: 12,
      data: { changed: true },
    });

    expect(normalized.state).toBe('completed');
    expect(normalized.success).toBe(true);
    expect(normalized.code).toBe('success');
    expect(normalized.audit_id).toBe(12);
    expect(normalized.result).toEqual({ changed: true });
  });

  it('normalizes approval required state from approval payload', () => {
    const normalized = normalizeClusterOperationResponse({
      state: 'approval_required',
      code: 'approval_required',
      approval: {
        required: true,
        ticket: 'k8s-appr-1',
        expires_at: '2026-04-03T12:00:00Z',
      },
      message: 'needs approval',
    });

    expect(normalized.state).toBe('approval_required');
    expect(normalized.success).toBe(false);
    expect(normalized.code).toBe('approval_required');
    expect(normalized.approval?.required).toBe(true);
    expect(normalized.approval?.ticket).toBe('k8s-appr-1');
    expect(normalized.approval?.expires_at).toBe('2026-04-03T12:00:00Z');
  });

  it('normalizes failed state with error code', () => {
    const normalized = normalizeClusterOperationResponse({
      state: 'failed',
      error_code: 'approval_token_replayed',
      error_message: 'approval token replayed',
      diagnostics: ['replayed token'],
    });

    expect(normalized.state).toBe('failed');
    expect(normalized.success).toBe(false);
    expect(normalized.code).toBe('approval_token_replayed');
    expect(normalized.error_code).toBe('approval_token_replayed');
    expect(normalized.diagnostics).toEqual(['replayed token']);
  });

  it('normalizes rejected state', () => {
    const normalized = normalizeClusterOperationResponse({
      status: 'rejected',
      message: 'permission denied',
      auditId: 'audit-9',
      approval: {
        required: true,
        reason: 'team policy',
      },
    });

    expect(normalized.state).toBe('rejected');
    expect(normalized.success).toBe(false);
    expect(normalized.code).toBe('approval_rejected');
    expect(normalized.audit_id).toBe('audit-9');
    expect(normalized.approval?.reason).toBe('team policy');
  });

  it('preserves legacy compatibility fields while normalizing canonical output', () => {
    const normalized = normalizeClusterOperationResponse({
      success: false,
      approval_required: true,
      approval_ticket: 'k8s-appr-legacy',
      approval_expires_at: '2026-04-03T12:00:00Z',
      error_code: 'approval_token_invalid',
      result: { recovered: false },
    });

    expect(normalized.state).toBe('approval_required');
    expect(normalized.success).toBe(false);
    expect(normalized.code).toBe('approval_required');
    expect(normalized.error_code).toBe('approval_required');
    expect(normalized.approval?.ticket).toBe('k8s-appr-legacy');
    expect(normalized.approval?.expires_at).toBe('2026-04-03T12:00:00Z');
    expect(normalized.result).toEqual({ recovered: false });
  });
});
