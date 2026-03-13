# Design: AI Module Redesign

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                     OpsPilot AI Module Architecture                              │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                           HTTP Handler Layer                               │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐         │ │
│  │  │ ChatHandler │ │SessionHandler│ │ ToolHandler │ │ApprovalHandler│        │ │
│  │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘         │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                          │
│                                       ▼                                          │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                           Orchestrator Layer                               │ │
│  │   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                │ │
│  │   │   Planner   │────▶│  Executor   │────▶│  Replanner  │──┐             │ │
│  │   └─────────────┘     └─────────────┘     └─────────────┘  │             │ │
│  │          ▲                                       │         │             │ │
│  │          └───────────────────────────────────────┘         │             │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                          │
│                                       ▼                                          │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                           Tool Layer                                       │ │
│  │  ┌─────────────────────────────────────────────────────────────────────┐  │ │
│  │  │ Tool Registry: K8sTools | HostTools | ServiceTools | DeployTools    │  │ │
│  │  └─────────────────────────────────────────────────────────────────────┘  │ │
│  │  ┌─────────────────────────────────────────────────────────────────────┐  │ │
│  │  │ Approval Gate: readonly → auto | mutating → interrupt → resume     │  │ │
│  │  └─────────────────────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                          │
│                                       ▼                                          │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                           Storage Layer                                    │ │
│  │  ┌───────────────────────┐     ┌───────────────────────┐                  │ │
│  │  │        Redis          │     │        MySQL          │                  │ │
│  │  │ Checkpoint/State Cache│     │ Session/Approval/Exec │                  │ │
│  │  └───────────────────────┘     └───────────────────────┘                  │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## 1. Runtime Layer

### 1.1 核心架构

使用 **Plan-Execute-Replanner** 模式，基于 Eino ADK 实现：

```
用户请求: "帮我扩容 nginx 部署到 3 副本"
                    │
                    ▼
              ┌──────────┐
              │ Planner  │ ─── 分解成步骤列表
              └──────────┘
                    │
                    ▼
        ┌─────────────────────────┐
        │ [                       │
        │   1. 查询 nginx 部署状态 │
        │   2. 执行扩容操作        │
        │   3. 确认扩容结果        │
        │ ]                       │
        └─────────────────────────┘
                    │
                    ▼
              ┌──────────┐
              │ Executor │ ─── 逐步骤执行，审批中断
              └──────────┘
                    │
                    ▼
              ┌───────────┐
              │ Replanner │ ─── 根据结果调整计划
              └───────────┘
```

### 1.2 混合上下文注入

采用 **混合模式** 注入场景上下文：

```
┌─────────────────────────────────────────────────────────────────────┐
│                         ContextProcessor                             │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ 1. System Prompt Builder  → 注入场景描述、项目约束           │   │
│  │ 2. Tool Filter            → 根据场景筛选可用工具             │   │
│  │ 3. Example Injector       → 注入场景相关的 few-shot 示例     │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.3 Runtime Interface

```go
// Runtime 定义 AI 运行时接口。
type Runtime interface {
    // Run 执行对话请求，通过 SSE 流式输出事件。
    Run(ctx context.Context, req RunRequest, emit StreamEmitter) error

    // Resume 恢复中断的执行（非流式）。
    Resume(ctx context.Context, req ResumeRequest) (*ResumeResult, error)

    // ResumeStream 恢复中断的执行（流式）。
    ResumeStream(ctx context.Context, req ResumeRequest, emit StreamEmitter) (*ResumeResult, error)
}

// RunRequest 对话请求参数。
type RunRequest struct {
    SessionID      string
    Message        string
    RuntimeContext RuntimeContext
}

// RuntimeContext 运行时上下文。
type RuntimeContext struct {
    Scene             string             // 场景标识
    SceneName         string             // 场景名称
    Route             string             // 路由路径
    ProjectID         string             // 项目 ID
    ProjectName       string             // 项目名称
    CurrentPage       string             // 当前页面
    SelectedResources []SelectedResource // 选中的资源
    UserContext       map[string]any     // 用户上下文
}

// SelectedResource 选中的资源。
type SelectedResource struct {
    Type      string `json:"type"`      // deployment, pod, service, etc.
    Name      string `json:"name"`      // 资源名称
    Namespace string `json:"namespace"` // 命名空间
}
```

### 1.4 Planner 设计

#### Prompt 结构

```
System Prompt:
├── PLANNING PRINCIPLES（规划原则）
├── RESOURCE AWARENESS（资源感知）
├── STEP DECOMPOSITION（步骤分解）
├── TOOL SELECTION（工具选择）
├── SCENE AWARENESS（场景感知）
├── EXAMPLES（示例）
└── RESTRICTIONS（约束）

