import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import LegacyGovernanceRedirect from '../../components/Auth/LegacyGovernanceRedirect';
import { LEGACY_GOVERNANCE_MENU_ITEMS } from '../layout/navigation.config';
import {
  ToolsPage,
  SettingsPage,
  AIModelSettingsPage,
  UsersPage,
  RolesPage,
  PermissionsPage,
  ServiceListPage,
  ServiceProvisionPage,
  ServiceDetailPage,
  ServiceDeployPage,
  ServiceVisibilityPage,
  CMDBPage,
  AutomationPage,
  CICDPage,
  HelpCenterPage,
} from './pages';

interface PlatformRoutesProps {
  withAuth: WithAuth;
  governanceMenuEnabled: boolean;
}

type LegacyGovernancePath = (typeof LEGACY_GOVERNANCE_MENU_ITEMS)[number]['key'];

const LEGACY_GOVERNANCE_PAGES: Record<LegacyGovernancePath, React.ReactElement> = {
  '/settings/users': <UsersPage />,
  '/settings/roles': <RolesPage />,
  '/settings/permissions': <PermissionsPage />,
};

export function renderPlatformRoutes({ withAuth, governanceMenuEnabled }: PlatformRoutesProps) {
  return (
    <>
    <Route path="/tools" element={<ToolsPage />} />
    <Route path="/tools/nightingale" element={<ToolsPage />} />
    <Route path="/tools/jenkins" element={<ToolsPage />} />
    <Route path="/tools/jumpserver" element={<ToolsPage />} />
    <Route path="/tools/kuboard" element={<ToolsPage />} />
    <Route path="/tools/cmdb" element={<ToolsPage />} />
    <Route path="/tools/archery" element={<ToolsPage />} />
    <Route path="/settings" element={<SettingsPage />} />
    <Route path="/settings/ai-models" element={<AIModelSettingsPage />} />
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
    <Route path="/services" element={withAuth('service', 'read', <ServiceListPage />)} />
    <Route path="/services/provision" element={withAuth('service', 'write', <ServiceProvisionPage />)} />
    <Route path="/services/:id" element={withAuth('service', 'read', <ServiceDetailPage />)} />
    <Route path="/services/:id/deploy" element={withAuth('service', 'deploy', <ServiceDeployPage />)} />
    <Route path="/services/:id/visibility" element={withAuth('service', 'write', <ServiceVisibilityPage />)} />
    <Route path="/cmdb" element={withAuth('cmdb', 'read', <CMDBPage />)} />
    <Route path="/automation" element={withAuth('automation', 'read', <AutomationPage />)} />
    <Route path="/cicd" element={withAuth('cicd', 'read', <CICDPage />)} />
      <Route path="/help" element={<HelpCenterPage />} />
    </>
  );
}
