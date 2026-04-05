# Cluster Detail UI 重构设计（信息降噪 + 紧凑交互 + 前后端对齐）

## 1. 背景与目标

当前 `ClusterDetailPage` 承载过多模块，信息密度与优先级失衡：

- 基础信息与高价值运行态信息混杂，关键决策路径不突出。
- 页面单体过大，交互反馈链路（审批/审计/失败处置）分散。
- 前后端能力映射缺少显式治理，容易出现前端入口与后端接口不一致。

本次重构目标：

1. 降低“基础信息”展示优先级（移出主视区）。
2. 以行动导向重建详情页首屏，提升紧凑度与交互效率。
3. 建立前后端能力矩阵，消除未对接接口风险。
4. 按领域重组 `internal/service/cluster`，降低维护复杂度。

优先级（已确认）：`A 信息层级 > B 页面紧凑 > C 交互体验`。

## 2. 方案评估与决策

### 2.1 方案候选

1. 轻改版：保留单页结构，仅调顺序与样式。
2. 中改版：首屏作战面板 + 功能 Tab 分区。
3. 重构版：拆分为总览 + 多子页面（本次选择）。

### 2.2 选择结论

采用 **方案 3（重构版）**：

- 中长期可维护性最佳，能彻底解决单体页面持续膨胀。
- 与后端领域拆分（handler/logic/repo）天然对齐。
- 可在保持 URL 可分享与审计跳转能力下，显著降低首屏噪声。

## 3. 信息架构（IA）

从“单页承载全部”改为 `1 + 5` 结构：

- 概览（Overview）：默认落地页
- 节点与容量（Nodes & Capacity）
- 工作负载（Workloads）
- 网络与流量（Network & Traffic）
- 安全与治理（Security & Governance）
- 基本信息（Info）

关键规则：

- `基本信息` 进入独立分区（用户确认为 B 路径：独立 Tab/页）。
- 首屏只保留“判断 + 决策 + 执行”相关内容。
- 所有低频字段（描述、创建更新时间、静态属性）默认不出现在首屏。

## 4. 交互模型与页面节奏

### 4.1 首屏结构

首屏采用双区模型：

- 主区：运行态总览（健康、告警、异常趋势、最近风险信号）
- 侧区：关键操作台（节点/证书/升级/治理入口）

次级页面承载深度数据与批量操作，不在首屏堆叠长表格。

### 4.2 统一操作反馈协议

所有高风险操作统一返回并渲染四态：

- `approval_required`：ticket、过期时间、继续执行入口
- `running`：阶段进度与最近事件
- `completed`：结果摘要 + audit_id + 跳转操作中心
- `failed/rejected`：错误摘要 + runbook 建议 + 重试入口

### 4.3 紧凑策略

- 表格使用小尺寸行高，默认展示关键列，次要列折叠。
- 指标卡限制为主值 + 趋势，不展示大段描述。
- 全局筛选（namespace / status / 时间）固定在二级导航下方工具条。
- 明细统一“按需展开”（Drawer/详情页），不在首屏长列表平铺。

## 5. 前后端能力对齐

新增 `Cluster Detail Capability Matrix`（设计期与开发期都必须维护）：

- 列：`页面能力` / `前端入口` / `API 契约` / `后端 handler/logic/repo` / `测试映射` / `状态`。
- 状态值：`ready` / `partial` / `missing`。

治理规则：

1. 任一能力为 `missing` 时，前端不得直接暴露可点击入口（需隐藏或降级）。
2. 新增页面分区前，必须先完成矩阵映射与契约确认。
3. 所有高风险动作统一复用治理信封，不允许页面侧自定义状态机。

## 6. `service/cluster` 目录化重组设计

目标：从平铺文件改为按领域目录组织，先“搬迁不改行为”，再逐步拆逻辑。

建议目录：

- `internal/service/cluster/handler/`
  - `overview_handler.go`
  - `node_handler.go`
  - `workload_handler.go`
  - `network_handler.go`
  - `security_handler.go`
  - `governance_handler.go`
- `internal/service/cluster/logic/`
  - `overview_logic.go`
  - `node_ops_logic.go`
  - `workload_ops_logic.go`
  - `network_policy_logic.go`
  - `security_runtime_logic.go`
- `internal/service/cluster/repository/`
  - `cluster_repo.go`
  - `node_repo.go`
  - `workload_repo.go`
  - `security_repo.go`
- `internal/service/cluster/contracts/`
  - DTO、统一响应结构、状态枚举
- `internal/service/cluster/routes/`
  - 按领域拆分路由注册

迁移策略：

1. 第 1 轮仅做“目录搬迁 + 导出适配层”，不改行为。
2. 第 2 轮逐步拆超大函数并补覆盖测试。
3. 每一轮迁移都保证可回滚（小步提交）。

## 7. 错误处理与可观测性

- 前端所有操作反馈都携带 `audit_id` 与状态码。
- 对 `approval_required`、`approval_rejected`、`suggest_only` 做统一 UI 语义映射。
- 首屏保留“最近失败操作摘要”与“快速跳转操作中心”。
- 关键交互埋点：入口点击、提交、审批待定、失败重试、详情跳转。

## 8. 测试策略

### 8.1 前端

- 页面级：概览页渲染优先级、基础信息分区隔离、紧凑布局断言。
- 交互级：高风险操作四态反馈、审计跳转、筛选联动。
- 契约级：`cluster.phase3` 与 cluster operation envelope 兼容测试。

### 8.2 后端

- handler 路由可达性（分目录后保持原路径兼容）。
- governance 信封一致性（approval/rejected/failed/completed）。
- 目录迁移回归（迁移前后接口行为一致）。

### 8.3 联调

- Capability Matrix 中 `ready` 项必须通过端到端联调。
- `partial/missing` 项必须有明确降级策略与 UI 保护。

## 9. 非目标（本次不做）

- 全量引入新设计系统或全站视觉改版。
- 改造非 cluster 领域页面结构。
- 引入新的后端基础设施依赖。

## 10. 交付物

1. 本设计文档。
2. Capability Matrix 文档（后续执行计划阶段创建）。
3. 分阶段实施计划（由 writing-plans 产出）。
