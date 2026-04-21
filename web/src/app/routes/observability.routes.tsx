import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import {
  DeploymentTopologyPage,
  MetricsDashboardPage,
  DeploymentAuditLogsPage,
  PolicyManagementPage,
  AIOpsInsightsPage,
  MonitorCenterLayout,
  MonitorPage,
  AlertsPage,
  AlertDetailPage,
  RulesConfigPage,
  ChannelsConfigPage,
  RoutingConfigPage,
  DeliveriesPage,
} from './pages';

export function renderObservabilityRoutes(withAuth: WithAuth) {
  return (
    <>
      <Route path="/deployment/observability/topology" element={withAuth('deploy:target', 'read', <DeploymentTopologyPage />)} />
      <Route path="/deployment/observability/metrics" element={withAuth('monitoring', 'read', <MetricsDashboardPage />)} />
      <Route path="/deployment/observability/audit-logs" element={withAuth('deploy:target', 'read', <DeploymentAuditLogsPage />)} />
      <Route path="/deployment/observability/policies" element={withAuth('deploy:target', 'write', <PolicyManagementPage />)} />
      <Route path="/deployment/observability/aiops" element={withAuth('monitoring', 'read', <AIOpsInsightsPage />)} />
      <Route path="/monitor" element={withAuth('monitoring', 'read', <MonitorCenterLayout />)}>
        <Route index element={<MonitorPage />} />
        <Route path="dashboard" element={<MonitorPage />} />
        <Route path="alerts" element={<AlertsPage />} />
        <Route path="alerts/:alertId" element={<AlertDetailPage />} />
        <Route path="rules" element={<RulesConfigPage />} />
        <Route path="channels" element={<ChannelsConfigPage />} />
        <Route path="routing" element={<RoutingConfigPage />} />
        <Route path="deliveries" element={<DeliveriesPage />} />
      </Route>
    </>
  );
}
