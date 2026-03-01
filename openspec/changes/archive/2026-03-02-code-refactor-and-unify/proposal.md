## Why

随着项目功能持续迭代，代码库积累了大量重复的权限判断逻辑、不一致的 HTTP 响应格式，以及过度膨胀的单文件（最大超过 1200 行），严重拖慢了新功能开发速度和多人协作效率。现在暂停功能开发，集中清理技术债，是建立可持续开发节奏的最佳时机。

## What Changes

- **提取 `internal/httpx` 公共包**：将分散在 10+ 个 handler 文件中重复定义的 `authorize()`、`isAdmin()`、`hasPermission()`、`toUint()`、`uintFromParam()`、`uintFromQuery()` 等函数统一收口，各 handler 改为调用公共实现
- **统一 HTTP 响应格式**：全量替换约 600 处内联 `c.JSON(...)` 为 `httpx.OK()` / `httpx.Fail()` / `httpx.BindErr()`，所有响应均返回 HTTP 200，通过 `code` 字段（xcode 体系）区分结果；废弃 `"success": false` 风格
- **拆分超大文件**：将 `deployment/logic.go`（1184行）、`service/logic.go`（1010行）、`cluster/handler/phase1.go`（1027行）按职责边界分拆，单文件原则上不超过 300 行
- **统一 API 契约层**：为尚未在 `api/<domain>/v1/` 下定义 types 的域（cluster、service、monitoring、automation、ai、aiops、host、rbac）补全类型定义，将 service 包内散落的 `types.go` 迁移至对应 `api/v1` 包
- **清除死代码**：删除 `xcode` 中的 gRPC 相关代码（`ToGrpcError`、`mapGrpcCode`、gRPC import）；删除 `api/user/user.go`、`api/node/node.go` 中未被实现的 interface 定义；清理 `rbac/handler/permission.go` 中的 `"log"` 残留 import

## Capabilities

### New Capabilities

- `http-response-convention`：统一的 HTTP 响应规范——所有接口返回 HTTP 200，body 结构为 `{code, msg, data}`，列表响应为 `{code, msg, data: {list, total}}`，由 `internal/httpx` 包提供 `OK()`、`Fail()`、`BindErr()` 函数强制实施
- `handler-auth-shared`：Handler 层公共授权工具——`internal/httpx` 提供 `UIDFromCtx()`、`IsAdmin()`、`HasPermission()`、`Authorize()` 等函数，替代各 handler 中 8+ 份重复实现，权限判断逻辑统一
- `api-contract-layer`：所有域的请求/响应类型统一定义在 `api/<domain>/v1/` 下，作为前后端稳定契约；service 内部 `types.go` 仅保留不对外暴露的内部结构

### Modified Capabilities

- `platform-capability-baseline`：handler 层的 RBAC 检查方式由各自内联实现改为调用 `httpx.Authorize()`，行为一致性增强（不改变权限模型本身）

## Impact

**受影响的代码路径：**
- `internal/service/*/handler*.go`（全部约 15 个 handler 文件）：响应格式、权限调用方式变更
- `internal/service/deployment/logic.go` → 拆分为 5 个文件
- `internal/service/service/logic.go` → 拆分为 6 个文件
- `internal/service/cluster/handler/phase1.go` → 拆分为 4 个文件
- `internal/xcode/code.go`：删除 gRPC 相关代码
- `api/*/`：新增 cluster、service、monitoring、automation、ai、aiops、host、rbac 的 v1 types；删除 user/user.go、node/node.go 中未实现的 interface
- `internal/httpx/`（新建）：公共 handler 工具包

**无数据库 migration，无 API path 变更，无前端改动，纯后端代码组织重构。**
