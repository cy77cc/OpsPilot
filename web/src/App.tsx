import React from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import AppLayout from './components/Layout/AppLayout';
import { PermissionProvider, Authorized } from './components/RBAC';
import { AuthProvider, useAuth } from './components/Auth/AuthContext';
import { NotificationProvider } from './contexts/NotificationContext';
import { PageTransition } from './components/Motion';
import Dashboard from './pages/Dashboard/Dashboard';
import HostListPage from './pages/Hosts/HostListPage';
import HostDetailPage from './pages/Hosts/HostDetailPage';
import HostOnboardingPage from './pages/Hosts/HostOnboardingPage';
import HostTerminalPage from './pages/Hosts/HostTerminalPage';
import HostKeysPage from './pages/Hosts/HostKeysPage';
import HostCloudImportPage from './pages/Hosts/HostCloudImportPage';
import HostVirtualizationPage from './pages/Hosts/HostVirtualizationPage';
import ConfigPage from './pages/Config/ConfigPage';
import TasksPage from './pages/Tasks/TasksPage';
import K8sPage from './pages/K8s/K8sPage';
import DeploymentPage from './pages/Deployment/DeploymentPage';
import DeploymentListPage from './pages/Deployment/DeploymentListPage';
import DeploymentOverviewPage from './pages/Deployment/DeploymentOverviewPage';
import DeploymentDetailPage from './pages/Deployment/DeploymentDetailPage';
import EnhancedDeploymentCreatePage from './pages/Deployment/EnhancedDeploymentCreatePage';
import ApprovalCenterPage from './pages/Deployment/ApprovalCenterPage';
import ClusterListPage from './pages/Deployment/Infrastructure/ClusterListPage';
import ClusterDetailPage from './pages/Deployment/Infrastructure/ClusterDetailPage';
import ClusterBootstrapWizard from './pages/Deployment/Infrastructure/ClusterBootstrapWizard';
import ClusterImportWizard from './pages/Deployment/Infrastructure/ClusterImportWizard';
import CredentialListPage from './pages/Deployment/Infrastructure/CredentialListPage';
import DeploymentTargetListPage from './pages/Deployment/Targets/DeploymentTargetListPage';
import DeploymentTargetDetailPage from './pages/Deployment/Targets/DeploymentTargetDetailPage';
import CreateTargetWizard from './pages/Deployment/Targets/CreateTargetWizard';
import EnvironmentBootstrapWizard from './pages/Deployment/Targets/EnvironmentBootstrapWizard';
import DeploymentTopologyPage from './pages/Deployment/Observability/DeploymentTopologyPage';
import MetricsDashboardPage from './pages/Deployment/Observability/MetricsDashboardPage';
import DeploymentAuditLogsPage from './pages/Deployment/Observability/AuditLogsPage';
import PolicyManagementPage from './pages/Deployment/Observability/PolicyManagementPage';
import AIOpsInsightsPage from './pages/Deployment/Observability/AIOpsInsightsPage';
import MonitorPage from './pages/Monitor/MonitorPage';
import ToolsPage from './pages/Tools/ToolsPage';
import SettingsPage from './pages/Settings/SettingsPage';
import UsersPage from './pages/Settings/UsersPage';
import RolesPage from './pages/Settings/RolesPage';
import PermissionsPage from './pages/Settings/PermissionsPage';
import ServiceListPage from './pages/Services/ServiceListPage';
import ServiceDetailPage from './pages/Services/ServiceDetailPage';
import ServiceProvisionPage from './pages/Services/ServiceProvisionPage';
import CMDBPage from './pages/CMDB/CMDBPage';
import AutomationPage from './pages/Automation/AutomationPage';
import CICDPage from './pages/CICD/CICDPage';
import JobListPage from './pages/Jobs/JobListPage';
import JobCreationPage from './pages/Jobs/JobCreationPage';
import ExecutionHistoryPage from './pages/Jobs/ExecutionHistoryPage';
import JobCalendarPage from './pages/Jobs/JobCalendarPage';
import ConfigAppsPage from './pages/ConfigCenter/ConfigAppsPage';
import ConfigListPage from './pages/ConfigCenter/ConfigListPage';
import ConfigDiffPage from './pages/ConfigCenter/ConfigDiffPage';
import ConfigMultiEnvPage from './pages/ConfigCenter/ConfigMultiEnvPage';
import ConfigCenterAuditLogsPage from './pages/ConfigCenter/AuditLogsPage';
import LoginPage from './pages/Auth/LoginPage';
import RegisterPage from './pages/Auth/RegisterPage';
import AICommandCenterPage from './pages/AI/AICommandCenterPage';
import HelpCenterPage from './pages/Help/HelpCenterPage';
import AccessDeniedPage from './components/Auth/AccessDeniedPage';
import LegacyGovernanceRedirect from './components/Auth/LegacyGovernanceRedirect';

const ProtectedRoute: React.FC<{ children: React.ReactElement }> = ({ children }) => {
  const { isAuthenticated, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return <div className="min-h-screen flex items-center justify-center">加载中...</div>;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  return children;
};

const ProtectedApp: React.FC = () => {
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
            {/* 旧主机路由重定向到新位置 */}
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
            <Route path="/cmdb" element={withAuth('cmdb', 'read', <CMDBPage />)} />
            <Route path="/automation" element={withAuth('automation', 'read', <AutomationPage />)} />
            <Route path="/cicd" element={withAuth('cicd', 'read', <CICDPage />)} />
            <Route path="/ai" element={withAuth('ai', 'read', <AICommandCenterPage />)} />
            <Route path="/help" element={<HelpCenterPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </PageTransition>
      </AppLayout>
      </NotificationProvider>
    </PermissionProvider>
  );
};

const App: React.FC = () => {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route
            path="/*"
            element={
              <ProtectedRoute>
                <ProtectedApp />
              </ProtectedRoute>
            }
          />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
};

export default App;
