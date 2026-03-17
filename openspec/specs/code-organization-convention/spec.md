# Code Organization Convention

## Purpose

定义项目代码的组织规范，要求使用目录形式进行功能分类，避免单个目录下堆积大量文件，提高代码可维护性和可发现性。
## Requirements
### Requirement: Directory Organization SHALL Use Functional Grouping

代码目录 SHALL 使用功能分组，禁止在单个目录下放置超过 10 个同级文件。当文件数量超过阈值时，必须按功能域拆分为子目录。

**Threshold Rules:**
| Category | Max Files Per Directory | Action When Exceeded |
|----------|------------------------|----------------------|
| Models | 10 | Group by domain |
| Handlers | 8 | Group by resource |
| Components | 12 | Group by feature |
| Hooks | 10 | Group by concern |
| Utils | 8 | Group by purpose |
| API Modules | 15 | Group by domain |

#### Scenario: model directory exceeds threshold

- **GIVEN** `internal/model/` contains more than 10 model files
- **WHEN** maintainers organize the codebase
- **THEN** the models SHALL be split into subdirectories by domain (e.g., `model/ai/`, `model/user/`, `model/cluster/`)
- **AND** each subdirectory SHALL contain a `doc.go` or have clear naming

#### Scenario: component directory exceeds threshold

- **GIVEN** `web/src/components/` contains more than 12 component directories
- **WHEN** maintainers organize the codebase
- **THEN** related components SHALL be grouped into feature directories (e.g., `components/Feedback/`, `components/DataDisplay/`)

---

### Requirement: AI Tools Directory SHALL Be Organized By Domain

AI 工具目录 SHALL 按领域域分组，每个领域一个子目录，根目录只保留核心类型和注册入口。

**Threshold Rules:**
| Category | Max Files Per Directory | Action When Exceeded |
|----------|------------------------|----------------------|
| Tools Root | 10 | Group implementations into `impl/` subdirectory |
| Param Utilities | 8 | Keep in `param/` subdirectory |

**Recommended Structure:**
```
internal/ai/tools/
├── contracts.go           # Core types (ToolMeta, ToolResult, errors)
├── registry.go            # Tool registration (BuildLocalTools)
├── runner.go              # Execution logic (runWithPolicyAndEvent)
├── wrapper.go             # Risk-based wrappers
├── builder.go             # Tool building utilities
├── category.go            # Scene-based tool filtering
├── param/                 # Parameter handling
│   ├── hints.go
│   ├── resolver.go
│   └── validator.go
└── impl/                  # Tool implementations by domain
    ├── kubernetes/
    │   └── tools.go
    ├── host/
    │   └── tools.go
    ├── service/
    │   └── tools.go
    ├── monitor/
    │   └── tools.go
    ├── cicd/
    │   └── tools.go
    ├── deployment/
    │   └── tools.go
    ├── governance/
    │   └── tools.go
    ├── infrastructure/
    │   └── tools.go
    └── mcp/
        ├── client.go
        └── proxy.go
```

#### Scenario: tools directory exceeds threshold

- **GIVEN** `internal/ai/tools/` contains more than 10 Go files at root level
- **WHEN** maintainers organize the codebase
- **THEN** tool implementations SHALL be moved to `impl/<domain>/` subdirectories
- **AND** each subdirectory SHALL contain related tools for one domain

#### Scenario: new domain tools are added

- **GIVEN** a new domain of AI tools needs to be implemented
- **WHEN** adding the tool implementation
- **THEN** a new subdirectory SHALL be created under `impl/` if the domain doesn't exist
- **AND** the tool file SHALL be placed in the corresponding domain subdirectory

---

### Requirement: Tool Implementation Files SHALL Follow Naming Convention

工具实现文件 SHALL 遵循命名规范，文件名反映工具领域。

**Naming Rules:**
- Implementation files: `tools.go` within each `impl/<domain>/` directory
- Core files: descriptive names (`contracts.go`, `registry.go`, `runner.go`)
- Parameter utilities: `param/<purpose>.go`

#### Scenario: naming a new tool file

- **GIVEN** a developer is creating a new tool implementation
- **WHEN** saving the file
- **THEN** the file SHALL be named `tools.go` and placed in the appropriate `impl/<domain>/` directory
- **AND** the package SHALL be named after the domain (e.g., `package kubernetes`)

---

### Requirement: Core Tool Types SHALL Remain in Root Package

核心工具类型 SHALL 保留在根包，确保实现包可以依赖统一类型定义。

**Types in Root Package:**
- `ToolMeta`, `ToolMode`, `ToolRisk`
- `ToolResult`, `ToolExecutionError`
- `ApprovalRequiredError`, `ConfirmationRequiredError`
- `PlatformDeps`, `RegisteredTool`
- Context helpers (`WithToolPolicyChecker`, `EmitToolEvent`)

