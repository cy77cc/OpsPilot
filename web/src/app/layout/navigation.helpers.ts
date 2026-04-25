import type { MenuPathEntry, MenuSection } from './navigation.types';

export const menuRouteOverrides: Record<string, string> = {
  '/delivery/services/all': '/delivery/services',
};

export function findSectionPath(sections: MenuSection[], targetKey: string): MenuPathEntry[] {
  for (const section of sections) {
    const item = section.items.find((entry) => entry.key === targetKey);
    if (item) {
      return [{ title: section.title }, { title: item.label, key: item.key }];
    }
  }
  return [];
}

export function getActiveMenuKey(pathname: string): string {
  // Observability
  if (pathname.startsWith('/observability/jobs')) return '/observability/tasks';
  if (pathname.startsWith('/observability/tasks')) return '/observability/tasks';
  if (pathname.startsWith('/observability/monitor')) return '/observability/monitor';
  if (pathname.startsWith('/observability/metrics')) return '/observability/metrics';
  if (pathname.startsWith('/observability/topology')) return '/observability/topology';
  if (pathname.startsWith('/observability/cmdb')) return '/observability/cmdb';
  if (pathname.startsWith('/observability/aiops')) return '/observability/aiops';

  // Delivery
  if (pathname.startsWith('/delivery/deployments')) return '/delivery/deployments';
  if (pathname.startsWith('/delivery/services')) return '/delivery/services';
  if (pathname.startsWith('/delivery/targets')) return '/delivery/targets';
  if (pathname.startsWith('/delivery/automation')) return '/delivery/automation';
  if (pathname.startsWith('/delivery/cicd')) return '/delivery/cicd';

  // Resources
  if (pathname.startsWith('/resources/clusters')) return '/resources/clusters';
  if (pathname.startsWith('/resources/hosts')) return '/resources/hosts';
  if (pathname.startsWith('/resources/credentials')) return '/resources/clusters';
  if (pathname.startsWith('/resources/projects')) return '/resources/projects';
  if (pathname.startsWith('/resources/nodes')) return '/resources/nodes';

  // Governance
  if (pathname.startsWith('/governance/org')) return '/governance/org';
  if (pathname.startsWith('/governance/approvals')) return '/governance/approvals';
  if (pathname.startsWith('/governance/audit-logs')) return '/governance/audit-logs';
  if (pathname.startsWith('/governance/users')) return '/governance/users';
  if (pathname.startsWith('/governance/roles')) return '/governance/roles';
  if (pathname.startsWith('/governance/permissions')) return '/governance/permissions';

  // AI
  if (pathname.startsWith('/ai/chat')) return '/ai/chat';
  if (pathname.startsWith('/ai/settings/models')) return '/ai/settings/models';
  if (pathname.startsWith('/ai/models')) return '/ai/models';
  if (pathname.startsWith('/ai/usage')) return '/ai/usage';

  // Legacy/Common
  if (pathname.startsWith('/tools')) return '/tools';
  if (pathname.startsWith('/settings')) return '/settings';
  if (pathname.startsWith('/help')) return '/help';
  
  return pathname;
}

export function getBreadcrumbItems(menuPath: MenuPathEntry[]): Array<{ title: string; path?: string }> {
  const items: Array<{ title: string; path?: string }> = [{ title: '首页', path: '/' }];
  for (const entry of menuPath) {
    items.push(entry.key ? { title: entry.title, path: entry.key } : { title: entry.title });
  }
  return items;
}
