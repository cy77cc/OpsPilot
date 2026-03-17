## MODIFIED Requirements

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
