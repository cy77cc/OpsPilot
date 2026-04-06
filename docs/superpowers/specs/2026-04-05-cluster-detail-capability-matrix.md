# Cluster Detail Capability Matrix

## Status Enum
- `ready`
- `partial`
- `missing`

## Release Gate
- Any `missing` capability MUST NOT expose clickable entry in UI.

| 页面能力 | 前端入口 | API 端点 | 后端 handler | 测试文件 | 状态 | 备注 |
|---------|---------|---------|------------|--------|------|------|
| 集群健康摘要 | `/deployment/infrastructure/clusters/:id` | `GET /clusters/:id` | `GetClusterDetail` | `ClusterDetailPage.test.tsx` | ready | 概览主卡片基础能力 |
| 最近失败操作 | `/deployment/infrastructure/clusters/:id` | `GET /clusters/:id/operations/history?status=failed&page_size=5` | `ListOperationHistory` | `ClusterOperationCenterPage.test.tsx` | ready | 复用现有操作中心历史 |
| 审批态操作反馈 | `/deployment/infrastructure/clusters/:id` | `POST /clusters/:id/* (operation envelope)` | `BuildOperationResponse` | `ClusterOperationCenterPage.test.tsx` | ready | 统一四态渲染：approval/running/completed/failed |
| 节点与容量列表 | `/deployment/infrastructure/clusters/:id/nodes` | `GET /clusters/:id/nodes` | `ListNodes` | `ClusterNodesPage.test.tsx` | ready | 计划拆分为独立页面 |
| 工作负载视图（Deployments/StatefulSets/Pods） | `/deployment/infrastructure/clusters/:id/workloads` | `GET /clusters/:id/resources?namespace=:ns` | `GetClusterResources` | `ClusterWorkloadsPage.test.tsx` | partial | 需补齐按资源类型分组契约 |
| 网络与流量总览（Service/Ingress/Gateway API） | `/deployment/infrastructure/clusters/:id/network` | `GET /clusters/:id/services` | `ListServices` | `ClusterNetworkTrafficPage.test.tsx` | partial | Gateway API 优先，Ingress 兼容 |
| 配置与存储（ConfigMap/Secret/PVC） | `/deployment/infrastructure/clusters/:id/config-storage` | `GET /clusters/:id/resources?scope=config-storage` | `GetClusterResources` | `ClusterConfigStoragePage.test.tsx` | partial | 当前接口需补返回聚合结构 |
| 安全告警摘要 | `/deployment/infrastructure/clusters/:id` | `GET /clusters/:id/security/alerts?severity=high&page_size=5` | `ListRuntimeAlerts` | `handler_phase3_runtime_test.go` | ready | 已支持 severity 与 page_size 过滤 |
| 运行时处置（contain） | `/deployment/infrastructure/clusters/:id/operations` | `POST /clusters/:id/security/alerts/:id/contain` | `ContainRuntimeAlert` | `handler_phase3_runtime_test.go` | partial | `external_managed` 走 `suggest_only` |
| 漂移治理入口 | `/deployment/infrastructure/clusters/:id/workloads` | `GET /clusters/:id/apps/:name/drift` | `N/A` | `N/A` | partial | Phase 3 C4 规划项，当前未暴露可点击入口 |
