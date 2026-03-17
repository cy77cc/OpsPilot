## Context

当前 `internal/service/` 下的服务模块已经出现三种主要组织形态：

1. `user`、`project`、`host`、`node` 等模块已经采用或接近采用 `routes.go + handler/ + logic/` 的目录结构。
2. `ai` 模块最近完成了 handler/logic 分层，但仍保留了 `handler/handler.go` 这类聚合入口文件，整体还需要和基准结构进一步对齐。
3. `automation`、`cmdb`、`dashboard`、`jobs`、`monitoring`、`topology` 等模块仍以 `handler.go`、`logic.go` 平铺在模块根目录；`cluster`、`deployment`、`service` 等复杂模块则存在分层不均衡、根目录文件过多的问题。

用户希望参考 `internal/service/user` 的文件组织方式，将其他服务模块整理为更统一、更易发现的结构。这是一个跨多个后端域的代码组织重构，需要先明确统一骨架、例外边界和分批迁移策略，避免在实施中引入 import 混乱或大范围行为回归。

## Goals / Non-Goals

**Goals:**
- 统一 `internal/service/<domain>/` 的基础目录骨架，默认采用 `routes.go + handler/ + logic/`
- 将平铺在模块根目录的 handler/logic 实现迁移到对应子目录，降低根目录复杂度
- 为复杂服务模块定义允许保留的辅助目录或领域文件边界，例如 `repo/`、`collector.go`、`policy.go`
- 明确实施顺序和兼容要求，确保重构过程中不改变现有 API、鉴权语义和业务流程
- 将 `internal/service/user` 提升为服务层组织的参考样板，供后续新模块遵循

**Non-Goals:**
- 不修改 HTTP 路径、请求响应字段、权限模型或数据库结构
- 不强制所有模块拆成完全相同的文件数量或完全同名文件
- 不借这次重构引入新的跨模块基类、框架或依赖注入机制
- 不处理与目录整理无关的业务重写、性能优化或前端联动改造

## Decisions

### Decision 1: 以 `internal/service/user` 为默认参考结构，而不是重新设计一套新模板

**Decision:** 后续服务模块整理默认对齐 `internal/service/user` 的结构：模块根目录保留 `routes.go` 和少量领域支撑文件，HTTP 入口放入 `handler/`，业务编排放入 `logic/`。

**Rationale:** `user` 已经是代码库里最稳定、最容易理解的一种组织方式，认知成本低，也和当前 OpenSpec 中对服务层分层的描述一致。沿用现有成功样板，比再引入一套“理想模型”更适合当前代码库。

**Alternatives considered:**
- 为每个资源再创建更深一层子包：层级更深、迁移成本更高，不利于当前项目渐进重构
- 继续允许每个模块自行组织：短期省事，但会继续累积不一致性

### Decision 2: 采用“默认骨架 + 受控例外”的规范，而不是一刀切清空所有根目录文件

**Decision:** 每个服务模块 SHALL 默认包含 `routes.go`、`handler/`、`logic/`；若存在明显的横切职责或基础设施代码，可保留少量例外文件或目录，例如：
- `repo/`：数据访问封装，如 `cicd/repo/`
- `collector.go` / `metrics.go`：模块级采集器或指标入口，如 `dashboard/collector.go`
- `policy.go` / `cache_policy.go`：跨 handler/logic 的共享策略定义
- `types.go`：仅在无法立即迁移、且内容属于模块内部私有结构时短期保留

**Rationale:** 当前模块复杂度差异很大，完全一刀切会逼出大量不自然的移动。受控例外既能统一主干结构，也能尊重复杂模块的实际职责边界。

**Alternatives considered:**
- 仅要求“有 handler 和 logic 即可”，不要求根目录骨架：约束太弱，难以真正统一
- 所有代码都必须进入子目录：会让 collector、policy、repo 等基础设施职责无处安放

### Decision 3: 重构优先迁移平铺模块，复杂模块采用二阶段归整

**Decision:** 实施分两类推进：
- 第一批直接整理平铺模块：`automation`、`cmdb`、`dashboard`、`jobs`、`monitoring`、`topology`
- 第二批整理部分分层但仍不规整的模块：`ai`、`cicd`、`cluster`、`deployment`、`service`

**Rationale:** 平铺模块收益最高、迁移风险最低，适合作为规范落地第一步。复杂模块往往已存在多文件拆分，需要在不破坏现有边界的前提下进一步归整，应该晚于简单模块执行。

**Alternatives considered:**
- 一次性全量迁移所有模块：评审、验证和回滚成本太高
- 只迁移简单模块，放弃复杂模块：无法完成“统一其他服务模块”的目标

### Decision 4: 目录重构必须保持导出构造器和路由注册入口稳定

**Decision:** 对外可见的包级入口 SHALL 保持稳定，包括：
- `Register<Domain>Handlers(...)` 的函数签名和注册行为不变
- 原有构造器如 `NewHandler`、`New<Domain>Handler` 在必要时通过包装或迁移后继续可用
- 外部引用模块包路径仍保持 `internal/service/<domain>`，不引入新的子包对调用方暴露

**Rationale:** 本次变更的核心是代码组织，不是公共 API 重写。保持包入口稳定，可以把影响限定在模块内部，实现更安全的渐进重构。

**Alternatives considered:**
- 同步重命名全部构造器和导出类型：规范性更强，但会把重构升级成大范围接口变更

## Risks / Trade-offs

- [复杂模块边界判断不一致] → 在 design 和 tasks 中先定义允许保留的例外类型，再逐模块迁移，避免边做边改口径
- [文件移动后 import 或测试引用失效] → 每个模块迁移后立即运行对应包测试或 `go test`/`go build` 做增量校验
- [为了凑结构而产生空洞的 logic/handler 拆分] → 仅移动真正属于 HTTP 或业务编排职责的代码，不为了目录形式拆出无意义文件
- [平铺模块整理后仍残留 `types.go` 等历史文件] → 允许短期保留，但在任务中要求记录保留原因并优先清理纯 handler/logic 平铺文件
- [开发中新增模块继续沿用旧结构] → 通过修改 OpenSpec 规范明确默认骨架和例外边界，作为后续评审依据

## Migration Plan

1. 更新 `code-organization-convention` 中关于服务模块组织的要求，明确默认骨架、例外边界和兼容要求
2. 先整理平铺模块，把根目录中的 `handler.go` / `logic.go` 迁移至 `handler/`、`logic/`
3. 再整理复杂模块，对 `ai`、`cicd`、`cluster`、`deployment`、`service` 做定向归整，保留必要的基础设施文件
4. 每个模块迁移后执行增量编译或测试，最后运行全量验证
5. 如出现回归，按模块粒度回滚对应文件移动；本次变更不涉及存储迁移，可直接通过 git 回退

## Open Questions

- 本次 proposal 先把 `rbac`、`notification` 视为非首批整理对象，因为它们当前文件量较少、且职责边界与典型业务服务不同。若实施时发现它们也需要纳入，可以在 apply 阶段再补充为追加任务。