User Message:
├── SCENE CONTEXT（场景上下文）
│   ├── Scene: 部署管理
│   ├── Project: Production (proj-123)
│   ├── Selected Resources: deployment: nginx
│   └── Scene Constraints: 生产环境变更需要审批
├── USER REQUEST（用户请求）
├── CURRENT TIME（当前时间）
└── Generate a plan...（生成计划）
```

#### Planner 实现

```go
func NewPlanner(ctx context.Context, deps PlannerDeps) (adk.Agent, error) {
    cm, err := newPlannerChatModel(ctx)
    if err != nil {
        return nil, err
    }

    return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
        ChatModelWithFormattedOutput: cm,
        GenInputFn:                   newPlannerInputGen(deps),
        NewPlan: func(ctx context.Context) planexecute.Plan {
            return &Plan{}
        },
    })
}
```

### 1.5 Executor 设计

#### Prompt 增强

在原有 Executor prompt 基础上增加：

1. **CONTEXT AWARENESS** - 场景、项目、选中资源感知
2. **TOOL CLASSIFICATION & APPROVAL** - 工具分类和审批说明
3. **场景约束** - 从 SceneConfig 获取

#### 执行流程

```
Plan Step
{ tool: "scale_deployment", params: {...} }
        │
        ▼
┌─────────────────┐
│ Tool Lookup     │ ─── 查找工具定义
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Risk Check      │ ─── 检查工具风险等级
│ mode: mutating  │
│ risk: medium    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│ readonly?       │──NO─▶│ ApprovalGate    │
└────────┬────────┘     │ Interrupt       │
         │              └────────┬────────┘
        YES                      │
         │                       ▼
         │              ┌─────────────────┐
         │              │ Wait for        │
         │              │ Approval        │
         │              └────────┬────────┘
         │                       │
         │              ┌────────┴────────┐
         │           approved          rejected
         │              │                 │
         ▼              ▼                 ▼
┌─────────────────────────────┐   ┌─────────────────┐
│     Execute Tool            │   │ Return          │
│     (InvokableRun)          │   │ "disapproved"   │
└─────────────────────────────┘   └─────────────────┘
```

### 1.6 Replanner 设计

Replanner 在每步执行后判断是否需要调整计划：

```
Decision:
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐   │
│   │ 计划完成？   │──YES──▶│ 提交结果    │     │             │   │
│   └──────┬──────┘     │submit_result│     │             │   │
│          │            └─────────────┘     │             │   │
│         NO                 │              │             │   │
│          │                 │              │             │   │
│          ▼                 ▼              ▼             │   │
│   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐   │
│   │ 计划需要    │──YES──▶│ 生成新计划  │     │ 继续执行    │   │
│   │ 调整？      │       │create_plan  │     │下一步骤     │   │
│   └──────┬──────┘     └─────────────┘     └─────────────┘   │
│          │                   │                ▲             │
│         NO                   └────────────────┘             │
│          │                                                  │
│          └──────────────────────────────────────────────────┘
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

## 2. 场景配置系统

### 2.1 硬编码默认 + 数据库覆盖

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          SceneConfigResolver                             │
│                                                                         │
│   ┌─────────────────┐         ┌─────────────────┐                      │
│   │ DefaultScenes   │         │ DB Overrides    │                      │
│   │ (硬编码)         │         │ (数据库)         │                      │
│   │                 │         │                 │                      │
│   │ deployment: {...}│  ─────▶ │ deployment: {...}│ (覆盖)              │
│   │ monitor: {...}   │         │ host: {...}     │ (覆盖)              │
│   │ host: {...}      │         │ new-scene: {...}│ (新增)              │
│   └─────────────────┘         └─────────────────┘                      │
│            │                           │                                │
│            └───────────┬───────────────┘                                │
│                        ▼                                                │
│              ┌─────────────────┐                                        │
│              │ Merged Config   │                                        │
│              │ (最终配置)       │                                        │
│              └─────────────────┘                                        │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 SceneConfig 结构