#### Scenario: implementation package imports core types

- **GIVEN** a tool implementation in `impl/kubernetes/tools.go`
- **WHEN** the implementation needs `ToolMeta` or `ToolResult`
- **THEN** it SHALL import from `github.com/cy77cc/k8s-manage/internal/ai/tools`
- **AND** use the types directly without redefining

## Backend Organization

### Requirement: Go Models SHALL Be Organized By Domain

后端数据模型 SHALL 按领域域分组，每个领域一个子目录。

**Recommended Structure:**
```
internal/model/
├── ai/                     # AI 相关模型
│   ├── chat.go
│   ├── checkpoint.go
│   ├── approval.go
│   └── doc.go
├── user/                   # 用户与权限模型
│   ├── user.go
│   ├── role.go
│   ├── permission.go
│   └── doc.go
├── infrastructure/         # 基础设施模型
│   ├── host.go
│   ├── cluster.go
│   ├── node.go
│   └── doc.go
├── deployment/             # 部署管理模型
│   ├── deployment.go
│   ├── target.go
│   ├── environment.go
│   └── doc.go
├── service/                # 服务管理模型
│   ├── service.go
│   ├── catalog.go
│   └── doc.go
├── observability/          # 可观测性模型
│   ├── monitoring.go
│   ├── alert.go
│   └── doc.go
└── common/                 # 通用模型
    ├── audit_log.go
    ├── notification.go
    └── doc.go
```

**Acceptance Criteria:**
- [ ] 每个子目录文件数不超过 8 个
- [ ] 每个子目录包含 `doc.go` 说明包用途
- [ ] 跨领域引用通过导入路径显式声明

---

### Requirement: Service Handlers SHALL Be Organized By Resource

服务模块 SHALL 默认采用与 `internal/service/user` 一致的基础目录结构：模块根目录保留 `routes.go` 作为路由注册入口，并使用 `handler/` 目录承载 HTTP 处理器、`logic/` 目录承载业务编排逻辑。模块内部的文件整理 MUST 优先围绕职责边界展开，而不是继续将 `handler.go`、`logic.go` 等核心实现长期平铺在模块根目录。

对于复杂模块，系统 MAY 在模块根目录或辅助子目录中保留少量横切基础设施文件，但这些例外 MUST 满足以下条件：
- 文件不属于 HTTP 请求处理实现本身，也不属于纯业务编排逻辑
- 保留在根目录能显著提升可发现性，或其职责天然跨越多个 handler/logic 文件
- 命名能够清晰表达用途，例如 `repo/`、`collector.go`、`metrics.go`、`policy.go`

所有服务模块重组 SHALL 保持外部调用兼容：`Register<Domain>Handlers(...)` 等路由注册入口、既有 HTTP API 路径、请求响应结构、鉴权语义 MUST NOT 因目录调整而改变。

**Recommended Structure:**
```
internal/service/
├── user/
│   ├── routes.go
│   ├── handler/
│   │   ├── auth.go
│   │   ├── permissions.go
│   │   └── users.go
│   └── logic/
│       ├── auth.go
│       └── users.go
├── automation/
│   ├── routes.go
│   ├── handler/
│   └── logic/
├── cluster/
│   ├── routes.go
│   ├── handler/
│   ├── logic/
│   └── policy.go
└── ...
```

**Acceptance Criteria:**
- [ ] 每个服务模块包含独立的 `routes.go`
- [ ] HTTP 处理器实现放在 `handler/` 目录
- [ ] 业务编排逻辑放在 `logic/` 目录
- [ ] 根目录仅保留模块入口或少量跨职责基础设施文件
- [ ] 目录重组不改变对外 API 与鉴权行为

#### Scenario: Reorganize a flat service module
- **WHEN** 一个服务模块仍将 `handler.go` 和 `logic.go` 平铺在模块根目录
- **THEN** 该模块 MUST 迁移为 `routes.go + handler/ + logic/` 的基础结构

#### Scenario: Keep an explicit infrastructure exception
- **WHEN** 一个模块包含跨多个处理器共享的采集器、策略或仓储实现
- **THEN** 该文件或目录 MAY 保留在根目录或独立辅助目录中，但 MUST 不替代 `handler/` 与 `logic/` 的主干职责划分

#### Scenario: Preserve external behavior during reorganization
- **WHEN** 服务模块按统一结构进行目录重组
- **THEN** 其路由注册入口、HTTP API 路径、请求响应契约与鉴权语义 MUST 保持兼容

---

## Frontend Organization

### Requirement: React Components SHALL Be Organized By Feature

