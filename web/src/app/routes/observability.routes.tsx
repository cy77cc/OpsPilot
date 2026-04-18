import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import {
  DeploymentTopologyPage,
  MetricsDashboardPage,
  DeploymentAuditLogsPage,
  PolicyManagementPage,
  AIOpsInsightsPage,
  MonitorPage,
  AlertsPage,
  AlertDetailPage,
} from './pages';

export function renderObservabilityRoutes(withAuth: WithAuth) {
  return (
  <>
    <Route path="/deployment/observability/topology" element={withAuth('deploy:target', 'read', <DeploymentTopologyPage />)} />
    <Route path="/deployment/observability/metrics" element={withAuth('monitoring', 'read', <MetricsDashboardPage />)} />
    <Route path="/deployment/observability/audit-logs" element={withAuth('deploy:target', 'read', <DeploymentAuditLogsPage />)} />
    <Route path="/deployment/observability/policies" element={withAuth('deploy:target', 'write', <PolicyManagementPage />)} />
    <Route path="/deployment/observability/aiops" element={withAuth('monitoring', 'read', <AIOpsInsightsPage />)} />
    <Route path="/monitor" element={withAuth('monitoring', 'read', <MonitorPage />)} />
    <Route path="/monitor/dashboard" element={withAuth('monitoring', 'read', <MonitorPage />)} />
    <Route path="/monitor/alerts" element={withAuth('monitoring', 'read', <AlertsPage />)} />
    <Route path="/monitor/alerts/:alertId" element={withAuth('monitoring', 'read', <AlertDetailPage />)} />
    <Route path="/monitor/rules" element={withAuth('monitoring', 'read', <MonitorPage />)} />
    <Route path="/monitoring/alerts" element={withAuth('monitoring', 'read', <AlertsPage />)} />
  </>
  );
}