```go
// SceneConfig 场景配置
type SceneConfig struct {
    Name         string        `json:"name"`
    Description  string        `json:"description"`
    Constraints  []string      `json:"constraints"`
    AllowedTools []string      `json:"allowed_tools"`
    BlockedTools []string      `json:"blocked_tools"`
    Examples     []string      `json:"examples"`

    // 审批配置
    ApprovalConfig *SceneApprovalConfig `json:"approval_config,omitempty"`
}

// SceneApprovalConfig 场景审批配置
type SceneApprovalConfig struct {
    DefaultPolicy       ApprovalPolicy                `json:"default_policy"`
    ToolOverrides       map[string]ToolApprovalOverride `json:"tool_overrides,omitempty"`
    EnvironmentPolicies map[string]ApprovalPolicy      `json:"environment_policies,omitempty"`
}

// ApprovalPolicy 审批策略
type ApprovalPolicy struct {
    RequireApprovalFor    []RiskLevel    `json:"require_approval_for"`
    RequireForAllMutating bool           `json:"require_for_all_mutating,omitempty"`
    SkipConditions        []SkipCondition `json:"skip_conditions,omitempty"`
}
```

### 2.3 预定义场景

```go
var defaultScenes = map[string]*SceneConfig{
    "deployment": {
        Name:        "部署管理",
        Description: "管理 Kubernetes 部署、扩缩容、回滚等操作",
        Constraints: []string{
            "跨命名空间操作需要明确指定",
            "生产环境变更需要审批",
        },
        AllowedTools: []string{
            "get_deployment", "list_deployments", "scale_deployment",
            "restart_deployment", "rollback_deployment",
            "list_pods", "get_pod_logs", "describe_pod",
        },
        BlockedTools: []string{
            "delete_cluster", "execute_host_command",
        },
        ApprovalConfig: &SceneApprovalConfig{
            DefaultPolicy: ApprovalPolicy{
                RequireApprovalFor: []RiskLevel{RiskMedium, RiskHigh},
                SkipConditions: []SkipCondition{
                    {Type: "environment", Pattern: "dev"},
                },
            },
            EnvironmentPolicies: map[string]ApprovalPolicy{
                "production": {
                    RequireForAllMutating: true,
                },
            },
        },
    },
    "monitor": {
        Name:        "监控中心",
        Description: "查看集群、应用、资源的监控指标和告警",
        Constraints: []string{"只读场景，不执行变更操作"},
        AllowedTools: []string{
            "get_cluster_info", "list_pods", "get_pod_metrics",
            "get_node_metrics", "query_alerts", "query_logs",
        },
    },
}
```

### 2.4 配置刷新机制

```
配置获取流程:
1. 检查本地缓存 (5min TTL)
   ├─ HIT → 直接返回
   └─ MISS → 继续
2. 查询数据库
   ├─ 找到 → 合并默认配置 + 数据库配置
   └─ 未找到 → 使用默认配置
3. 写入缓存，返回结果

配置更新流程:
1. 管理员通过 API 更新配置
2. 保存到 MySQL
3. 发布 Redis 通知: "scene:config:changed"
4. 所有实例订阅通知，清除本地缓存
5. 下次请求重新加载最新配置
```

---

## 3. 审批流程

### 3.1 审批粒度：场景+工具级

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          审批判断流程                                         │
│                                                                              │
│   ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐       │
│   │ ToolMeta        │     │ SceneConfig     │     │ Environment     │       │
│   │ (工具定义)       │     │ (场景配置)       │     │ (环境策略)       │       │
│   │                 │     │                 │     │                 │       │
│   │ mode: mutating  │     │ requireApproval │     │ production:     │       │
│   │ risk: medium    │     │ for: [medium,   │     │   all mutating  │       │
│   │                 │     │      high]      │     │ staging:        │       │
│   │                 │     │                 │     │   medium+high   │       │
│   └────────┬────────┘     └────────┬────────┘     └────────┬────────┘       │
│            │                       │                       │                 │
│            └───────────────────────┼───────────────────────┘                 │
│                                    │                                         │
│                                    ▼                                         │
│                        ┌─────────────────────┐                               │
│                        │ ApprovalDecision    │                               │
│                        │ needApproval: bool  │                               │
│                        │ reason: string      │                               │
│                        └─────────────────────┘                               │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 决策矩阵

```
┌─────────────────┬───────────┬────────────┬────────────┬────────────┐
│ 工具            │ 风险等级   │ dev        │ staging    │ production │
├─────────────────┼───────────┼────────────┼────────────┼────────────┤
│ get_deployment  │ low       │ 无需审批    │ 无需审批    │ 无需审批    │
│ scale_deployment│ medium    │ 无需审批    │ 需要审批    │ 需要审批    │
│ restart_pod     │ medium    │ 无需审批    │ 需要审批    │ 需要审批    │
│ delete_deployment│ high     │ 需要审批    │ 需要审批    │ 需要审批    │
└─────────────────┴───────────┴────────────┴────────────┴────────────┘
```

