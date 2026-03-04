# README.md 愿景 vs 实现差距分析

**Last Updated**: 2026-03-04
**Purpose**: 对照 README.md 规划，分析当前实现状态，识别功能缺口

---

## 1. 核心资源编排 (The Infrastructure)

> 这是平台的"引擎"，负责与底层基础设施（通常是 K8s）交互。

| 功能 | README 描述 | 实现状态 | 证据 | 差距说明 |
|------|-------------|----------|------|----------|
| 多租户隔离 | 基于 Namespace 的资源物理隔离及基于 RBAC 的逻辑隔离 | ⚠️ In Progress | `internal/service/rbac/routes.go` | RBAC 有基础，但 admin 临时全量放行策略待收敛 |
| 计算资源管理 | 支持容器（Pod）的规格定义、扩缩容策略（HPA/VPA） | ⚠️ 部分实现 | `internal/service/cluster/handler/handler_hpa.go` | HPA Done，VPA 未实现 |
| 存储与网络 | 自动化配置 Ingress/Gateway、动态关联 PV/PVC | ⚠️ 部分实现 | 集群路由有基础 | Ingress 配置有 API，Gateway/PVC 动态关联不完整 |
| 自定义资源 (CRD) | 使用 Go 编写 Operator，将复杂中间件抽象为 Paas 资源 | ❌ 未实现 | - | 完全缺失 |
| 配额管理 | 限制单个项目或团队的最大 CPU/内存使用量 | ✅ Done | `internal/service/cluster/logic_quota.go` | 已实现 quota/limit-range |

---

## 2. 应用生命周期管理 (ALM)

> 这是用户感知最明显的部分，即"应用是如何跑起来的"。

| 功能 | README 描述 | 实现状态 | 证据 | 差距说明 |
|------|-------------|----------|------|----------|
| 代码到镜像 | 集成 Cloud Native Buildpacks 或 Dockerfile 自动构建，支持代码仓库 Webhook | ⚠️ 骨架实现 | `internal/service/cicd/routes.go` | 有 CI 配置 API，缺少 Webhook 自动触发、镜像构建流水线 |
| 部署流水线 | 支持灰度发布（Canary）、蓝绿部署（Blue-Green） | ⚠️ 部分实现 | `web/src/pages/CICD/CICDPage.tsx` | API 支持 rolling/blue-green/canary，前端有基础表单 |
| 环境管理 | 一键克隆"开发、测试、生产"多套环境 | ⚠️ 有基础 | `web/src/pages/Deployment/Targets/` | 环境管理有页面，克隆功能不完整 |
| 配置中心 | 类似 Apollo 或 ConfigMap 的管理，支持热更新及敏感信息（Secret）加密 | ✅ In Progress | `web/src/pages/ConfigCenter/` | 有 ConfigAppsPage, ConfigListPage, ConfigDiffPage, ConfigMultiEnvPage |

---

## 3. 开发者体验 (Developer Experience)

> 一个好的 Paas 平台必须让开发人员"用得爽"。

| 功能 | README 描述 | 实现状态 | 证据 | 差距说明 |
|------|-------------|----------|------|----------|
| **服务管理可见性与中间件能力** | 在统一服务入口中管理中间件/业务服务并支持可见性授权 | ⚠️ In Progress | `internal/service/service/` + `web/src/pages/Services/` | 正在将 catalog 合并到 services |
| 日志与链路追踪 | 统一收集 stdout 日志，集成 OpenTelemetry 进行链路追踪（Tracing） | ❌ 未实现 | - | 无 EFK/Loki 集成，无 Tracing 可视化 |
| Web 终端 | 允许开发者直接通过浏览器进入容器控制台（使用 Gorilla WebSocket 实现） | ✅ Done | `web/src/pages/Hosts/HostTerminalPage.tsx` | 已实现，使用 WebSocket |
| API 网关/服务网格 | 自动注入 Sidecar（如 Istio），实现服务间流量加密和负载均衡 | ❌ 未实现 | - | 完全缺失 |

---

## 4. 运营与治理 (Governance)

| 功能 | README 描述 | 实现状态 | 证据 | 差距说明 |
|------|-------------|----------|------|----------|
| 计量计费 (Metering) | 统计各租户的资源消耗情况，生成账单 | ❌ 未实现 | - | 完全缺失 |
| 监控告警 | 集成 Prometheus 抓取指标，支持邮件、钉钉、Slack 告警推送 | ⚠️ 骨架实现 | `internal/service/monitoring/notifier.go` | 有 alerts/alert-rules API，通知适配器是 skeleton，注释: "Skeleton adapter. Real HTTP delivery can replace this without changing handler contract" |
| 配额管理 (Quota) | 限制单个项目或团队的最大 CPU/内存使用量 | ✅ Done | 同上 | 已实现 |

