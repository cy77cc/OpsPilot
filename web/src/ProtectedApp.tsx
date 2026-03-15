import React, { Suspense, lazy } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { PermissionProvider, Authorized } from './components/RBAC';
import { useAuth } from './components/Auth/AuthContext';
import { NotificationProvider } from './contexts/NotificationContext';
import { PageTransition } from './components/Motion';
import AccessDeniedPage from './components/Auth/AccessDeniedPage';
import LegacyGovernanceRedirect from './components/Auth/LegacyGovernanceRedirect';

const AppLayout = lazy(() => import('./components/Layout/AppLayout'));
const Dashboard = lazy(() => import('./pages/Dashboard/Dashboard'));
const HostListPage = lazy(() => import('./pages/Hosts/HostListPage'));
const HostDetailPage = lazy(() => import('./pages/Hosts/HostDetailPage'));
const HostOnboardingPage = lazy(() => import('./pages/Hosts/HostOnboardingPage'));
const HostTerminalPage = lazy(() => import('./pages/Hosts/HostTerminalPage'));
const HostKeysPage = lazy(() => import('./pages/Hosts/HostKeysPage'));
const HostCloudImportPage = lazy(() => import('./pages/Hosts/HostCloudImportPage'));
const HostVirtualizationPage = lazy(() => import('./pages/Hosts/HostVirtualizationPage'));
const ConfigPage = lazy(() => import('./pages/Config/ConfigPage'));
const TasksPage = lazy(() => import('./pages/Tasks/TasksPage'));
const K8sPage = lazy(() => import('./pages/K8s/K8sPage'));
const DeploymentListPage = lazy(() => import('./pages/Deployment/DeploymentListPage'));
const DeploymentOverviewPage = lazy(() => import('./pages/Deployment/DeploymentOverviewPage'));
const DeploymentDetailPage = lazy(() => import('./pages/Deployment/DeploymentDetailPage'));
const EnhancedDeploymentCreatePage = lazy(() => import('./pages/Deployment/EnhancedDeploymentCreatePage'));
const ApprovalCenterPage = lazy(() => import('./pages/Deployment/ApprovalCenterPage'));
const ClusterListPage = lazy(() => import('./pages/Deployment/Infrastructure/ClusterListPage'));
const ClusterDetailPage = lazy(() => import('./pages/Deployment/Infrastructure/ClusterDetailPage'));
const ClusterBootstrapWizard = lazy(() => import('./pages/Deployment/Infrastructure/ClusterBootstrapWizard'));
const ClusterImportWizard = lazy(() => import('./pages/Deployment/Infrastructure/ClusterImportWizard'));
const CredentialListPage = lazy(() => import('./pages/Deployment/Infrastructure/CredentialListPage'));
const DeploymentTargetListPage = lazy(() => import('./pages/Deployment/Targets/DeploymentTargetListPage'));
const DeploymentTargetDetailPage = lazy(() => import('./pages/Deployment/Targets/DeploymentTargetDetailPage'));
const CreateTargetWizard = lazy(() => import('./pages/Deployment/Targets/CreateTargetWizard'));
const EnvironmentBootstrapWizard = lazy(() => import('./pages/Deployment/Targets/EnvironmentBootstrapWizard'));
const DeploymentTopologyPage = lazy(() => import('./pages/Deployment/Observability/DeploymentTopologyPage'));
const MetricsDashboardPage = lazy(() => import('./pages/Deployment/Observability/MetricsDashboardPage'));
const DeploymentAuditLogsPage = lazy(() => import('./pages/Deployment/Observability/AuditLogsPage'));
const PolicyManagementPage = lazy(() => import('./pages/Deployment/Observability/PolicyManagementPage'));
const AIOpsInsightsPage = lazy(() => import('./pages/Deployment/Observability/AIOpsInsightsPage'));
const MonitorPage = lazy(() => import('./pages/Monitor/MonitorPage'));
const ToolsPage = lazy(() => import('./pages/Tools/ToolsPage'));
const SettingsPage = lazy(() => import('./pages/Settings/SettingsPage'));
const UsersPage = lazy(() => import('./pages/Settings/UsersPage'));
const RolesPage = lazy(() => import('./pages/Settings/RolesPage'));
const PermissionsPage = lazy(() => import('./pages/Settings/PermissionsPage'));
const ServiceListPage = lazy(() => import('./pages/Services/ServiceListPage'));
const ServiceDetailPage = lazy(() => import('./pages/Services/ServiceDetailPage'));
const ServiceProvisionPage = lazy(() => import('./pages/Services/ServiceProvisionPage'));
const ServiceDeployPage = lazy(() => import('./pages/Services/ServiceDeployPage'));
const ServiceVisibilityPage = lazy(() => import('./pages/Services/ServiceVisibilityPage'));
const CMDBPage = lazy(() => import('./pages/CMDB/CMDBPage'));
const AutomationPage = lazy(() => import('./pages/Automation/AutomationPage'));
const CICDPage = lazy(() => import('./pages/CICD/CICDPage'));
const JobListPage = lazy(() => import('./pages/Jobs/JobListPage'));
const JobCreationPage = lazy(() => import('./pages/Jobs/JobCreationPage'));
const ExecutionHistoryPage = lazy(() => import('./pages/Jobs/ExecutionHistoryPage'));
const JobCalendarPage = lazy(() => import('./pages/Jobs/JobCalendarPage'));
const ConfigAppsPage = lazy(() => import('./pages/ConfigCenter/ConfigAppsPage'));
const ConfigListPage = lazy(() => import('./pages/ConfigCenter/ConfigListPage'));
const ConfigDiffPage = lazy(() => import('./pages/ConfigCenter/ConfigDiffPage'));
const ConfigMultiEnvPage = lazy(() => import('./pages/ConfigCenter/ConfigMultiEnvPage'));
const ConfigCenterAuditLogsPage = lazy(() => import('./pages/ConfigCenter/AuditLogsPage'));
const HelpCenterPage = lazy(() => import('./pages/Help/HelpCenterPage'));
const AIUsagePage = lazy(() => import('./pages/AI/Usage'));