### 3.3 ApprovalGate 中间件

```go
// ApprovalGate 审批门禁中间件
type ApprovalGate struct {
    inner           tool.InvokableTool
    meta            common.ToolMeta
    decisionMaker   *runtime.ApprovalDecisionMaker
    summaryRenderer *SummaryRenderer
}

func (g *ApprovalGate) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
    // 1. 解析参数
    var params map[string]any
    json.Unmarshal([]byte(argumentsInJSON), &params)

    // 2. 获取运行时上下文
    rc := runtime.GetRuntimeContext(ctx)

    // 3. 判断是否需要审批
    decision, _ := g.decisionMaker.Decide(ctx, runtime.ApprovalCheckRequest{
        ToolName:    g.meta.Name,
        Scene:       rc.Scene,
        Environment: rc.Environment,
        Namespace:   getStringParam(params, "namespace"),
        Params:      params,
    })

    // 4. 不需要审批，直接执行
    if !decision.NeedApproval {
        return g.inner.InvokableRun(ctx, argumentsInJSON, opts...)
    }

    // 5. 检查是否是恢复执行
    wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
    if wasInterrupted {
        return g.handleResume(ctx, storedArgs, opts...)
    }

    // 6. 触发中断
    return g.triggerInterrupt(ctx, argumentsInJSON, decision, params)
}
```

### 3.4 审批流程图

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ tool_call   │────▶│ ApprovalGate│────▶│ Interrupt   │
│ (mutating)  │     │   检查      │     │   等待      │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ 继续执行    │◀────│   Resume    │◀────│ 用户审批    │
│ tool_result │     │   恢复      │     │ 批准/拒绝   │
└─────────────┘     └─────────────┘     └─────────────┘
```

---

## 4. 工具系统

### 4.1 ToolMeta 定义

```go
// ToolMeta 工具元数据
type ToolMeta struct {
    Name        string    `json:"name"`         // 工具名称
    DisplayName string    `json:"display_name"` // 显示名称
    Description string    `json:"description"`  // 描述
    Mode        ToolMode  `json:"mode"`         // readonly | mutating
    Risk        RiskLevel `json:"risk"`         // low | medium | high
    Category    string    `json:"category"`     // kubernetes, host, deployment...
    Tags        []string  `json:"tags"`         // 标签：用于场景匹配

    // 参数提示定义
    ParamHints map[string]ParamHint `json:"param_hints"`

    // 参数依赖关系
    ParamDependencies map[string][]string `json:"param_dependencies,omitempty"`
}
```

### 4.2 动态参数提示：混合模式

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          参数提示系统架构                                     │
│                                                                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                      ParamHint Definition                            │   │
│   │                                                                      │   │
│   │  static: [value1, value2, ...]           ← 静态枚举                  │   │
│   │  dynamic: {                             ← 动态数据源                 │   │
│   │    source: "deployments",                                            │   │
│   │    query_params: {"namespace": "{namespace}"}                        │   │
│   │  }                                                                   │   │
│   │  remote: {                             ← 远程接口                    │   │
│   │    endpoint: "/api/v1/deployments",                                  │   │
│   │    value_field: "name",                                              │   │
│   │    label_field: "display_name"                                       │   │
│   │  }                                                                   │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│                              │                                               │
│                              ▼                                               │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                      ParamHintResolver                               │   │
│   │                                                                      │   │
│   │   ┌───────────────┐ ┌───────────────┐ ┌───────────────┐            │   │
│   │   │ StaticResolver│ │DynamicResolver│ │ RemoteResolver│            │   │
│   │   └───────────────┘ └───────────────┘ └───────────────┘            │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.3 参数提示定义

```go
// ParamHint 参数提示定义
type ParamHint struct {
    Name        string `json:"name"`
    Type        string `json:"type"`         // string, integer, boolean, array
    Required    bool   `json:"required"`
    Description string `json:"description"`
    Default     any    `json:"default,omitempty"`

    // 验证规则
    Min   *int   `json:"min,omitempty"`
    Max   *int   `json:"max,omitempty"`
    Regex string `json:"regex,omitempty"`

    // 提示配置（三选一）
    StaticOptions  []StaticOption   `json:"static_options,omitempty"`
    DynamicSource  *DynamicSource   `json:"dynamic_source,omitempty"`
    RemoteEndpoint *RemoteEndpoint  `json:"remote_endpoint,omitempty"`
}

