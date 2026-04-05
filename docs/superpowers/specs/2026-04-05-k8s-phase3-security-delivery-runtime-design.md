# K8s Phase 3 设计：安全与交付平台化（A -> C -> B）

## 1. 背景与决策

基于路线图 `docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md`，Phase 3 目标是将平台从“可操作与可治理”升级为“生产门禁级安全与交付平台”。

已确认约束：

- 执行顺序：`A -> C -> B`
  - A：准入与镜像扫描门禁
  - C：GitOps + 应用市场
  - B：运行时安全治理
- 规划原则：以完整功能闭环为主，不按固定周期切分。
- 验收口径：按生产门禁标准验收（多集群灰度、SLO/SLA、灾备演练、值班 Runbook 全套）。

## 2. 总体目标与范围

### 2.1 总体目标

1. 建立从镜像供应链到集群准入的安全门禁闭环。
2. 建立从应用模板到多环境发布的标准化交付闭环。
3. 建立从运行时检测到自动处置与复盘的安全运营闭环。
4. 统一审计、回滚、告警与演练体系，满足生产门禁要求。

### 2.2 范围

- In Scope：
  - 准入与镜像扫描门禁（A）
  - GitOps 与应用市场标准化（C）
  - 运行时安全检测与处置（B）
  - 跨域治理（审计、SLO/SLA、演练、Runbook）
- Out of Scope：
  - 业务侧自定义安全编排 DSL
  - 跨组织多控制面联邦治理

## 3. 架构边界与治理复用

### 3.1 `external_managed` 集群权限模型（P0）

针对现有 `external_managed` 变更受限事实，本阶段明确“双运行模式”：

1. `platform_managed`：
  - A/C/B 全能力可执行（含自动处置、自动回滚、自动修复）。
2. `external_managed`：
  - A：可执行“准入前校验/策略建议”，但仅在授权范围内下发策略。
  - C：默认只读观测 + GitOps 状态回传；发布执行需外部控制面完成。
  - B：默认“检测 + 告警 + 审计”，自动隔离/阻断降级为“处置建议 + 工单”。

禁止在规格中假设 external_managed 集群具备强制写权限。

### 3.2 复用现有 Governance 基础设施（P0/P1）

Phase 3 不重新发明审批/审计模型，复用现有能力：

- `OperationApproval`（审批生命周期）
- `OperationAudit`（操作审计）
- `Scope`/`RiskLevel`（作用域与风险等级）
- `approval_policy.go` 与 `policy_release.go`（策略发布/回滚模式）

统一约定：

1. A/C/B 的阻断、熔断、处置动作均落 `OperationAudit`。
2. 需要人工确认的动作统一走 `OperationApproval`。
3. 风险分级统一采用 low/medium/high/critical 四级。

## 4. 技术栈与部署策略

采用已确认方案（A 方案）：

- A：`OPA Gatekeeper + Trivy + Kyverno(可选策略包)`
- C：`ArgoCD + Harbor(Helm OCI)`
- B：`Falco + Tetragon`

部署策略（P2）：

1. 集群内组件默认 Helm 安装，并接入现有集群引导流程。
2. 多集群分发采用“中心策略源 + 集群 Agent 拉取状态 + 平台推送控制”混合模式：
  - 配置源：平台数据库 + Git 仓库（GitOps 定义）
  - 状态源：集群回传（ArgoCD/Falco/Tetragon）
3. 关键组件版本纳入平台“集群能力清单”，支持升级与回滚记录。

## 5. 路线与任务结构（A -> C -> B）

### 5.1 P3-A：准入与镜像扫描门禁

1. A1：镜像供应链基线
  - Harbor 项目分级、签名策略、SBOM 与漏洞元数据落库
2. A2：Trivy 扫描门禁
  - 漏洞分级阈值、扫描报告持久化、阻断判定
3. A3：Admission 策略包
  - Gatekeeper/Kyverno 策略模板、版本化、回滚
  - 准入拒绝写入 `OperationAudit(code=admission_denied)`
4. A4：例外机制
  - 时效豁免 + 审批 + 自动过期回收 + 审计
5. A5：多集群一致性
  - 策略分发、版本比对、一致性校验、失败回滚

### 5.2 P3-C：GitOps + 应用市场

1. C1：ArgoCD 多集群接管
  - Project/RBAC 隔离、Sync Waves、健康检查
2. C2：应用市场
  - Harbor Helm OCI、模板门禁（schema/lint/安全基线）
3. C3：环境晋级发布
  - DEV -> STG -> PROD promotion、审批卡点
4. C4：漂移治理
  - 漂移检测、冲突提示、自动修复策略
5. C5：回滚与熔断
  - 版本回滚、配置回滚、发布失败自动熔断

### 5.3 P3-B：运行时安全治理

1. B1：运行时检测基线
  - Falco/Tetragon 规则分层（基础/加强/业务特化）
2. B2：告警与处置
  - 告警分级，`platform_managed` 支持自动处置，`external_managed` 降级为工单建议
3. B3：事件取证
  - 进程/网络/容器元数据快照，证据留存与审计关联
4. B4：运营闭环
  - 误报治理、规则灰度、命中率与噪声比看板
5. B5：策略反哺
  - 运行时事件反哺准入策略与发布策略

### 5.4 跨域治理任务（贯穿 A/C/B）

