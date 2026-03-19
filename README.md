# OpsPilot / k8s-manage

OpsPilot 是一个面向平台工程、运维和研发团队的智能 PaaS 控制面。项目当前已经具备 Kubernetes/主机资源管理、服务与发布管理、监控告警、RBAC 治理，以及 AI Copilot 的对话、工具调用、审批恢复和流式可视化能力。

项目的长期目标不是“给 PaaS 加一个聊天框”，而是演进为一个 AI 驱动的智能 PaaS 平台：让用户通过统一控制面完成资源接入、应用交付、环境治理、运行诊断和受控变更，并让 AI 逐步承担分析、编排、执行建议和审批协同的职责。

## 项目目标

围绕“AI 驱动的智能 PaaS 平台”，当前目标可以概括为四条主线：

- 统一基础设施控制面：纳管 Kubernetes 集群、主机、项目、服务、部署目标和运行环境。
- 统一应用交付链路：覆盖服务建模、配置注入、发布、回滚、可观测与审计。
- 统一治理与安全边界：通过 JWT、Casbin、审批流和权限模型保证多角色受控操作。
- 统一 AI 操作入口：让 AI 不只做问答，还能结合场景上下文、平台工具和人工审批参与平台运行。

## 当前能力概览

后端已注册的核心业务域集中在 `/api/v1`，包括：

- 用户认证与访问治理：`user`、`rbac`
- 基础设施与资源：`host`、`cluster`、`node`、`project`
- 平台交付：`service`、`deployment`、`cicd`、`automation`
- 运维观测：`monitoring`、`dashboard`、`cmdb`、`topology`、`jobs`
- AI 控制面：`ai`

AI 模块当前不是独立 Demo，而是平台的一等能力，已具备：

- SSE 流式对话与 turn/block 结构化消息渲染
- 场景化 AI 会话、工具执行和执行过程可视化
- 审批中断与恢复能力，支持 human-in-the-loop
- 面向主机、Kubernetes、服务、发布、监控、治理等领域的工具集成

## 整体架构

系统采用单 Go 服务控制面，开发态前后端分离，生产态由后端嵌入前端静态资源。

```text
                           +----------------------+
                           |   React 19 + Vite    |
                           |   Ant Design 6 UI    |
                           +----------+-----------+
                                      |
                                      | HTTP / SSE / WebSocket
                                      v
+---------------------+    +----------+-----------+    +----------------------+
|  Browser / Operator | -> | Gin API / Route Hub  | -> | Domain Services      |
|  Dev / Ops / SRE    |    | /api/v1 + /ws        |    | host/cluster/...     |
+---------------------+    +----------+-----------+    +----------+-----------+
                                      |                           |
                                      |                           |
                                      v                           v
                           +----------+-----------+    +----------------------+
                           | AI Runtime / Tools   |    | DB / Redis / Casbin  |
                           | Eino + tool calling  |    | Prometheus / K8s API |
                           +----------------------+    +----------------------+
```

### 后端架构

- 入口：`main.go` -> `internal/cmd` -> `internal/server`
- Web 服务：Gin 提供 `/api/health`、`/api/v1/*` 和 `/ws/notifications`
- 服务装配：`internal/service/service.go` 统一注册各领域路由
- 运行依赖：`internal/svc` 负责装配 DB、Redis、本地缓存、Casbin、Prometheus 等上下文
- 配置管理：`internal/config` 基于 Viper 读取 `configs/config.yaml` 和环境变量
- 数据访问：GORM + `storage/migrations` 管理持久化与迁移

### AI 架构

AI 能力围绕 `internal/ai` 与 `internal/service/ai` 组织，核心组成包括：

- `internal/ai/runtime`：对话事件、流式投影、turn/block 生命周期
- `internal/ai/agents`：诊断、问答、变更等 agent 组织层
- `internal/ai/tools`：Kubernetes、host、service、deployment、monitor、governance 等工具
- `internal/service/ai`：AI HTTP 接口、SSE 输出、会话/审批/恢复处理

