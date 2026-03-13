# Spec: AI Module Detailed Technical Specification

## Overview

本文档记录 AI 模块重构的详细技术规格，包括 Runtime 实现、场景配置、审批流程、工具系统的具体设计。

---

## 1. Runtime 实现

### 1.1 Planner Prompt 规格

#### System Prompt 结构

```
You are an expert platform SRE planner specializing in Kubernetes and cloud operations.

## PLANNING PRINCIPLES

**1. Understand the Goal:**
- Analyze the user's request in the context of the current scene and selected resources.
- Identify the target resources and the desired outcome.

**2. Resource Awareness:**
- If the user has selected specific resources, those should be the primary targets.
- Always include namespace in your steps when operating on Kubernetes resources.
- Consider project boundaries and access permissions.

**3. Step Decomposition:**
- Break down complex tasks into atomic, verifiable steps.
- Each step should have a single, clear objective.
- For mutating operations, include verification steps before and after.

**4. Tool Selection:**
- Choose the most appropriate tool for each step.
- Prefer readonly tools for information gathering.
- Be aware that mutating tools will require approval before execution.

## SCENE AWARENESS

You are operating in the **{scene_name}** scene. Follow these constraints:
{scene_constraints}

## EXAMPLES

{examples}

## RESTRICTIONS

- Do not generate steps that operate on resources outside the current project.
- Always include verification steps for mutating operations.
- Do not skip steps for the sake of brevity.
```

#### User Message 结构

```
## SCENE CONTEXT
- **Scene**: {scene_name} - {scene_description}
- **Project**: {project_name} ({project_id})
- **Current Page**: {current_page}
{selected_resources}

## USER REQUEST
{input}

## CURRENT TIME
{current_time}

Generate a plan to fulfill the user's request.
```

#### Plan 输出格式

```json
{
  "steps": [
    {
      "instruction": "查询 nginx 部署当前状态",
      "tool": "get_deployment",
      "params": {"name": "nginx", "namespace": "default"},
      "reason": "验证当前副本数"
    },
    {
      "instruction": "执行扩容操作",
      "tool": "scale_deployment",
      "params": {"name": "nginx", "namespace": "default", "replicas": 3},
      "reason": "用户请求扩容到 3 副本"
    }
  ]
}
```

### 1.2 Executor Prompt 规格

#### System Prompt 结构

```
You are a diligent and meticulous platform SRE executor working in a Kubernetes and cloud operations environment.

## EXECUTION PRINCIPLES
- Stay focused on the current step while keeping the overall objective in mind.
- Prefer tool-based verification over assumptions.
- Use the most relevant domain tools for the task.
- Base every conclusion on concrete tool output.

## CONTEXT AWARENESS
- Always consider the current scene, project, and selected resources.
- If the user has selected specific resources, use them as the primary target.
- Respect project boundaries.
- Be aware of the current page context.

## TOOL CLASSIFICATION & APPROVAL
- **readonly** tools (mode=readonly): Safe to execute directly.
- **mutating** tools (mode=mutating): Will modify state, require approval.
  - **low risk**: Routine changes, auto-approved in non-production.
  - **medium risk**: Significant changes, always require approval.
  - **high risk**: Critical changes, require explicit approval.

## RESPONSE REQUIREMENTS
- Report what you checked, what tools you used, and what evidence you found.
- Summarize the result of the current step clearly and concisely.
```

#### User Message 结构

```
## SCENE CONTEXT
- **Scene**: {scene_name} - {scene_description}
- **Project**: {project_name} ({project_id})
- **Current Page**: {current_page}
{selected_resources}

## SCENE CONSTRAINTS
{scene_constraints}

## OBJECTIVE
{input}

## GIVEN PLAN
{plan}

## COMPLETED STEPS & RESULTS
{executed_steps}

## YOUR TASK
Execute the first step, which is:
{step}
```

### 1.3 Replanner Prompt 规格

#### System Prompt 结构

```
You are an expert platform SRE replanner specializing in Kubernetes and cloud operations.

## REPLANNING PRINCIPLES

**1. Evaluate Execution Results:**
- Analyze whether the current step achieved its intended goal.
- Check if the result reveals new information that affects subsequent steps.
- Identify any errors or unexpected conditions.

**2. Decision Making:**

When to **submit_result** (plan is complete):
- All steps have been executed successfully.
- The user's original request has been fully addressed.

When to **create_plan** (plan needs modification):
- An error occurred that requires a different approach.
- New information suggests a better path forward.
- A step was rejected (e.g., approval denied).

When to continue (no tool call needed):
- The plan is progressing as expected.

## ERROR HANDLING

| Error Type | Replanning Action |
|------------|-------------------|
| Resource not found | Try alternative queries or ask for clarification |
| Permission denied | Report issue, suggest alternative |
| Approval rejected | Propose alternative or report back to user |
| Timeout | Consider retry or alternative approach |
| Tool failure | Analyze cause, adjust plan or report |
```

---

## 2. 场景配置规格

### 2.1 SceneConfig 数据结构

```go
type SceneConfig struct {
    // 基础信息
    Name        string   `json:"name"`        // 场景名称
    Description string   `json:"description"` // 场景描述
    Constraints []string `json:"constraints"` // 场景约束

    // 工具配置
    AllowedTools []string `json:"allowed_tools"` // 允许的工具（白名单）
    BlockedTools []string `json:"blocked_tools"` // 禁用的工具（黑名单）
    Examples     []string `json:"examples"`      // 场景示例 ID 列表

    // 审批配置
    ApprovalConfig *SceneApprovalConfig `json:"approval_config,omitempty"`
}

type SceneApprovalConfig struct {
    // 默认审批策略
    DefaultPolicy ApprovalPolicy `json:"default_policy"`

    // 工具级覆盖配置
    ToolOverrides map[string]ToolApprovalOverride `json:"tool_overrides,omitempty"`

    // 基于环境的审批要求
    EnvironmentPolicies map[string]ApprovalPolicy `json:"environment_policies,omitempty"`
}

type ApprovalPolicy struct {
    // 需要审批的风险等级阈值
    RequireApprovalFor []RiskLevel `json:"require_approval_for"`

    // 是否对所有变更操作都需要审批
    RequireForAllMutating bool `json:"require_for_all_mutating,omitempty"`

    // 跳过审批的条件
    SkipConditions []SkipCondition `json:"skip_conditions,omitempty"`
}

type ToolApprovalOverride struct {
    ForceApproval   bool   `json:"force_approval,omitempty"`
    SkipApproval    bool   `json:"skip_approval,omitempty"`
    SummaryTemplate string `json:"summary_template,omitempty"`
}

type SkipCondition struct {
    Type    string `json:"type"`    // environment, namespace, label
    Pattern string `json:"pattern"` // 匹配模式
}
```

