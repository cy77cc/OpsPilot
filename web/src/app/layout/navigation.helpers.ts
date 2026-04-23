import type { MenuPathEntry, MenuSection } from './navigation.types';

export const menuRouteOverrides: Record<string, string> = {
  '/services/all': '/services',
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
  if (pathname.startsWith('/jobs')) {return '/tasks';}
  if (pathname.startsWith('/k8s')) {return '/deployment';}
  if (pathname.startsWith('/deployment/overview')) {return '/deployment';}
  if (pathname.startsWith('/deployment/create')) {return '/deployment';}
  if (pathname.startsWith('/deployment/approvals')) {return '/deployment';}
  if (pathname.startsWith('/deployment/targets')) {return '/deployment/targets';}
  if (pathname.startsWith('/deployment/infrastructure/clusters')) {return '/deployment/infrastructure/clusters';}
  if (pathname.startsWith('/deployment/infrastructure/credentials')) {return '/deployment/infrastructure/clusters';}
  if (pathname.startsWith('/deployment/infrastructure/hosts')) {return '/deployment/infrastructure/hosts';}
  if (pathname.startsWith('/hosts')) {return '/deployment/infrastructure/hosts';}
  if (pathname.startsWith('/deployment/observability/metrics')) {return '/monitor';}
  if (pathname.startsWith('/deployment/observability/topology')) {return '/deployment/observability/topology';}
  if (pathname.startsWith('/deployment/observability/audit-logs')) {return '/deployment/observability/audit-logs';}
  if (pathname.startsWith('/deployment/observability/policies')) {return '/deployment/observability/policies';}
  if (pathname.startsWith('/deployment/observability/aiops')) {return '/deployment/observability/aiops';}
  if (pathname.startsWith('/monitoring')) {return '/monitor';}
  if (pathname.startsWith('/monitor')) {return '/monitor';}
  if (pathname.startsWith('/automation')) {return '/automation';}
  if (pathname.startsWith('/cicd')) {return '/cicd';}
  if (pathname.startsWith('/cmdb')) {return '/cmdb';}
  if (pathname.startsWith('/tools')) {return '/tools';}
  if (pathname.startsWith('/services')) {return '/services';}
  if (pathname.startsWith('/settings/ai-models')) {return '/settings/ai-models';}
  if (pathname.startsWith('/governance')) {return '/governance/users';}
  if (pathname.startsWith('/settings/users')) {return '/settings/users';}
  if (pathname.startsWith('/settings/roles')) {return '/settings/roles';}
  if (pathname.startsWith('/settings/permissions')) {return '/settings/permissions';}
  if (pathname.startsWith('/settings')) {return '/settings';}
  if (pathname.startsWith('/help')) {return '/help';}
  if (pathname.startsWith('/deployment/') && pathname !== '/deployment/targets') {return '/deployment';}
  return pathname;
}

export function getBreadcrumbItems(menuPath: MenuPathEntry[]): Array<{ title: string; path?: string }> {
  const items: Array<{ title: string; path?: string }> = [{ title: '首页', path: '/' }];
  for (const entry of menuPath) {
    items.push(entry.key ? { title: entry.title, path: entry.key } : { title: entry.title });
  }
  return items;
}
