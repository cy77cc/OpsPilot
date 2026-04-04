# K8s 管理平台路线图与第一阶段设计（Roadmap + Phase 1）

## 1. 背景与目标

当前集群详情页以“只读列表”为主，缺乏高频运维与资源操作能力，导致平台在日常生产运维场景中无法形成闭环。

本设计目标分两层：

1. 长期：沉淀完整的生产就绪 K8s 管理平台路线图，覆盖集群生命周期、工作负载、网络、可观测、安全、交付六大域。
2. 近期：定义 Phase 1（6-8 周）可落地范围，以“可见价值优先 + 最小底座能力”方式快速形成可验收版本。

已确认约束：

- 执行策略：70% 可交付能力 + 30% 基础能力（混合）
- 周期：6-8 周
- 团队规模：6 人以上
- 推进方式：以纵向场景主线交付，并并行补齐主线必需底座

## 2. 能力域全景（长期路线图）

### 2.1 多集群与生命周期管理

- 集群纳管与创建：支持私有云、公有云与边缘节点的一键创建或导入。
- 节点池管理：支持弹性扩缩容、GPU 节点调度与隔离。
- 升级与备份恢复：支持控制平面/节点平滑升级、etcd 备份与灾备恢复。

### 2.2 资源与工作负载管理

- 资源编排：可视化管理 Deployment、StatefulSet、DaemonSet、Job/CronJob。
- 配置与存储映射：统一管理 ConfigMap、Secret、PV/PVC/StorageClass、CSI 状态。
- 弹性伸缩：支持 HPA（CPU/内存/自定义指标）与 VPA。

### 2.3 网络与流量管理

- 服务暴露与网关：管理 Service、Ingress、域名与 SSL 证书。
- CNI 可视化：展示插件状态与 IPAM 分配。
- 网络策略控制：可视化 NetworkPolicy、跨 Namespace 隔离。
- 服务网格：集成 Istio/Linkerd，支持灰度、熔断、限流。

### 2.4 可观测性与智能运维

- 监控大盘：Prometheus/Grafana 指标可视化。
- 日志聚合：EFK/Loki 统一日志检索。
- 链路与告警：调用链、阈值告警、Webhook/邮件/IM 通知。
- AIOps：自动化诊断与初步根因分析。

### 2.5 平台安全与合规

- RBAC 与多租户隔离：最小权限原则。
- 镜像安全扫描：部署前漏洞拦截。
- 运行时安全：异常行为、越权与逃逸检测。

### 2.6 应用商店与 CI/CD 交付

- Helm 私有仓库：企业应用市场与中间件一键部署。
- GitOps：基于 ArgoCD/Flux 的声明式持续交付。

## 3. 分阶段路线图（建议）

### Phase 1（当前设计，6-8 周）

主题：集群详情页从“只读”升级为“可操作闭环”。

### Phase 2（下一阶段）

主题：网络策略与可观测深度能力。

- NetworkPolicy 可视化策略编排
- Service/Ingress 与证书全流程治理
- 指标、日志、链路三位一体定位

### Phase 3（中长期）

主题：安全治理与交付平台化。

- 准入控制与镜像扫描门禁
- 运行时安全治理
- 应用市场 + GitOps 标准化交付

## 4. Phase 1 范围（已确认）

## 4.1 可交付主线（70%）

1. 节点操作闭环
- 支持 `cordon/uncordon/drain/remove`
- 支持节点标签/污点维护
- 支持操作结果反馈与状态追踪

2. 工作负载基础操作
- Deployment/StatefulSet：扩缩容、重启、滚动状态查看
- Pod：查看与基础处置动作

3. 服务与流量基础操作
- Service/Ingress：列表 + 基础创建/编辑/删除
- 暂不引入 Service Mesh 深度治理能力

4. 详情页体验升级
- 从纯列表升级为“资源视图 + 操作面板 + 状态反馈 + 历史追踪入口”

## 4.2 基础能力（30%，仅主线必需）

1. 统一操作任务模型
- 异步执行与标准状态机：`pending/running/completed/failed/approval_required/rejected`
- 标准化返回结构（含 `audit_id`、message、approval 信息）

2. 权限与审批最小闭环
- 基于 RBAC 的操作授权
- 高风险动作审批 token 流程