### 2.2 预定义场景

| Scene | Name | Allowed Tools | Blocked Tools | 审批策略 |
|-------|------|---------------|---------------|----------|
| deployment | 部署管理 | get_deployment, scale_deployment, restart_deployment... | delete_cluster, execute_host_command | medium+high 需审批，dev 环境跳过 |
| monitor | 监控中心 | get_cluster_info, list_pods, query_alerts... | 所有 mutating 工具 | 只读场景，无需审批 |
| host | 主机管理 | list_hosts, get_host_info, execute_host_command... | - | 所有变更需审批 |
| cicd | CI/CD | list_pipelines, trigger_pipeline... | - | medium+high 需审批 |

### 2.3 场景配置 API

| 接口 | 方法 | 描述 |
|------|------|------|
| /ai/scene/configs | GET | 获取所有场景配置 |
| /ai/scene/configs/:scene | GET | 获取指定场景配置 |
| /ai/scene/configs/:scene | PUT | 更新场景配置 |
| /ai/scene/configs/:scene | DELETE | 删除场景配置（恢复默认） |

---

## 3. 审批流程规格

### 3.1 审批决策流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                      ApprovalDecisionMaker                           │
│                                                                      │
│  Input: ApprovalCheckRequest                                        │
│  { tool_name, scene, environment, namespace, params }               │
│                                                                      │
│  Step 1: 获取 ToolMeta                                              │
│          ↓                                                           │
│  Step 2: 检查 Mode == readonly?                                     │
│          ├── YES → 返回 NeedApproval: false                         │
│          └── NO  → 继续                                             │
│                                                                      │
│  Step 3: 获取 SceneConfig                                           │
│          ↓                                                           │
│  Step 4: 检查 ToolOverrides                                         │
│          ├── ForceApproval: true → 返回 NeedApproval: true          │
│          ├── SkipApproval: true  → 返回 NeedApproval: false         │
│          └── 无覆盖 → 继续                                          │
│                                                                      │
│  Step 5: 检查 EnvironmentPolicies                                   │
│          ├── RequireForAllMutating: true → 返回 NeedApproval: true  │
│          └── 继续                                                   │
│                                                                      │
│  Step 6: 应用 DefaultPolicy                                         │
│          ├── Risk in RequireApprovalFor → 返回 NeedApproval: true  │
│          └── 否则 → 返回 NeedApproval: false                        │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 ApprovalInfo 结构

```go
type ApprovalInfo struct {
    ID              string            `json:"id,omitempty"`
    ToolName        string            `json:"tool_name"`
    ToolDisplayName string            `json:"tool_display_name"`
    ArgumentsInJSON string            `json:"arguments_json"`
    RiskLevel       RiskLevel         `json:"risk_level"`
    Mode            string            `json:"mode"`
    Summary         string            `json:"summary"`
    Params          map[string]any    `json:"params"`
    CreatedAt       time.Time         `json:"created_at"`
    ExpiresAt       time.Time         `json:"expires_at"`
}
```

### 3.3 ApprovalResult 结构

```go
type ApprovalResult struct {
    Approved bool    `json:"approved"`
    Reason   *string `json:"reason,omitempty"`
}
```

### 3.4 SSE approval_required 事件

```json
{
  "event": "approval_required",
  "data": {
    "id": "approval-uuid-123",
    "tool_name": "scale_deployment",
    "tool_display_name": "扩缩容部署",
    "risk_level": "medium",
    "mode": "mutating",
    "summary": "扩缩容部署 nginx 到 3 副本 (命名空间: default)",
    "params": {
      "name": "nginx",
      "namespace": "default",
      "replicas": 3
    },
    "created_at": "2026-03-13T16:00:00Z",
    "expires_at": "2026-03-14T16:00:00Z"
  }
}
```

---

## 4. 工具系统规格

### 4.1 ParamHint 类型

| 类型 | 说明 | 示例 |
|------|------|------|
| static | 静态枚举值 | restart_strategy: [graceful, force, rolling] |
| dynamic | 动态数据源 | namespace: 从数据库查询 |
| remote | 远程接口 | host: 从 /api/v1/hosts 查询 |

### 4.2 内置数据源

| Source | 说明 | 查询参数 |
|--------|------|----------|
| namespaces | 命名空间列表 | project_id |
| deployments | 部署列表 | namespace, project_id |
| pods | Pod 列表 | namespace, label_selector |
| services | 服务列表 | namespace, project_id |
| hosts | 主机列表 | - |
| clusters | 集群列表 | project_id |
| configmaps | ConfigMap 列表 | namespace |
| secrets | Secret 列表 | namespace |

### 4.3 参数依赖拓扑

工具参数可能存在依赖关系，需要按拓扑顺序解析：

```
scale_deployment:
  namespace → name → replicas

参数解析顺序:
1. namespace (无依赖，可先解析)
2. name (依赖 namespace，需等 namespace 选择后再解析)
3. replicas (无依赖，可独立解析)
```

### 4.4 参数提示 API

**请求:**
```
GET /api/v1/ai/tools/scale_deployment/params/hints?
    namespace=default&
    dependencies={"namespace":"default"}
```

