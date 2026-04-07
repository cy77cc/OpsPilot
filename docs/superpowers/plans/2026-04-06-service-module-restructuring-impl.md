# Service 模块目录结构重组实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立统一的目录结构规范：**顶层只保留 routes.go，所有 Go 文件移入子目录**

**Architecture:** 按业务层级分离：handler（HTTP处理层）→ logic（业务逻辑层）→ domain（领域细分，仅复杂模块）

**Tech Stack:** Go, Gin framework, 目录重构

---

## 核心原则

```
service/xxxx/
├── handler/      # HTTP 处理层
├── logic/        # 业务逻辑层
├── domain/       # 领域细分（可选，仅复杂模块）
└── routes.go     # 唯一的顶层文件：路由入口
```

**命名规范**：
- ✅ `handler/release.go` - 按业务概念命名
- ❌ `handler/handler_release.go` - 禁止重复前缀

---

## 实际勘误（与设计文档的差异）

| 模块 | 设计文档预期 | 实际现状 | 结论 |
|------|-------------|---------|------|
| topology | 需从 routes.go 拆分 | 已有 `handler/handler.go` + `logic/logic.go` | **已符合规范** |
| dashboard | 新建 handler/，移动 collector | 已有 handler/logic，collector.go 在根目录 | 移动 collector.go |
| deployment | handler/logic 空，15个 flat 文件 | handler/logic 仅有 doc.go 占位符 | 全量重组 |
| cluster | handler/logic 存在 | handler/ 有实际文件，logic/ 仅 doc.go | 需补充整理 |

---

## Phase 1: notification 模块（复杂度：低）

**现状**：3 个 flat 文件在根目录
```
notification/
├── notification.go  # 包含 handler + 路由注册混合
├── provider.go
└── integration.go
```

**目标结构**：
```
notification/
├── handler/
│   ├── notification.go
│   ├── provider.go
│   └── integration.go
├── logic/           # 空目录占位
└── routes.go        # 从 notification.go 提取路由注册
```

### Task 1.1: 创建 handler 和 logic 子目录

**Files:**
- Create: `internal/service/notification/handler/`
- Create: `internal/service/notification/logic/`

- [ ] **Step 1: 创建目录结构**

```bash
mkdir -p internal/service/notification/handler
mkdir -p internal/service/notification/logic
touch internal/service/notification/logic/.gitkeep
```

- [ ] **Step 2: 验证目录创建成功**

Run: `ls -la internal/service/notification/`
Expected: handler/, logic/ 目录存在

### Task 1.2: 移动 provider.go 和 integration.go

**Files:**
- Move: `internal/service/notification/provider.go` → `internal/service/notification/handler/provider.go`
- Move: `internal/service/notification/integration.go` → `internal/service/notification/handler/integration.go`

- [ ] **Step 1: 使用 git mv 移动文件**

```bash
git mv internal/service/notification/provider.go internal/service/notification/handler/provider.go
git mv internal/service/notification/integration.go internal/service/notification/handler/integration.go
```

- [ ] **Step 2: 更新 provider.go 的 package 声明**

**Modify**: `internal/service/notification/handler/provider.go`

```go
// 从:
package notification

// 改为:
package handler
```

- [ ] **Step 3: 更新 integration.go 的 package 声明**

**Modify**: `internal/service/notification/handler/integration.go`

```go
package handler
```

### Task 1.3: 分析并拆分 notification.go

**Files:**
- Read: `internal/service/notification/notification.go`
- Create: `internal/service/notification/handler/notification.go`
- Create: `internal/service/notification/routes.go`

- [ ] **Step 1: 读取 notification.go 内容分析结构**

Run: Read the file to identify:
1. Handler struct and methods (HTTP handlers)
2. Route registration logic (RegisterRoutes or similar)

- [ ] **Step 2: 创建 handler/notification.go**

提取 handler 部分，更新 package 为 handler。

- [ ] **Step 3: 创建 routes.go**

提取路由注册部分，更新 import 路径指向 handler 包。

### Task 1.4: 删除原 notification.go

**Files:**
- Delete: `internal/service/notification/notification.go`

- [ ] **Step 1: 确认新结构可编译后删除原文件**

```bash
go build ./internal/service/notification/...
rm internal/service/notification/notification.go
```

### Task 1.5: 验证编译并提交

- [ ] **Step 1: 验证编译**

```bash
go build ./internal/service/notification/...
```

Expected: 无错误

- [ ] **Step 2: 提交 Phase 1**

```bash
git add internal/service/notification/
git commit -m "refactor(notification): restructure to handler/logic layout

- Move provider.go, integration.go to handler/
- Split notification.go into handler/notification.go and routes.go
- Top-level only keeps routes.go

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Phase 2: dashboard 模块（复杂度：低）

**现状**：
```
dashboard/
├── handler/handler.go    # 已存在
├── logic/logic.go        # 已存在
├── collector.go          # 需移入 logic/
└── routes.go
```

**目标结构**：
```
dashboard/
├── handler/
│   └── handler.go
├── logic/
│   ├── logic.go
│   └── collector.go      # 从根目录移入
└── routes.go
```

### Task 2.1: 移动 collector.go 到 logic/

**Files:**
- Move: `internal/service/dashboard/collector.go` → `internal/service/dashboard/logic/collector.go`

- [ ] **Step 1: 使用 git mv 移动文件**

```bash
git mv internal/service/dashboard/collector.go internal/service/dashboard/logic/collector.go
```

- [ ] **Step 2: 更新 collector.go 的 package 声明**

**Modify**: `internal/service/dashboard/logic/collector.go`

```go
// 从:
package dashboard

