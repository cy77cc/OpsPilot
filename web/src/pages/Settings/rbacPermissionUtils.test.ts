import { describe, expect, it } from 'vitest';
import {
  getFilteredPermissionCodes,
  groupPermissions,
  inverseSelection,
  summarizePermissionChanges,
} from './rbacPermissionUtils';

const permissions = [
  { id: '1', code: 'rbac:read', name: '读取权限', description: '', category: 'rbac', createdAt: '' },
  { id: '2', code: 'rbac:write', name: '写入权限', description: '', category: 'rbac', createdAt: '' },
  { id: '3', code: 'host:read', name: '主机读取', description: '', category: 'host', createdAt: '' },
];

describe('rbacPermissionUtils', () => {
  it('groups permissions by category', () => {
    const grouped = groupPermissions(permissions);
    expect(grouped).toHaveLength(2);
    expect(grouped[0].key).toBe('host');
    expect(grouped[1].key).toBe('rbac');
  });

  it('returns filtered permission codes by query', () => {
    const codes = getFilteredPermissionCodes(permissions, 'rbac');
    expect(codes).toEqual(['rbac:read', 'rbac:write']);
  });

  it('inverses scoped selection', () => {
    const next = inverseSelection(['rbac:read'], ['rbac:read', 'rbac:write']);
    expect(next.sort()).toEqual(['rbac:write']);
  });

  it('summarizes added/removed permission counts', () => {
    const summary = summarizePermissionChanges(['rbac:read', 'host:read'], ['rbac:read', 'rbac:write']);
    expect(summary).toEqual({ added: 1, removed: 1 });
  });
});
