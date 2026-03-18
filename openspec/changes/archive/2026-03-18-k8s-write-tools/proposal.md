## Why

当前的 AI Tool 仅支持 K8s 只读操作（查询资源、事件、日志），无法执行变更类操作。运维场景中常见的扩缩容、重启、删除 Pod 等操作仍需人工介入。同时，AI 执行操作时需要知道资源 ID（cluster_id、host_id 等），但缺乏资源发现能力，导致操作流程不顺畅。

## What Changes

- 新增 `platform_discover_resources` 资源发现工具，支持查询集群、主机、服务、命名空间和指标元数据
- 新增 K8s 写操作工具集（P0 优先级）：
  - `k8s_scale_deployment` - Deployment 扩缩容
  - `k8s_restart_deployment` - Deployment 滚动重启
  - `k8s_delete_pod` - 删除 Pod
- 新增 K8s 写操作工具集（P1 优先级）：
  - `k8s_rollback_deployment` - Deployment 回滚
  - `k8s_delete_deployment` - 删除 Deployment（高风险）
- 所有写操作工具集成审批中间件，支持 Human-in-the-Loop 工作流

## Capabilities

### New Capabilities

- `platform-discovery`: 资源发现能力，AI 可查询平台内可用资源（集群、主机、服务、命名空间、指标）
- `k8s-workload-operations`: K8s 工作负载变更操作（扩缩容、重启、删除）

### Modified Capabilities

无。现有 K8s 只读工具保持不变。

## Impact

**新增文件：**
- `internal/ai/tools/platform/tools.go` - 资源发现工具
- `internal/ai/tools/kubernetes/write.go` - K8s 写操作工具

**修改文件：**
- `internal/ai/tools/kubernetes/tools.go` - 注册写操作工具
- `internal/ai/tools/tools.go` - 更新工具集入口
- `internal/ai/tools/middleware/approval.go` - 补充 K8s 工具的预览生成器

**依赖影响：**
- 复用现有 `internal/ai/tools/common/approval.go` 审批类型
- 复用现有 `internal/ai/tools/middleware/approval.go` 审批中间件
- 复用现有 K8s client 解析逻辑 `resolveK8sClient`
