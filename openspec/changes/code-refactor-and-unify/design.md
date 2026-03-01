## Context

项目 `k8s-manage` 是一个 Go（Gin + GORM）后端的 Kubernetes/PaaS 运维平台。经历多轮快速迭代后，后端代码存在三个核心结构问题：

1. **权限工具代码散落**：`authorize()`、`isAdmin()`、`hasPermission()`、`toUint()` 等工具函数在 10+ 个包中各自实现，逻辑存在细微差异（部分只查 username，部分同时查 role.code），导致权限行为不一致且难以维护。
2. **响应格式双轨并行**：`"success": false` 风格（约 170 处）与 `code: 1000` 风格（约 428 处）混用，`internal/response` 和 `internal/xcode` 两套机制已建立但未被全面使用，前后端协作成本高。
3. **超大单文件**：`deployment/logic.go`（1184行）、`service/logic.go`（1010行）、`cluster/handler/phase1.go`（1027行）混合了多个职责域，阅读和修改代价极高。
4. **API 契约层残缺**：只有 user、node、cicd、project、deployment、cmdb 6 个域在 `api/<domain>/v1/` 下有类型定义；cluster、service、monitoring、automation、ai、aiops、host、rbac 8 个域的类型散落在各自 service 包内，没有稳定的对外契约。
5. **死代码**：`xcode` 中的 gRPC 桩代码（项目不使用 gRPC）、未被实现的 interface（`api/user/user.go`、`api/node/node.go`）。

## Goals / Non-Goals

**Goals:**
- 提取 `internal/httpx` 包，统一 handler 层的响应输出和权限调用接口
- 全量将响应格式收口到 `httpx.OK()` / `httpx.Fail()` / `httpx.BindErr()`，所有接口返回 HTTP 200
- 拆分 3 个超大文件，单文件行数控制在 300 行以内
- 为 8 个缺失域补全 `api/<domain>/v1/` 类型定义，将 service 内部 types 迁移过去
- 删除 gRPC 死代码和未实现 interface

**Non-Goals:**
- 不改变任何 API 路径、请求/响应字段结构（纯代码组织重构，前端零改动）
- 不引入 DI 框架或替换 Gin/GORM
- 不建立完整 DAO 层（监控、部署等模块 logic 直接访问 DB 的模式保持不变）
- 不修改权限模型本身（Casbin 策略、权限码定义不变）
- 不补充测试（保持现有测试覆盖率，重构后不破坏已有测试）

## Decisions

### 决策 1：新建 `internal/httpx` 而非扩展现有 `internal/response`

**决策**：创建新包 `internal/httpx`，覆盖现有 `internal/response` 包的职责，并扩展权限工具函数。

**理由**：`internal/response` 命名偏窄，只表达"响应"，而我们还需要放 `UIDFromCtx`、`Authorize`、`uintFromParam` 等 handler 层工具。`httpx` 是 Go 社区惯用命名（参考 go-zero/httpx），语义更广，适合作为 handler 层的公共工具包。现有 `internal/response` 在三个文件中有使用，迁移后删除旧包。

**备选方案**：扩展 `internal/response` — 被否，命名语义不匹配导致后续贡献者困惑。

---

### 决策 2：响应格式 — 始终返回 HTTP 200

**决策**：所有业务接口（包括权限拒绝、参数错误、资源不存在）均返回 HTTP 200，通过 body 中的 `code` 字段（xcode 体系）区分。

**理由**：项目前端已经按此约定开发，强制改为语义 HTTP Status 会引入前端改动（超出本次范围）。`xcode` 体系已经设计了完整的 1000/2000/3000/4000 分层，语义表达能力足够。

**备选方案**：REST 语义状态码 — 被否，需要同步前端，超出本次重构范围。

**响应结构约定**：
```
成功：   { "code": 1000, "msg": "请求成功", "data": <T> }
列表：   { "code": 1000, "msg": "请求成功", "data": { "list": [...], "total": N } }
错误：   { "code": <xcode>, "msg": "<描述>" }
```

---

### 决策 3：文件拆分策略 — 按职责域命名，不引入子包

**决策**：拆分超大文件时在同一包内新增文件（如 `logic_release.go`、`logic_target.go`），不创建子包。

**理由**：当前每个 service 域已经是一个包（如 `package deployment`），拆子包会引入循环导入风险，且对调用方透明度最好（不改变任何 import 路径）。文件名前缀约定 `handler_<domain>.go` / `logic_<domain>.go` 使职责一目了然。