// 改为:
package logic
```

### Task 2.2: 更新 handler.go 的 import（如需要）

**Files:**
- Modify: `internal/service/dashboard/handler/handler.go`

- [ ] **Step 1: 检查 handler.go 是否引用 collector**

Run: `grep -n "collector\|Collector" internal/service/dashboard/handler/handler.go`

- [ ] **Step 2: 如有引用，更新 import 路径**

```go
// 添加 import:
import "github.com/cy77cc/OpsPilot/internal/service/dashboard/logic"

// 更新调用:
// 从: dashboard.NewCollector(...)
// 改为: logic.NewCollector(...)
```

### Task 2.3: 验证编译并提交

- [ ] **Step 1: 验证编译**

```bash
go build ./internal/service/dashboard/...
```

- [ ] **Step 2: 提交 Phase 2**

```bash
git add internal/service/dashboard/
git commit -m "refactor(dashboard): move collector.go to logic/

Top-level only keeps routes.go

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Phase 3: governance 模块（复杂度：中）

**现状**：
```
governance/
├── approval/   # 已存在子目录
├── audit/
├── envelope/
├── policy/
├── errors.go   # flat 文件
├── service.go  # flat 文件
├── types.go    # flat 文件
└── service_test.go
```

**目标结构**：
```
governance/
├── handler/
│   ├── errors.go
│   └── service.go
├── logic/
│   └── types.go          # 领域类型移入 logic
├── approval/, audit/, envelope/, policy/  # 保持不变
└── routes.go             # 新建路由入口
```

### Task 3.1: 创建 handler 和 logic 子目录

**Files:**
- Create: `internal/service/governance/handler/`
- Create: `internal/service/governance/logic/`

- [ ] **Step 1: 创建目录结构**

```bash
mkdir -p internal/service/governance/handler
mkdir -p internal/service/governance/logic
```

### Task 3.2: 移动 errors.go 和 service.go 到 handler/

**Files:**
- Move: `internal/service/governance/errors.go` → `internal/service/governance/handler/errors.go`
- Move: `internal/service/governance/service.go` → `internal/service/governance/handler/service.go`
- Move: `internal/service/governance/service_test.go` → `internal/service/governance/handler/service_test.go`

- [ ] **Step 1: 移动文件**

```bash
git mv internal/service/governance/errors.go internal/service/governance/handler/errors.go
git mv internal/service/governance/service.go internal/service/governance/handler/service.go
git mv internal/service/governance/service_test.go internal/service/governance/handler/service_test.go
```

- [ ] **Step 2: 更新 package 声明**

**Modify**: `internal/service/governance/handler/errors.go`

```go
package handler
```

**Modify**: `internal/service/governance/handler/service.go`

```go
package handler
```

**Modify**: `internal/service/governance/handler/service_test.go`

```go
package handler
```

### Task 3.3: 移动 types.go 到 logic/

**Files:**
- Move: `internal/service/governance/types.go` → `internal/service/governance/logic/types.go`

- [ ] **Step 1: 移动 types.go**

```bash
git mv internal/service/governance/types.go internal/service/governance/logic/types.go
```

- [ ] **Step 2: 更新 package 声明**

**Modify**: `internal/service/governance/logic/types.go`

```go
package logic
```

### Task 3.4: 检查外部引用并更新

**Files:**
- Search: 外部模块对 governance 包的引用

- [ ] **Step 1: 搜索外部引用**

```bash
grep -rn "governance\.Service\|governance\.GovError\|governance\.Types" --include="*.go" internal/service/ | grep -v "governance/handler\|governance/logic"
```

- [ ] **Step 2: 更新引用路径**

如有引用，更新 import 为:
```go
import (
    governancehandler "github.com/cy77cc/OpsPilot/internal/service/governance/handler"
    governancelogic "github.com/cy77cc/OpsPilot/internal/service/governance/logic"
)
```

### Task 3.5: 验证编译并提交

- [ ] **Step 1: 验证编译**

```bash
go build ./internal/service/governance/...
go test ./internal/service/governance/...
```

- [ ] **Step 2: 提交 Phase 3**

