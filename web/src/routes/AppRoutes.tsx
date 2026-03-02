import React, { lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import LoadingSkeleton from '../components/LoadingSkeleton';

// 立即加载的核心页面
import AppLayout from '../components/Layout/AppLayout';
import Dashboard from '../pages/Dashboard/Dashboard';

// 懒加载的页面组件
const ServiceListPage = lazy(() => import('../pages/Services/ServiceListPage'));
const ServiceDetailPage = lazy(() => import('../pages/Services/ServiceDetailPage'));
const ServiceProvisionPage = lazy(() => import('../pages/Services/ServiceProvisionPage'));

const DeploymentListPage = lazy(() => import('../pages/Deployment/DeploymentListPage'));
const DeploymentCreatePage = lazy(() => import('../pages/Deployment/DeploymentCreatePage'));
const EnhancedDeploymentCreatePage = lazy(() => import('../pages/Deployment/EnhancedDeploymentCreatePage'));
const DeploymentDetailPage = lazy(() => import('../pages/Deployment/DeploymentDetailPage'));
const DeploymentOverviewPage = lazy(() => import('../pages/Deployment/DeploymentOverviewPage'));
const ApprovalCenterPage = lazy(() => import('../pages/Deployment/ApprovalCenterPage'));

// 部署管理 - 基础设施
const CredentialListPage = lazy(() => import('../pages/Deployment/Infrastructure/CredentialListPage'));
const ClusterListPage = lazy(() => import('../pages/Deployment/Infrastructure/ClusterListPage'));
const ClusterDetailPage = lazy(() => import('../pages/Deployment/Infrastructure/ClusterDetailPage'));
const ClusterBootstrapWizard = lazy(() => import('../pages/Deployment/Infrastructure/ClusterBootstrapWizard'));
const ClusterImportWizard = lazy(() => import('../pages/Deployment/Infrastructure/ClusterImportWizard'));

// 部署管理 - 部署目标
const DeploymentTargetListPage = lazy(() => import('../pages/Deployment/Targets/DeploymentTargetListPage'));
const DeploymentTargetDetailPage = lazy(() => import('../pages/Deployment/Targets/DeploymentTargetDetailPage'));
const CreateTargetWizard = lazy(() => import('../pages/Deployment/Targets/CreateTargetWizard'));
const EnvironmentBootstrapWizard = lazy(() => import('../pages/Deployment/Targets/EnvironmentBootstrapWizard'));

// 部署管理 - 可观测性
const AuditLogsPage = lazy(() => import('../pages/Deployment/Observability/AuditLogsPage'));
const DeploymentTopologyPage = lazy(() => import('../pages/Deployment/Observability/DeploymentTopologyPage'));
const PolicyManagementPage = lazy(() => import('../pages/Deployment/Observability/PolicyManagementPage'));
const MetricsDashboardPage = lazy(() => import('../pages/Deployment/Observability/MetricsDashboardPage'));
const AIOpsInsightsPage = lazy(() => import('../pages/Deployment/Observability/AIOpsInsightsPage'));

const HostListPage = lazy(() => import('../pages/Hosts/HostListPage'));
const HostDetailPage = lazy(() => import('../pages/Hosts/HostDetailPage'));

// 其他页面
const CMDBPage = lazy(() => import('../pages/CMDB/CMDBPage'));
const AutomationPage = lazy(() => import('../pages/Automation/AutomationPage'));
const CICDPage = lazy(() => import('../pages/CICD/CICDPage'));
const AICommandCenterPage = lazy(() => import('../pages/AI/AICommandCenterPage'));
const HelpCenterPage = lazy(() => import('../pages/Help/HelpCenterPage'));
const MonitorPage = lazy(() => import('../pages/Monitor/MonitorPage'));
const ToolsPage = lazy(() => import('../pages/Tools/ToolsPage'));

const ConfigPage = lazy(() => import('../pages/Config/ConfigPage'));
const TasksPage = lazy(() => import('../pages/Tasks/TasksPage'));
const SettingsPage = lazy(() => import('../pages/Settings/SettingsPage'));

// 懒加载包装器
const LazyPage: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <Suspense fallback={<LoadingSkeleton type="detail" count={1} />}>{children}</Suspense>
);

/**
 * 应用路由配置
 *
 * 特性:
 * - 路由级代码分割
 * - 懒加载非核心页面
 * - 加载时显示骨架屏
 */
