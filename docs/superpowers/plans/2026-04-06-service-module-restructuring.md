# Service 模块目录结构重组实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立统一的 `handler/logic/domain` 目录结构规范，实现"目录即索引"

**Architecture:** 按业务层级分离：handler（HTTP处理）→ logic（业务逻辑）→ domain（领域细分，仅复杂模块）

**Tech Stack:** Go, Gin framework, 目录重构

---

## 概述

本计划将 `internal/service/` 下的模块按 handler/logic/domain 目录结构重组。

**发现的问题（实际勘误）**：
- `topology` 已正确结构化，无需修改
- `notification` 的 `notification.go` 同时包含 handler 和路由注册，需拆分
- `dashboard` 的 `collector.go` 是采集逻辑，应移入 `logic/`

---

## Phase 1: notification 模块

**目标**: 将 3 个 flat 文件重组为 `handler/` 结构

### Task 1.1: 创建 handler 子目录

- [ ] 创建目录 `internal/service/notification/handler/`

```bash
mkdir -p internal/service/notification/handler
```

### Task 1.2: 创建 handler/notification.go

**Create**: `internal/service/notification/handler/notification.go`

从 `notification.go` 提取 handler 方法，保留 `NotificationService` struct 和所有 HTTP 处理方法。

```go
package handler

import (
	"strconv"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NotificationService struct {
	svcCtx *svc.ServiceContext
}

func NewNotificationService(svcCtx *svc.ServiceContext) *NotificationService {
	return &NotificationService{svcCtx: svcCtx}
}

// ListNotifications 获取通知列表...
// (提取自 notification.go 的所有方法)
```

### Task 1.3: 更新 routes.go import 路径

**Modify**: `internal/service/notification/routes.go`

将 `RegisterNotificationHandlers` 中的 import 从 `github.com/cy77cc/OpsPilot/internal/service/notification` 改为 `github.com/cy77cc/OpsPilot/internal/service/notification/handler`

```go
import (
	notificationhandler "github.com/cy77cc/OpsPilot/internal/service/notification/handler"
)

// RegisterNotificationHandlers 中:
// svc := NewNotificationService(svcCtx)
svc := notificationhandler.NewNotificationService(svcCtx)
```

### Task 1.4: 删除原 notification.go

**Delete**: `internal/service/notification/notification.go`

确认新结构工作后，删除旧文件。

### Task 1.5: 移动 provider.go 和 integration.go

```bash
# 使用 git mv 保留历史
git mv internal/service/notification/provider.go internal/service/notification/handler/provider.go
git mv internal/service/notification/integration.go internal/service/notification/handler/integration.go
```

### Task 1.6: 创建 logic 空目录占位

```bash
mkdir -p internal/service/notification/logic
# 添加 .gitkeep 或空文件确保目录存在
touch internal/service/notification/logic/.gitkeep
```

### Task 1.7: 验证编译

```bash
go build ./internal/service/notification/...
```

---

## Phase 2: dashboard 模块

**目标**: 将 `collector.go` 移入 `logic/`

### Task 2.1: 创建 logic 子目录

```bash
mkdir -p internal/service/dashboard/logic
```

### Task 2.2: 移动 collector.go

```bash
git mv internal/service/dashboard/collector.go internal/service/dashboard/logic/collector.go
```

### Task 2.3: 更新 collector.go 的 package 声明

**Modify**: `internal/service/dashboard/logic/collector.go`

```go
// 从:
package dashboard
// 改为:
package logic
```

### Task 2.4: 更新 handler.go 的 import 路径

**Modify**: `internal/service/dashboard/handler/handler.go`

如果 `handler.go` 导入了 `dashboard` 包（用于 `NewCollector`），需更新 import。

### Task 2.5: 验证编译

```bash
go build ./internal/service/dashboard/...
```

---

## Phase 3: governance 模块

**目标**: 将 `errors.go` 和 `service.go` 移入 `handler/`

### Task 3.1: 创建 handler 子目录

```bash
mkdir -p internal/service/governance/handler
```

### Task 3.2: 移动并检查 errors.go

```bash
git mv internal/service/governance/errors.go internal/service/governance/handler/errors.go
```

检查 `errors.go` 是否被其他模块引用：
```bash
grep -r "governance.*CodeSuccess\|governance.*GovError" --include="*.go" internal/
```

### Task 3.3: 移动 service.go

```bash
git mv internal/service/governance/service.go internal/service/governance/handler/service.go
```

### Task 3.4: 检查 service.go 的引用

```bash
grep -r "governance\.NewService\|governance\.Service " --include="*.go" internal/
```

### Task 3.5: 创建 logic 空目录占位