**响应:**
```json
{
  "code": 1000,
  "data": {
    "namespace": {
      "name": "namespace",
      "type": "string",
      "required": true,
      "options": [
        {"value": "default", "label": "default"},
        {"value": "production", "label": "production"}
      ]
    },
    "name": {
      "name": "name",
      "type": "string",
      "required": true,
      "options": [
        {"value": "nginx", "label": "nginx (3 replicas)"},
        {"value": "redis", "label": "redis (1 replica)"}
      ]
    },
    "replicas": {
      "name": "replicas",
      "type": "integer",
      "required": true,
      "default": 1
    }
  }
}
```

---

## 5. 工具元数据规格

### 5.1 工具分类

| 工具 | Mode | Risk | Category | 说明 |
|-----|------|------|----------|------|
| get_cluster_info | readonly | low | kubernetes | 获取集群信息 |
| list_pods | readonly | low | kubernetes | 列出 Pod |
| get_deployment | readonly | low | kubernetes | 获取部署详情 |
| scale_deployment | mutating | medium | kubernetes | 扩缩容部署 |
| restart_deployment | mutating | medium | kubernetes | 重启部署 |
| restart_pod | mutating | medium | kubernetes | 重启 Pod |
| delete_deployment | mutating | high | kubernetes | 删除部署 |
| execute_host_command | mutating | high | host | 执行主机命令 |
| trigger_pipeline | mutating | medium | cicd | 触发流水线 |

### 5.2 风险等级定义

| 等级 | 说明 | 示例操作 |
|------|------|----------|
| low | 低风险，只读或轻微变更 | 查询资源、获取日志 |
| medium | 中风险，可能影响服务 | 扩缩容、重启、配置变更 |
| high | 高风险，可能导致数据丢失或服务中断 | 删除资源、执行命令、关键配置变更 |

---

## 6. 完整流程示例

### 6.1 扩容部署流程

```
用户请求: "帮我扩容 nginx 部署到 3 副本"
场景: deployment
环境: production

Step 1: Planner 生成计划
├── 查询 nginx 部署当前状态 (get_deployment)
├── 执行扩容操作 (scale_deployment)
└── 确认扩容结果 (list_pods)

Step 2: Executor 执行步骤 1
├── get_deployment (readonly, low risk)
├── ApprovalDecision: NeedApproval = false
├── 直接执行
└── 结果: nginx 当前有 1 副本

Step 3: Executor 执行步骤 2
├── scale_deployment (mutating, medium risk)
├── ApprovalDecision:
│   ├── ToolMeta: mode=mutating, risk=medium
│   ├── SceneConfig: deployment
│   ├── EnvironmentPolicies["production"]: RequireForAllMutating=true
│   └── NeedApproval = true
├── ApprovalGate.triggerInterrupt()
│   ├── 生成摘要: "扩缩容部署 nginx 到 3 副本"
│   ├── 发送 approval_required SSE 事件
│   └── 暂停执行，等待审批

Step 4: 用户审批
├── POST /ai/approvals/{id}/approve
└── 返回: approved = true

Step 5: Resume 恢复执行
├── ApprovalGate.handleResume()
│   ├── 获取 ApprovalResult: Approved = true
│   └── 执行 scale_deployment
└── 结果: 扩容成功

Step 6: Executor 执行步骤 3
├── list_pods (readonly, low risk)
├── 直接执行
└── 结果: 3 个 Pod 运行中

Step 7: Replanner 决策
├── 所有步骤执行成功
└── submit_result: "扩容成功，nginx 部署现在有 3 个副本在运行"
```

---

## 7. 配置示例

### 7.1 生产环境场景配置

```json
{
  "scene": "deployment",
  "name": "部署管理",
  "description": "管理 Kubernetes 部署、扩缩容、回滚等操作",
  "constraints": [
    "跨命名空间操作需要明确指定",
    "生产环境变更需要审批"
  ],
  "allowed_tools": [
    "get_deployment", "list_deployments", "scale_deployment",
    "restart_deployment", "rollback_deployment",
    "list_pods", "get_pod_logs", "describe_pod"
  ],
  "blocked_tools": [
    "delete_cluster", "execute_host_command"
  ],
  "approval_config": {
    "default_policy": {
      "require_approval_for": ["medium", "high"],
      "skip_conditions": [
        {"type": "environment", "pattern": "dev"}
      ]
    },
    "tool_overrides": {
      "delete_deployment": {
        "force_approval": true,
        "summary_template": "删除部署 {name} (命名空间: {namespace}) - 此操作不可恢复"
      }
    },
    "environment_policies": {
      "production": {
        "require_for_all_mutating": true,
        "require_approval_for": ["low", "medium", "high"]
      },
      "staging": {
        "require_approval_for": ["medium", "high"]
      },
      "dev": {
        "require_approval_for": ["high"]
      }
    }
  }
}
```

---

## 8. 错误码定义

| Code | Message | Description |
|------|---------|-------------|
| 4001 | Tool not found | 工具不存在 |
| 4002 | Invalid parameters | 参数校验失败 |
| 4003 | Approval required | 变更工具需要审批 |
| 4004 | Invalid approval token | 审批 token 无效 |
| 4005 | Approval expired | 审批已过期 |
| 4006 | Execution not found | 执行记录不存在 |
| 4007 | Scene not found | 场景不存在 |
| 4008 | Tool blocked | 工具在当前场景被禁用 |
| 4009 | Approval rejected | 审批已拒绝 |

---

## 9. API 层规格