---

## 5. AI 能力阶段分析

> README 场景分析：AI 能为 K8s 管理平台做什么？

| 阶段 | 功能描述 | 实现状态 | 证据 | 差距说明 |
|------|-------------|----------|------|----------|
| 阶段一: K8s 智能助手 | 回答 K8s 基础知识，生成 YAML 文件，解释错误日志 | ✅ In Progress | `internal/service/ai/` | Eino + Ollama，有 chat handler |
| 阶段二: 集群诊断 (RAG/Context) | 用户问 "为什么我的 Pod 挂了？"，AI 自动读取集群 Event 和 Log 分析 | ✅ In Progress | `docs/ai/help-knowledge-base.md` | 有知识库，可注入 Context |
| 阶段三: 运维 Agent (Tool Use) | 用户说 "帮我扩容 nginx 到 3 个副本"，AI 调用 client-go 执行操作 | ⚠️ 部分实现 | `openspec/specs/ai-assistant-command-bridge/` | 审批流程有基础，执行审计有内存态路径 |

---

## 6. PaaS 功能模块架构规划对照

### 资源编排模块
- ✅ 集群管理：支持多 K8s 集群接入，监控节点（Node）健康状态
- ✅ 多租户隔离：通过 K8s Namespace 实现物理隔离，通过 RBAC 维护不同团队的权限
- ✅ 配额管理：限制每个项目能使用的 CPU、内存和磁盘配额

### 应用交付模块 (CD)
- ✅ 应用定义：支持通过 Web 界面配置环境变量、端口映射、启动脚本
- ⚠️ 发布策略：集成蓝绿发布、滚动更新和金丝雀发布
- ✅ 弹性伸缩：根据 CPU 利用率自动增减 Pod 数量（HPA）

### 构建与镜像模块 (CI)
- ⚠️ 自动化构建：集成 Git Webhook，代码提交后触发 Docker 镜像构建
- ❌ 内置镜像仓库：私有镜像托管（可集成 Harbor 接口）
- ⚠️ 制品管理：记录每次构建的版本、作者、Commit ID，支持一键回滚

### 运维治理模块 (Observability)
- ❌ 日志中心：集成 EFK 或 Loki，实现在线查看实时日志
- ⚠️ 监控告警：基于 Prometheus 和 Grafana，提供 CPU/内存/网络 IO 的可视化看板
- ✅ Web Shell：通过 WebSocket 实现浏览器终端

### 开发者门户
- ⚠️ 中间件服务入口：通过服务管理统一提供中间件创建与部署能力
- ⚠️ 域名/网关管理：自动配置 Ingress，管理 SSL 证书
- ❌ API 开放平台：允许外部系统通过 API 调用平台功能

---

## 7. 功能缺失优先级

### P0 - 阻塞性缺失 (影响核心价值)

| # | 功能 | 影响 | 建议 |
|---|------|------|------|
| 1 | 服务管理统一入口 | 目录与服务双入口带来认知和维护成本 | **进行中：并入 services** |
| 2 | 日志中心 | 运维排查问题的核心能力，完全缺失 | 需要 EFK/Loki 集成 |
| 3 | 真实通知推送 | 告警系统只有骨架，无法真正通知用户 | 补充钉钉/Slack/邮件 Provider |

### P1 - 体验性缺失 (影响用户体验)

| # | 功能 | 影响 | 建议 |
|---|------|------|------|
| 4 | 镜像仓库集成 | 没有内置 Harbor 集成或私有仓库 | 集成 Harbor API |
| 5 | 链路追踪 | OpenTelemetry 集成缺失 | 需要 Tracing 可视化 |
| 6 | 构建流水线 | Git Webhook 自动触发未完整 | 完善 CI/CD 集成 |

### P2 - 扩展性缺失 (影响高级场景)

| # | 功能 | 影响 | 建议 |
|---|------|------|------|
| 7 | CRD Operator | 自定义资源编排能力 | 编写 Go Operator |
| 8 | 服务网格 | Istio Sidecar 注入 | 需要 Istio 集成 |
| 9 | 计量计费 | 多租户资源统计和账单 | 需要计量服务 |
| 10 | API 开放平台 | 外部系统集成能力 | 需要 API Gateway |

---

## 8. 相关文档

- 平台状态矩阵: `docs/platform-status-matrix.md`
- 项目路线图: `docs/roadmap.md`
- 进度记录: `docs/progress.md`
- OpenSpec Specs: `openspec/specs/`