// DynamicSource 动态数据源
type DynamicSource struct {
    Source      string            `json:"source"`       // deployments, namespaces, pods...
    QueryParams map[string]string `json:"query_params"` // 查询参数模板
    ValueField  string            `json:"value_field"`
    LabelField  string            `json:"label_field"`
    DependsOn   []string          `json:"depends_on,omitempty"`
}

// RemoteEndpoint 远程接口
type RemoteEndpoint struct {
    Endpoint     string `json:"endpoint"`
    Method       string `json:"method,omitempty"`
    ValueField   string `json:"value_field"`
    LabelField   string `json:"label_field"`
    DataPath     string `json:"data_path"`
    CacheTTL     int    `json:"cache_ttl,omitempty"`
}
```

### 4.4 工具定义示例

```go
var ScaleDeploymentToolMeta = common.ToolMeta{
    Name:        "scale_deployment",
    DisplayName: "扩缩容部署",
    Description: "调整部署的副本数量",
    Mode:        common.ModeMutating,
    Risk:        common.RiskMedium,
    Category:    "kubernetes",
    Tags:        []string{"deployment", "scale"},
    ParamDependencies: map[string][]string{
        "name": {"namespace"},
    },
    ParamHints: map[string]common.ParamHint{
        "namespace": {
            Name:     "namespace",
            Type:     "string",
            Required: true,
            DynamicSource: &common.DynamicSource{
                Source:     "namespaces",
                ValueField: "name",
                LabelField: "name",
            },
        },
        "name": {
            Name:     "name",
            Type:     "string",
            Required: true,
            DynamicSource: &common.DynamicSource{
                Source:     "deployments",
                QueryParams: map[string]string{"namespace": "{namespace}"},
                ValueField: "name",
                LabelField: "display_name",
            },
            DependsOn: []string{"namespace"},
        },
        "replicas": {
            Name:        "replicas",
            Type:        "integer",
            Required:    true,
            Description: "目标副本数",
            Default:     1,
        },
    },
}
```

### 4.5 内置数据源

```go
// 注册内置数据源
r.dataSources["namespaces"] = &NamespaceDataSource{db: r.db}
r.dataSources["deployments"] = &DeploymentDataSource{db: r.db}
r.dataSources["pods"] = &PodDataSource{db: r.db}
r.dataSources["services"] = &ServiceDataSource{db: r.db}
r.dataSources["hosts"] = &HostDataSource{db: r.db}
r.dataSources["clusters"] = &ClusterDataSource{db: r.db}
r.dataSources["configmaps"] = &ConfigMapDataSource{db: r.db}
r.dataSources["secrets"] = &SecretDataSource{db: r.db}
```

---

## 5. API 层

### 5.1 接口概览

**已实现 (9个):**
- POST /ai/chat
- POST /ai/chat/stream
- POST /ai/resume
- POST /ai/resume/stream
- GET /ai/sessions
- GET /ai/sessions/:id
- DELETE /ai/sessions/:id
- POST /ai/feedback
- GET /ai/knowledge

**缺失 (12个):**
- GET /ai/capabilities
- GET /ai/tools/:name/params/hints
- POST /ai/tools/preview
- POST /ai/tools/execute
- GET /ai/executions/:id
- POST /ai/approvals
- GET /ai/approvals
- GET /ai/approvals/:id
- POST /ai/approvals/:id/approve
- POST /ai/approvals/:id/reject
- GET /ai/scene/:scene/tools
- GET /ai/scene/:scene/prompts

### 5.2 实现优先级

| 优先级 | 接口组 | 说明 |
|--------|--------|------|
| **P0** | 审批流程 | approve/reject + approvals CRUD |
| **P0** | 工具执行 | execute + executions |
| **P1** | 能力查询 | capabilities + hints |
| **P2** | 场景相关 | scene tools + prompts |

### 5.3 SSE Event Types

```
event: meta           - 会话元信息
event: delta          - 文本内容片段
event: thinking_delta - 思考过程
event: tool_call      - 工具调用
event: tool_result    - 工具结果
event: approval_required - 审批请求
event: done           - 完成
event: error          - 错误
```

---

## 6. Storage Layer

### 6.1 Redis Schema

```
ai:checkpoint:{session_id}     -> Checkpoint JSON    (TTL: 24h)
ai:execution:{exec_id}         -> Execution State    (TTL: 1h)
ai:approval:{approval_id}      -> Approval State     (TTL: 24h)
ai:scene:config:{scene}        -> SceneConfig JSON   (TTL: 5min)
```

### 6.2 MySQL Tables

```sql
-- AI 会话表
CREATE TABLE ai_sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    scene VARCHAR(50),
    title VARCHAR(255),
    messages JSON,
    turns JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_scene (user_id, scene)
);

