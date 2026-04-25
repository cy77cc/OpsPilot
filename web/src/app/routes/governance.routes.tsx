import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import LegacyGovernanceRedirect from '../../components/Auth/LegacyGovernanceRedirect';
import { LEGACY_GOVERNANCE_MENU_ITEMS } from '../layout/navigation.config';
import {
  ToolsPage,
  SettingsPage,
  UsersPage,
  RolesPage,
  PermissionsPage,
  AccessControlPage,
  ApprovalCenterPage,
  DeploymentAuditLogsPage,
  HelpCenterPage,
} from './pages';

interface GovernanceRoutesProps {
  withAuth: WithAuth;
  governanceMenuEnabled: boolean;
}

type LegacyGovernancePath = (typeof LEGACY_GOVERNANCE_MENU_ITEMS)[number]['key'];

const LEGACY_GOVERNANCE_PAGES: Record<LegacyGovernancePath, React.ReactElement> = {
  '/settings/users': <UsersPage />,
  '/settings/roles': <RolesPage />,
  '/settings/permissions': <PermissionsPage />,
};

export function renderGovernanceRoutes({ withAuth, governanceMenuEnabled }: GovernanceRoutesProps) {
  return (
    <>
      <Route path="/tools" element={<ToolsPage />} />
      <Route path="/tools/nightingale" element={<ToolsPage />} />
      <Route path="/tools/jenkins" element={<ToolsPage />} />
      <Route path="/tools/jumpserver" element={<ToolsPage />} />
      <Route path="/tools/kuboard" element={<ToolsPage />} />
      <Route path="/tools/archery" element={<ToolsPage />} />
      <Route path="/settings" element={<SettingsPage />} />
      
      <Route path="/governance/org" element={withAuth('rbac', 'read', <AccessControlPage />)} />
      <Route path="/governance/approvals" element={withAuth('deploy:target', 'read', <ApprovalCenterPage />)} />
      <Route path="/governance/audit-logs" element={withAuth('deploy:target', 'read', <DeploymentAuditLogsPage />)} />
      <Route path="/governance/users" element={withAuth('rbac', 'read', <UsersPage />)} />
      <Route path="/governance/roles" element={withAuth('rbac', 'read', <RolesPage />)} />
      <Route path="/governance/permissions" element={withAuth('rbac', 'read', <PermissionsPage />)} />

      {LEGACY_GOVERNANCE_MENU_ITEMS.map(({ key: legacyPath }) => (
        <Route
          key={legacyPath}
          path={legacyPath}
          element={
            governanceMenuEnabled
              ? <LegacyGovernanceRedirect to={legacyPath.replace('/settings/', '/governance/')} />
              : withAuth('rbac', 'read', LEGACY_GOVERNANCE_PAGES[legacyPath])
          }
        />
      ))}

      <Route path="/help" element={<HelpCenterPage />} />
    </>
  );
}