3. 审计与可追踪
- 每次操作写入审计记录
- 支持从详情页跳转操作中心回溯链路

4. 失败恢复策略
- 高风险动作提供补救指引（runbook）
- Phase 1 不承诺全自动回滚

### 4.2.1 操作响应契约冻结（Phase 1）

为避免前后端在审批与失败语义上发生漂移，Phase 1 固定以下响应契约：

| 字段 | 类型 | 说明 |
|------|------|------|
| `state` | string | 传输状态，固定为 `completed` / `approval_required` / `rejected` / `failed` |
| `code` | string | 业务码，固定为 `success` / `approval_required` / `approval_rejected` / `failed`（或明确的错误码） |
| `approval` | object? | 审批信息，`state=approval_required` 时必有 `required=true` 或可推导审批信息 |
| `audit_id` | string\\|number? | 审计记录 ID，用于跳转操作中心追踪 |
| `message` | string | 面向用户的结果说明 |

状态与业务码映射：

- `state=completed` -> `code=success`
- `state=approval_required` -> `code=approval_required`
- `state=rejected` -> `code=approval_rejected`
- `state=failed` -> `code=failed` 或具体失败码（如 token/permission 相关）

兼容性约束：

- 历史返回中的 `approval_rejected` 仍必须归一化为 `state=rejected`，即便同时包含 approval 元数据。
- 前端允许从 `audit_id|auditId|operation_id|operationId` 兼容提取审计标识，但输出字段统一为 `audit_id`。

## 4.3 非目标（Phase 1 不做）

- Service Mesh 深度能力（灰度、熔断、限流等）
- CNI/IPAM 深层可视化与 eBPF/iptables 映射
- 全量 GitOps 与应用市场高级能力
- VPA 全面生产化与运行时安全深度检测

## 5. Phase 1 参考架构

### 5.1 分层设计

- 展示层：集群详情页各 Tab 的资源视图与操作入口
- 应用层：统一操作调用封装（发起、审批、轮询、反馈）
- 领域层：NodeOps / WorkloadOps / ServiceOps 资源操作处理器
- 平台层：RBAC、审批流、审计日志、任务执行器

### 5.2 统一交互数据流

1. 用户在详情页触发操作。
2. 前端提交标准操作请求。
3. 后端返回状态与 `audit_id`。
4. 若为 `approval_required`，前端发起审批补充并重试。
5. 前端展示操作状态与审计跳转。
6. 用户在操作中心查看完整执行链路。

## 6. 错误处理与恢复策略

- 可重试错误：超时、短暂网络抖动、瞬时依赖失败；前端提供重试。
- 不可重试错误：权限不足、参数非法、策略拒绝；返回明确原因与修复建议。
- 高风险失败：必须保留审计与处置建议，避免“无声失败”。

## 7. 测试与验收策略

### 7.1 测试层次

- 单元测试：状态机分支、审批分支、错误分支、UI 条件渲染
- 集成测试：节点/工作负载/服务关键操作链路
- 回归测试：RBAC 权限矩阵、审计完整性、关键页面可用性

### 7.2 验收标准

高风险操作必须满足“四可”：

- 可授权（权限/审批闭环）
- 可执行（执行成功或明确失败）
- 可追踪（状态可见）
- 可审计（审计链路完整）

## 8. 里程碑（6-8 周建议）

- Week 1-2：任务/审计/权限最小底座 + 节点操作联调
- Week 3-4：工作负载操作 + 详情页交互改造
- Week 5-6：Service/Ingress 基础操作 + 失败处理增强
- Week 7-8：稳定性、回归、验收与上线准备

## 9. 风险与应对

- 范围扩张风险：严格按 Phase 1 非目标收敛。
- 跨端协同风险：统一操作协议与错误码，前后端并行对齐。
- 审批链路复杂度：优先保障高风险动作，低风险动作直通。
- 上线质量风险：把权限与审计回归列为阻塞验收项。

## 10. 后续拆分建议（文档层）

建议在本总纲基础上补充子文档：

- `cluster-lifecycle.md`
- `workload-management.md`
- `network-traffic.md`
- `observability-aiops.md`
- `security-compliance.md`
- `app-catalog-delivery.md`

每个子文档聚焦领域模型、接口契约、权限矩阵、验收用例。
