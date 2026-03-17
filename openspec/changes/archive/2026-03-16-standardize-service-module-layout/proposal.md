## Why

`internal/service/user` 已经形成了较清晰的 `routes.go + handler/ + logic/` 分层，但其他服务模块仍存在多种组织方式并存：有的把 handler/logic 平铺在根目录，有的文件数量过多，有的职责边界不清。继续在这种不一致的结构上迭代，会持续放大维护成本、影响新人理解速度，也让跨模块重构变得更脆弱。

## What Changes

- 以 `internal/service/user` 作为参考结构，统一其他服务模块的目录组织方式
- 为服务模块定义一致的最小骨架：`routes.go`、`handler/`、`logic/`，必要时保留 `repo/` 或少量领域支撑文件
- 将当前平铺或混杂的 handler/logic 文件迁移到对应子目录，按资源或职责拆分，降低单目录和单文件复杂度
- 明确哪些模块属于本次首批整理范围，以及迁移过程中必须保持不变的兼容边界
- 为后续增量开发补充统一约束，避免新模块继续沿用旧式结构

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `code-organization-convention`: 细化服务层模块目录规范，要求服务模块默认采用与 `internal/service/user` 一致的 `routes.go + handler/ + logic/` 结构，并定义允许保留的例外边界

## Impact

- 受影响代码主要位于 `internal/service/ai`、`internal/service/automation`、`internal/service/cicd`、`internal/service/cmdb`、`internal/service/dashboard`、`internal/service/jobs`、`internal/service/monitoring`、`internal/service/topology` 等尚未完全按统一结构组织的模块
- 可能涉及 `internal/service/cluster`、`internal/service/deployment`、`internal/service/service` 等已有部分分层但仍需进一步归整的模块
- 不改变已有 HTTP API 路径、请求响应结构、鉴权语义或数据库 schema
- 影响开发规范与代码组织约定，需要同步补充 OpenSpec delta spec 与实施任务拆解
