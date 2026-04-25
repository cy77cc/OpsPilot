import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import {
  DeploymentListPage,
  DeploymentOverviewPage,
  EnhancedDeploymentCreatePage,
  DeploymentDetailPage,
  ApprovalCenterPage,
} from './pages';

export function renderDeploymentRoutes(withAuth: WithAuth) {
  return (
  <>
    <Route path="/deployment" element={withAuth('deploy:target', 'read', <DeploymentListPage />)} />
    <Route path="/deployment/overview" element={withAuth('deploy:target', 'read', <DeploymentOverviewPage />)} />
    <Route path="/deployment/create" element={withAuth('deploy:target', 'write', <EnhancedDeploymentCreatePage />)} />
    <Route path="/deployment/:id" element={withAuth('deploy:target', 'read', <DeploymentDetailPage />)} />
    <Route path="/deployment/approvals" element={withAuth('deploy:target', 'read', <ApprovalCenterPage />)} />
  </>
  );
}
