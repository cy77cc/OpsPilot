# Navigation System

前端导航系统规范，包括侧边栏菜单、路由配置和页面结构。

## Requirements

### REQ-001: 侧边栏菜单结构

侧边栏采用层级结构，支持二级和三级菜单。

```
仪表板 (/)
服务管理
CMDB (/cmdb)
自动化
CI/CD (/cicd)
AI命令中心
帮助中心
配置中心
任务中心
部署管理 (/deployment)
  ├─ 基础设施
  │   ├─ 集群管理 (/deployment/infrastructure/clusters)
  │   ├─ 凭证管理 (/deployment/infrastructure/credentials)
  │   └─ 主机管理 (/deployment/infrastructure/hosts)
  ├─ 部署目标
  │   ├─ 目标列表 (/deployment/targets)
  │   └─ 创建目标 (/deployment/targets/create)
  ├─ 发布管理
  │   ├─ 发布概览 (/deployment/overview)
  │   ├─ 创建发布 (/deployment/create)
  │   ├─ 发布历史 (/deployment)
  │   └─ 审批中心 (/deployment/approvals)
  └─ 可观测性
      ├─ 部署拓扑 (/deployment/observability/topology)
      ├─ 指标仪表板 (/deployment/observability/metrics)
      ├─ 审计日志 (/deployment/observability/audit-logs)
      ├─ 策略管理 (/deployment/observability/policies)
      └─ AIOps 洞察 (/deployment/observability/aiops)
监控中心
工具箱
系统设置
```

### REQ-002: 菜单滚动

侧边栏菜单区域需要支持滚动，防止菜单项溢出屏幕。

- Logo 区域固定在顶部
- 菜单区域可滚动 (`overflow-y-auto`)
- 折叠按钮固定在底部

### REQ-003: 菜单激活状态

根据当前路由自动高亮对应菜单项。

- 主机相关页面 (`/deployment/infrastructure/hosts/*`) 激活主机管理菜单
- 集群相关页面激活集群管理菜单
- 支持 URL 前缀匹配

### REQ-004: 响应式设计

侧边栏支持响应式布局。

- 桌面端：固定侧边栏，支持折叠
- 移动端：抽屉式侧边栏

## Route Configuration

### 主应用路由 (App.tsx)

所有路由在 `App.tsx` 中集中配置，使用 `withAuth` 包装需要权限的路由。

```typescript
// 部署管理路由
<Route path="/deployment" element={<DeploymentListPage />} />
<Route path="/deployment/overview" element={<DeploymentOverviewPage />} />
<Route path="/deployment/create" element={<EnhancedDeploymentCreatePage />} />
<Route path="/deployment/:id" element={<DeploymentDetailPage />} />

// 基础设施路由
<Route path="/deployment/infrastructure/clusters" element={<ClusterListPage />} />
<Route path="/deployment/infrastructure/hosts" element={<HostListPage />} />

// ... 其他路由
```

### 旧路由重定向

保留旧路由兼容性，自动重定向到新路径。

```typescript
// 主机路由迁移
<Route path="/hosts" element={<Navigate to="/deployment/infrastructure/hosts" replace />} />
```

## Implementation Files

| 文件 | 用途 |
|------|------|
| `web/src/App.tsx` | 主应用路由配置 |
| `web/src/routes/AppRoutes.tsx` | 备用路由配置（部分使用） |
| `web/src/components/Layout/AppLayout.tsx` | 侧边栏菜单配置 |

## Migration Notes

### 2026-03-02: 主机路由迁移

主机管理从顶级菜单迁移到部署管理 > 基础设施下：

- `/hosts` → `/deployment/infrastructure/hosts`
- `/hosts/onboarding` → `/deployment/infrastructure/hosts/onboarding`
- `/hosts/:id` → `/deployment/infrastructure/hosts/:id`
- 旧路由保留重定向