```bash
mkdir -p internal/service/governance/logic
touch internal/service/governance/logic/.gitkeep
```

### Task 3.6: 验证编译

```bash
go build ./internal/service/governance/...
```

---

## Phase 4: deployment 模块

**目标**: 将 15 个 flat 文件重组为 handler/logic 结构

**文件清单**:
- `handler.go` → `handler/release.go`
- `handler_environment.go` → `handler/environment.go`
- `logic.go` → `logic/deployment.go` (保留根目录作为入口)
- `logic_bootstrap.go` → `logic/bootstrap.go`
- `logic_compose.go` → `logic/compose.go`
- `logic_governance.go` → `logic/governance.go`
- `logic_release.go` → `logic/release.go`
- `logic_target.go` → `logic/target.go`
- `logic_util.go` → `logic/util.go`
- `bootstrap.go` → 留在根目录（模块初始化）
- `audit.go` → 留在根目录
- `metrics.go` → 留在根目录
- `policy.go` → 留在根目录
- `topology.go` → 留在根目录
- `types.go` → 留在根目录

### Task 4.1: 创建子目录

```bash
mkdir -p internal/service/deployment/handler
mkdir -p internal/service/deployment/logic
```

### Task 4.2: 移动 handler 文件

```bash
git mv internal/service/deployment/handler.go internal/service/deployment/handler/release.go
git mv internal/service/deployment/handler_environment.go internal/service/deployment/handler/environment.go
```

### Task 4.3: 更新 handler/release.go 的 package

**Modify**: `internal/service/deployment/handler/release.go`

```go
package handler
```

### Task 4.4: 更新 handler/environment.go 的 package

**Modify**: `internal/service/deployment/handler/environment.go`

```go
package handler
```

### Task 4.5: 移动 logic 文件

```bash
git mv internal/service/deployment/logic.go internal/service/deployment/logic/deployment.go
git mv internal/service/deployment/logic_bootstrap.go internal/service/deployment/logic/bootstrap.go
git mv internal/service/deployment/logic_compose.go internal/service/deployment/logic/compose.go
git mv internal/service/deployment/logic_governance.go internal/service/deployment/logic/governance.go
git mv internal/service/deployment/logic_release.go internal/service/deployment/logic/release.go
git mv internal/service/deployment/logic_target.go internal/service/deployment/logic/target.go
git mv internal/service/deployment/logic_util.go internal/service/deployment/logic/util.go
```

### Task 4.6: 更新所有移动文件的 package 声明

将 `logic_*.go` 改名后的文件 package 从 `deployment` 改为 `logic`。

### Task 4.7: 更新 routes.go import

**Modify**: `internal/service/deployment/routes.go`

更新所有 import 路径，指向新的 handler 和 logic 包。

### Task 4.8: 验证编译

```bash
go build ./internal/service/deployment/...
```

---

## Phase 5: service → application 重命名

**目标**: 将 `service/service/` 目录重命名为 `service/application/`

### Task 5.1: 创建 application 目录结构

```bash
mkdir -p internal/service/application/handler
mkdir -p internal/service/application/logic
```

### Task 5.2: 移动文件并重命名

```bash
# Handler
git mv internal/service/service/handler.go internal/service/application/handler/handler.go

# Logic files (去掉 logic_ 前缀)
git mv internal/service/service/logic.go internal/service/application/logic/application.go
git mv internal/service/service/logic_deploy.go internal/service/application/logic/deploy.go
git mv internal/service/service/logic_env_match.go internal/service/application/logic/env_match.go
git mv internal/service/service/logic_render.go internal/service/application/logic/render.go
git mv internal/service/service/logic_revision.go internal/service/application/logic/revision.go
git mv internal/service/service/logic_service.go internal/service/application/logic/service.go
git mv internal/service/service/logic_util.go internal/service/application/logic/util.go
git mv internal/service/service/logic_variable.go internal/service/application/logic/variable.go
git mv internal/service/service/render.go internal/service/application/logic/render.go  # 如果存在且不同

# 其他文件
git mv internal/service/service/template_vars.go internal/service/application/
git mv internal/service/service/types.go internal/service/application/
git mv internal/service/service/routes.go internal/service/application/
```

### Task 5.3: 更新所有 package 声明

将所有移动的文件 package 从 `service` 改为 `application`。

### Task 5.4: 更新 routes.go 中的 import

**Modify**: `internal/service/application/routes.go`

更新所有 import 路径。

### Task 5.5: 验证编译

```bash
go build ./internal/service/application/...
```

---

## Phase 6: cluster 模块

**目标**: 建立 `handler/`、`logic/`、`domain/policy/`、`domain/security/` 结构

### Task 6.1: 创建 domain 子目录

