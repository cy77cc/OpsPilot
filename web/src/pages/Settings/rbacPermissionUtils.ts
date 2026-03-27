import type { Permission } from '../../api/modules/rbac';

export interface PermissionGroup {
  key: string;
  label: string;
  permissions: Permission[];
}

export const getPermissionGroupKey = (permission: Permission): string => {
  const category = String(permission.category || '').trim();
  if (category) {
    return category;
  }
  const code = String(permission.code || '').trim();
  const prefix = code.split(':')[0];
  return prefix || 'other';
};

export const groupPermissions = (permissions: Permission[]): PermissionGroup[] => {
  const grouped = new Map<string, Permission[]>();
  for (const permission of permissions) {
    const key = getPermissionGroupKey(permission);
    const current = grouped.get(key) || [];
    current.push(permission);
    grouped.set(key, current);
  }

  return Array.from(grouped.entries())
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([key, items]) => ({
      key,
      label: key,
      permissions: items.sort((x, y) => x.code.localeCompare(y.code)),
    }));
};

export const filterPermissions = (permissions: Permission[], query: string): Permission[] => {
  const q = query.trim().toLowerCase();
  if (!q) {
    return permissions;
  }
  return permissions.filter((item) =>
    [item.code, item.name, item.description, item.category]
      .some((value) => String(value || '').toLowerCase().includes(q)),
  );
};

export const getFilteredPermissionCodes = (permissions: Permission[], query: string): string[] => {
  return filterPermissions(permissions, query).map((item) => item.code);
};

export const inverseSelection = (current: string[], scope: string[]): string[] => {
  const currentSet = new Set(current);
  const scopeSet = new Set(scope);
  const result = new Set(current);

  for (const code of scopeSet) {
    if (currentSet.has(code)) {
      result.delete(code);
    } else {
      result.add(code);
    }
  }

  return Array.from(result);
};

export const summarizePermissionChanges = (origin: string[], next: string[]): { added: number; removed: number } => {
  const originSet = new Set(origin);
  const nextSet = new Set(next);

  let added = 0;
  let removed = 0;

  for (const code of nextSet) {
    if (!originSet.has(code)) {
      added += 1;
    }
  }

  for (const code of originSet) {
    if (!nextSet.has(code)) {
      removed += 1;
    }
  }

  return { added, removed };
};