### 9.1 接口概览与优先级

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          API 实现优先级                                      │
│                                                                              │
│  P0 (阻塞前端)                    P1 (核心功能)                             │
│  ┌─────────────────────────┐     ┌─────────────────────────┐               │
│  │ 审批流程 (5个)           │     │ 能力查询 (2个)           │               │
│  │ - POST /approvals       │     │ - GET /capabilities     │               │
│  │ - GET /approvals        │     │ - GET /tools/:name/     │               │
│  │ - GET /approvals/:id    │     │   params/hints          │               │
│  │ - POST /approve         │     └─────────────────────────┘               │
│  │ - POST /reject          │                                              │
│  │                         │     P2 (体验优化)                             │
│  │ 工具执行 (2个)           │     ┌─────────────────────────┐               │
│  │ - POST /tools/execute   │     │ 场景相关 (3个)           │               │
│  │ - GET /executions/:id   │     │ - GET /scene/:scene/    │               │
│  └─────────────────────────┘     │   tools                 │               │
│                                  │ - GET /scene/:scene/    │               │
│                                  │   prompts               │               │
│                                  │ - POST /tools/preview   │               │
│                                  └─────────────────────────┘               │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 9.2 审批流程接口

#### 9.2.1 POST /ai/approvals - 创建审批

**请求:**
```json
{
  "tool_name": "scale_deployment",
  "params": {
    "name": "nginx",
    "namespace": "default",
    "replicas": 3
  },
  "session_id": "session-uuid",
  "checkpoint_id": "checkpoint-uuid"
}
```

**响应:**
```json
{
  "code": 1000,
  "data": {
    "id": "approval-uuid",
    "tool_name": "scale_deployment",
    "tool_display_name": "扩缩容部署",
    "risk_level": "medium",
    "mode": "mutating",
    "summary": "扩缩容部署 nginx 到 3 副本 (命名空间: default)",
    "params": { "name": "nginx", "namespace": "default", "replicas": 3 },
    "status": "pending",
    "expires_at": "2026-03-14T16:00:00Z",
    "created_at": "2026-03-13T16:00:00Z"
  }
}
```

**实现逻辑:**
1. 获取工具元数据 ToolMeta
2. 调用 ApprovalDecisionMaker.Decide() 判断是否需要审批
3. 生成审批记录，存储到 MySQL
4. 存储临时状态到 Redis (TTL: 24h)

#### 9.2.2 GET /ai/approvals - 审批列表

**请求:**
```
GET /ai/approvals?status=pending&page=1&page_size=20
```

**响应:**
```json
{
  "code": 1000,
  "data": {
    "items": [
      {
        "id": "approval-uuid-1",
        "tool_name": "scale_deployment",
        "tool_display_name": "扩缩容部署",
        "risk_level": "medium",
        "summary": "扩缩容部署 nginx 到 3 副本",
        "status": "pending",
        "created_at": "2026-03-13T16:00:00Z"
      }
    ],
    "total": 5,
    "page": 1,
    "page_size": 20
  }
}
```

#### 9.2.3 GET /ai/approvals/:id - 审批详情

**响应:**
```json
{
  "code": 1000,
  "data": {
    "id": "approval-uuid",
    "tool_name": "scale_deployment",
    "tool_display_name": "扩缩容部署",
    "risk_level": "medium",
    "mode": "mutating",
    "summary": "扩缩容部署 nginx 到 3 副本 (命名空间: default)",
    "params": { "name": "nginx", "namespace": "default", "replicas": 3 },
    "status": "pending",
    "session_id": "session-uuid",
    "checkpoint_id": "checkpoint-uuid",
    "expires_at": "2026-03-14T16:00:00Z",
    "created_at": "2026-03-13T16:00:00Z",
    "approved_at": null,
    "rejected_at": null,
    "reason": null
  }
}
```

#### 9.2.4 POST /ai/approvals/:id/approve - 批准审批

**请求:**
```json
{
  "reason": "确认扩容"
}
```

**响应:**
```json
{
  "code": 1000,
  "data": {
    "approval": {
      "id": "approval-uuid",
      "status": "approved"
    },
    "execution": {
      "id": "exec-uuid",
      "status": "running"
    }
  }
}
```

**实现逻辑:**
1. 获取审批记录，检查状态和过期时间
2. 更新审批状态为 approved
3. 创建执行记录 (AiExecution)
4. 异步调用 Runtime.Resume() 恢复 Agent 执行

#### 9.2.5 POST /ai/approvals/:id/reject - 拒绝审批

**请求:**
```json
{
  "reason": "取消操作"
}
```

**响应:**
```json
{
  "code": 1000,
  "data": {
    "id": "approval-uuid",
    "status": "rejected"
  }
}
```

**实现逻辑:**
1. 更新审批状态为 rejected
2. 异步通知 Agent 恢复（返回拒绝结果）

### 9.3 工具执行接口

#### 9.3.1 POST /ai/tools/execute - 工具执行

**请求:**
```json
{
  "tool": "scale_deployment",
  "params": {
    "name": "nginx",
    "namespace": "default",
    "replicas": 3
  },
  "checkpoint_id": "cp-abc-def-ghi"
}
```

**响应:**
```json
{
  "code": 1000,
  "data": {
    "id": "exec-uuid",
    "tool": "scale_deployment",
    "status": "running",
    "created_at": "2026-03-13T16:00:00Z"
  }
}
```

**错误响应（需要审批）:**
```json
{
  "code": 4003,
  "msg": "Approval required",
  "data": {
    "approval_required": true,
    "risk_level": "medium",
    "summary": "扩缩容部署 nginx 到 3 副本"
  }
}
```

**实现逻辑:**
1. 获取工具元数据
2. 调用 ApprovalDecisionMaker.Decide() 判断是否需要审批
3. 如需审批，验证 checkpoint_id
4. 创建执行记录
5. 异步执行工具

#### 9.3.2 GET /ai/executions/:id - 执行状态

**响应:**
```json
{
  "code": 1000,
  "data": {
    "id": "exec-uuid",
    "tool": "scale_deployment",
    "params": { "name": "nginx", "namespace": "default", "replicas": 3 },
    "status": "completed",
    "result": {
      "message": "Deployment nginx scaled to 3 replicas",
      "replicas": 3
    },
    "created_at": "2026-03-13T16:00:00Z",
    "finished_at": "2026-03-13T16:00:05Z"
  }
}
```