1. G1：统一审计模型
2. G2：SLO/SLA 与告警分级体系
3. G3：多集群灰度与灾备演练剧本
4. G4：值班 Runbook 与升级/降级手册

## 6. API 契约草案（P0）

### 6.1 A 门禁域

- `POST /api/v1/clusters/:id/admission/policies`
- `GET /api/v1/clusters/:id/admission/results`
- `POST /api/v1/clusters/:id/admission/exemptions`
- `POST /api/v1/clusters/:id/admission/exemptions/:exemption_id/revoke`

### 6.2 C GitOps 域

- `POST /api/v1/clusters/:id/apps`
- `GET /api/v1/clusters/:id/apps/:name`
- `POST /api/v1/clusters/:id/apps/:name/sync`
- `POST /api/v1/clusters/:id/apps/:name/rollback`

### 6.3 B 运行时域

- `GET /api/v1/clusters/:id/security/alerts`
- `GET /api/v1/clusters/:id/security/events/:event_id`
- `POST /api/v1/clusters/:id/security/alerts/:alert_id/resolve`
- `POST /api/v1/clusters/:id/security/alerts/:alert_id/contain`

## 7. 数据模型草案（P1）

新增或扩展实体：

1. `admission_policies`
  - 策略模板、版本、目标集群、发布状态
2. `admission_exemptions`
  - scope（cluster/namespace/workload）、expires_at、approval_id、status
3. `image_scan_reports`
  - image_digest、scanner、severity_summary、sbom_ref、policy_decision
4. `gitops_app_releases`
  - app、env、git_revision、sync_result、rollback_ref、audit_id
5. `runtime_security_events`
  - cluster_id、namespace、workload、rule_id、severity、raw_payload、dispose_status
6. `runtime_disposal_actions`
  - event_id、action、mode(auto/manual/suggest_only)、approval_id、audit_id

与既有表复用规则：

- 可复用 `OperationAudit` 的动作，优先复用，不创建冗余审计表。
- 审批统一复用 `OperationApproval`；资源级例外通过 scope 扩展字段或 `resource_ref` 落地。

## 8. 关键依赖与一致性架构（P2）

1. `A -> C/B`：准入门禁是交付与运行时安全前置约束。
2. `C -> B`：GitOps 变更记录作为运行时归因基线。
3. `B -> A/C`：运行时事件反馈推动策略持续收敛。

多集群一致性机制：

- 分发模式：中心 Push 下发策略版本，集群 Pull 上报生效状态。
- 基准源：平台策略版本库（单一真相源）。
- 演练触发：支持手动触发和计划任务触发两种模式。

## 9. 生产门禁级 DoD（量化）

### 9.1 A（准入与镜像）

1. Critical 漏洞（CVSS >= 9.0 或扫描器 Critical）在 PROD 拦截率 `100%`。
2. 未签名/签名无效镜像在 PROD 拦截率 `100%`。
3. 准入判定响应 p95 `< 800ms`。
4. 例外必须具备审批记录与过期时间，过期回收成功率 `100%`。
5. 多集群策略一致性检查通过率 `>= 99.9%`。

### 9.2 C（GitOps + 应用市场）

1. PROD 变更中 GitOps 路径覆盖率 `100%`。
2. 模板门禁漏检率 `0`（schema/lint/security）。
3. 发布失败熔断触发时间 `< 60s`。
4. 一键回滚成功率 `>= 99%`（平台托管集群）。
5. 漂移检测周期 `<= 5min`，关键漂移告警到达率 `100%`。

### 9.3 B（运行时安全）

1. 高危规则事件检测到告警延迟 p95 `< 30s`。
2. 告警关联到发布版本的可追溯率 `>= 99%`。
3. `platform_managed` 自动处置成功率 `>= 95%`。
4. `external_managed` 处置建议工单生成率 `100%`。
5. 误报率（月）`<= 10%`，持续下降。

### 9.4 跨域上线门槛

1. 多集群灰度演练：至少 1 次完整记录并通过复盘。
2. 灾备演练：准入/交付/运行时三链路各至少 1 次可复现演练。
3. 值班 Runbook 覆盖：A/C/B 全域关键故障路径覆盖率 `100%`。
4. 审计完整性：关键动作审计记录完整率 `100%`。

## 10. 风险与控制

1. 策略过严导致交付受阻  
   - 控制：灰度策略、时效豁免、审批、策略版本回滚。
2. 运行时告警噪声过高  
   - 控制：规则分层、误报治理、阈值调优。
3. 多集群策略漂移  
   - 控制：统一版本源、漂移告警、自动修复。
4. GitOps 与人工操作冲突  
   - 控制：PROD 禁止旁路发布，越权操作强审计。

## 11. 交付物清单

1. Phase 3 设计文档（本文件）
2. Phase 3 执行计划（后续 `writing-plans` 产出）
3. API 契约与数据模型文档（A/C/B 分域）
4. 子域 Runbook（准入、交付、运行时）
5. 生产门禁验收记录模板（灰度、SLO/SLA、灾备、值班）

## 12. 与路线图对齐

与 `2026-04-04-k8s-management-roadmap-chart.md` 一致：

- Phase 3 主题：安全与交付平台化
- 核心目标：准入/镜像扫描、运行时安全、GitOps/应用市场
- 在主题不变前提下，明确顺序 `A -> C -> B`，并增加可落地 API、数据模型、部署与量化 DoD。