```bash
git add internal/service/governance/
git commit -m "refactor(governance): move all Go files to handler/logic

- service.go, errors.go → handler/
- types.go → logic/
- Top-level only keeps routes.go (to be created if needed)

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Phase 4: deployment 模块（复杂度：高）

**现状**：15+ flat 文件在根目录
```
deployment/
├── handler/doc.go       # 占位符
├── logic/doc.go         # 占位符
├── handler.go           # HTTP handler
├── handler_environment.go
├── logic.go             # logic 入口
├── logic_bootstrap.go
├── logic_compose.go
├── logic_governance.go
├── logic_release.go
├── logic_target.go
├── logic_util.go
├── bootstrap.go         # 模块初始化
├── audit.go
├── metrics.go
├── policy.go
├── topology.go
├── types.go
└── routes.go
```

**目标结构**：
```
deployment/
├── handler/
│   ├── release.go       # 原 handler.go
│   └── environment.go   # 原 handler_environment.go
├── logic/
│   ├── deployment.go    # 原 logic.go（入口）
│   ├── bootstrap.go     # 原 logic_bootstrap.go
│   ├── compose.go       # 原 logic_compose.go
│   ├── governance.go    # 原 logic_governance.go
│   ├── release.go       # 原 logic_release.go
│   ├── target.go        # 原 logic_target.go
│   ├── util.go          # 原 logic_util.go
│   ├── audit.go         # 从根目录移入
│   ├── bootstrap_init.go # 原 bootstrap.go（模块初始化）
│   ├── metrics.go       # 从根目录移入
│   ├── policy.go        # 从根目录移入
│   ├── topology.go      # 从根目录移入
│   └── types.go         # 从根目录移入
└── routes.go
```

### Task 4.1: 移动 handler 文件

**Files:**
- Move: `internal/service/deployment/handler.go` → `internal/service/deployment/handler/release.go`
- Move: `internal/service/deployment/handler_environment.go` → `internal/service/deployment/handler/environment.go`

- [ ] **Step 1: 移动 handler.go 到 handler/release.go**

```bash
git mv internal/service/deployment/handler.go internal/service/deployment/handler/release.go
```

- [ ] **Step 2: 移动 handler_environment.go**

```bash
git mv internal/service/deployment/handler_environment.go internal/service/deployment/handler/environment.go
```

- [ ] **Step 3: 更新 handler/release.go package**

**Modify**: `internal/service/deployment/handler/release.go`

```go
package handler
```

- [ ] **Step 4: 更新 handler/environment.go package**

**Modify**: `internal/service/deployment/handler/environment.go`

```go
package handler
```

- [ ] **Step 5: 删除 handler/doc.go 占位符**

```bash
rm internal/service/deployment/handler/doc.go
```

### Task 4.2: 移动 logic_*.go 文件到 logic/

**Files:**
- Move: `internal/service/deployment/logic.go` → `internal/service/deployment/logic/deployment.go`
- Move: `internal/service/deployment/logic_bootstrap.go` → `internal/service/deployment/logic/bootstrap.go`
- Move: `internal/service/deployment/logic_compose.go` → `internal/service/deployment/logic/compose.go`
- Move: `internal/service/deployment/logic_governance.go` → `internal/service/deployment/logic/governance.go`
- Move: `internal/service/deployment/logic_release.go` → `internal/service/deployment/logic/release.go`
- Move: `internal/service/deployment/logic_target.go` → `internal/service/deployment/logic/target.go`
- Move: `internal/service/deployment/logic_util.go` → `internal/service/deployment/logic/util.go`

- [ ] **Step 1: 批量移动 logic_*.go 文件**

```bash
git mv internal/service/deployment/logic.go internal/service/deployment/logic/deployment.go
git mv internal/service/deployment/logic_bootstrap.go internal/service/deployment/logic/bootstrap.go
git mv internal/service/deployment/logic_compose.go internal/service/deployment/logic/compose.go
git mv internal/service/deployment/logic_governance.go internal/service/deployment/logic/governance.go
git mv internal/service/deployment/logic_release.go internal/service/deployment/logic/release.go
git mv internal/service/deployment/logic_target.go internal/service/deployment/logic/target.go
git mv internal/service/deployment/logic_util.go internal/service/deployment/logic/util.go
```

- [ ] **Step 2: 批量更新 package 声明**

```bash
for f in internal/service/deployment/logic/*.go; do
  sed -i 's/^package deployment$/package logic/' "$f"
done
```

### Task 4.3: 移动根目录其他 Go 文件到 logic/

**Files:**
- Move: `internal/service/deployment/bootstrap.go` → `internal/service/deployment/logic/bootstrap_init.go`
- Move: `internal/service/deployment/audit.go` → `internal/service/deployment/logic/audit.go`
- Move: `internal/service/deployment/metrics.go` → `internal/service/deployment/logic/metrics.go`
- Move: `internal/service/deployment/policy.go` → `internal/service/deployment/logic/policy.go`
- Move: `internal/service/deployment/topology.go` → `internal/service/deployment/logic/topology.go`
- Move: `internal/service/deployment/types.go` → `internal/service/deployment/logic/types.go`

- [ ] **Step 1: 移动根目录文件到 logic/**

```bash
git mv internal/service/deployment/bootstrap.go internal/service/deployment/logic/bootstrap_init.go
git mv internal/service/deployment/audit.go internal/service/deployment/logic/audit.go
git mv internal/service/deployment/metrics.go internal/service/deployment/logic/metrics.go
git mv internal/service/deployment/policy.go internal/service/deployment/logic/policy.go
git mv internal/service/deployment/topology.go internal/service/deployment/logic/topology.go
git mv internal/service/deployment/types.go internal/service/deployment/logic/types.go
```

- [ ] **Step 2: 批量更新 package 声明**

```bash
for f in internal/service/deployment/logic/bootstrap_init.go internal/service/deployment/logic/audit.go internal/service/deployment/logic/metrics.go internal/service/deployment/logic/policy.go internal/service/deployment/logic/topology.go internal/service/deployment/logic/types.go; do
  sed -i 's/^package deployment$/package logic/' "$f"
done
```

- [ ] **Step 3: 删除 logic/doc.go 占位符**

```bash
rm internal/service/deployment/logic/doc.go
```

### Task 4.4: 更新 routes.go import 路径

**Files:**
- Modify: `internal/service/deployment/routes.go`

- [ ] **Step 1: 更新 import 路径**

```go
import (
    deploymenthandler "github.com/cy77cc/OpsPilot/internal/service/deployment/handler"
    deploymentlogic "github.com/cy77cc/OpsPilot/internal/service/deployment/logic"
)
```

### Task 4.5: 验证编译并提交

- [ ] **Step 1: 验证编译**

```bash
go build ./internal/service/deployment/...
```

- [ ] **Step 2: 提交 Phase 4**

```bash
git add internal/service/deployment/
git commit -m "refactor(deployment): move all Go files to handler/logic

- handler.go → handler/release.go
- handler_environment.go → handler/environment.go
- logic*.go → logic/*.go (remove logic_ prefix)
- bootstrap.go → logic/bootstrap_init.go
- audit.go, metrics.go, policy.go, topology.go, types.go → logic/
- Top-level only keeps routes.go

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Phase 5: service → application 重命名（复杂度：中）

**现状**：
```
service/
├── handler/doc.go       # 占位符
├── logic/doc.go         # 占位符
├── handler.go           # HTTP handler
├── logic.go             # logic 入口
├── logic_deploy.go
├── logic_env_match.go
├── logic_render.go
├── logic_revision.go
├── logic_service.go
├── logic_util.go
├── logic_variable.go
├── render.go
├── template_vars.go
├── types.go
└── routes.go
```

**目标结构**：
```
application/             # 重命名，避免 service 关键字冲突
├── handler/
│   └── handler.go       # 原 handler.go
├── logic/
│   ├── application.go   # 原 logic.go
│   ├── deploy.go        # 原 logic_deploy.go
│   ├── env_match.go     # 原 logic_env_match.go
│   ├── render.go        # 原 logic_render.go 或 render.go
│   ├── revision.go      # 原 logic_revision.go
│   ├── service.go       # 原 logic_service.go
│   ├── util.go          # 原 logic_util.go
│   ├── variable.go      # 原 logic_variable.go
│   ├── template_vars.go # 从根目录移入
│   └── types.go         # 从根目录移入
└── routes.go
```

### Task 5.1: 创建 application 目录结构

**Files:**
- Create: `internal/service/application/`
- Create: `internal/service/application/handler/`
- Create: `internal/service/application/logic/`

- [ ] **Step 1: 创建目录**

```bash
mkdir -p internal/service/application/handler
mkdir -p internal/service/application/logic
```

### Task 5.2: 移动 handler 文件

**Files:**
- Move: `internal/service/service/handler.go` → `internal/service/application/handler/handler.go`

- [ ] **Step 1: 移动 handler.go**

```bash
git mv internal/service/service/handler.go internal/service/application/handler/handler.go
```

- [ ] **Step 2: 更新 package 声明**

**Modify**: `internal/service/application/handler/handler.go`

```go
package handler
```

### Task 5.3: 移动 logic 文件并重命名

**Files:**
- Move: `internal/service/service/logic.go` → `internal/service/application/logic/application.go`
- Move: `internal/service/service/logic_deploy.go` → `internal/service/application/logic/deploy.go`
- Move: `internal/service/service/logic_env_match.go` → `internal/service/application/logic/env_match.go`
- Move: `internal/service/service/logic_render.go` → `internal/service/application/logic/render.go`
- Move: `internal/service/service/logic_revision.go` → `internal/service/application/logic/revision.go`
- Move: `internal/service/service/logic_service.go` → `internal/service/application/logic/service.go`
- Move: `internal/service/service/logic_util.go` → `internal/service/application/logic/util.go`
- Move: `internal/service/service/logic_variable.go` → `internal/service/application/logic/variable.go`

- [ ] **Step 1: 批量移动 logic 文件**

```bash
git mv internal/service/service/logic.go internal/service/application/logic/application.go
git mv internal/service/service/logic_deploy.go internal/service/application/logic/deploy.go
git mv internal/service/service/logic_env_match.go internal/service/application/logic/env_match.go
git mv internal/service/service/logic_render.go internal/service/application/logic/render.go
git mv internal/service/service/logic_revision.go internal/service/application/logic/revision.go
git mv internal/service/service/logic_service.go internal/service/application/logic/service.go
git mv internal/service/service/logic_util.go internal/service/application/logic/util.go
git mv internal/service/service/logic_variable.go internal/service/application/logic/variable.go
```

- [ ] **Step 2: 批量更新 package 声明**

```bash
for f in internal/service/application/logic/*.go; do
  sed -i 's/^package service$/package logic/' "$f"
done
```

### Task 5.4: 移动根目录其他 Go 文件到 logic/

**Files:**
- Move: `internal/service/service/render.go` → `internal/service/application/logic/render_core.go`（若与 logic_render.go 不同）
- Move: `internal/service/service/template_vars.go` → `internal/service/application/logic/template_vars.go`
- Move: `internal/service/service/types.go` → `internal/service/application/logic/types.go`

- [ ] **Step 1: 检查 render.go 是否与 logic_render.go 重复**

```bash
cat internal/service/service/render.go | head -20
```

若与 logic_render.go 不同，移动并重命名为 `render_core.go`；若相同，删除。

- [ ] **Step 2: 移动保留文件**

```bash
git mv internal/service/service/template_vars.go internal/service/application/logic/template_vars.go
git mv internal/service/service/types.go internal/service/application/logic/types.go
```

- [ ] **Step 3: 更新 package 声明**

```bash
for f in internal/service/application/logic/template_vars.go internal/service/application/logic/types.go; do
  sed -i 's/^package service$/package logic/' "$f"
done
```

### Task 5.5: 移动 routes.go 并更新 import 路径

**Files:**
- Move: `internal/service/service/routes.go` → `internal/service/application/routes.go`

- [ ] **Step 1: 移动 routes.go**

```bash
git mv internal/service/service/routes.go internal/service/application/routes.go
```

- [ ] **Step 2: 更新 import 路径**

**Modify**: `internal/service/application/routes.go`

```go
import (
    applicationhandler "github.com/cy77cc/OpsPilot/internal/service/application/handler"
    applicationlogic "github.com/cy77cc/OpsPilot/internal/service/application/logic"
)
```

### Task 5.6: 删除原 service 目录

**Files:**
- Delete: `internal/service/service/` 目录

- [ ] **Step 1: 验证编译后删除原目录**

```bash
go build ./internal/service/application/...
rm -rf internal/service/service/
```

### Task 5.7: 更新外部引用

**Files:**
- Search: 全项目对 `service/service` 的引用

- [ ] **Step 1: 搜索外部引用**

```bash
grep -rn "service/service" --include="*.go" internal/ | grep -v "application"
```

- [ ] **Step 2: 批量更新引用路径**

将所有 `github.com/cy77cc/OpsPilot/internal/service/service` 改为 `github.com/cy77cc/OpsPilot/internal/service/application`

### Task 5.8: 验证编译并提交

- [ ] **Step 1: 验证编译**

```bash
go build ./internal/service/application/...
go build ./internal/service/...
```

- [ ] **Step 2: 提交 Phase 5**

```bash
git add internal/service/application/
git commit -m "refactor: rename service/service to application

- Avoid 'service' keyword conflict
- handler.go → handler/handler.go
- logic*.go → logic/*.go (remove logic_ prefix)
- template_vars.go, types.go → logic/
- Top-level only keeps routes.go

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Phase 6: cluster 模块（复杂度：最高）

**现状**：大量 flat 文件需整理
```
cluster/
├── handler/
│   ├── handler_approval.go
│   ├── handler_hpa.go
│   ├── handler_namespace.go
│   ├── handler_rollout.go
│   ├── policy.go
│   └── resource.go
├── logic/doc.go          # 仅占位符
├── contracts/, integration/, security/  # 子目录
├── handler.go            # flat 文件
├── handler_approvals.go
├── handler_delivery_gitops.go
├── handler_operations.go
├── handler_policy.go
├── handler_security_admission.go
├── handler_security_runtime.go
├── logic_advanced.go
├── logic_bootstrap.go
├── logic_import.go
├── logic_nodes.go
├── logic_resources.go
├── logic_services.go
├── logic_*.go
├── policy_*.go          # 策略相关
├── security_repository.go
├── security_slo_logic.go
├── security_types.go
├── approval_policy.go
├── cache_policy.go
├── collector.go
├── delivery_consistency_logic.go
├── governance_audit.go
├── operation_response.go
├── redaction.go
├── repository.go
├── types.go
└── routes.go
```

**目标结构**：
```
cluster/
├── handler/
│   ├── handler.go        # 原 handler.go（基础）
│   ├── approvals.go      # 原 handler_approvals.go
│   ├── delivery_gitops.go
│   ├── operations.go
│   ├── policy.go
│   ├── security_admission.go
│   ├── security_runtime.go
│   ├── approval.go       # 原 handler/handler_approval.go
│   ├── hpa.go            # 原 handler/handler_hpa.go
│   ├── namespace.go      # 原 handler/handler_namespace.go
│   ├── rollout.go        # 原 handler/handler_rollout.go
│   └── resource.go       # 保留
├── logic/
│   ├── advanced.go
│   ├── bootstrap.go
│   ├── import.go
│   ├── nodes.go
│   ├── resources.go
│   ├── services.go
│   ├── *_ops_logic.go
│   ├── approval_policy.go  # 从根目录移入
│   ├── cache_policy.go     # 从根目录移入
│   ├── collector.go        # 从根目录移入
│   ├── delivery_consistency.go  # 原 delivery_consistency_logic.go
│   ├── audit.go            # 原 governance_audit.go
│   ├── operation_response.go
│   ├── redaction.go
│   ├── repository.go
│   └── types.go
├── domain/
│   ├── policy/
│   │   ├── adapter_calico.go
│   │   ├── adapter_cilium.go
│   │   ├── adapter_flannel.go
│   │   ├── definition.go
│   │   ├── metrics.go
│   │   ├── release.go
│   │   └── simulation.go
│   └── security/
│       ├── repository.go
│       ├── slo_logic.go
│       └── types.go
├── contracts/           # 保持不变
├── integration/         # 保持不变
└── routes.go
```

### Task 6.1: 创建 domain 子目录

**Files:**
- Create: `internal/service/cluster/domain/policy/`
- Create: `internal/service/cluster/domain/security/`

- [ ] **Step 1: 创建目录**

```bash
mkdir -p internal/service/cluster/domain/policy
mkdir -p internal/service/cluster/domain/security
```

### Task 6.2: 移动 handler_*.go 文件到 handler/ 并重命名

**Files:**
- Move: `internal/service/cluster/handler.go` → `internal/service/cluster/handler/handler.go`
- Move: `internal/service/cluster/handler_approvals.go` → `internal/service/cluster/handler/approvals.go`
- Move: `internal/service/cluster/handler_delivery_gitops.go` → `internal/service/cluster/handler/delivery_gitops.go`
- Move: `internal/service/cluster/handler_operations.go` → `internal/service/cluster/handler/operations.go`
- Move: `internal/service/cluster/handler_policy.go` → `internal/service/cluster/handler/policy.go`
- Move: `internal/service/cluster/handler_security_admission.go` → `internal/service/cluster/handler/security_admission.go`
- Move: `internal/service/cluster/handler_security_runtime.go` → `internal/service/cluster/handler/security_runtime.go`

- [ ] **Step 1: 批量移动 handler 文件**

```bash
git mv internal/service/cluster/handler.go internal/service/cluster/handler/handler.go
git mv internal/service/cluster/handler_approvals.go internal/service/cluster/handler/approvals.go
git mv internal/service/cluster/handler_delivery_gitops.go internal/service/cluster/handler/delivery_gitops.go
git mv internal/service/cluster/handler_operations.go internal/service/cluster/handler/operations.go
git mv internal/service/cluster/handler_policy.go internal/service/cluster/handler/policy.go
git mv internal/service/cluster/handler_security_admission.go internal/service/cluster/handler/security_admission.go
git mv internal/service/cluster/handler_security_runtime.go internal/service/cluster/handler/security_runtime.go
```

- [ ] **Step 2: 重命名 handler/ 内的现有文件**

```bash
git mv internal/service/cluster/handler/handler_approval.go internal/service/cluster/handler/approval.go
git mv internal/service/cluster/handler/handler_hpa.go internal/service/cluster/handler/hpa.go
git mv internal/service/cluster/handler/handler_namespace.go internal/service/cluster/handler/namespace.go
git mv internal/service/cluster/handler/handler_rollout.go internal/service/cluster/handler/rollout.go
```

- [ ] **Step 3: 批量更新 package 声明**

所有移入 handler/ 的文件，确保 package 为 `handler`。

### Task 6.3: 移动 logic_*.go 文件到 logic/

**Files:**
- Move: `internal/service/cluster/logic_advanced.go` → `internal/service/cluster/logic/advanced.go`
- Move: `internal/service/cluster/logic_bootstrap.go` → `internal/service/cluster/logic/bootstrap.go`
- Move: `internal/service/cluster/logic_import.go` → `internal/service/cluster/logic/import.go`
- Move: `internal/service/cluster/logic_nodes.go` → `internal/service/cluster/logic/nodes.go`
- Move: `internal/service/cluster/logic_resources.go` → `internal/service/cluster/logic/resources.go`
- Move: `internal/service/cluster/logic_services.go` → `internal/service/cluster/logic/services.go`
- Move: `internal/service/cluster/logic_advanced_ops_logic.go` → `internal/service/cluster/logic/advanced_ops_logic.go`
- Move: `internal/service/cluster/logic_node_ops_logic.go` → `internal/service/cluster/logic/node_ops_logic.go`
- Move: `internal/service/cluster/logic_service_ops_logic.go` → `internal/service/cluster/logic/service_ops_logic.go`
- Move: `internal/service/cluster/logic_workload_ops_logic.go` → `internal/service/cluster/logic/workload_ops_logic.go`

- [ ] **Step 1: 批量移动 logic 文件**

```bash
git mv internal/service/cluster/logic_advanced.go internal/service/cluster/logic/advanced.go
git mv internal/service/cluster/logic_bootstrap.go internal/service/cluster/logic/bootstrap.go
git mv internal/service/cluster/logic_import.go internal/service/cluster/logic/import.go
git mv internal/service/cluster/logic_nodes.go internal/service/cluster/logic/nodes.go
git mv internal/service/cluster/logic_resources.go internal/service/cluster/logic/resources.go
git mv internal/service/cluster/logic_services.go internal/service/cluster/logic/services.go
git mv internal/service/cluster/logic_advanced_ops_logic.go internal/service/cluster/logic/advanced_ops_logic.go
git mv internal/service/cluster/logic_node_ops_logic.go internal/service/cluster/logic/node_ops_logic.go
git mv internal/service/cluster/logic_service_ops_logic.go internal/service/cluster/logic/service_ops_logic.go
git mv internal/service/cluster/logic_workload_ops_logic.go internal/service/cluster/logic/workload_ops_logic.go
```

- [ ] **Step 2: 批量更新 package 声明**

```bash
for f in internal/service/cluster/logic/*.go; do
  sed -i 's/^package cluster$/package logic/' "$f"
done
```

- [ ] **Step 3: 删除 logic/doc.go 占位符**

```bash
rm internal/service/cluster/logic/doc.go
```

### Task 6.4: 移动 policy_*.go 文件到 domain/policy/

**Files:**
- Move: `internal/service/cluster/policy_adapter_calico.go` → `internal/service/cluster/domain/policy/adapter_calico.go`
- Move: `internal/service/cluster/policy_adapter_cilium.go` → `internal/service/cluster/domain/policy/adapter_cilium.go`
- Move: `internal/service/cluster/policy_adapter_flannel.go` → `internal/service/cluster/domain/policy/adapter_flannel.go`
- Move: `internal/service/cluster/policy_definition.go` → `internal/service/cluster/domain/policy/definition.go`
- Move: `internal/service/cluster/policy_metrics.go` → `internal/service/cluster/domain/policy/metrics.go`
- Move: `internal/service/cluster/policy_release.go` → `internal/service/cluster/domain/policy/release.go`
- Move: `internal/service/cluster/policy_simulation.go` → `internal/service/cluster/domain/policy/simulation.go`

- [ ] **Step 1: 批量移动 policy 文件**

```bash
git mv internal/service/cluster/policy_adapter_calico.go internal/service/cluster/domain/policy/adapter_calico.go
git mv internal/service/cluster/policy_adapter_cilium.go internal/service/cluster/domain/policy/adapter_cilium.go
git mv internal/service/cluster/policy_adapter_flannel.go internal/service/cluster/domain/policy/adapter_flannel.go
git mv internal/service/cluster/policy_definition.go internal/service/cluster/domain/policy/definition.go
git mv internal/service/cluster/policy_metrics.go internal/service/cluster/domain/policy/metrics.go
git mv internal/service/cluster/policy_release.go internal/service/cluster/domain/policy/release.go
git mv internal/service/cluster/policy_simulation.go internal/service/cluster/domain/policy/simulation.go
```

- [ ] **Step 2: 批量更新 package 声明**

```bash
for f in internal/service/cluster/domain/policy/*.go; do
  sed -i 's/^package cluster$/package policy/' "$f"
done
```

- [ ] **Step 3: 移动测试文件**

```bash
git mv internal/service/cluster/policy_adapter_calico_test.go internal/service/cluster/domain/policy/adapter_calico_test.go 2>/dev/null || true
git mv internal/service/cluster/policy_adapter_cilium_test.go internal/service/cluster/domain/policy/adapter_cilium_test.go 2>/dev/null || true
git mv internal/service/cluster/policy_adapter_flannel_test.go internal/service/cluster/domain/policy/adapter_flannel_test.go 2>/dev/null || true
git mv internal/service/cluster/policy_definition_test.go internal/service/cluster/domain/policy/definition_test.go 2>/dev/null || true
git mv internal/service/cluster/policy_release_test.go internal/service/cluster/domain/policy/release_test.go 2>/dev/null || true
git mv internal/service/cluster/policy_simulation_test.go internal/service/cluster/domain/policy/simulation_test.go 2>/dev/null || true
```

### Task 6.5: 移动 security_*.go 文件到 domain/security/

**Files:**
- Move: `internal/service/cluster/security_repository.go` → `internal/service/cluster/domain/security/repository.go`
- Move: `internal/service/cluster/security_slo_logic.go` → `internal/service/cluster/domain/security/slo_logic.go`
- Move: `internal/service/cluster/security_types.go` → `internal/service/cluster/domain/security/types.go`

- [ ] **Step 1: 批量移动 security 文件**

```bash
git mv internal/service/cluster/security_repository.go internal/service/cluster/domain/security/repository.go
git mv internal/service/cluster/security_slo_logic.go internal/service/cluster/domain/security/slo_logic.go
git mv internal/service/cluster/security_types.go internal/service/cluster/domain/security/types.go
```

- [ ] **Step 2: 批量更新 package 声明**

```bash
for f in internal/service/cluster/domain/security/*.go; do
  sed -i 's/^package cluster$/package security/' "$f"
done
```

- [ ] **Step 3: 移动测试文件**

```bash
git mv internal/service/cluster/security_repository_test.go internal/service/cluster/domain/security/repository_test.go 2>/dev/null || true
git mv internal/service/cluster/security_slo_logic_test.go internal/service/cluster/domain/security/slo_logic_test.go 2>/dev/null || true
```

### Task 6.6: 移动根目录其他 Go 文件到 logic/

**Files:**
- Move: `internal/service/cluster/approval_policy.go` → `internal/service/cluster/logic/approval_policy.go`
- Move: `internal/service/cluster/cache_policy.go` → `internal/service/cluster/logic/cache_policy.go`
- Move: `internal/service/cluster/collector.go` → `internal/service/cluster/logic/collector.go`
- Move: `internal/service/cluster/delivery_consistency_logic.go` → `internal/service/cluster/logic/delivery_consistency.go`
- Move: `internal/service/cluster/governance_audit.go` → `internal/service/cluster/logic/audit.go`
- Move: `internal/service/cluster/operation_response.go` → `internal/service/cluster/logic/operation_response.go`
- Move: `internal/service/cluster/redaction.go` → `internal/service/cluster/logic/redaction.go`
- Move: `internal/service/cluster/repository.go` → `internal/service/cluster/logic/repository.go`
- Move: `internal/service/cluster/types.go` → `internal/service/cluster/logic/types.go`

- [ ] **Step 1: 批量移动根目录文件**

```bash
git mv internal/service/cluster/approval_policy.go internal/service/cluster/logic/approval_policy.go
git mv internal/service/cluster/cache_policy.go internal/service/cluster/logic/cache_policy.go
git mv internal/service/cluster/collector.go internal/service/cluster/logic/collector.go
git mv internal/service/cluster/delivery_consistency_logic.go internal/service/cluster/logic/delivery_consistency.go
git mv internal/service/cluster/governance_audit.go internal/service/cluster/logic/audit.go
git mv internal/service/cluster/operation_response.go internal/service/cluster/logic/operation_response.go
git mv internal/service/cluster/redaction.go internal/service/cluster/logic/redaction.go
git mv internal/service/cluster/repository.go internal/service/cluster/logic/repository.go
git mv internal/service/cluster/types.go internal/service/cluster/logic/types.go
```

- [ ] **Step 2: 批量更新 package 声明**

```bash
for f in internal/service/cluster/logic/approval_policy.go internal/service/cluster/logic/cache_policy.go internal/service/cluster/logic/collector.go internal/service/cluster/logic/delivery_consistency.go internal/service/cluster/logic/audit.go internal/service/cluster/logic/operation_response.go internal/service/cluster/logic/redaction.go internal/service/cluster/logic/repository.go internal/service/cluster/logic/types.go; do
  sed -i 's/^package cluster$/package logic/' "$f"
done
```

- [ ] **Step 3: 移动相关测试文件**

```bash
git mv internal/service/cluster/delivery_consistency_logic_test.go internal/service/cluster/logic/delivery_consistency_test.go 2>/dev/null || true
git mv internal/service/cluster/operation_response_test.go internal/service/cluster/logic/operation_response_test.go 2>/dev/null || true
```

### Task 6.7: 更新 routes.go import 路径

**Files:**
- Modify: `internal/service/cluster/routes.go`

- [ ] **Step 1: 更新 import 路径**

```go
import (
    clusterhandler "github.com/cy77cc/OpsPilot/internal/service/cluster/handler"
    clusterlogic "github.com/cy77cc/OpsPilot/internal/service/cluster/logic"
    clusterpolicy "github.com/cy77cc/OpsPilot/internal/service/cluster/domain/policy"
    clustersecurity "github.com/cy77cc/OpsPilot/internal/service/cluster/domain/security"
)
```

### Task 6.8: 验证编译并提交

- [ ] **Step 1: 验证编译**

```bash
go build ./internal/service/cluster/...
go test ./internal/service/cluster/...
```

- [ ] **Step 2: 提交 Phase 6**

```bash
git add internal/service/cluster/
git commit -m "refactor(cluster): move all Go files to handler/logic/domain

- Move handler_*.go to handler/ (remove handler_ prefix)
- Move logic_*.go to logic/ (remove logic_ prefix)
- Create domain/policy/ for policy_*.go files
- Create domain/security/ for security_*.go files
- Move all root-level Go files to logic/
- Top-level only keeps routes.go

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 最终验证

### Task F.1: 全量编译检查

- [ ] **Step 1: 编译整个 service 目录**

```bash
go build ./internal/service/...
```

### Task F.2: 运行测试

- [ ] **Step 1: 运行所有测试**

```bash
go test ./internal/service/...
```

### Task F.3: 检查顶层目录是否符合规范

- [ ] **Step 1: 验证各模块顶层只有 routes.go**

```bash
for dir in notification dashboard governance deployment application cluster; do
  echo "=== $dir ==="
  ls internal/service/$dir/*.go 2>/dev/null || echo "No Go files in root"
done
```

Expected: 每个模块只显示 `routes.go`

### Task F.4: 最终提交

- [ ] **Step 1: 汇总提交（如之前未单独提交）**

```bash
git add -A
git commit -m "refactor: complete service module restructuring

Principle: Top-level only keeps routes.go, all Go files moved to subdirectories

Phase 1: notification - handler/ subdirectory
Phase 2: dashboard - move collector.go to logic/
Phase 3: governance - handler/logic/ subdirectories
Phase 4: deployment - handler/logic/ subdirectories (all 15+ files moved)
Phase 5: service → application rename (all files moved)
Phase 6: cluster - handler/logic/domain/ structure

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 附录：各模块最终结构速查

```
notification/
├── handler/
│   ├── notification.go
│   ├── provider.go
│   └── integration.go
├── logic/
│   └── .gitkeep
└── routes.go

dashboard/
├── handler/
│   └── handler.go
├── logic/
│   ├── logic.go
│   └── collector.go
└── routes.go

governance/
├── handler/
│   ├── errors.go
│   └── service.go
├── logic/
│   └── types.go
├── approval/, audit/, envelope/, policy/
└── routes.go

deployment/
├── handler/
│   ├── release.go
│   └── environment.go
├── logic/
│   ├── deployment.go
│   ├── bootstrap.go
│   ├── bootstrap_init.go
│   ├── compose.go
│   ├── governance.go
│   ├── release.go
│   ├── target.go
│   ├── util.go
│   ├── audit.go
│   ├── metrics.go
│   ├── policy.go
│   ├── topology.go
│   └── types.go
└── routes.go

application/
├── handler/
│   └── handler.go
├── logic/
│   ├── application.go
│   ├── deploy.go
│   ├── env_match.go
│   ├── render.go
│   ├── revision.go
│   ├── service.go
│   ├── util.go
│   ├── variable.go
│   ├── template_vars.go
│   └── types.go
└── routes.go

cluster/
├── handler/
│   ├── handler.go
│   ├── approvals.go
│   ├── delivery_gitops.go
│   ├── operations.go
│   ├── policy.go
│   ├── security_admission.go
│   ├── security_runtime.go
│   ├── approval.go
│   ├── hpa.go
│   ├── namespace.go
│   ├── rollout.go
│   └── resource.go
├── logic/
│   ├── advanced.go
│   ├── bootstrap.go
│   ├── import.go
│   ├── nodes.go
│   ├── resources.go
│   ├── services.go
│   ├── *_ops_logic.go
│   ├── approval_policy.go
│   ├── cache_policy.go
│   ├── collector.go
│   ├── delivery_consistency.go
│   ├── audit.go
│   ├── operation_response.go
│   ├── redaction.go
│   ├── repository.go
│   └── types.go
├── domain/
│   ├── policy/
│   │   ├── adapter_calico.go
│   │   ├── adapter_cilium.go
│   │   ├── adapter_flannel.go
│   │   ├── definition.go
│   │   ├── metrics.go
│   │   ├── release.go
│   │   └── simulation.go
│   └── security/
│       ├── repository.go
│       ├── slo_logic.go
│       └── types.go
├── contracts/
├── integration/
└── routes.go
```