**实现逻辑:**
1. 先从 Redis 查询（快速路径）
2. Redis 未命中则从 MySQL 查询

### 9.4 能力查询接口

#### 9.4.1 GET /ai/capabilities - 工具能力列表

**响应:**
```json
{
  "code": 1000,
  "data": [
    {
      "name": "get_deployment",
      "display_name": "获取部署详情",
      "description": "获取指定部署的详细信息",
      "mode": "readonly",
      "risk": "low",
      "category": "kubernetes",
      "tags": ["deployment", "read"]
    },
    {
      "name": "scale_deployment",
      "display_name": "扩缩容部署",
      "description": "调整部署的副本数量",
      "mode": "mutating",
      "risk": "medium",
      "category": "kubernetes",
      "tags": ["deployment", "scale"]
    }
  ]
}
```

#### 9.4.2 GET /ai/tools/:name/params/hints - 参数提示

详见第 4.4 节。

### 9.5 场景相关接口

#### 9.5.1 GET /ai/scene/:scene/tools - 场景工具

**响应:**
```json
{
  "code": 1000,
  "data": {
    "scene": "deployment",
    "scene_name": "部署管理",
    "tools": [
      {
        "name": "get_deployment",
        "display_name": "获取部署详情",
        "description": "获取指定部署的详细信息",
        "usage_hint": "用于查询部署状态、副本数、镜像版本等"
      },
      {
        "name": "scale_deployment",
        "display_name": "扩缩容部署",
        "description": "调整部署的副本数量",
        "usage_hint": "用于扩容或缩容，需要审批",
        "requires_approval": true
      }
    ]
  }
}
```

#### 9.5.2 GET /ai/scene/:scene/prompts - 场景提示词

**响应:**
```json
{
  "code": 1000,
  "data": {
    "scene": "deployment",
    "prompts": [
      {
        "id": "prompt-1",
        "text": "帮我扩容 {deployment} 到 {replicas} 副本",
        "type": "template",
        "variables": ["deployment", "replicas"]
      },
      {
        "id": "prompt-2",
        "text": "查看 {deployment} 的日志",
        "type": "template",
        "variables": ["deployment"]
      }
    ]
  }
}
```

#### 9.5.3 POST /ai/tools/preview - 工具预览

**请求:**
```json
{
  "tool": "scale_deployment",
  "params": { "name": "nginx", "namespace": "default", "replicas": 3 }
}
```

**响应:**
```json
{
  "code": 1000,
  "data": {
    "preview": {
      "current_state": {
        "replicas": 1,
        "available_replicas": 1
      },
      "target_state": {
        "replicas": 3
      },
      "impact": {
        "pods_to_create": 2,
        "resource_change": {
          "cpu": "+200m",
          "memory": "+256Mi"
        }
      },
      "warnings": []
    },
    "dry_run": true
  }
}
```

### 9.6 前端对齐

#### 9.6.1 SSE 事件格式

前端使用 `@ant-design/x-sdk` 的 `useXChat` hook，后端 SSE 格式：

```
event: delta
data: {"content": "正在查询..."}

event: tool_call
data: {"tool": "get_deployment", "params": {...}}

event: approval_required
data: {"id": "...", "summary": "...", ...}

event: done
data: {}
```

#### 9.6.2 PlatformChatProvider 适配器

```typescript
export class PlatformChatProvider implements ChatProvider {
  async *[Symbol.asyncIterator](messages: ChatMessage[]): AsyncGenerator<ChatMessage> {
    const response = await fetch('/api/v1/ai/chat/stream', {
      method: 'POST',
      body: JSON.stringify({
        message: messages[messages.length - 1].content,
        scene: this.scene,
        selected_resources: this.selectedResources,
      }),
    });

    // 解析 SSE 流，转换为 ChatMessage
    // ...
  }
}
```

### 9.7 前后端对齐规格

#### 9.7.1 审批 API 路径

**聊天窗口确认 (SSE 流式):**

```
POST /ai/resume/step/stream
Content-Type: application/json

{
  "checkpoint_id": "cp-abc-def-ghi",
  "approved": true,
  "reason": "确认扩容"
}

Response: SSE Stream
event: stage_delta
data: {"stage":"user_action","status":"success","description":"已确认执行"}

event: step_update
data: {"step_id":"step-2","label":"scale_deployment","status":"success","content":"扩容成功"}

event: delta
data: {"content_chunk":"扩容成功..."}

event: done
data: {}
```

**审批中心确认 (异步执行):**

```
POST /ai/approvals/{id}/approve
Content-Type: application/json

{
  "reason": "确认扩容"
}

Response:
{
  "code": 1000,
  "data": {
    "approval": {"id": "approval-uuid", "status": "approved"},
    "execution": {"id": "exec-uuid", "status": "running"}
  }
}
```

#### 9.7.2 SSE 事件类型（ThoughtChain 统一模型）

**事件流程示例:**