const AppRoutes: React.FC = () => {
  return (
    <Routes>
      <Route path="/" element={<AppLayout><Dashboard /></AppLayout>} />

      {/* 服务管理 */}
      <Route
        path="/services"
        element={
          <AppLayout>
            <LazyPage>
              <ServiceListPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/services/:id"
        element={
          <AppLayout>
            <LazyPage>
              <ServiceDetailPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/services/provision"
        element={
          <AppLayout>
            <LazyPage>
              <ServiceProvisionPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 部署管理 */}
      <Route
        path="/deployment"
        element={
          <AppLayout>
            <LazyPage>
              <DeploymentListPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/overview"
        element={
          <AppLayout>
            <LazyPage>
              <DeploymentOverviewPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/approvals"
        element={
          <AppLayout>
            <LazyPage>
              <ApprovalCenterPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/create"
        element={
          <AppLayout>
            <LazyPage>
              <EnhancedDeploymentCreatePage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/:id"
        element={
          <AppLayout>
            <LazyPage>
              <DeploymentDetailPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 部署管理 - 基础设施 */}
      <Route
        path="/deployment/infrastructure/credentials"
        element={
          <AppLayout>
            <LazyPage>
              <CredentialListPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/infrastructure/clusters"
        element={
          <AppLayout>
            <LazyPage>
              <ClusterListPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/infrastructure/clusters/:id"
        element={
          <AppLayout>
            <LazyPage>
              <ClusterDetailPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/infrastructure/clusters/bootstrap"
        element={
          <AppLayout>
            <LazyPage>
              <ClusterBootstrapWizard />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/infrastructure/clusters/import"
        element={
          <AppLayout>
            <LazyPage>
              <ClusterImportWizard />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 部署管理 - 部署目标 */}
      <Route
        path="/deployment/targets"
        element={
          <AppLayout>
            <LazyPage>
              <DeploymentTargetListPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/targets/:id"
        element={
          <AppLayout>
            <LazyPage>
              <DeploymentTargetDetailPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/targets/create"
        element={
          <AppLayout>
            <LazyPage>
              <CreateTargetWizard />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/targets/:targetId/bootstrap/:jobId?"
        element={
          <AppLayout>
            <LazyPage>
              <EnvironmentBootstrapWizard />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 部署管理 - 可观测性 */}
      <Route
        path="/deployment/observability/audit-logs"
        element={
          <AppLayout>
            <LazyPage>
              <AuditLogsPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/observability/topology"
        element={
          <AppLayout>
            <LazyPage>
              <DeploymentTopologyPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/observability/policies"
        element={
          <AppLayout>
            <LazyPage>
              <PolicyManagementPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/observability/metrics"
        element={
          <AppLayout>
            <LazyPage>
              <MetricsDashboardPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/observability/aiops"
        element={
          <AppLayout>
            <LazyPage>
              <AIOpsInsightsPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 主机管理 - 重定向到新路由 */}
      <Route path="/hosts" element={<Navigate to="/deployment/infrastructure/hosts" replace />} />
      <Route path="/hosts/:path*" element={<Navigate to="/deployment/infrastructure/hosts/:path*" replace />} />

      {/* 主机管理 - 新路由位置 */}
      <Route
        path="/deployment/infrastructure/hosts"
        element={
          <AppLayout>
            <LazyPage>
              <HostListPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/deployment/infrastructure/hosts/:id"
        element={
          <AppLayout>
            <LazyPage>
              <HostDetailPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 其他页面 */}
      <Route
        path="/config"
        element={
          <AppLayout>
            <LazyPage>
              <ConfigPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/tasks"
        element={
          <AppLayout>
            <LazyPage>
              <TasksPage />
            </LazyPage>
          </AppLayout>
        }
      />
      <Route
        path="/settings"
        element={
          <AppLayout>
            <LazyPage>
              <SettingsPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* CMDB */}
      <Route
        path="/cmdb"
        element={
          <AppLayout>
            <LazyPage>
              <CMDBPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 自动化 */}
      <Route
        path="/automation"
        element={
          <AppLayout>
            <LazyPage>
              <AutomationPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* CI/CD */}
      <Route
        path="/cicd"
        element={
          <AppLayout>
            <LazyPage>
              <CICDPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* AI命令中心 */}
      <Route
        path="/ai"
        element={
          <AppLayout>
            <LazyPage>
              <AICommandCenterPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 帮助中心 */}
      <Route
        path="/help"
        element={
          <AppLayout>
            <LazyPage>
              <HelpCenterPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 监控中心 */}
      <Route
        path="/monitor"
        element={
          <AppLayout>
            <LazyPage>
              <MonitorPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 工具箱 */}
      <Route
        path="/tools"
        element={
          <AppLayout>
            <LazyPage>
              <ToolsPage />
            </LazyPage>
          </AppLayout>
        }
      />

      {/* 默认重定向 */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
};

export default AppRoutes;
