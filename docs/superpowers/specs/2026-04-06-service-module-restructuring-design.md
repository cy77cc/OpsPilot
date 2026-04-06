# Service 模块目录结构重组设计

## 1. 目标

建立统一的目录结构规范：**目录即索引** —— 通过目录结构即可定位代码、理解职责。

## 2. 核心原则

| 层级 | 职责 | 说明 |
|------|------|------|
| `handler/` | HTTP 处理层 | 接收请求、参数校验、调用 logic、返回响应 |
| `logic/` | 业务逻辑层 | 核心算法、业务编排、领域规则 |
| `domain/` | 领域细分（可选） | 复杂模块按领域进一步拆分 |
| `routes.go` | 路由注册 | 保持不变，声明式注册路由 |

## 3. 模块重组清单

### 3.1 deployment - 最高优先级

**现状**：15个 flat 文件在根目录
```
deployment/
├── handler.go, routes.go
├── logic.go, logic_*.go (5个)
├── bootstrap.go, logic_bootstrap.go
├── audit.go, metrics.go, policy.go, topology.go, types.go
└── handler_environment.go
```

**目标结构**：
```
deployment/
├── handler/
│   ├── handler.go          # 基础 handler
│   └── environment.go      # 环境相关 handler
├── logic/
│   ├── logic.go            # 基础逻辑
│   ├── logic_bootstrap.go
│   ├── logic_compose.go
│   ├── logic_governance.go
│   ├── logic_release.go
│   ├── logic_target.go
│   └── logic_util.go
├── bootstrap.go            # 启动逻辑
├── audit.go                # 审计相关
├── metrics.go              # 指标相关
├── policy.go               # 策略相关
├── topology.go             # 拓扑相关
├── types.go                # 类型定义
└── routes.go               # 路由注册
```

### 3.2 notification

**现状**：3个 flat 文件
```
notification/
├── notification.go
├── provider.go
└── integration.go
```

**目标结构**：
```
notification/
├── handler/
│   ├── notification.go    # 主 handler
│   ├── provider.go          # Provider 配置 handler
│   └── integration.go      # 集成 handler
├── logic/                   # 业务逻辑（暂无，新建）
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
│   └── handler.go          # 从 routes.go 拆分出 handler
├── logic/
│   └── logic.go            # 从 routes.go 拆分出 logic
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
│   ├── errors.go           # 错误处理
│   ├── service.go          # 服务 handler
│   └── types.go            # 类型 handler
├── logic/                  # 新建，整理业务逻辑
├── approval/, audit/, envelope/, policy/  # 保持不变
└── routes.go
```

### 3.5 cluster

**现状**：handler/logic 存在 + 20个 flat 文件
```
cluster/
├── handler/, logic/        # 已存在
├── policy_*.go (6个), security_*.go (4个)
├── handler_policy.go, handler_security_*.go
├── logic_advanced.go, logic_services.go
├── repository.go, security_repository.go
├── bootstrap.go, collector.go
├── delivery_consistency_logic.go
├── governance_audit.go
├── redaction.go, operation_response.go
├── types.go
└── routes.go
```

**目标结构**：
```
cluster/
├── handler/
│   ├── handler.go          # 基础 handler
│   ├── approvals.go        # 审批相关
│   ├── delivery_gitops.go  # GitOps 发布
│   ├── operations.go       # 运维操作
│   ├── policy.go           # 策略
│   ├── security_admission.go
│   └── security_runtime.go
├── logic/
│   ├── logic.go            # 基础逻辑
│   ├── logic_advanced.go
│   ├── logic_services.go
│   └── logic_nodes.go
├── domain/
│   ├── policy/             # 策略领域
│   │   ├── adapter_calico.go
│   │   ├── adapter_cilium.go
│   │   ├── adapter_flannel.go
│   │   ├── definition.go
│   │   ├── metrics.go
│   │   ├── release.go
│   │   └── simulation.go
│   └── security/           # 安全领域
│       ├── repository.go
│       ├── slo_logic.go
│       └── types.go
├── repository.go           # 数据访问
├── bootstrap.go            # 启动逻辑
├── collector.go            # 采集
├── delivery_consistency_logic.go
├── governance_audit.go
├── redaction.go
├── operation_response.go
├── types.go
└── routes.go
```

### 3.6 service → application（重命名）

**现状**：模块名与服务名相同，路径为 `service/service/`

**目标结构**：
```
application/                # 重命名，避免 service 关键字冲突
├── handler/
│   └── handler.go
├── logic/
│   ├── logic.go, logic_deploy.go
│   ├── logic_env_match.go, logic_render.go
│   ├── logic_revision.go, logic_service.go
│   ├── logic_util.go, logic_variable.go
│   └── render.go
├── template_vars.go
├── types.go
└── routes.go
```

### 3.7 dashboard

**现状**：collector.go + routes.go
```
dashboard/
├── collector.go
└── routes.go
```

**目标结构**：
```
dashboard/
├── handler/
│   └── dashboard.go        # handler（暂无，新建）
├── logic/
│   └── collector.go        # 采集逻辑
└── routes.go
```

## 4. 实施顺序

1. **Phase 1**: deployment, notification（改动最小，先行）
2. **Phase 2**: topology, governance（需要拆分现有文件）
3. **Phase 3**: dashboard, service → application（涉及重命名）
4. **Phase 4**: cluster（最复杂，文件最多）

## 5. 注意事项

- 所有文件移动后需更新 `package` 声明
- `routes.go` 中的 import 路径需同步更新
- Git history 保留：使用 `git mv` 而非直接 mv
- 先备份/确认 CI 通过后再合并
