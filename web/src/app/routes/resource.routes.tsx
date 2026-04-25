import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import {
  ClusterListPage,
  ClusterDetailPage,
  ClusterNodesPage,
  ClusterWorkloadsPage,
  ClusterNetworkTrafficPage,
  ClusterConfigStoragePage,
  ClusterOperationCenterPage,
  ClusterPolicyCenterPage,
  ClusterSecurityCenterPage,
  ClusterBootstrapWizard,
  ClusterImportWizard,
  CredentialListPage,
  HostListPage,
  HostOnboardingPage,
  HostCredentialsPage,
  HostCloudImportPage,
  HostVirtualizationPage,
  HostDetailPage,
  HostTerminalPage,
  DeploymentTargetListPage,
  CreateTargetWizard,
  DeploymentTargetDetailPage,
  EnvironmentBootstrapWizard,
} from './pages';

export function renderInfrastructureRoutes(withAuth: WithAuth) {
  return (
  <>
    <Route path="/deployment/infrastructure/clusters" element={withAuth('cluster', 'read', <ClusterListPage />)} />
    <Route path="/deployment/infrastructure/clusters/:id" element={withAuth('cluster', 'read', <ClusterDetailPage />)} />
    <Route path="/deployment/infrastructure/clusters/:id/nodes" element={withAuth('cluster', 'read', <ClusterNodesPage />)} />
    <Route path="/deployment/infrastructure/clusters/:id/workloads" element={withAuth('cluster', 'read', <ClusterWorkloadsPage />)} />
    <Route path="/deployment/infrastructure/clusters/:id/network" element={withAuth('cluster', 'read', <ClusterNetworkTrafficPage />)} />
    <Route path="/deployment/infrastructure/clusters/:id/config-storage" element={withAuth('cluster', 'read', <ClusterConfigStoragePage />)} />
    <Route path="/deployment/infrastructure/clusters/:id/operations" element={withAuth('cluster', 'read', <ClusterOperationCenterPage />)} />
    <Route path="/deployment/infrastructure/clusters/:id/policies" element={withAuth('cluster', 'read', <ClusterPolicyCenterPage />)} />
    <Route path="/deployment/infrastructure/clusters/:id/security" element={withAuth('cluster', 'read', <ClusterSecurityCenterPage />)} />
    <Route path="/deployment/infrastructure/clusters/bootstrap" element={withAuth('cluster', 'write', <ClusterBootstrapWizard />)} />
    <Route path="/deployment/infrastructure/clusters/import" element={withAuth('cluster', 'write', <ClusterImportWizard />)} />
    <Route path="/deployment/infrastructure/credentials" element={withAuth('cluster', 'read', <CredentialListPage />)} />
    <Route path="/deployment/infrastructure/hosts" element={withAuth('host', 'read', <HostListPage />)} />
    <Route path="/deployment/infrastructure/hosts/onboarding" element={withAuth('host', 'write', <HostOnboardingPage />)} />
    <Route path="/deployment/infrastructure/hosts/keys" element={withAuth('host', 'write', <HostCredentialsPage />)} />
    <Route path="/deployment/infrastructure/hosts/credentials" element={withAuth('host', 'write', <HostCredentialsPage />)} />
    <Route path="/deployment/infrastructure/hosts/cloud-import" element={withAuth('host', 'write', <HostCloudImportPage />)} />
    <Route path="/deployment/infrastructure/hosts/virtualization" element={withAuth('host', 'write', <HostVirtualizationPage />)} />
    <Route path="/deployment/infrastructure/hosts/:id" element={withAuth('host', 'read', <HostDetailPage />)} />
    <Route path="/deployment/infrastructure/hosts/:id/terminal" element={withAuth('host', 'write', <HostTerminalPage />)} />
    <Route path="/deployment/targets" element={withAuth('deploy:target', 'read', <DeploymentTargetListPage />)} />
    <Route path="/deployment/targets/create" element={withAuth('deploy:target', 'write', <CreateTargetWizard />)} />
    <Route path="/deployment/targets/:id" element={withAuth('deploy:target', 'read', <DeploymentTargetDetailPage />)} />
    <Route path="/deployment/targets/:targetId/bootstrap/:jobId?" element={withAuth('deploy:target', 'write', <EnvironmentBootstrapWizard />)} />
  </>
  );
}
