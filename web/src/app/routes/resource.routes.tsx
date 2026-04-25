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
} from './pages';

export function renderResourceRoutes(withAuth: WithAuth) {
  return (
  <>
    <Route path="/resources/clusters" element={withAuth('cluster', 'read', <ClusterListPage />)} />
    <Route path="/resources/clusters/:id" element={withAuth('cluster', 'read', <ClusterDetailPage />)} />
    <Route path="/resources/clusters/:id/nodes" element={withAuth('cluster', 'read', <ClusterNodesPage />)} />
    <Route path="/resources/clusters/:id/workloads" element={withAuth('cluster', 'read', <ClusterWorkloadsPage />)} />
    <Route path="/resources/clusters/:id/network" element={withAuth('cluster', 'read', <ClusterNetworkTrafficPage />)} />
    <Route path="/resources/clusters/:id/config-storage" element={withAuth('cluster', 'read', <ClusterConfigStoragePage />)} />
    <Route path="/resources/clusters/:id/operations" element={withAuth('cluster', 'read', <ClusterOperationCenterPage />)} />
    <Route path="/resources/clusters/:id/policies" element={withAuth('cluster', 'read', <ClusterPolicyCenterPage />)} />
    <Route path="/resources/clusters/:id/security" element={withAuth('cluster', 'read', <ClusterSecurityCenterPage />)} />
    <Route path="/resources/clusters/bootstrap" element={withAuth('cluster', 'write', <ClusterBootstrapWizard />)} />
    <Route path="/resources/clusters/import" element={withAuth('cluster', 'write', <ClusterImportWizard />)} />
    <Route path="/resources/credentials" element={withAuth('cluster', 'read', <CredentialListPage />)} />
    <Route path="/resources/hosts" element={withAuth('host', 'read', <HostListPage />)} />
    <Route path="/resources/hosts/onboarding" element={withAuth('host', 'write', <HostOnboardingPage />)} />
    <Route path="/resources/hosts/keys" element={withAuth('host', 'write', <HostCredentialsPage />)} />
    <Route path="/resources/hosts/credentials" element={withAuth('host', 'write', <HostCredentialsPage />)} />
    <Route path="/resources/hosts/cloud-import" element={withAuth('host', 'write', <HostCloudImportPage />)} />
    <Route path="/resources/hosts/virtualization" element={withAuth('host', 'write', <HostVirtualizationPage />)} />
    <Route path="/resources/hosts/:id" element={withAuth('host', 'read', <HostDetailPage />)} />
    <Route path="/resources/hosts/:id/terminal" element={withAuth('host', 'write', <HostTerminalPage />)} />
  </>
  );
}