-- AI 审批表
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

-- AI 工具执行表
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

-- AI 场景配置表
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

---

## 7. Error Handling

| 错误场景 | 处理方式 |
|---------|---------|
| Redis 不可用 | 降级到内存 Checkpoint Store |
| LLM 超时 | 返回 timeout 错误，提示用户重试 |
| 工具执行失败 | 返回 tool_result error，LLM 可以重试 |
| 审批超时 | 标记审批 expired，需要重新发起 |
| 场景配置缺失 | 使用默认配置 |
| 数据源查询失败 | 返回空选项列表，记录日志 |

---

## 8. API Layer

### 8.1 接口分组与优先级

```
P0 (阻塞前端):
├── 审批流程 (5个)
│   ├── POST /ai/approvals          - 创建审批
│   ├── GET /ai/approvals           - 审批列表
│   ├── GET /ai/approvals/:id       - 审批详情
│   ├── POST /ai/approvals/:id/approve - 批准审批
│   └── POST /ai/approvals/:id/reject - 拒绝审批
│
└── 工具执行 (2个)
    ├── POST /ai/tools/execute      - 执行工具
    └── GET /ai/executions/:id      - 执行状态

P1 (核心功能):
└── 能力查询 (2个)
    ├── GET /ai/capabilities        - 工具能力列表
    └── GET /ai/tools/:name/params/hints - 参数提示

P2 (体验优化):
└── 场景相关 (3个)
    ├── GET /ai/scene/:scene/tools   - 场景工具
    ├── GET /ai/scene/:scene/prompts - 场景提示词
    └── POST /ai/tools/preview       - 工具预览
```

### 8.2 审批接口核心流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         审批流程                                             │
│                                                                              │
│  1. 创建审批                                                                 │
│     POST /ai/approvals                                                       │
│     ├── 获取 ToolMeta                                                        │
│     ├── 判断是否需要审批                                                     │
│     ├── 生成审批记录 (MySQL)                                                 │
│     └── 存储临时状态 (Redis)                                                 │
│                                                                              │
│  2. 用户审批                                                                 │
│     POST /ai/approvals/:id/approve                                           │
│     ├── 检查审批状态和过期时间                                               │
│     ├── 更新审批状态 (MySQL)                                                 │
│     ├── 创建执行记录 (MySQL)                                                 │
│     └── 异步调用 Runtime.Resume()                                            │
│                                                                              │
│  3. Agent 恢复执行                                                           │
│     Runtime.Resume()                                                         │
│     ├── 获取 ApprovalResult                                                  │
│     └── 继续工具执行                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 8.3 工具执行接口核心流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         工具执行                                             │
│                                                                              │
│  POST /ai/tools/execute                                                      │
│  ├── 获取工具元数据                                                          │
│  ├── 判断是否需要审批                                                        │
│  │   ├── 需要: 验证 checkpoint_id                                            │
│  │   └── 不需要: 直接执行                                                    │
│  ├── 创建执行记录                                                            │
│  └── 异步执行工具                                                            │
│                                                                              │
│  GET /ai/executions/:id                                                      │
│  ├── 优先从 Redis 查询（快速）                                               │
│  └── Redis 未命中则从 MySQL 查询                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 8.4 SSE 事件协议

```
前端期望格式 (@ant-design/x-sdk):

event: delta
data: {"content": "正在查询..."}

event: tool_call
data: {"tool": "get_deployment", "params": {...}}

event: approval_required
data: {"id": "...", "summary": "...", ...}

event: done
data: {}

event: error
data: {"message": "..."}
```

### 8.5 前端适配器

```typescript
// PlatformChatProvider 适配后端 SSE 格式
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
  }
}
```

---

## 9. 前后端对齐决议

### 9.1 审批 API 路径

**决议：两者都提供**

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        API 分层设计                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  审批管理 API (CRUD)                                                    │
│  ├── POST /ai/approvals              # 创建审批                         │
│  ├── GET /ai/approvals               # 审批列表                         │
│  ├── GET /ai/approvals/:id           # 审批详情                         │
│  ├── POST /ai/approvals/:id/approve  # 批准 (返回 execution_id)         │
│  └── POST /ai/approvals/:id/reject   # 拒绝                             │
│                                                                         │
│  Agent 恢复 API (执行流)                                                │
│  ├── POST /ai/resume/step            # 非流式恢复                       │
│  └── POST /ai/resume/step/stream     # SSE 流式恢复                     │
│                                                                         │
│  执行查询 API                                                           │
│  └── GET /ai/executions/:id          # 执行状态                         │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘

