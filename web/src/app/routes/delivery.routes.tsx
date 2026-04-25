import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import {
  DeploymentListPage,
  DeploymentOverviewPage,
  EnhancedDeploymentCreatePage,
  DeploymentDetailPage,
  ServiceListPage,
  ServiceProvisionPage,
  ServiceDetailPage,
  ServiceDeployPage,
  ServiceVisibilityPage,
  AutomationPage,
  CICDPage,
  DeploymentTargetListPage,
  CreateTargetWizard,
  DeploymentTargetDetailPage,
  EnvironmentBootstrapWizard,
} from './pages';

export function renderDeliveryRoutes(withAuth: WithAuth) {
  return (
  <>
    <Route path="/delivery/deployments" element={withAuth('deploy:target', 'read', <DeploymentListPage />)} />
    <Route path="/delivery/deployments/overview" element={withAuth('deploy:target', 'read', <DeploymentOverviewPage />)} />
    <Route path="/delivery/deployments/create" element={withAuth('deploy:target', 'write', <EnhancedDeploymentCreatePage />)} />
    <Route path="/delivery/deployments/:id" element={withAuth('deploy:target', 'read', <DeploymentDetailPage />)} />
    
    <Route path="/delivery/services" element={withAuth('service', 'read', <ServiceListPage />)} />
    <Route path="/delivery/services/provision" element={withAuth('service', 'write', <ServiceProvisionPage />)} />
    <Route path="/delivery/services/:id" element={withAuth('service', 'read', <ServiceDetailPage />)} />
    <Route path="/delivery/services/:id/deploy" element={withAuth('service', 'deploy', <ServiceDeployPage />)} />
    <Route path="/delivery/services/:id/visibility" element={withAuth('service', 'write', <ServiceVisibilityPage />)} />
    
    <Route path="/delivery/automation" element={withAuth('automation', 'read', <AutomationPage />)} />
    <Route path="/delivery/cicd" element={withAuth('cicd', 'read', <CICDPage />)} />

    <Route path="/delivery/targets" element={withAuth('deploy:target', 'read', <DeploymentTargetListPage />)} />
    <Route path="/delivery/targets/create" element={withAuth('deploy:target', 'write', <CreateTargetWizard />)} />
    <Route path="/delivery/targets/:id" element={withAuth('deploy:target', 'read', <DeploymentTargetDetailPage />)} />
    <Route path="/delivery/targets/:targetId/bootstrap/:jobId?" element={withAuth('deploy:target', 'write', <EnvironmentBootstrapWizard />)} />
  </>
  );
}
