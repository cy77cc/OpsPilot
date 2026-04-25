import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import {
  DeploymentTopologyPage,
  MetricsDashboardPage,
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
  CMDBPage,
  TasksPage,
  JobListPage,
  JobCreationPage,
  ExecutionHistoryPage,
  JobCalendarPage,
} from './pages';

export function renderObservabilityRoutes(withAuth: WithAuth) {
  return (
    <>
      <Route path="/observability/topology" element={withAuth('deploy:target', 'read', <DeploymentTopologyPage />)} />
      <Route path="/observability/metrics" element={withAuth('monitoring', 'read', <MetricsDashboardPage />)} />
      <Route path="/observability/policies" element={withAuth('deploy:target', 'write', <PolicyManagementPage />)} />
      <Route path="/observability/aiops" element={withAuth('monitoring', 'read', <AIOpsInsightsPage />)} />
      <Route path="/observability/monitor" element={withAuth('monitoring', 'read', <MonitorCenterLayout />)}>
        <Route index element={<MonitorPage />} />
        <Route path="dashboard" element={<MonitorPage />} />
        <Route path="alerts" element={<AlertsPage />} />
        <Route path="alerts/:alertId" element={<AlertDetailPage />} />
        <Route path="rules" element={<RulesConfigPage />} />
        <Route path="channels" element={<ChannelsConfigPage />} />
        <Route path="routing" element={<RoutingConfigPage />} />
        <Route path="deliveries" element={<DeliveriesPage />} />
      </Route>
      
      <Route path="/observability/cmdb" element={withAuth('cmdb', 'read', <CMDBPage />)} />

      <Route path="/observability/tasks" element={withAuth('task', 'read', <TasksPage />)} />
      <Route path="/observability/tasks/create" element={withAuth('task', 'write', <TasksPage />)} />
      <Route path="/observability/tasks/:id" element={withAuth('task', 'read', <TasksPage />)} />
      <Route path="/observability/jobs" element={withAuth('task', 'read', <JobListPage />)} />
      <Route path="/observability/jobs/create" element={withAuth('task', 'write', <JobCreationPage />)} />
      <Route path="/observability/jobs/:id/edit" element={withAuth('task', 'write', <JobCreationPage />)} />
      <Route path="/observability/jobs/:jobId/history" element={withAuth('task', 'read', <ExecutionHistoryPage />)} />
      <Route path="/observability/jobs/calendar" element={withAuth('task', 'read', <JobCalendarPage />)} />
    </>
  );
}
