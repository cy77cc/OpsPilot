## Context

当前 AI Tool 体系存在两个问题：
1. **缺乏资源发现能力**：AI 执行操作需要知道 `cluster_id`、`host_id` 等参数，但没有便捷的方式获取这些信息
2. **缺乏 K8s 写操作能力**：运维场景常见的扩缩容、重启、删除 Pod 等操作无法通过 AI 完成

现有的审批中间件架构已完备，支持 `StatefulInterrupt` → SSE → 用户确认 → `ResumeWithParams` 流程。

## Goals / Non-Goals

**Goals:**
- 实现 `platform_discover_resources` 工具，支持查询集群、主机、服务、命名空间、指标元数据
- 实现 K8s 写操作工具集（P0: scale/restart/delete-pod，P1: rollback/delete-deployment）
- 所有写操作集成审批中间件，确保 Human-in-the-Loop

**Non-Goals:**
- 不实现 Alertmanager API 集成（告警数据流已正确）
- 不实现 K8s 资源创建/更新 YAML 的工具（P2 范围）
- 不修改前端审批 UI（复用现有组件）

## Decisions

### 1. 资源发现工具设计

**决策**: 单一工具 `platform_discover_resources` 支持多种资源类型查询

**理由**:
- AI 可以通过一次调用获取上下文，减少工具调用次数
- 与现有 `cluster_list_inventory`、`host_list_inventory` 等工具命名风格一致

**替代方案**:
- ❌ 拆分为多个工具 (`discover_clusters`, `discover_hosts`...) - 增加工具数量，AI 需多次调用

### 2. K8s 写操作工具参数设计

**决策**: 使用 `cluster_id` + `namespace` + `name` 三元组定位资源

**理由**:
- 与现有只读工具参数风格一致
- `resolveK8sClient` 已实现从 `cluster_id` 获取 K8s client 的逻辑

**输入参数**:
```go
type K8sScaleDeploymentInput struct {
    ClusterID int    `json:"cluster_id" jsonschema_description:"required,cluster id"`
    Namespace string `json:"namespace" jsonschema_description:"required,kubernetes namespace"`
    Name      string `json:"name" jsonschema_description:"required,deployment name"`
    Replicas  int    `json:"replicas" jsonschema_description:"required,target replica count"`
}
```

### 3. 滚动重启实现方式

**决策**: 通过更新 `spec.template.metadata.annotations` 触发滚动更新

**理由**:
- Kubernetes 原生支持，无需额外依赖
- 保持 Deployment 历史记录
- 不影响 `spec.revisionHistoryLimit`

```go
annotation := "kubectl.kubernetes.io/restartedAt"
deployment.Spec.Template.Annotations[annotation] = time.Now().Format(time.RFC3339)
```

### 4. 审批流程集成

**决策**: 写操作工具通过中间件自动触发审批，不需要工具内部处理

**理由**:
- 审批逻辑集中在中间件，便于维护
- 工具实现更简洁，只需关注业务逻辑
- 已有 `DefaultNeedsApproval` 和 `DefaultPreviewGenerator` 覆盖 K8s 工具

## Risks / Trade-offs

| 风险 | 缓解措施 |
|------|----------|
| 删除 Pod 可能导致服务短暂中断 | 审批预览明确提示 "控制器会重建新 Pod" |
| 扩缩容到 0 副本相当于停止服务 | 审批预览显示目标副本数，用户可拒绝 |
| 滚动重启可能导致服务不稳定 | 风险等级设为 medium，提示可能的短暂不稳定 |
| AI 可能误删重要资源 | 删除 Deployment 风险等级设为 critical |

## Migration Plan

无需迁移。新工具增量添加，不影响现有功能。

**部署顺序**:
1. 先部署资源发现工具
2. 再部署 K8s 写操作工具
3. 验证审批流程正常工作
