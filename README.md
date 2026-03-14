# k8s-manage

|目录/文件名称 |	说明 |	描述 |
|--|--|--|
|api| 	对外接口 | 	对外提供服务的输入/输出数据结构定义。考虑到版本管理需要，往往以 api/xxx/v1... 存在。|
|hack| 	工具脚本| 	存放项目开发工具、脚本等内容。例如，CLI 工具的配置，各种 shell/bat 脚本等文件。|
|internal| 	内部逻辑| 	业务逻辑存放目录。通过 Golang internal 特性对外部隐藏可见性|
|    cmd	| 入口指令| 	命令行管理目录。可以管理维护多个命令行。|
|    consts	| 常量定义	| 项目所有常量定义。|
|    controller| 	接口处理| 	接收/解析用户输入参数的入口/接口层。|
|    dao	| 数据访问| 	数据访问对象，这是一层抽象对象，用于和底层数据库交互，仅包含最基础的 CRUD 方法。|
|    logic	| 业务封装| 	业务逻辑封装管理，特定的业务逻辑实现和封装。往往是项目中最复杂的部分。|
|    model	| 结构模型	| 数据结构管理模块，管理数据实体对象，以及输入与输出数据结构定义。|
|        do	| 领域对象	| 用于 dao 数据操作中业务模型与实例模型转换，由工具维护，用户不能修改。|
|        entity	| 数据模型	| 数据模型是模型与数据集合的一对一关系，由工具维护，用户不能修改。|
|    service	| 业务接口	| 用于业务模块解耦的接口定义层。具体的接口实现在 logic 中进行注入。|
|manifest	| 交付清单	| 包含程序编译、部署、运行、配置的文件。常见内容如下：|
|    config	| 配置管理	| 配置文件存放目录。|
|    docker	| 镜像文件	| Docker 镜像相关依赖文件，脚本文件等等。|
|    deploy	| 部署文件	| 部署相关的文件。默认提供了 Kubernetes 集群化部署的 Yaml 模板，通过 kustomize 管理。|
|    protobuf	| 协议文件	| GRPC 协议时使用的 protobuf 协议定义文件，协议文件编译后生成 go 文件到 api 目录。|
|resource	| 静态资源| 	静态资源文件。这些文件往往可以通过资源打包/镜像编译的形式注入到发布文件中。|
|go.mod	| 依赖管理	| 使用 Go Module 包管理的依赖描述文件。|
|main.go|	入口文件|	程序入口文件。|

## Go 驱动的 Paas 平台功能规划图

### 节点管理

### 1. 核心资源编排 (The Infrastructure)
这是平台的“引擎”，负责与底层基础设施（通常是 K8s）交互。
多租户隔离 (Multi-tenancy): 基于 Namespace 的资源物理隔离及基于 RBAC 的逻辑隔离。
计算资源管理: 支持容器（Pod）的规格定义（CPU/Memory Limit）、扩缩容策略（HPA/VPA）。
存储与网络: 自动化配置 Ingress/Gateway、动态关联 PV/PVC 存储卷。
自定义资源 (CRD): 使用 Go 编写 Operator，将复杂的中间件（Redis, MySQL）抽象为 Paas 资源。

### 2. 应用生命周期管理 (ALM)
这是用户感知最明显的部分，即“应用是如何跑起来的”。
代码到镜像 (Build Service): 集成 Cloud Native Buildpacks 或 Dockerfile 自动构建，支持代码仓库 Webhook。
部署流水线 (CI/CD): 支持灰度发布（Canary）、蓝绿部署（Blue-Green）。
环境管理: 一键克隆“开发、测试、生产”多套环境。
配置中心: 类似 Apollo 或 ConfigMap 的管理，支持热更新及敏感信息（Secret）加密。

### 3. 开发者体验 (Developer Experience)
一个好的 Paas 平台必须让开发人员“用得爽”。
服务目录 (Service Catalog): 预置常用的中间件（DB, Message Queue）模板，点击即部署。
日志与链路追踪: 统一收集 stdout 日志，集成 OpenTelemetry 进行链路追踪（Tracing）。
Web 终端: 允许开发者直接通过浏览器进入容器控制台（使用 Gorilla WebSocket 实现）。
API 网关/服务网格: 自动注入 Sidecar（如 Istio），实现服务间流量加密和负载均衡。