```
用户发送: "帮我扩容 nginx 部署到 3 副本"

event: meta
data: {"session_id":"session-1"}

event: stage_delta
data: {"stage":"rewrite","status":"success","content":"用户想要扩容 nginx 部署"}

event: stage_delta
data: {"stage":"plan","status":"loading","description":"正在生成执行计划..."}

event: stage_delta
data: {"stage":"plan","status":"success","description":"生成 3 个步骤","content":"1. 查询 nginx 部署状态\n2. 执行扩容操作\n3. 确认扩容结果"}

event: stage_delta
data: {"stage":"execute","status":"loading","description":"正在执行..."}

event: step_update
data: {"step_id":"step-1","label":"get_deployment","status":"loading","content":"正在查询 nginx 部署..."}

event: step_update
data: {"step_id":"step-1","label":"get_deployment","status":"success","content":"nginx 当前 1 副本","data":{"result":{"ok":true,"replicas":1}}}

event: step_update
data: {"step_id":"step-2","label":"scale_deployment","status":"loading","content":"正在扩容到 3 副本..."}

event: approval_required
data: {"id":"approval-1","checkpoint_id":"cp-xxx","tool_name":"scale_deployment","tool_display_name":"扩缩容部署","risk_level":"medium","summary":"扩缩容部署 nginx 到 3 副本","params":{"name":"nginx","namespace":"default","replicas":3}}

--- 用户审批后继续 ---

event: step_update
data: {"step_id":"step-2","label":"scale_deployment","status":"success","content":"扩容成功"}

event: step_update
data: {"step_id":"step-3","label":"list_pods","status":"loading","content":"正在确认扩容结果..."}

event: step_update
data: {"step_id":"step-3","label":"list_pods","status":"success","content":"3 个 Pod 运行中"}

event: stage_delta
data: {"stage":"execute","status":"success","description":"执行完成"}

event: stage_delta
data: {"stage":"summary","status":"loading"}

event: delta
data: {"content_chunk":"扩容成功，"}

event: delta
data: {"content_chunk":"nginx 部署现在有 3 个副本在运行。"}

event: stage_delta
data: {"stage":"summary","status":"success"}

event: done
data: {"session_id":"session-1"}
```

**前端 ThoughtChain 数据结构:**

```typescript
type ThoughtStageKey = 'rewrite' | 'plan' | 'execute' | 'user_action' | 'summary';
type ThoughtStageStatus = 'loading' | 'success' | 'error' | 'abort';

interface ThoughtStageItem {
  key: ThoughtStageKey;
  title: string;
  status: ThoughtStageStatus;
  description?: string;
  content?: string;
  details?: ThoughtStageDetailItem[];  // execute 阶段的工具调用列表
  collapsible?: boolean;
  blink?: boolean;
}

interface ThoughtStageDetailItem {
  id: string;
  label: string;
  status: ThoughtStageStatus;
  content?: string;
  data?: {
    tool?: string;
    params?: Record<string, unknown>;
    result?: { ok: boolean; data?: unknown; error?: string; latency_ms?: number };
  };
}
```

**前端事件处理（简化版）:**

```typescript
const handlers = {
  onStageDelta: (data) => {
    patchAssistantMessage((message) => ({
      ...message,
      thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
        key: data.stage,
        title: resolveThoughtStageTitle(data.stage),
        status: data.status,
        description: data.description,
        content: data.content,
      }),
    }));
  },

  onStepUpdate: (data) => {
    patchAssistantMessage((message) => {
      const currentExecute = message.thoughtChain?.find(s => s.key === 'execute');
      return {
        ...message,
        thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
          key: 'execute',
          status: 'loading',
          details: upsertDetail(currentExecute?.details || [], {
            id: data.step_id,
            label: data.label,
            status: data.status,
            content: data.content,
            data: data.data,
          }),
        }),
      };
    });
  },

  onApprovalRequired: (data) => {
    patchAssistantMessage((message) => ({
      ...message,
      thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
        key: 'user_action',
        title: '等待你确认',
        status: 'loading',
        description: data.summary,
      }),
    }));
    setPendingConfirmation({...});
  },

  onDelta: (data) => {
    assistantContent += data.contentChunk || '';
    patchAssistantMessage((message) => ({
      ...message,
      content: assistantContent,
      thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
        key: 'summary',
        status: 'loading',
      }),
    }));
  },

  onDone: () => {
    patchAssistantMessage((message) => ({
      ...message,
      thoughtChain: message.thoughtChain?.map(s => ({
        ...s,
        status: s.status === 'loading' ? 'success' : s.status,
      })),
    }));
  },
};
```

#### 9.7.3 ThoughtChain UI 设计规范

##### 9.7.3.1 整体布局

```
┌─────────────────────────────────────────────────────────────┐
│  AI 助手                                                [X] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  💭 思维链 (ThoughtChain)                            │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  [rewrite]  ✓ 理解需求                              │   │
│  │  [plan]     ✓ 执行计划 (点击展开)                    │   │
│  │  [execute]  ◐ 执行中...                             │   │
│  │  [user_action] ◐ 等待确认 (审批面板展开)             │   │
│  │  [summary]  ○ 待执行                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  ⚠️ 操作确认                                         │   │
│  │  ───────────────────────────────────────────────────│   │
│  │  工具: 扩缩容部署                                    │   │
│  │  风险: 中等风险                                      │   │
│  │  操作: 扩缩容部署 nginx 到 3 副本                    │   │
│  │  ───────────────────────────────────────────────────│   │
│  │  参数:                                               │   │
│  │    • name: nginx                                     │   │
│  │    • namespace: default                              │   │
│  │    • replicas: 3                                     │   │
│  │  ───────────────────────────────────────────────────│   │
│  │                              [取消]  [确认执行]       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  📝 最终回复                                         │   │
│  │  扩容成功，nginx 部署现在有 3 个副本在运行...        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  💡 推荐操作                                         │   │
│  │  ───────────────────────────────────────────────────│   │
│  │  • 查看部署详情                                      │   │
│  │  • 检查 Pod 状态                                     │   │
│  │  • 查看资源使用情况                                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  [输入框...]                                           [发送]│
└─────────────────────────────────────────────────────────────┘
```

##### 9.7.3.2 阶段卡片设计

**状态图标与颜色:**

| 状态 | 图标 | 颜色 | 说明 |
|------|------|------|------|
| pending | ○ | `#d9d9d9` (灰色) | 待执行 |
| loading | ◐ | `#1890ff` (蓝色) | 执行中，带旋转动画 |
| success | ✓ | `#52c41a` (绿色) | 执行成功 |
| error | ✗ | `#ff4d4f` (红色) | 执行失败 |
| abort | ◊ | `#faad14` (橙色) | 用户取消 |

**卡片交互:**
- **执行中 (loading)**: 卡片自动展开，显示详细内容
- **执行完成后**: 卡片自动折叠，保留标题和状态
- **用户手动**: 点击卡片可手动展开/折叠

##### 9.7.3.3 各阶段详细设计