场景 1: 聊天窗口内确认 (SSE 流式)
────────────────────────────────────
用户收到 approval_required 事件 → POST /ai/resume/step/stream → SSE 流继续

场景 2: 审批中心确认 (异步执行)
────────────────────────────────────
用户打开审批中心 → POST /ai/approvals/:id/approve → 异步执行 → 轮询结果
```

### 9.2 SSE 事件类型

**决议：统一到 ThoughtChain 模型**

放弃 Turn/Block 模型，使用 ThoughtChain 统一驱动进度展示和消息内容渲染。

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        ThoughtChain 统一模型                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ThoughtStage (执行阶段)                                                 │
│  ├── key: rewrite | plan | execute | user_action | summary              │
│  ├── status: loading | success | error | abort                          │
│  ├── title: 阶段标题                                                     │
│  ├── description: 阶段描述                                               │
│  ├── content: 阶段内容（计划文本、工具结果等）                            │
│  └── details: ThoughtStageDetailItem[] (execute 阶段的工具调用列表)      │
│                                                                         │
│  ThoughtStageDetail (执行详情 - 每个工具调用)                            │
│  ├── id: step-id                                                        │
│  ├── label: 工具名称                                                     │
│  ├── status: loading | success | error                                  │
│  ├── content: 结果摘要                                                   │
│  └── data: { tool, params, result }                                     │
│                                                                         │
│  阶段内容承载:                                                           │
│  ├── rewrite    → 改写后的用户问题                                       │
│  ├── plan       → 计划步骤列表                                           │
│  ├── execute    → 工具调用详情（details 数组）                           │
│  ├── user_action→ 审批/澄清内容                                          │
│  └── summary    → 最终回复文本（通过 delta 事件增量）                     │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**SSE 事件列表（简化后）:**

| 事件类型 | 用途 | 数据结构 |
|----------|------|----------|
| `meta` | 会话元信息 | `{ session_id }` |
| `stage_delta` | 阶段状态更新 | `{ stage, status, description, content }` |
| `step_update` | 步骤/工具状态更新 | `{ step_id, label, status, content, data }` |
| `approval_required` | 需要审批 | `{ id, checkpoint_id, tool_name, summary, params }` |
| `delta` | 文本内容增量 | `{ content_chunk }` |
| `thinking_delta` | 思考过程增量 | `{ content_chunk }` |
| `done` | 完成 | `{ session_id }` |
| `error` | 错误 | `{ stage, message }` |

**后端事件转换器:**

```go
// SSEEventConverter 将 Eino 输出转换为 ThoughtChain 事件
type SSEEventConverter struct {
    sessionID string
    stages    map[string]*ThoughtStage
    steps     map[string]*ThoughtStep
}

func (c *SSEEventConverter) OnPlanCreated(plan *Plan) SSEEvent {
    return SSEEvent{
        Type: "stage_delta",
        Data: map[string]any{
            "stage":       "plan",
            "status":      "success",
            "description": fmt.Sprintf("生成 %d 个步骤", len(plan.Steps)),
            "content":     formatPlanContent(plan),
        },
    }
}

func (c *SSEEventConverter) OnToolCallStart(toolName string, params map[string]any) SSEEvent {
    stepID := fmt.Sprintf("step-%d", len(c.steps)+1)
    return SSEEvent{
        Type: "step_update",
        Data: map[string]any{
            "step_id": stepID,
            "label":   toolName,
            "status":  "loading",
            "content": fmt.Sprintf("正在执行 %s...", toolName),
        },
    }
}

func (c *SSEEventConverter) OnToolResult(toolName string, result ToolResult) SSEEvent {
    return SSEEvent{
        Type: "step_update",
        Data: map[string]any{
            "step_id": c.findStepByTool(toolName),
            "label":   toolName,
            "status":  result.OK ? "success" : "error",
            "content": result.Summary,
            "data":    map[string]any{"result": result},
        },
    }
}

func (c *SSEEventConverter) OnApprovalRequired(approval *ApprovalInfo) []SSEEvent {
    return []SSEEvent{
        {Type: "stage_delta", Data: map[string]any{
            "stage": "user_action", "status": "loading",
            "description": "等待确认", "content": approval.Summary,
        }},
        {Type: "approval_required", Data: map[string]any{
            "id": approval.ID, "checkpoint_id": approval.CheckpointID,
            "tool_name": approval.ToolName, "summary": approval.Summary,
        }},
    }
}