```bash
mkdir -p internal/service/cluster/domain/policy
mkdir -p internal/service/cluster/domain/security
```

### Task 6.2: 移动 handler 文件

```bash
# handler/ 已有，整理命名
git mv internal/service/cluster/handler_approvals.go internal/service/cluster/handler/approvals.go
git mv internal/service/cluster/handler_delivery_gitops.go internal/service/cluster/handler/delivery_gitops.go
git mv internal/service/cluster/handler_policy.go internal/service/cluster/handler/policy.go
git mv internal/service/cluster/handler_security_admission.go internal/service/cluster/handler/security_admission.go
git mv internal/service/cluster/handler_security_runtime.go internal/service/cluster/handler/security_runtime.go
```

### Task 6.3: 移动 logic 文件

```bash
git mv internal/service/cluster/logic_advanced.go internal/service/cluster/logic/advanced.go
git mv internal/service/cluster/logic_services.go internal/service/cluster/logic/services.go
# logic_nodes.go 已存在
```

### Task 6.4: 移动 domain/policy 文件

```bash
git mv internal/service/cluster/policy_adapter_calico.go internal/service/cluster/domain/policy/adapter_calico.go
git mv internal/service/cluster/policy_adapter_cilium.go internal/service/cluster/domain/policy/adapter_cilium.go
git mv internal/service/cluster/policy_adapter_flannel.go internal/service/cluster/domain/policy/adapter_flannel.go
git mv internal/service/cluster/policy_definition.go internal/service/cluster/domain/policy/definition.go
git mv internal/service/cluster/policy_metrics.go internal/service/cluster/domain/policy/metrics.go
git mv internal/service/cluster/policy_release.go internal/service/cluster/domain/policy/release.go
git mv internal/service/cluster/policy_simulation.go internal/service/cluster/domain/policy/simulation.go
```

### Task 6.5: 移动 domain/security 文件

```bash
git mv internal/service/cluster/security_repository.go internal/service/cluster/domain/security/repository.go
git mv internal/service/cluster/security_slo_logic.go internal/service/cluster/domain/security/slo_logic.go
git mv internal/service/cluster/security_types.go internal/service/cluster/domain/security/types.go
```

### Task 6.6: 整理其他文件

```bash
# 根目录保留
git mv internal/service/cluster/delivery_consistency_logic.go internal/service/cluster/delivery_consistency.go
git mv internal/service/cluster/governance_audit.go internal/service/cluster/audit.go
```

### Task 6.7: 更新所有 package 声明

批量更新所有移动文件的 package 声明。

### Task 6.8: 更新 routes.go import

**Modify**: `internal/service/cluster/routes.go`

更新所有 import 路径。

### Task 6.9: 验证编译

```bash
go build ./internal/service/cluster/...
```

---

## 最终验证

### Task F.1: 全量编译检查

```bash
go build ./internal/service/...
```

### Task F.2: 运行测试

```bash
go test ./internal/service/...
```

### Task F.3: 提交所有更改

```bash
git add -A
git commit -m "refactor: restructure service modules to handler/logic/domain layout

Phase 1: notification - handler/ subdirectory
Phase 2: dashboard - move collector.go to logic/
Phase 3: governance - handler/ subdirectory
Phase 4: deployment - handler/logic/ subdirectories
Phase 5: service -> application rename
Phase 6: cluster - domain/policy and domain/security

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 附录：各模块目标结构速查

```
notification/
├── handler/
│   ├── notification.go
│   ├── provider.go
│   └── integration.go
├── logic/               # 空目录占位
└── routes.go

dashboard/
├── handler/
│   └── handler.go
├── logic/
│   └── collector.go
└── routes.go

governance/
├── handler/
│   ├── errors.go
│   └── service.go
├── logic/               # 空目录占位
├── approval/, audit/, envelope/, policy/
└── routes.go

deployment/
├── handler/
│   ├── release.go
│   └── environment.go
├── logic/
│   ├── deployment.go
│   ├── bootstrap.go
│   ├── compose.go
│   ├── governance.go
│   ├── release.go
│   ├── target.go
│   └── util.go
├── bootstrap.go, audit.go, metrics.go, policy.go, topology.go, types.go
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
│   └── variable.go
├── template_vars.go, types.go
└── routes.go

cluster/
├── handler/
│   ├── handler.go
│   ├── approvals.go
│   ├── delivery_gitops.go
│   ├── operations.go
│   ├── policy.go
│   ├── security_admission.go
│   └── security_runtime.go
├── logic/
│   ├── advanced.go
│   ├── nodes.go
│   └── services.go
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
├── bootstrap.go, collector.go, delivery_consistency.go, audit.go
├── redaction.go, operation_response.go, repository.go, types.go
└── routes.go
```