**1. rewrite (需求理解)**

```
┌─────────────────────────────────────────────────────────┐
│ ✓ 理解需求                                    [展开/折叠]│
├─────────────────────────────────────────────────────────┤
│ 用户想要扩容 nginx 部署到 3 个副本                       │
│ 意图: 执行操作 | 置信度: 95%                             │
└─────────────────────────────────────────────────────────┘
```

**2. plan (执行计划)**

```
┌─────────────────────────────────────────────────────────┐
│ ✓ 执行计划 (3 步骤)                           [展开/折叠]│
├─────────────────────────────────────────────────────────┤
│ 1. 查询 nginx 部署当前状态                               │
│ 2. 执行扩容操作到 3 副本 (需审批)                        │
│ 3. 确认扩容结果                                          │
└─────────────────────────────────────────────────────────┘
```

**3. execute (执行过程)**

```
┌─────────────────────────────────────────────────────────┐
│ ◐ 执行中 (2/3)                               [展开/折叠]│
├─────────────────────────────────────────────────────────┤
│ ✓ get_deployment (320ms)                                │
│   nginx 当前 1 副本，镜像 nginx:1.25                     │
│                                                         │
│ ◐ scale_deployment                                      │
│   正在扩容到 3 副本...                                   │
│                                                         │
│ ○ list_pods (待执行)                                    │
└─────────────────────────────────────────────────────────┘
```

**4. user_action (用户确认)**

当 `approval_required` 事件触发时:

```
┌─────────────────────────────────────────────────────────┐
│ ◐ 等待你确认                                 [展开/折叠]│
├─────────────────────────────────────────────────────────┤
│ 需要确认以下操作:                                        │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ ⚠️ 扩缩容部署                                      │ │
│ │ ───────────────────────────────────────────────────│ │
│ │ 风险等级: 中等风险                                  │ │
│ │ 操作摘要: 扩缩容部署 nginx 到 3 副本                │ │
│ │ ───────────────────────────────────────────────────│ │
│ │ 参数预览:                                           │ │
│ │   • name: nginx                                     │ │
│ │   • namespace: default                              │ │
│ │   • replicas: 3                                     │ │
│ │ ───────────────────────────────────────────────────│ │
│ │                                    [取消] [确认执行] │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

用户确认后:
- 状态变为 `success`
- 显示确认信息 "已确认执行"

用户取消后:
- 状态变为 `abort`
- 显示取消信息 "用户取消操作"

**5. summary (结果总结)**

```
┌─────────────────────────────────────────────────────────┐
│ ✓ 结果总结                                    [展开/折叠]│
├─────────────────────────────────────────────────────────┤
│ 执行耗时: 2.3s                                          │
│ 工具调用: 3 次                                          │
│ 状态变更: nginx 副本数 1 → 3                            │
└─────────────────────────────────────────────────────────┘
```

##### 9.7.3.4 推荐操作区域

根据执行结果动态生成推荐:

```typescript
interface RecommendedAction {
  id: string;
  label: string;
  action: 'navigate' | 'query' | 'command';
  params?: Record<string, unknown>;
}

// 根据工具执行结果生成推荐
const generateRecommendations = (result: ToolResult): RecommendedAction[] => {
  const recommendations: RecommendedAction[] = [];

  // 扩容后推荐查看 Pod 状态
  if (result.tool === 'scale_deployment') {
    recommendations.push(
      { id: '1', label: '查看 Pod 状态', action: 'query', params: { command: '查看 nginx 的 Pod 状态' } },
      { id: '2', label: '查看资源使用', action: 'query', params: { command: '查看 nginx 的资源使用情况' } },
    );
  }

  // 创建资源后推荐查看详情
  if (result.tool === 'create_deployment') {
    recommendations.push(
      { id: '1', label: '查看部署详情', action: 'navigate', params: { path: '/deployment/nginx' } },
      { id: '2', label: '配置服务暴露', action: 'query', params: { command: '为 nginx 创建 Service' } },
    );
  }

  return recommendations;
};
```

##### 9.7.3.5 动画效果

**阶段切换动画:**
- 新阶段开始: 淡入 + 向下滑动 (200ms ease-out)
- 阶段完成: 状态图标颜色渐变 (150ms)
- 展开/折叠: 高度动画 (200ms ease-in-out)

**执行中动画:**
- loading 图标: CSS 旋转动画 (1s linear infinite)
- 工具调用行: 添加左侧蓝色边框高亮
- 文字流: 打字机效果显示 `content_chunk`

**审批弹入:**
- 审批面板: 从底部滑入 (250ms ease-out)
- 遮罩层: 淡入 (150ms)

##### 9.7.3.6 响应式设计

| 断点 | ThoughtChain 宽度 | 审批面板 |
|------|-------------------|----------|
| > 768px | 100% | 内嵌显示 |
| ≤ 768px | 100% | 全屏弹窗 |

##### 9.7.3.7 暗色模式

```css
/* 暗色模式配色 */
.thought-chain.dark {
  --tc-bg: #1f1f1f;
  --tc-border: #434343;
  --tc-text: #ffffffd9;
  --tc-text-secondary: #ffffff73;
  --tc-success: #52c41a;
  --tc-error: #ff4d4f;
  --tc-warning: #faad14;
  --tc-loading: #1890ff;
}
```

#### 9.7.4 场景 Key 处理

**后端 SceneResolver 实现:**

```go
type SceneResolver struct {
    domainConfigs map[string]SceneConfig    // 一级场景配置
    subSceneRules map[string]SubSceneRule   // 子场景规则（硬编码）
}

func (r *SceneResolver) Resolve(sceneKey string) ResolvedScene {
    parts := strings.Split(sceneKey, ":")
    domain := parts[0]
    sub := ""
    if len(parts) > 1 {
        sub = parts[1]
    }

    config := r.domainConfigs[domain]

    if sub != "" {
        if rule, ok := r.subSceneRules[sub]; ok {
            return r.applyRule(config, rule)
        }
    }

    return ResolvedScene{Config: config}
}