### 4. 运营与治理 (Governance)
计量计费 (Metering): 统计各租户的资源消耗情况，生成账单。
监控告警: 集成 Prometheus 抓取指标，支持邮件、钉钉、Slack 告警推送。
配额管理 (Quota): 限制单个项目或团队的最大 CPU/内存使用量。

## PaaS 平台功能模块架构规划

### 1. 资源编排模块 (Resource Orchestration)
这是平台的“大脑”，负责管理底层的计算资源。
集群管理：支持多 K8s 集群接入，监控节点（Node）健康状态。
多租户隔离：通过 K8s Namespace 实现物理隔离，通过 RBAC 维护不同团队的权限。
配额管理 (Resource Quota)：限制每个项目能使用的 CPU、内存和磁盘配额。

### 2. 应用交付模块 (Application Delivery / CD)
解决“代码如何变成运行中的服务”的问题。
应用定义：支持通过 Web 界面配置环境变量、端口映射、启动脚本。
发布策略：集成蓝绿发布、滚动更新（Rolling Update）和金丝雀发布（Canary）。
弹性伸缩 (Auto-scaling)：根据 CPU 利用率或自定义指标自动增减 Pod 数量（HPA）。

### 3. 构建与镜像模块 (CI/Image Management)
自动化构建：集成 Git Webhook，代码提交后触发 Docker 镜像构建。
内置镜像仓库：私有镜像托管（可集成 Harbor 接口）。
制品管理：记录每次构建的版本、作者、Commit ID，支持一键回滚。

### 4. 运维治理模块 (Observability & Ops)
日志中心：集成 EFK (Elasticsearch + Fluentd + Kibana) 或 Loki，实现在线查看实时日志。
监控告警：基于 Prometheus 和 Grafana，提供 CPU/内存/网络 IO 的可视化看板。
Web Shell：通过 WebSocket 实现浏览器终端，直接进入容器排查问题。

### 5. 开发者门户 (Developer Portal)
服务目录 (Marketplace)：预置 MySQL、Redis、Kafka 等中间件，用户一键申请。
域名/网关管理：自动配置 Ingress，管理 SSL 证书。
API 开放平台：允许外部系统通过 API 调用平台功能。

### 场景分析：AI 能为 K8s 管理平台做什么？
在写代码之前，我们要明确 AI 在这个项目里的“角色”。基于 Eino 的能力，我们可以实现以下几个阶段的功能：

- 阶段一：K8s 智能助手 (Copilot)
  - 功能 : 回答 K8s 基础知识，生成 YAML 文件，解释错误日志。
  - 实现 : 基础的 Prompt -> LLM -> Output 链路。
- 阶段二：集群诊断 (RAG/Context)
  - 功能 : 用户问 "为什么我的 Pod 挂了？"，AI 自动读取当前集群的 Event 和 Log 进行分析。
  - 实现 : 将 K8s 查询结果作为 Context 注入 Prompt。
- 阶段三：运维 Agent (Tool Use)
  - 功能 : 用户说 "帮我扩容 nginx 到 3 个副本"，AI 调用 client-go 执行操作。
  - 实现 : 使用 Eino 的 ToolsNode 封装 Service 层的方法。

## 本地开发（前后端分离）

```bash
# 终端 1: 启动后端开发模式（只提供 API / WebSocket）
make dev-backend

# 终端 2: 启动前端 Vite 开发服务
make dev-frontend
```

说明：

- 前端开发地址默认是 `http://127.0.0.1:5173`
- 后端开发地址默认是 `http://127.0.0.1:8080`
- 前端通过 Vite proxy 转发 `/api` 和 `/ws` 到后端
- 开发模式下后端不再加载 embed 的前端静态资源，因此不需要每次改前端都先重新构建 `web/dist`