const RouteFallback: React.FC = () => (
  <div className="min-h-screen flex items-center justify-center">加载中...</div>
);

export default function ProtectedApp() {
  const { user } = useAuth();
  const governanceMenuEnabled = import.meta.env.VITE_FEATURE_GOVERNANCE_MENU !== 'false';
  const withAuth = (resource: string, action: string, element: React.ReactElement) => (
    <Authorized resource={resource} action={action} fallback={<AccessDeniedPage />}>
      {element}
    </Authorized>
  );

  return (
    <PermissionProvider>
      <NotificationProvider userId={user?.id}>
        <Suspense fallback={<RouteFallback />}>
          <AppLayout>
            <PageTransition>
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/config" element={withAuth('config', 'read', <ConfigPage />)} />
                <Route path="/config/:id" element={withAuth('config', 'write', <ConfigPage />)} />
                <Route path="/configcenter/apps" element={withAuth('config', 'read', <ConfigAppsPage />)} />
                <Route path="/configcenter/list" element={withAuth('config', 'read', <ConfigListPage />)} />
                <Route path="/configcenter/diff" element={withAuth('config', 'read', <ConfigDiffPage />)} />
                <Route path="/configcenter/multienv" element={withAuth('config', 'write', <ConfigMultiEnvPage />)} />
                <Route path="/configcenter/logs" element={withAuth('config', 'read', <ConfigCenterAuditLogsPage />)} />
                <Route path="/tasks" element={withAuth('task', 'read', <TasksPage />)} />
                <Route path="/tasks/create" element={withAuth('task', 'write', <TasksPage />)} />
                <Route path="/tasks/:id" element={withAuth('task', 'read', <TasksPage />)} />
                <Route path="/jobs" element={withAuth('task', 'read', <JobListPage />)} />
                <Route path="/jobs/create" element={withAuth('task', 'write', <JobCreationPage />)} />
                <Route path="/jobs/:id/edit" element={withAuth('task', 'write', <JobCreationPage />)} />
                <Route path="/jobs/:jobId/history" element={withAuth('task', 'read', <ExecutionHistoryPage />)} />
                <Route path="/jobs/calendar" element={withAuth('task', 'read', <JobCalendarPage />)} />
                <Route path="/deployment" element={withAuth('deploy:target', 'read', <DeploymentListPage />)} />
                <Route path="/deployment/overview" element={withAuth('deploy:target', 'read', <DeploymentOverviewPage />)} />
                <Route path="/deployment/create" element={withAuth('deploy:target', 'write', <EnhancedDeploymentCreatePage />)} />
                <Route path="/deployment/:id" element={withAuth('deploy:target', 'read', <DeploymentDetailPage />)} />
                <Route path="/deployment/approvals" element={withAuth('deploy:target', 'read', <ApprovalCenterPage />)} />
                <Route path="/deployment/infrastructure/clusters" element={withAuth('cluster', 'read', <ClusterListPage />)} />
                <Route path="/deployment/infrastructure/clusters/:id" element={withAuth('cluster', 'read', <ClusterDetailPage />)} />
                <Route path="/deployment/infrastructure/clusters/bootstrap" element={withAuth('cluster', 'write', <ClusterBootstrapWizard />)} />
                <Route path="/deployment/infrastructure/clusters/import" element={withAuth('cluster', 'write', <ClusterImportWizard />)} />
                <Route path="/deployment/infrastructure/credentials" element={withAuth('cluster', 'read', <CredentialListPage />)} />
                <Route path="/deployment/infrastructure/hosts" element={withAuth('host', 'read', <HostListPage />)} />
                <Route path="/deployment/infrastructure/hosts/onboarding" element={withAuth('host', 'write', <HostOnboardingPage />)} />
                <Route path="/deployment/infrastructure/hosts/keys" element={withAuth('host', 'write', <HostKeysPage />)} />
                <Route path="/deployment/infrastructure/hosts/cloud-import" element={withAuth('host', 'write', <HostCloudImportPage />)} />
                <Route path="/deployment/infrastructure/hosts/virtualization" element={withAuth('host', 'write', <HostVirtualizationPage />)} />
                <Route path="/deployment/infrastructure/hosts/:id" element={withAuth('host', 'read', <HostDetailPage />)} />
                <Route path="/deployment/infrastructure/hosts/:id/terminal" element={withAuth('host', 'write', <HostTerminalPage />)} />
                <Route path="/deployment/targets" element={withAuth('deploy:target', 'read', <DeploymentTargetListPage />)} />
                <Route path="/deployment/targets/create" element={withAuth('deploy:target', 'write', <CreateTargetWizard />)} />
                <Route path="/deployment/targets/:id" element={withAuth('deploy:target', 'read', <DeploymentTargetDetailPage />)} />
                <Route path="/deployment/targets/:targetId/bootstrap/:jobId?" element={withAuth('deploy:target', 'write', <EnvironmentBootstrapWizard />)} />
                <Route path="/deployment/observability/topology" element={withAuth('deploy:target', 'read', <DeploymentTopologyPage />)} />
                <Route path="/deployment/observability/metrics" element={withAuth('monitoring', 'read', <MetricsDashboardPage />)} />
                <Route path="/deployment/observability/audit-logs" element={withAuth('deploy:target', 'read', <DeploymentAuditLogsPage />)} />
                <Route path="/deployment/observability/policies" element={withAuth('deploy:target', 'write', <PolicyManagementPage />)} />
                <Route path="/deployment/observability/aiops" element={withAuth('monitoring', 'read', <AIOpsInsightsPage />)} />
                <Route path="/k8s" element={<Navigate to="/deployment" replace />} />
                <Route path="/k8s/:cluster" element={<Navigate to="/deployment" replace />} />
                <Route path="/k8s-legacy" element={withAuth('kubernetes', 'read', <K8sPage />)} />
                <Route path="/hosts" element={<Navigate to="/deployment/infrastructure/hosts" replace />} />
                <Route path="/hosts/onboarding" element={<Navigate to="/deployment/infrastructure/hosts/onboarding" replace />} />
                <Route path="/hosts/keys" element={<Navigate to="/deployment/infrastructure/hosts/keys" replace />} />
                <Route path="/hosts/cloud-import" element={<Navigate to="/deployment/infrastructure/hosts/cloud-import" replace />} />
                <Route path="/hosts/virtualization" element={<Navigate to="/deployment/infrastructure/hosts/virtualization" replace />} />
                <Route path="/hosts/detail/:id" element={<Navigate to="/deployment/infrastructure/hosts/:id" replace />} />
                <Route path="/hosts/terminal/:id" element={<Navigate to="/deployment/infrastructure/hosts/:id/terminal" replace />} />
                <Route path="/monitor" element={withAuth('monitoring', 'read', <MonitorPage />)} />
                <Route path="/monitor/dashboard" element={withAuth('monitoring', 'read', <MonitorPage />)} />
                <Route path="/monitor/alerts" element={withAuth('monitoring', 'read', <MonitorPage />)} />
                <Route path="/monitor/rules" element={withAuth('monitoring', 'read', <MonitorPage />)} />
                <Route path="/monitoring/alerts" element={withAuth('monitoring', 'read', <MonitorPage />)} />
                <Route path="/tools" element={<ToolsPage />} />
                <Route path="/tools/nightingale" element={<ToolsPage />} />
                <Route path="/tools/jenkins" element={<ToolsPage />} />
                <Route path="/tools/jumpserver" element={<ToolsPage />} />
                <Route path="/tools/kuboard" element={<ToolsPage />} />
                <Route path="/tools/cmdb" element={<ToolsPage />} />
                <Route path="/tools/archery" element={<ToolsPage />} />
                <Route path="/settings" element={<SettingsPage />} />
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
                <Route path="/ai/usage" element={<AIUsagePage />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </PageTransition>
          </AppLayout>
        </Suspense>
      </NotificationProvider>
    </PermissionProvider>
  );
}