func (r *SceneResolver) applyRule(config SceneConfig, rule SubSceneRule) ResolvedScene {
    resolved := ResolvedScene{Config: config}

    // 应用工具过滤
    if len(rule.IncludeTools) > 0 {
        resolved.AllowedTools = rule.IncludeTools
    }
    if len(rule.ExcludeTools) > 0 {
        resolved.AllowedTools = removeItems(resolved.AllowedTools, rule.ExcludeTools)
    }

    // 添加额外约束
    resolved.Constraints = append(config.Constraints, rule.ExtraConstraints...)

    return resolved
}
```

**子场景规则定义:**

```go
var subSceneRules = map[string]SubSceneRule{
    "clusters": {
        IncludeTools:     []string{"get_cluster_info", "list_nodes", "upgrade_cluster"},
        ExcludeTools:     []string{"execute_host_command"},
        ExtraConstraints: []string{"集群操作需要明确指定目标集群"},
    },
    "hosts": {
        IncludeTools:     []string{"list_hosts", "get_host_info", "execute_host_command"},
        ExtraConstraints: []string{"主机命令执行需要审批"},
        ApprovalAdjust:   &ApprovalAdjustment{ForceApprovalFor: []string{"execute_host_command"}},
    },
    "metrics": {
        IncludeTools:     []string{"get_cluster_info", "list_pods", "query_alerts"},
        ExcludeTools:     []string{"scale_deployment", "restart_deployment", "execute_host_command"},
        ExtraConstraints: []string{"当前为只读监控场景，不支持变更操作"},
    },
}
```

#### 9.7.4 审批标识符

**前端类型定义:**

```typescript
// AIInterruptApprovalResponse 审批响应
export interface AIInterruptApprovalResponse {
  checkpoint_id: string;  // Eino 检查点 ID
  approved: boolean;
  reason?: string;
}

// ApprovalRequiredEvent 审批事件
export interface ApprovalRequiredEvent {
  id: string;
  checkpoint_id: string;
  tool_name: string;
  tool_display_name: string;
  risk_level: 'low' | 'medium' | 'high';
  summary: string;
  params: Record<string, any>;
}
```

**前端处理逻辑:**

```typescript
// useAIChat.ts
const handleApprovalRequired = useCallback((payload: ApprovalRequiredEvent) => {
  const confirmation: ConfirmationRequest = {
    id: payload.id,
    title: payload.tool_display_name,
    description: payload.summary,
    risk: payload.risk_level,
    details: payload as unknown as Record<string, unknown>,
    onConfirm: () => confirmApproval(true, payload.checkpoint_id),
    onCancel: () => confirmApproval(false, payload.checkpoint_id),
  };
  setPendingConfirmation(confirmation);
}, []);

const confirmApproval = useCallback(async (approved: boolean, checkpoint_id: string) => {
  await aiApi.respondApprovalStream({
    checkpoint_id,
    approved,
  }, handlers);
}, []);
```

### 9.8 API 实现总结

| 接口 | 优先级 | 核心逻辑 | 数据库操作 |
|------|--------|----------|-----------|
| POST /approvals | P0 | 生成审批记录，判断风险等级 | INSERT ai_approvals |
| GET /approvals | P0 | 分页查询，状态筛选 | SELECT ai_approvals |
| GET /approvals/:id | P0 | 获取详情 | SELECT ai_approvals |
| POST /approvals/:id/approve | P0 | 更新状态，恢复 Agent 执行 | UPDATE + INSERT ai_executions |
| POST /approvals/:id/reject | P0 | 更新状态，通知 Agent | UPDATE |
| POST /tools/execute | P0 | 异步执行工具，返回执行 ID | INSERT ai_executions |
| GET /executions/:id | P0 | 查询执行状态 | SELECT ai_executions |
| GET /capabilities | P1 | 读取工具元数据 | 无 |
| GET /tools/:name/params/hints | P1 | 解析参数提示，查询数据源 | SELECT (根据 source) |
| GET /scene/:scene/tools | P2 | 读取场景配置 | SELECT ai_scene_configs |
| GET /scene/:scene/prompts | P2 | 读取预定义提示词 | SELECT ai_scene_prompts |
| POST /tools/preview | P2 | 执行 dry-run，返回预期变更 | 无 |

---

## 10. 数据库表结构

### 10.1 ai_approvals 表

```sql
CREATE TABLE ai_approvals (
    id VARCHAR(36) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    tool_mode VARCHAR(20) NOT NULL,
    risk_level VARCHAR(20) NOT NULL,
    params JSON,
    summary VARCHAR(512),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    session_id VARCHAR(36),
    checkpoint_id VARCHAR(36),
    expires_at TIMESTAMP NOT NULL,
    approved_at TIMESTAMP NULL,
    rejected_at TIMESTAMP NULL,
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_status (user_id, status),
    INDEX idx_session (session_id)
);
```

### 10.2 ai_executions 表

```sql
CREATE TABLE ai_executions (
    id VARCHAR(36) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    params JSON,
    mode VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    approval_id VARCHAR(36),
    result JSON,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP NULL,
    INDEX idx_user_status (user_id, status),
    INDEX idx_approval (approval_id)
);
```

### 10.3 ai_scene_configs 表

```sql
CREATE TABLE ai_scene_configs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    scene VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(128),
    description VARCHAR(512),
    constraints TEXT,
    allowed_tools TEXT,
    blocked_tools TEXT,
    examples TEXT,
    approval_config JSON,
    is_enabled BOOLEAN DEFAULT TRUE,
    priority INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### 10.4 ai_scene_prompts 表

```sql
CREATE TABLE ai_scene_prompts (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    scene VARCHAR(64) NOT NULL,
    text VARCHAR(512) NOT NULL,
    type VARCHAR(20) DEFAULT 'template',
    variables JSON,
    sort_order INT DEFAULT 0,
    is_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_scene (scene)
);
```