## 本地构建与运行（生产式一体）

```bash
# 1) 编译前端静态资源到 web/dist
make web-build

# 2) 编译后端（会 embed 当前 web/dist）
make build

# 或一步完成
make build-all

# 本地启动
make run
```

说明：

- 生产构建模式下，服务启动后访问 `/` 会直接加载 embed 的前端页面。
- API 统一前缀为 `/api/v1`。

## Monitoring Deployment

Prometheus and Alertmanager compose configs:

- `deploy/compose/prometheus/docker-compose.yml`
- `deploy/compose/alertmanager/docker-compose.yml`

Quick start:

```bash
docker compose -f deploy/compose/prometheus/docker-compose.yml up -d
docker compose -f deploy/compose/alertmanager/docker-compose.yml up -d
```

Rule sync API:

```bash
curl -X POST http://127.0.0.1:8080/api/v1/alerts/rules/sync
```

Related docs:

- Monitoring API: `docs/ops/monitoring-alerting-api.md`
- Alerting config guide: `docs/ops/alerting-configuration-guide.md`

## 帮助文档与 AI 知识库

- 平台帮助文档（给用户）：`docs/user/help-center-manual.md`
- 运维值班 FAQ 100 题：`docs/user/ops-faq-100.md`
- FAQ 一题一条 JSONL：`docs/ai/ops-faq-100.jsonl`
- AI 帮助知识库（给模型）：`docs/ai/help-knowledge-base.md`
- FAQ 喂料说明：`docs/ai/ops-faq-100-kb.md`
- AI 分块知识（RAG/向量检索）：`docs/ai/help-knowledge-base.jsonl`

后端 AI 聊天在识别“帮助/如何操作”类问题时，会自动注入对应帮助知识片段，提升回答一致性与可执行性。

当前 AI 抽屉已经升级为 `turn -> blocks` 的流式渲染模型：

- 后端在 `/api/v1/ai/chat` 和 `/api/v1/ai/resume/step/stream` 中同时输出兼容 SSE 事件与原生 turn/block 生命周期事件
- 前端优先用 turn/block 事件驱动状态、工具、审批、证据和最终回答渲染，并保留 legacy message 兼容路径
- 审批恢复会继续写入原 assistant turn，历史会话回放也会优先读取结构化 `turns` 合同

当前已接入 `plan-execute-replan-visualization` 主流程，可视化语义包括：

- `phase_started / phase_complete`：显示 `planning / executing / replanning` 阶段切换
- `plan_generated`：显示结构化步骤列表
- `step_started / step_complete`：显示步骤状态推进
- `tool_call / tool_result`：显示工具参数与执行结果
- `approval_required`：显示审批确认卡片

实现主路径：

- 后端：`internal/ai/orchestrator.go` -> `internal/ai/runtime/*`
- 前端：`web/src/components/AI/Copilot.tsx` -> `turnLifecycle.ts` -> `messageBlocks.ts` -> `AssistantMessageBlocks.tsx`

更多实现映射见：

- `docs/refactor/plan-execute-replan-implementation-remap.md`
- `docs/ai-api.md`

### AIV2 单 Agent Runtime

AI 网关默认启用 `internal/aiv2`，也可以通过 `feature_flags.ai_assistant_v2` 显式切换：

- 未配置 / `true`：使用新的单 `ChatModelAgent + Runner` runtime
- `false`：回退到 legacy 多阶段 runtime（`rewrite -> planner -> executor -> summarizer`）

`aiv2` 的设计目标：

- 直接复用现有 host / k8s / service / delivery / observability tools
- 去掉 `expert agent as tool` 的额外模型跳转
- 用 Eino ADK `Interrupt / ResumeWithParams` 做 human-in-the-loop 审批
- 继续复用现有 `/api/v1/ai/chat`、`/api/v1/ai/resume/step`、`/api/v1/ai/resume/step/stream` 和前端 SSE 契约

回滚方式：

- 将 `feature_flags.ai_assistant_v2` 设回 `false`
- 网关会自动恢复到 legacy runtime，无需改前端路由
