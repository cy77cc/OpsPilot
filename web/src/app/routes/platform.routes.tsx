import React from 'react';
import { Navigate, Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import LegacyGovernanceRedirect from '../../components/Auth/LegacyGovernanceRedirect';
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
    <Route
      path="/settings/users"
      element={governanceMenuEnabled ? <LegacyGovernanceRedirect to="/governance/users" /> : <Navigate to="/settings" replace />}
    />
    <Route
      path="/settings/roles"
      element={governanceMenuEnabled ? <LegacyGovernanceRedirect to="/governance/roles" /> : <Navigate to="/settings" replace />}
    />
    <Route
      path="/settings/permissions"
      element={governanceMenuEnabled ? <LegacyGovernanceRedirect to="/governance/permissions" /> : <Navigate to="/settings" replace />}
    />
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