这意味着项目已经从“AI 辅助说明文档”进入“AI 参与平台执行链路”的阶段，后续可以继续往 AIOps、智能交付和策略执行演进。

### 前端架构

前端位于 `web/`，基于 React 19、TypeScript、Vite、Ant Design 6 和 `@ant-design/x` 构建，主要特征包括：

- `web/src/pages`：按业务域组织页面
- `web/src/components`：沉淀 AI、RBAC、布局、交互和可视化组件
- `web/src/api/modules`：统一的前端 API 调用入口
- `web/src/ProtectedApp.tsx`：受保护路由、菜单能力与域页面编排

当前前端已覆盖的主要平台页面包括：

- Dashboard
- 主机接入与终端
- 集群与部署基础设施
- 部署目标、环境引导、发布与审批
- 服务目录与服务部署
- 监控、CMDB、自动化、CI/CD、帮助中心
- AI Copilot 与相关运行态组件

## 仓库结构

```text
.
|-- api/                 # 各业务域 API 契约（v1）
|-- configs/             # 配置文件
|-- deploy/              # Docker Compose、K8s 部署清单
|-- docs/                # AI 设计与工程设计文档
|-- e2e/                 # E2E 与性能测试
|-- internal/
|   |-- ai/              # AI runtime、agent、tools、state
|   |-- cmd/             # Cobra 命令入口
|   |-- config/          # 全局配置
|   |-- dao/             # DAO 层
|   |-- model/           # 领域模型
|   |-- server/          # HTTP 服务启动
|   |-- service/         # 各业务域路由、handler、logic
|   `-- svc/             # ServiceContext 依赖装配
|-- resource/            # SQL、Casbin 等资源文件
|-- storage/             # DB 初始化与迁移
|-- web/                 # React 前端
`-- openspec/            # 规格、变更和能力基线
```

## 技术栈

### 后端

- Go 1.26.x
- Gin
- Cobra
- GORM
- Viper
- Redis
- Casbin
- Prometheus
- `client-go`
- CloudWeGo Eino

### 前端

- React 19
- TypeScript
- Vite
- Ant Design 6
- `@ant-design/x`
- Tailwind CSS
- React Router
- Axios

## 开发方式

### 本地开发

```bash
make dev-backend
make dev-frontend
```

默认地址：

- 前端：`http://127.0.0.1:5173`
- 后端：`http://127.0.0.1:8080`

开发态下：

- 前端由 Vite 提供
- 后端只提供 API 和 WebSocket
- 前端通过代理转发 `/api` 与 `/ws`

### 本地构建

```bash
make web-build
make build

# 或一步完成
make build-all

# 运行
make run
```

生产式构建下，后端会嵌入 `web/dist`，访问 `/` 时直接回落到前端 SPA。

### 数据库迁移

```bash
make migrate-up
make migrate-status
make migrate-down
```

## 测试命令

```bash
make test
make web-test
make test-all
```

也可以按领域执行：

- `make test-ai`
- `make test-cluster`
- `make test-deployment`
- `make test-notification`

## 关键文档

- 项目上下文：`openspec/project.md`
- OpenSpec 使用说明：`openspec/README.md`
- AI 路线图：`docs/ai/roadmap.md`
- AI Phase 1/2 设计：`docs/ai/phase1-phase2-technical-design.md`
- AI Phase 3/4 设计：`docs/ai/phase3-phase4-technical-design.md`
- 平台能力基线：`openspec/specs/platform-capability-baseline/spec.md`

## 演进方向

面向智能 PaaS 平台，后续建议持续增强以下能力：

- 让 AI 会话深度绑定主机、集群、服务、部署等业务场景
- 将诊断、变更建议、审批、执行和审计整合为闭环
- 提升服务目录、环境引导、部署模板和可观测的联动程度
- 逐步构建 AIOps、智能巡检、风险评估和策略驱动执行能力

如果把今天的 OpsPilot 视为“一个具备 AI 能力的 PaaS 控制面”，那么目标就是把它继续推进成“由 AI 驱动的智能 PaaS 平台”。
