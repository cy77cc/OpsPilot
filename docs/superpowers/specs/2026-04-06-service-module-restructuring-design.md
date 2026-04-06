# Service 模块目录结构重组设计

## 1. 目标

建立统一的目录结构规范：**目录即索引** —— 通过目录结构即可定位代码、理解职责。

## 2. 核心原则

| 层级 | 职责 | 文件命名规范 |
|------|------|-------------|
| `handler/` | HTTP 处理层 | `xxx.go` - 按业务概念命名，不重复 handler_ 前缀 |
| `logic/` | 业务逻辑层 | `xxx.go` - 按业务概念命名，不重复 logic_ 前缀 |
| `domain/` | 领域细分（可选） | 复杂模块按领域进一步拆分 |
| `routes.go` | 路由注册 | 保持不变 |

**命名规范示例**：
- ✅ `handler/release.go` - 释放处理
- ❌ `handler/handler_release.go` - 重复前缀，冗余

## 3. 模块重组清单

### 3.1 deployment - 最高优先级

**现状**：15个 flat 文件在根目录
```
deployment/
├── handler.go, routes.go
├── logic.go, logic_bootstrap.go, logic_compose.go, logic_governance.go
├── logic_release.go, logic_target.go, logic_util.go
├── bootstrap.go, audit.go, metrics.go, policy.go, topology.go, types.go
└── handler_environment.go
```

**目标结构**：
```
deployment/
├── handler/
│   ├── release.go           # 发布处理（原 handler.go）
│   ├── environment.go       # 环境相关（自 handler_environment.go）
│   └── bootstrap.go          # 启动处理
├── logic/
│   ├── compose.go           # Compose 编排逻辑
│   ├── governance.go        # 治理逻辑
│   ├── release.go            # 发布逻辑
│   ├── target.go             # 目标逻辑
│   └── util.go               # 工具逻辑
├── bootstrap.go              # 引导逻辑（留在根目录，模块级初始化）
├── audit.go                  # 审计
├── metrics.go                # 指标
├── policy.go                 # 策略
├── topology.go               # 拓扑
├── types.go                  # 类型定义
└── routes.go                 # 路由注册
```

### 3.2 notification

**现状**：3个 flat 文件
```
notification/
├── notification.go, provider.go, integration.go
```

**目标结构**：
```
notification/
├── handler/
│   ├── notification.go       # 通知处理
│   ├── provider.go           # Provider 配置
│   └── integration.go        # 集成处理
├── logic/                    # 业务逻辑（暂无，新建空目录占位）
└── routes.go
```

### 3.3 topology

**现状**：只有 routes.go
```
topology/
└── routes.go
```

**目标结构**：
```
topology/
├── handler/
│   └── handler.go            # 从 routes.go 拆分
├── logic/
│   └── logic.go              # 从 routes.go 拆分
└── routes.go
```

### 3.4 governance

**现状**：部分子目录 + 3个 flat 文件
```
governance/
├── approval/, audit/, envelope/, policy/  # 子目录
├── errors.go, service.go, types.go       # flat 文件
```

**目标结构**：
```
governance/
├── handler/
│   ├── errors.go             # 错误处理
│   └── service.go            # 服务处理（自 service.go）
├── logic/                     # 业务逻辑（暂无，新建空目录占位）
├── approval/, audit/, envelope/, policy/  # 保持不变
└── routes.go
```
> 注：governance/types.go 若为领域类型定义，保留根目录；若是 DTO 则移入 handler/

### 3.5 cluster

**现状**：handler/logic 存在 + 20个 flat 文件
```
cluster/
├── handler/, logic/           # 已存在
├── policy_*.go (6个), security_*.go (4个)
├── handler_policy.go, handler_security_*.go
├── logic_advanced.go, logic_services.go
├── repository.go, security_repository.go
├── bootstrap.go, collector.go
├── delivery_consistency_logic.go
├── governance_audit.go
├── redaction.go, operation_response.go
└── types.go, routes.go
```

**目标结构**：
```
cluster/
├── handler/
│   ├── handler.go            # 基础 handler
│   ├── approvals.go          # 审批（自 handler_approvals.go）
│   ├── delivery_gitops.go     # GitOps（自 handler_delivery_gitops.go）
│   ├── operations.go          # 运维操作（自 handler_operations.go）
│   ├── policy.go              # 策略（自 handler_policy.go）
│   ├── security_admission.go  # 安全准入（自 handler_security_admission.go）
│   └── security_runtime.go    # 安全运行时（自 handler_security_runtime.go）
├── logic/
│   ├── advanced.go            # 高级逻辑（自 logic_advanced.go）
│   └── services.go            # 服务逻辑（自 logic_services.go）
├── domain/
│   ├── policy/                # 策略领域
│   │   ├── adapter_calico.go
│   │   ├── adapter_cilium.go
│   │   ├── adapter_flannel.go
│   │   ├── definition.go      # 策略定义
│   │   ├── metrics.go
│   │   ├── release.go         # 策略发布
│   │   └── simulation.go      # 策略模拟
│   └── security/              # 安全领域
│       ├── repository.go      # 安全数据访问
│       ├── slo_logic.go       # SLO 逻辑
│       └── types.go           # 安全类型
├── repository.go              # 数据访问
├── bootstrap.go               # 引导
├── collector.go               # 采集
├── delivery_consistency.go    # 交付一致性（自 delivery_consistency_logic.go）
├── audit.go                   # 治理审计（自 governance_audit.go）
├── redaction.go               # 脱敏
├── operation_response.go      # 操作响应
└── routes.go
```

### 3.6 service → application（重命名）

**现状**：模块名与服务名相同，路径为 `service/service/`

**目标结构**：
```
application/                   # 重命名，避免 service 关键字冲突
├── handler/
│   └── handler.go
├── logic/
│   ├── deploy.go              # 部署逻辑
│   ├── env_match.go           # 环境匹配
│   ├── render.go              # 渲染
│   ├── revision.go            # 版本
│   ├── service.go             # 服务逻辑
│   ├── util.go                # 工具
│   └── variable.go            # 变量
├── template_vars.go
├── types.go
└── routes.go
```

### 3.7 dashboard

**现状**：collector.go + routes.go
```
dashboard/
├── collector.go, routes.go
```

**目标结构**：
```
dashboard/
├── handler/
│   └── dashboard.go           # 新建
├── logic/
│   └── collector.go           # 采集逻辑
└── routes.go
```

## 4. 实施顺序

| Phase | 模块 | 复杂度 | 说明 |
|-------|------|--------|------|
| 1 | notification | 低 | 3个文件，直接移动 |
| 2 | dashboard | 低 | 新建 handler/，移动 collector.go |
| 3 | topology | 中 | 需从 routes.go 拆分逻辑 |
| 4 | governance | 中 | 整理 flat 文件到 handler/ |
| 5 | deployment | 高 | 15个文件，重新归类 |
| 6 | application | 中 | 仅重命名 |
| 7 | cluster | 最高 | 最多文件，domain/ 子目录 |

## 5. 实施注意事项

1. **Git History**：使用 `git mv` 保留历史
2. **Package 声明**：移动后需更新 `package`（通常与目录名一致）
3. **Import 路径**：`routes.go` 中的 import 需同步更新
4. **CI 验证**：每个 Phase 完成后运行测试，确认无破坏
5. **先小后大**：从简单模块开始，积累经验后再处理复杂模块
