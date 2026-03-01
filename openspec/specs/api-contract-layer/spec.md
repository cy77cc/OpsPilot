# API Contract Layer

## Purpose

定义各业务域在 `api/<domain>/v1/` 下维护纯数据类型契约文件的规范，确保 handler 请求/响应 struct 集中定义、禁止混入 interface，并约束 xcode 包和大文件拆分的边界。

## Requirements

### Requirement: 每个业务域在 api/<domain>/v1/ 下有完整的类型定义

系统中的每个业务域（cluster、service、monitoring、automation、ai、aiops、host、rbac）SHALL 在 `api/<domain>/v1/<domain>.go` 文件中定义其 HTTP handler 使用的所有请求/响应 struct。这些文件 MUST 是纯数据类型定义（不含 interface、不含业务逻辑），作为前后端稳定的 API 契约。

已有域（user、node、cicd、project、deployment、cmdb）的现有定义保持不变。

#### Scenario: 新增域类型文件
- **WHEN** 开发者为 `monitoring` 域新增 API
- **THEN** 请求/响应类型定义在 `api/monitoring/v1/monitoring.go`，不在 `internal/service/monitoring/` 内

#### Scenario: service 内部 types 迁移后无残留
- **WHEN** 完成迁移
- **THEN** `internal/service/*/types.go` 中不含对外暴露（用于 handler 请求/响应绑定）的 struct；仅保留包内私有类型

---

### Requirement: api/<domain>/v1/ 不含 interface 定义

`api/<domain>/` 目录 SHALL 只含请求/响应 struct 类型（数据契约），MUST NOT 含 Go interface 定义。现有的 `api/user/user.go` 和 `api/node/node.go` 中的 interface SHALL 被删除。

#### Scenario: 删除未实现 interface
- **WHEN** `api/user/user.go` 中定义的 `User`、`Auth`、`Roles`、`Permissions`、`RBAC` interface 从未被任何类型实现
- **THEN** 这些 interface 文件被删除，`go build ./...` 仍然通过

---

### Requirement: xcode 包不含 gRPC 相关代码

`internal/xcode/code.go` MUST NOT 含 gRPC 依赖（`google.golang.org/grpc`）和相关函数（`ToGrpcError`、`CodeFromGrpcError`、`mapGrpcCode`、`ToGrpcError`）。这些代码 SHALL 被删除，`go.mod` 中若 gRPC 仅因此引入则同步移除。

#### Scenario: gRPC 代码删除后编译通过
- **WHEN** 删除 xcode 中的 gRPC 相关函数和 import
- **THEN** `go build ./...` 通过，无其他包引用这些被删除的函数

---

### Requirement: 文件拆分后原包对外接口不变

对超大文件（`deployment/logic.go`、`service/logic.go`、`cluster/handler/phase1.go`）的拆分 SHALL 不改变任何包名、导出类型名、导出函数名或方法签名。拆分是纯文件组织变更，对其他包透明。

#### Scenario: 拆分后构建验证
- **WHEN** 完成文件拆分
- **THEN** `go build ./...` 通过，所有现有测试（`go test ./...`）通过

#### Scenario: 单文件行数约束
- **WHEN** 拆分完成
- **THEN** 每个拆分后的文件行数不超过 400 行（含注释和空行）