func (c *SSEEventConverter) OnTextDelta(chunk string) SSEEvent {
    return SSEEvent{Type: "delta", Data: map[string]any{"content_chunk": chunk}}
}
```

**ThoughtChain UI 设计规范:**

详见 `specs/ai-detailed-tech-spec.md` 第 9.7.3 节，包含:
- 整体布局设计
- 阶段卡片设计（状态图标、颜色、交互）
- 各阶段详细设计（rewrite, plan, execute, user_action, summary）
- 推荐操作区域
- 动画效果
- 响应式设计和暗色模式

**关键交互:**
- 阶段执行中 → 卡片自动展开
- 阶段完成后 → 卡片自动折叠
- 用户可手动展开/折叠

### 9.3 场景 Key 格式

**决议：混合方案（后端一级存储 + 子场景规则）**

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        场景处理架构                                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  前端传递: "deployment:clusters" (两级场景)                              │
│                                                                         │
│  后端处理:                                                               │
│  1. 解析场景: domain="deployment", sub="clusters"                       │
│  2. 加载 domain 场景配置（数据库或硬编码）                                │
│  3. 根据 sub 应用子场景规则（硬编码）:                                    │
│     - clusters: 集群工具白名单，集群约束                                  │
│     - hosts: 主机工具白名单，执行命令强制审批                             │
│     - metrics: 只读工具，无变更操作                                       │
│  4. 注入到 Planner/Executor Prompt                                      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**子场景规则定义:**

```go
type SubSceneRule struct {
    IncludeTools     []string  // 工具白名单
    ExcludeTools     []string  // 工具黑名单
    ExtraConstraints []string  // 额外约束
    ApprovalAdjust   *ApprovalAdjustment
}

var subSceneRules = map[string]SubSceneRule{
    "clusters": {
        IncludeTools: []string{"get_cluster_info", "list_nodes", "upgrade_cluster"},
        ExcludeTools: []string{"execute_host_command"},
        ExtraConstraints: []string{"集群操作需要明确指定目标集群"},
    },
    "hosts": {
        IncludeTools: []string{"list_hosts", "get_host_info", "execute_host_command"},
        ApprovalAdjust: &ApprovalAdjustment{ForceApprovalFor: []string{"execute_host_command"}},
    },
    "metrics": {
        IncludeTools: []string{"get_cluster_info", "list_pods", "query_alerts"},
        ExcludeTools: []string{"scale_deployment", "restart_deployment", "execute_host_command"},
    },
}
```

### 9.4 审批标识符

**决议：前端适配后端，使用 checkpoint_id**

```typescript
// 前端类型定义修改
export interface AIInterruptApprovalResponse {
  checkpoint_id: string;  // 使用 Eino 的 checkpoint_id
  approved: boolean;
  reason?: string;
}

// approval_required 事件数据
interface ApprovalRequiredEvent {
  id: string;
  checkpoint_id: string;  // 包含 checkpoint_id
  tool_name: string;
  tool_display_name: string;
  risk_level: string;
  summary: string;
  params: Record<string, any>;
}
```

**后端处理:**

```go
// 中断时返回 checkpoint_id
func (g *ApprovalGate) triggerInterrupt(ctx context.Context, ...) {
    checkpointID := tool.GetCheckpointID(ctx)
    g.sendEvent("approval_required", map[string]any{
        "id":              approvalID,
        "checkpoint_id":   checkpointID,
        "tool_name":       g.meta.Name,
        // ...
    })
}

// 恢复时使用 checkpoint_id
func (h *Handler) ResumeStepStream(ctx context.Context, req ResumeStepRequest) {
    params := &adk.ResumeParams{
        Input: map[string]any{"approved": req.Approved, "reason": req.Reason},
    }
    h.orchestrator.ResumeWithParams(ctx, req.CheckpointID, params)
}
```

### 9.5 审批恢复方式

**决议：两种都支持（SSE 流式 + 异步执行）**

| 方式 | 场景 | API | 返回 |
|------|------|-----|------|
| **SSE 流式** | 聊天窗口确认 | `POST /ai/resume/step/stream` | SSE 事件流 |
| **异步执行** | 审批中心确认 | `POST /ai/approvals/:id/approve` | execution_id |

---

## 10. Testing Strategy

1. **单元测试**
   - Runtime 接口实现
   - ApprovalGate 中间件
   - Checkpoint Store
   - 场景配置解析
   - 审批决策器
   - 参数提示解析器

2. **集成测试**
   - SSE 流式对话
   - Interrupt/Resume 流程
   - 审批流程端到端
   - 场景上下文注入

3. **E2E 测试**
   - 前端对话功能
   - 审批 UI 交互
   - 参数提示级联查询
