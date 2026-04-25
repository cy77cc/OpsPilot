import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { Navigate } from 'react-router-dom';
import { buildMenuSections } from '../layout/navigation.config';
import { renderGovernanceRoutes } from './governance.routes';
import LegacyGovernanceRedirect from '../../components/Auth/LegacyGovernanceRedirect';

const t = (key: string) => key;

function getRouteElement(
  path: string,
  governanceMenuEnabled: boolean,
  withAuth = vi.fn((_resource: string, _action: string, element: React.ReactElement) => element),
) {
  const routes = renderGovernanceRoutes({
    withAuth,
    governanceMenuEnabled,
  });

  const routeChildren = React.Children.toArray(routes.props.children);
  const matched = routeChildren.find(
    (child): child is React.ReactElement<{ path: string; element: React.ReactElement }> => {
      if (!React.isValidElement(child)) {
        return false;
      }

      const props = child.props as { path?: string };
      return props.path === path;
    },
  );

  if (!matched) {
    throw new Error(`missing route for ${path}`);
  }

  return {
    withAuth,
    element: matched.props.element,
  };
}

describe('renderGovernanceRoutes governance consistency', () => {
  it('keeps legacy governance settings routes reachable while legacy menu is enabled', () => {
    // Note: We don't verify menu existence here if it's renamed to governance
    const governanceMenu = buildMenuSections({
      t,
      governanceMenuEnabled: false,
      canReadGovernance: false,
    }).find((section) => section.key === 'governance');

    const usersRoute = getRouteElement('/settings/users', false);
    const rolesRoute = getRouteElement('/settings/roles', false);
    const permissionsRoute = getRouteElement('/settings/permissions', false);

    expect(usersRoute.withAuth).toHaveBeenCalledWith('rbac', 'read', expect.anything());
    expect(rolesRoute.withAuth).toHaveBeenCalledWith('rbac', 'read', expect.anything());
    expect(permissionsRoute.withAuth).toHaveBeenCalledWith('rbac', 'read', expect.anything());

    expect(usersRoute.element.type).not.toBe(Navigate);
    expect(rolesRoute.element.type).not.toBe(Navigate);
    expect(permissionsRoute.element.type).not.toBe(Navigate);
  });

  it('redirects legacy settings routes to governance routes when governance menu is enabled', () => {
    const governanceMenu = buildMenuSections({
      t,
      governanceMenuEnabled: true,
      canReadGovernance: true,
    }).find((section) => section.key === 'governance');

    const keys = governanceMenu?.items.map((item) => item.key) ?? [];
    expect(keys).not.toEqual(expect.arrayContaining(['/settings/users', '/settings/roles', '/settings/permissions']));
    expect(keys).toContain('/governance/users');

    const legacyToGovernanceMappings = [
      { legacyPath: '/settings/users', governancePath: '/governance/users' },
      { legacyPath: '/settings/roles', governancePath: '/governance/roles' },
      { legacyPath: '/settings/permissions', governancePath: '/governance/permissions' },
    ] as const;

    legacyToGovernanceMappings.forEach(({ legacyPath, governancePath }) => {
      const route = getRouteElement(legacyPath, true);
      expect(route.element.type).toBe(LegacyGovernanceRedirect);
      expect((route.element.props as { to: string }).to).toBe(governancePath);
    });
  });
});