前端组件 SHALL 按功能特性分组，相关组件放在同一目录下。

**Recommended Structure:**
```
web/src/components/
├── AI/                     # AI 助手组件
│   ├── index.ts
│   ├── Copilot.tsx
│   ├── hooks/
│   │   ├── useAIChat.ts
│   │   └── useSSEAdapter.ts
│   └── components/
│       ├── MessageActions.tsx
│       └── ToolCard.tsx
├── Layout/                 # 布局组件
├── Feedback/               # 反馈组件
│   ├── Notification/
│   └── Loading/
├── DataDisplay/            # 数据展示组件
│   ├── Table/
│   └── Charts/
├── Form/                   # 表单组件
└── common/                 # 通用小组件
```

**Acceptance Criteria:**
- [ ] 每个功能目录包含 `index.ts` 导出
- [ ] 复杂组件拆分为子组件目录
- [ ] 目录层级不超过 3 层

---

### Requirement: API Modules SHALL Be Organized By Domain

API 模块 SHALL 按领域分组，避免 `modules/` 目录下堆积过多文件。

**Recommended Structure:**
```
web/src/api/
├── modules/
│   ├── ai/                 # AI 领域 API
│   │   ├── index.ts
│   │   ├── chat.ts
│   │   ├── session.ts
│   │   └── tools.ts
│   ├── infrastructure/     # 基础设施 API
│   │   ├── index.ts
│   │   ├── hosts.ts
│   │   └── clusters.ts
│   ├── deployment/         # 部署管理 API
│   ├── user/               # 用户与权限 API
│   └── observability/      # 可观测性 API
├── api.ts                  # API 客户端
└── types.ts                # 公共类型
```

**Acceptance Criteria:**
- [ ] 每个领域目录文件数不超过 6 个
- [ ] 每个目录包含 `index.ts` 导出所有 API
- [ ] 类型定义与 API 放在同一目录

---

### Requirement: Hooks SHALL Be Organized By Concern

自定义 Hooks SHALL 按关注点分组，相关 hooks 放在同一目录。

**Recommended Structure:**
```
web/src/hooks/
├── data/                   # 数据相关 hooks
│   ├── usePolling.ts
│   └── useRetry.ts
├── ui/                     # UI 相关 hooks
│   ├── useKeyboardShortcuts.ts
│   └── useDebounce.ts
├── notification/           # 通知相关 hooks
│   ├── useNotification.ts
│   └── useNotificationWebSocket.ts
├── auth/                   # 认证相关 hooks
└── index.ts                # 统一导出
```

**Acceptance Criteria:**
- [ ] 每个分组目录文件数不超过 8 个
- [ ] 每个目录包含 `index.ts` 导出
- [ ] 通用 hook 放在根目录

---

### Requirement: Utils SHALL Be Organized By Purpose

工具函数 SHALL 按用途分组，避免 `utils/` 目录杂乱。

**Recommended Structure:**
```
web/src/utils/
├── http/                   # HTTP 相关
│   ├── apiErrorHandler.ts
│   └── requestCache.ts
├── performance/            # 性能相关
│   └── performanceMonitor.ts
├── browser/                # 浏览器相关
│   ├── browserNotification.ts
│   └── tokenManager.ts
├── animation/              # 动画相关
│   └── animationOptimization.ts
└── index.ts                # 统一导出
```

**Acceptance Criteria:**
- [ ] 每个分组目录文件数不超过 6 个
- [ ] 单一职责的工具函数可放根目录
- [ ] 测试文件与源文件放同一目录

---

## Migration Guidelines

### Requirement: Code Reorganization SHALL Be Incremental

代码重组 SHALL 采用增量迁移，每次迁移一个领域，保持代码可编译状态。

**Migration Steps:**
1. 创建目标目录结构
2. 移动文件到新位置
3. 更新所有导入路径
4. 运行测试验证
5. 提交变更

**Acceptance Criteria:**
- [ ] 每次迁移影响范围可控
- [ ] 迁移后测试通过
- [ ] 导入路径使用绝对路径

---

## Enforcement

### Requirement: CI SHALL Check Directory File Count

CI 流程 SHALL 检查目录文件数量，超过阈值时发出警告。

**Check Script Example:**
```bash
#!/bin/bash
# Check directory file count
THRESHOLD=10
find internal/model -maxdepth 1 -type f -name "*.go" | wc -l | \
  xargs -I {} bash -c 'if [ {} -gt $THRESHOLD ]; then echo "Warning: too many files"; exit 1; fi'
```

#### Scenario: CI detects threshold violation

- **GIVEN** a directory contains more files than allowed
- **WHEN** CI runs the organization check
- **THEN** the build SHALL fail with a clear message indicating which directory needs reorganization