**拆分规划**：
```
deployment/logic.go (1184行) →
  logic_target.go      # DeploymentTarget CRUD + 节点绑定
  logic_release.go     # Release 生命周期（Preview/Apply/Approve/Rollback）
  logic_bootstrap.go   # 集群 Bootstrap + Task 查询
  logic_compose.go     # Compose 运行时部署执行
  logic_governance.go  # 治理策略（Governance）
  logic_util.go        # 纯函数工具（token/hash/json/字符串等）

service/logic.go (1010行) →
  logic_service.go     # Service CRUD + 列表
  logic_render.go      # 模板渲染/预览/转换
  logic_variable.go    # 变量管理（schema/values）
  logic_revision.go    # 版本管理
  logic_deploy.go      # 部署操作（Deploy/DeployPreview/HelmImport）
  logic_util.go        # 工具函数（normalizeStringMap/mustJSON 等）

cluster/handler/phase1.go (1027行) →
  handler_namespace.go # Namespace CRUD + 绑定
  handler_hpa.go       # HPA/Quota/LimitRange
  handler_rollout.go   # Rollout 操作（Preview/Apply/Promote/Abort）
  handler_approval.go  # 审批流程（CreateApproval/ConfirmApproval）
```

---

### 决策 4：API 契约层 — `api/<domain>/v1/` 只放纯类型，不放 interface

**决策**：删除 `api/user/user.go` 和 `api/node/node.go` 中的 interface 定义，`api/<domain>/v1/` 只保留请求/响应 struct 定义（纯数据契约）。

**理由**：这些 interface 从未被 service 层实现（go 编译器验证），实际上是过时的设计文档，造成误导。Go 项目惯用模式是 interface 定义在消费方，不在库层。保留纯 struct 的 v1 包是合理的"契约文件"角色。

---

### 决策 5：`httpx.Authorize()` 的权限实现使用 cluster handler 版本作为基准

**决策**：以 `cluster/handler/policy.go` 中的 `hasAnyPermission()` 为权威实现（最完整：同时查 username=="admin" 和 role.code=="admin"，支持 `*:*` 和 `<domain>:*` 通配），其余各模块的简化版本废弃。

**理由**：各模块的 `isAdmin()` 实现存在差异，cluster 版本覆盖了所有场景（用户名 admin + 角色 admin），是最安全的基准。统一后权限判断行为更一致。

## Risks / Trade-offs

| 风险 | 缓解策略 |
|------|---------|
| 大规模改动引入编译错误 | 按模块逐步改动，每个模块改完后运行 `go build ./...` 验证 |
| 响应格式替换遗漏，导致部分接口仍返回旧格式 | 用 `grep -rn '"success"' internal/service` 做完工验证，确认零残留 |
| 文件拆分后包内 unexported 符号引用断裂 | 拆分时检查每个函数/变量的 unexported 引用，确保同包可见 |
| `httpx.Authorize()` 统一后部分模块权限变严格 | 对比旧实现，记录差异点（monitoring 的 `authorize` 参数签名不同），逐一确认行为预期 |
| api/v1 types 迁移导致 import 路径变更 | 先新建 api/v1 文件，再修改 service 层 import，最后删除旧 types.go；保持 `go build` 绿色 |

## Migration Plan

**执行顺序**（各阶段独立可验证）：

1. **Phase 1 - 基础设施**：新建 `internal/httpx`，实现所有工具函数；删除 gRPC 死代码；删除未实现 interface
2. **Phase 2 - 文件拆分**：拆 `deployment/logic.go` → 5文件；拆 `service/logic.go` → 6文件；拆 `cluster/handler/phase1.go` → 4文件；每步后验证 `go build`
3. **Phase 3 - API 契约层**：补全 8 个域的 `api/<domain>/v1/` types；迁移 service 内 types.go 引用
4. **Phase 4 - 响应格式统一**：全量替换所有 handler 的 `c.JSON` 内联调用为 `httpx.*`；权限调用改为 `httpx.Authorize()`；删除各 handler 内的重复工具函数

**回滚策略**：纯代码重组，无数据库变更，git revert 即可完整回滚。

## Open Questions

- 无。所有关键决策已由用户确认（HTTP 全 200、api/v1 统一、gRPC 代码可删）。
