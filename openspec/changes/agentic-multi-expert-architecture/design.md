# Design: Agentic Multi-Expert Architecture

## 1. 架构原则与运行时语义

### 1.1 运行时分层

四层架构在运行时中分为两个层次：

1. **阶段型 Agent**: Planner / Executor / Summarizer
2. **能力型 Agent**: HostOpsExpert / K8sExpert / ServiceExpert / DeliveryExpert / ObservabilityExpert

其中阶段型 Agent 负责流程推进，能力型 Agent 负责领域执行。阶段型 Agent 不直接暴露平台领域工具，能力型 Agent 通过 `AgentAsTool` 暴露给 Executor 调用。

```text
HTTP/SSE Gateway
    -> AI Facade / Orchestrator
        -> Planner Agent
        -> Executor Agent
            -> AgentAsTool(HostOpsExpert)
            -> AgentAsTool(K8sExpert)
            -> AgentAsTool(ServiceExpert)
            -> AgentAsTool(DeliveryExpert)
            -> AgentAsTool(ObservabilityExpert)
        -> Summarizer Agent
```

### 1.2 ADK 协作语义

该方案统一采用 Eino ADK 的两种协作语义：

- `Transfer`: 用于阶段间流转，保留上游对话上下文
- `ToolCall`: 用于 Executor 将专家作为工具调用，输入为新的结构化任务描述

对应关系如下：

```text
Planner --transfer--> Executor --toolcall--> Experts --transfer--> Summarizer
```

约束：

- Planner 不直接调用领域执行工具
- Executor 不持有平台工具，仅持有专家目录和执行控制能力
- Expert 持有本领域工具，通过 `AgentAsTool` 被 Executor 调用
- Summarizer 不回查平台工具，只消费 Planner/Executor/Expert 的结构化结果

### 1.3 确定性与模型自治边界

为了避免编排层再次退化为“大一统 ReAct Agent”，明确以下边界：

- **模型负责**: 意图理解、资源消歧、任务分解、专家内工具选择、结果总结
- **代码负责**: DAG 调度、重试、超时、审批、中断恢复、SSE 编排、状态持久化

因此：

- Planner 生成结构化计划
- Executor 以 Go 代码执行 DAG 调度
- Expert 内部允许使用 ToolCalling Model 自主选择工具
- Summarizer 生成最终答复和补充调查判断

## 2. 核心接口设计

### 2.1 Planner Agent

```go
// internal/ai/planner/planner.go

package planner

import (
    "context"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/schema"
)

// PlannerConfig Planner 配置
type PlannerConfig struct {
    Model           model.ToolCallingChatModel
    ExpertRegistry  ExpertRegistry
    MaxIterations   int
}

// Planner Agent 负责意图理解、资源解析、权限检查和计划制定
type Planner struct {
    agent    *react.Agent
    registry ExpertRegistry
    tools    []tool.BaseTool
}

// PlannerOutput Planner 输出
type PlannerOutput struct {
    // 需要用户澄清
    NeedsClarification bool              `json:"needs_clarification"`
    ClarificationMsg   string            `json:"clarification_msg,omitempty"`

    // 可以执行的计划
    Plan               *ExecutionPlan    `json:"plan,omitempty"`

    // 流式输出内容
    StreamingContent   string            `json:"streaming_content"`
}

// ExecutionPlan 执行计划
type ExecutionPlan struct {
    ID          string          `json:"id"`
    Goal        string          `json:"goal"`
    Resolved    ResolvedResources `json:"resolved"`
    Assumptions []string        `json:"assumptions,omitempty"`
    Steps       []PlanStep      `json:"steps"`
    StopConditions []string     `json:"stop_conditions,omitempty"`
    Permission  PermissionStatus `json:"permission"`
    TraceID     string          `json:"trace_id,omitempty"`
}

// ResolvedResources 已解析的资源
type ResolvedResources struct {
    ServiceID   *uint64 `json:"service_id,omitempty"`
    ServiceName string  `json:"service_name,omitempty"`
    ClusterID   *uint64 `json:"cluster_id,omitempty"`
    ClusterName string  `json:"cluster_name,omitempty"`
    HostIDs     []uint64 `json:"host_ids,omitempty"`
    Namespace   string  `json:"namespace,omitempty"`
}

// PlanStep 计划步骤
type PlanStep struct {
    ID          string   `json:"id"`
    Expert      string   `json:"expert"`
    Intent      string   `json:"intent"`
    Task        string   `json:"task"`
    Input       map[string]any `json:"input,omitempty"`
    DependsOn   []string `json:"depends_on,omitempty"`
    Context     map[string]any `json:"context,omitempty"`
    RetryPolicy RetryPolicy `json:"retry_policy,omitempty"`
    TimeoutSec  int      `json:"timeout_sec,omitempty"`
}

type RetryPolicy struct {
    MaxRetry      int    `json:"max_retry"`
    BackoffPolicy string `json:"backoff_policy,omitempty"`
}

// PermissionStatus 权限状态
type PermissionStatus struct {
    Allowed bool   `json:"allowed"`
    Reason  string `json:"reason,omitempty"`
}

// NewPlanner 创建 Planner 实例
func NewPlanner(ctx context.Context, cfg *PlannerConfig) (*Planner, error) {
    // 构建 Planner 专用工具
    tools := buildPlannerTools(cfg.ExpertRegistry)

    agent, err := react.NewAgent(ctx, &react.AgentConfig{
        ToolCallingModel: cfg.Model,
        ToolsConfig: compose.ToolsNodeConfig{
            Tools: tools,
        },
        MaxStep: 5, // Planner 最多调用 5 次工具
    })
    if err != nil {
        return nil, err
    }

    return &Planner{
        agent:    agent,
        registry: cfg.ExpertRegistry,
        tools:    tools,
    }, nil
}

// Stream 流式执行规划
func (p *Planner) Stream(ctx context.Context, req *PlannerRequest, emit func(event string, payload map[string]any) bool) (*PlannerOutput, error) {
    // 构建消息
    messages := p.buildMessages(req)

    // 流式调用
    stream, err := p.agent.Stream(ctx, messages)
    if err != nil {
        return nil, err
    }
    defer stream.Close()

    output := &PlannerOutput{}
    var fullContent strings.Builder

    for {
        chunk, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, err
        }

        if chunk.Content != "" {
            fullContent.WriteString(chunk.Content)
            if emit != nil {
                emit("delta", map[string]any{"contentChunk": chunk.Content})
            }
        }
    }

    // 解析输出
    output = p.parseOutput(fullContent.String())

    return output, nil
}

// parseOutput 解析 Planner 输出
func (p *Planner) parseOutput(content string) *PlannerOutput {
    // 检查是否需要澄清
    if strings.Contains(content, "[NEEDS_CLARIFICATION]") {
        return &PlannerOutput{
            NeedsClarification: true,
            ClarificationMsg:   extractClarification(content),
            StreamingContent:   content,
        }
    }

    // 解析执行计划
    plan := p.parsePlan(content)
    if plan != nil {
        return &PlannerOutput{
            NeedsClarification: false,
            Plan:               plan,
            StreamingContent:   content,
        }
    }

    // 默认：直接回复
    return &PlannerOutput{
        NeedsClarification: false,
        StreamingContent:   content,
    }
}
```

Planner 的输出采用“双通道”：

- 面向用户：保留流式自然语言内容，通过 `delta` 事件发送
- 面向系统：输出严格结构化的 `ExecutionPlan` 或 `ClarificationResponse`

系统不得依赖自然语言正文中的隐式语义驱动 Executor。真正进入执行阶段的数据必须是结构化对象。

建议 Planner 输出契约如下：

```go
type ClarificationResponse struct {
    NeedClarification bool     `json:"need_clarification"`
    Message           string   `json:"message"`
    Candidates        []string `json:"candidates,omitempty"`
}
```

Planner 最终输出建议统一为以下四类决策之一：

```go
type PlannerDecisionType string

const (
    PlannerDecisionClarify PlannerDecisionType = "clarify"
    PlannerDecisionReject  PlannerDecisionType = "reject"
    PlannerDecisionPlan    PlannerDecisionType = "plan"
    PlannerDecisionDirect  PlannerDecisionType = "direct_reply"
)

type PlannerDecision struct {
    Type       PlannerDecisionType `json:"type"`
    Clarify    *ClarificationResponse `json:"clarify,omitempty"`
    Plan       *ExecutionPlan      `json:"plan,omitempty"`
    DirectText string              `json:"direct_text,omitempty"`
    RejectReason string            `json:"reject_reason,omitempty"`
}
```

补充约束：

- `ExecutionPlan.ID`、`PlanStep.ID` 必须稳定且可追踪，不使用临时序号作为唯一标识
- `PlanStep.Intent` 建议收敛为有限集合：`investigate`、`verify`、`mutate`、`collect_evidence`、`summarize_context`
- `ResolvedResources` 只存放已确认的资源标识；推断、假设类信息进入 `Assumptions`
- `PlanStep.Input` 仅存放该 step 的最小必要输入，避免重复复制整份上下文
- 对用户可见的自然语言说明不得作为 Executor 的执行依据

### 2.1.1 Planner 实现级接口草案

```go
// internal/ai/planner/types.go

package planner

import "context"

type EmitFn func(event string, payload map[string]any) bool

type Request struct {
    SessionID      string         `json:"session_id"`
    Message        string         `json:"message"`
    UserID         uint64         `json:"user_id"`
    RuntimeContext map[string]any `json:"runtime_context,omitempty"`
    Iteration      int            `json:"iteration,omitempty"`
    PriorPlanHints []string       `json:"prior_plan_hints,omitempty"`
}

type Response struct {
    Decision *PlannerDecision `json:"decision"`
    Content  string           `json:"content,omitempty"`
}

type Interface interface {
    Plan(ctx context.Context, req *Request, emit EmitFn) (*Response, error)
}
```

实现约束：

- `Plan(...)` 是 Planner 唯一对外入口
- `Response.Decision` 是 Orchestrator 的唯一结构化输入
- `Content` 仅保留用户可见说明，不参与执行判断
- Planner 不向上层暴露内部 tools/model/prompt 细节

### 2.2 Planner 工具实现

```go
// internal/ai/planner/tools.go

package planner

import (
    "context"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/tool/utils"
)

// buildPlannerTools 构建 Planner 专用工具
func buildPlannerTools(registry ExpertRegistry) []tool.BaseTool {
    var tools []tool.BaseTool

    // resolve_service: 服务名称解析
    tools = append(tools, utils.NewTool(
        &schema.ToolInfo{
            Name: "resolve_service",
            Desc: "根据服务名称或关键词解析服务ID。支持模糊匹配，返回匹配的服务列表。",
        },
        resolveServiceTool,
    ))

    // resolve_cluster: 集群名称解析
    tools = append(tools, utils.NewTool(
        &schema.ToolInfo{
            Name: "resolve_cluster",
            Desc: "根据集群名称或环境解析集群ID。",
        },
        resolveClusterTool,
    ))

    // resolve_host: 主机名称解析
    tools = append(tools, utils.NewTool(
        &schema.ToolInfo{
            Name: "resolve_host",
            Desc: "根据主机名、IP或关键词解析主机ID。",
        },
        resolveHostTool,
    ))

    // check_permission: 权限检查
    tools = append(tools, utils.NewTool(
        &schema.ToolInfo{
            Name: "check_permission",
            Desc: "检查当前用户对指定资源的操作权限。",
        },
        checkPermissionTool,
    ))

    // get_user_context: 获取用户上下文
    tools = append(tools, utils.NewTool(
        &schema.ToolInfo{
            Name: "get_user_context",
            Desc: "获取用户当前上下文信息，包括当前页面、选中的资源等。",
        },
        getUserContextTool,
    ))

    return tools
}

// ResolveServiceInput 服务解析输入
type ResolveServiceInput struct {
    Keyword   string `json:"keyword" jsonschema:"description=服务名称或关键词"`
    Env       string `json:"env,omitempty" jsonschema:"description=环境过滤(prod/staging/dev)"`
    Exact     bool   `json:"exact,omitempty" jsonschema:"description=是否精确匹配"`
}

// ResolveServiceOutput 服务解析输出
type ResolveServiceOutput struct {
    Found    bool           `json:"found"`
    Services []ServiceMatch `json:"services,omitempty"`
    Message  string         `json:"message,omitempty"`
}

type ServiceMatch struct {
    ID          uint64 `json:"id"`
    Name        string `json:"name"`
    Environment string `json:"environment"`
    Status      string `json:"status"`
}

func resolveServiceTool(ctx context.Context, input ResolveServiceInput) (*ResolveServiceOutput, error) {
    // 从 context 获取 DB
    db := getDBFromContext(ctx)

    var services []model.Service
    query := db.Model(&model.Service{})

    if input.Exact {
        query = query.Where("name = ?", input.Keyword)
    } else {
        query = query.Where("name LIKE ?", "%"+input.Keyword+"%")
    }

    if input.Env != "" {
        query = query.Where("environment = ?", input.Env)
    }

    if err := query.Limit(10).Find(&services).Error; err != nil {
        return nil, err
    }

    if len(services) == 0 {
        return &ResolveServiceOutput{
            Found:   false,
            Message: fmt.Sprintf("未找到匹配 '%s' 的服务", input.Keyword),
        }, nil
    }

    matches := make([]ServiceMatch, len(services))
    for i, s := range services {
        matches[i] = ServiceMatch{
            ID:          s.ID,
            Name:        s.Name,
            Environment: s.Environment,
            Status:      s.Status,
        }
    }

    return &ResolveServiceOutput{
        Found:    true,
        Services: matches,
        Message:  fmt.Sprintf("找到 %d 个匹配的服务", len(services)),
    }, nil
}
```

### 2.2.1 Resolve 与 Inventory 分层

Planner 的 `resolve_service`、`resolve_cluster`、`resolve_host` 不应直接复制底层查询逻辑，而应建立两层结构：

```text
Planner
  -> resolve_service
       -> service_list_inventory
  -> resolve_cluster
       -> cluster_list_inventory
  -> resolve_host
       -> host_list_inventory
```

职责边界：

- `*_list_inventory`：负责列出候选，不负责最终选择
- `resolve_*`：负责结合上下文做候选收敛、歧义判断和最终选择
- Planner：根据 `resolve_*` 的结果决定是继续规划、要求澄清还是拒绝执行

建议统一协议如下：

```go
type ResolveStatus string

const (
    ResolveExact     ResolveStatus = "exact"
    ResolveAmbiguous ResolveStatus = "ambiguous"
    ResolveMissing   ResolveStatus = "missing"
)

type ResolveCandidate struct {
    ID       uint64         `json:"id"`
    Name     string         `json:"name"`
    Score    float64        `json:"score,omitempty"`
    Reason   string         `json:"reason,omitempty"`
    Metadata map[string]any `json:"metadata,omitempty"`
}

type ResolveResult struct {
    Status     ResolveStatus      `json:"status"`
    Resource   string             `json:"resource"`
    Selected   *ResolveCandidate  `json:"selected,omitempty"`
    Candidates []ResolveCandidate `json:"candidates,omitempty"`
    Message    string             `json:"message,omitempty"`
}
```

Resolve 行为约束：

- `exact`：直接写入 `ResolvedResources`
- `ambiguous`：Planner 必须返回澄清，不进入 Executor
- `missing`：关键资源缺失时澄清或拒绝；非关键资源可进入 `Assumptions`

推荐优先级：

1. 用户当前页面和选中资源
2. 用户显式指定的环境、集群、命名空间
3. 精确匹配
4. 模糊匹配

补充约束：

- 模糊匹配命中多个候选时，Planner 不得擅自选择
- inventory 负责“列”，resolve 负责“选”，两者不能互相越界
- resolve 的评分和过滤逻辑应统一实现，避免散落到多个 expert 中

### 2.2.1.1 Resolve 评分与排序规则

为了避免不同资源类型各自实现“拍脑袋匹配”，建议统一采用分项加权评分：

```go
type ResolveScore struct {
    ExactNameMatch     float64 `json:"exact_name_match"`
    PrefixMatch        float64 `json:"prefix_match"`
    FuzzyMatch         float64 `json:"fuzzy_match"`
    EnvMatch           float64 `json:"env_match"`
    NamespaceMatch     float64 `json:"namespace_match,omitempty"`
    SelectedContextHit float64 `json:"selected_context_hit"`
    SceneHintMatch     float64 `json:"scene_hint_match,omitempty"`
    RecencyHint        float64 `json:"recency_hint,omitempty"`
    Total              float64 `json:"total"`
}
```

推荐默认权重：

- 精确名称命中：`1.00`
- 前缀命中：`0.80`
- 模糊命中：`0.40 - 0.70`
- 环境命中：`+0.20`
- 命名空间命中：`+0.10`
- 当前页面已选资源命中：`+0.40`
- 场景提示命中：`+0.10`

约束：

- `SelectedContextHit` 只作为加分，不可单独覆盖完全不匹配的名称
- `EnvMatch` 和 `NamespaceMatch` 只能在名称已有基础匹配时生效
- `RecencyHint` 仅用于排序，不得改变 exact/ambiguous/missing 判定

### 2.2.1.2 Resolve 判定阈值

建议将最终判定标准显式化：

```text
1. 无候选:
   -> missing

2. Top1 分数 >= 0.90 且 Top1 - Top2 >= 0.20:
   -> exact

3. Top1 分数 >= 0.75 且 Top1 - Top2 < 0.20:
   -> ambiguous

4. Top1 分数 < 0.75 且存在候选:
   -> ambiguous
```

补充规则：

- 当用户明确传入 `Exact=true` 且未命中精确名称时，直接 `missing`
- 当当前页面已选资源与 top1 完全一致时，可降低 `exact` 阈值到 `0.80`
- 当候选数 > 10 时，应先截断为 TopN 再进入澄清流程，避免一次返回过多候选

### 2.2.1.3 Resolve 澄清规则

澄清不是“把所有候选都甩给用户”，而应有收敛策略：

```go
type ClarificationCandidate struct {
    ID      uint64 `json:"id"`
    Label   string `json:"label"`
    Reason  string `json:"reason,omitempty"`
    Env     string `json:"env,omitempty"`
    Status  string `json:"status,omitempty"`
}
```

建议：

- 默认最多返回 `3-5` 个候选
- 候选展示优先包含：名称、环境、状态、命中原因
- 候选排序按 `TotalScore` 降序
- 如同名跨环境冲突，优先用环境信息做澄清，而不是展示内部 ID

推荐澄清文案策略：

- 同名跨环境：
  “找到多个同名资源，请确认环境”
- 同环境多实例：
  “找到多个相似资源，请确认完整名称”
- 无高置信候选：
  “未能准确识别资源，请补充环境、命名空间或更完整名称”

### 2.2.1.4 各资源类型差异规则

虽然 resolve 框架统一，但各资源仍需补充差异规则：

#### `resolve_service`

- 名称匹配优先级最高
- `env` 是主过滤条件
- 当前页面若已在服务详情页，`SelectedServiceID` 应优先参与加分
- `namespace` 只作为辅助信息，不应覆盖服务名称判断

#### `resolve_cluster`

- `env` 和 cluster 名称同等重要
- 如果当前页面已选 cluster，则应显著加分
- 对 prod/staging/dev 别名应统一规范化后再参与匹配

#### `resolve_host`

- 主机名、IP、别名都可作为候选名称
- IP 精确匹配可直接视为高置信命中
- 若 `scope=node` 或来自 K8s 场景，可优先提升与当前 cluster 关联主机的分数

### 2.2.1.5 Resolve 输出增强

建议在 `ResolveCandidate.Metadata` 中补充可解释字段，方便 Planner 做澄清和日志记录：

```go
type ResolveCandidateMeta struct {
    Env        string `json:"env,omitempty"`
    Namespace  string `json:"namespace,omitempty"`
    Status     string `json:"status,omitempty"`
    MatchField string `json:"match_field,omitempty"` // name | alias | ip | selected_context
    ScoreNote  string `json:"score_note,omitempty"`
}
```

这样 Planner 不仅知道“选中了谁”，还知道“为什么选中它”。

### 2.2.2 Planner 工具文件划分

建议将 Planner 工具层拆成 4 个文件，分别承载装配、解析、上下文和权限逻辑：

```text
internal/ai/planner/
  tools.go         # buildPlannerTools 装配入口
  resolve.go       # resolve_service / resolve_cluster / resolve_host
  context.go       # get_user_context
  permission.go    # check_permission
```

职责约束：

- `tools.go` 只负责工具注册，不放业务逻辑
- `resolve.go` 只负责资源解析与候选收敛
- `context.go` 只负责运行时上下文抽取与标准化
- `permission.go` 只负责权限预检查，不发起审批

### 2.2.3 Resolve 工具接口草案

建议三个 resolve 工具共享统一模式，只在资源类型和 inventory 来源上区分：

```go
// internal/ai/planner/resolve.go

package planner

import "context"

type ResolveServiceInput struct {
    Keyword   string `json:"keyword"`
    Env       string `json:"env,omitempty"`
    Exact     bool   `json:"exact,omitempty"`
    Namespace string `json:"namespace,omitempty"`
}

type ResolveClusterInput struct {
    Keyword string `json:"keyword"`
    Env     string `json:"env,omitempty"`
    Exact   bool   `json:"exact,omitempty"`
}

type ResolveHostInput struct {
    Keyword string `json:"keyword"`
    Exact   bool   `json:"exact,omitempty"`
    Scope   string `json:"scope,omitempty"`
}

type Resolver interface {
    ResolveService(ctx context.Context, in *ResolveServiceInput) (*ResolveResult, error)
    ResolveCluster(ctx context.Context, in *ResolveClusterInput) (*ResolveResult, error)
    ResolveHost(ctx context.Context, in *ResolveHostInput) (*ResolveResult, error)
}
```

实现约束：

- `ResolveService` 复用 `service_list_inventory`
- `ResolveCluster` 复用 `cluster_list_inventory`
- `ResolveHost` 复用 `host_list_inventory`
- 三者统一返回 `ResolveResult`
- 三者都必须消费 `UserContext` 提供的候选过滤条件

### 2.2.4 UserContext 工具接口草案

`get_user_context` 不应把网关原始上下文直接暴露给 Planner，而应做一次标准化：

```go
// internal/ai/planner/context.go

package planner

import "context"

type UserContext struct {
    CurrentPage       string         `json:"current_page,omitempty"`
    SelectedServiceID *uint64        `json:"selected_service_id,omitempty"`
    SelectedClusterID *uint64        `json:"selected_cluster_id,omitempty"`
    SelectedHostIDs   []uint64       `json:"selected_host_ids,omitempty"`
    DefaultEnv        string         `json:"default_env,omitempty"`
    DefaultNamespace  string         `json:"default_namespace,omitempty"`
    Raw               map[string]any `json:"raw,omitempty"`
}

type UserContextProvider interface {
    GetUserContext(ctx context.Context) (*UserContext, error)
}
```

实现约束：

- `Raw` 仅用于保留无法标准化的上下文，不作为 Planner 直接判断依据
- `CurrentPage`、选中资源、默认环境和默认命名空间必须优先标准化
- `get_user_context` 必须稳定返回，即使没有上下文也返回空对象而不是错误

### 2.2.5 权限检查工具接口草案

`check_permission` 是 Planner 的预检查工具，不等于审批流本身：

```go
// internal/ai/planner/permission.go

package planner

import "context"

type PermissionCheckInput struct {
    ResourceType string         `json:"resource_type"`
    ResourceID   string         `json:"resource_id,omitempty"`
    Action       string         `json:"action"`
    Context      map[string]any `json:"context,omitempty"`
}

type PermissionCheckResult struct {
    Allowed bool     `json:"allowed"`
    Reason  string   `json:"reason,omitempty"`
    Missing []string `json:"missing,omitempty"`
}

type PermissionChecker interface {
    CheckPermission(ctx context.Context, in *PermissionCheckInput) (*PermissionCheckResult, error)
}
```

实现约束：

- `Allowed=false` 时必须返回 `Reason`
- `check_permission` 只做“是否允许进入执行链路”的预判断
- 对高风险变更，Planner 可以提示“执行阶段可能触发审批”，但不能在该阶段生成审批 token

### 2.2.6 Planner 工具调用顺序建议

Planner 内部建议按固定顺序使用工具，而不是完全放任模型随机调用：

```text
1. get_user_context
2. resolve_service / resolve_cluster / resolve_host
3. check_permission
4. 生成 PlannerDecision
```

补充约束：

- 如 `UserContext` 已有明确选中资源，应优先使用它缩小 resolve 范围
- 如多个资源都需解析，优先解析主资源，再解析从属资源
- `check_permission` 必须基于已解析出的稳定资源标识执行，而不是基于模糊名称执行

### 2.3 Executor Agent

```go
// internal/ai/executor/executor.go

package executor

import (
    "context"
    "sync"
    "time"
)

// ExecutorConfig Executor 配置
type ExecutorConfig struct {
    ExpertRegistry  ExpertRegistry
    MaxRetry        int
    ExpertTimeout   time.Duration
}

// Executor Agent 负责解析计划、调度专家、收集结果
type Executor struct {
    registry ExpertRegistry
    maxRetry int
    timeout  time.Duration
}

// ExecutionResult 执行结果
type ExecutionResult struct {
    PlanID       string                 `json:"plan_id"`
    Success      bool                   `json:"success"`
    StepResults  map[string]ExpertResult   `json:"step_results"`
    Errors       []StepError            `json:"errors,omitempty"`
    Duration     time.Duration          `json:"duration"`
}

// ExpertResult 专家执行结果
type ExpertResult struct {
    ExpertName string        `json:"expert_name"`
    Success    bool          `json:"success"`
    Output     string        `json:"output"`
    ToolCalls  []ToolCallInfo `json:"tool_calls,omitempty"`
    Duration   time.Duration `json:"duration"`
    Error      string        `json:"error,omitempty"`
}

// StepError 步骤错误
type StepError struct {
    StepID   string `json:"step_id"`
    Expert   string `json:"expert"`
    Error    string `json:"error"`
    Retry    int    `json:"retry"`
}

// Execute 执行计划
func (e *Executor) Execute(ctx context.Context, plan *ExecutionPlan, emit func(event string, payload map[string]any) bool) (*ExecutionResult, error) {
    start := time.Now()
    result := &ExecutionResult{
        PlanID:      plan.ID,
        StepResults: make(map[int]ExpertResult),
    }

    // 构建依赖图
    dag := e.buildDAG(plan.Steps)

    // 按依赖顺序执行
    for batch := range dag.TopologicalBatches() {
        // 并行执行同一 batch 的步骤
        var wg sync.WaitGroup
        var mu sync.Mutex

        for _, step := range batch {
            wg.Add(1)
            go func(s PlanStep) {
                defer wg.Done()

                // 发送步骤开始事件
                if emit != nil {
                    emit("step_start", map[string]any{
                        "step_id": s.ID,
                        "expert": s.Expert,
                    })
                    emit("expert_progress", map[string]any{
                        "expert": s.Expert,
                        "status": "running",
                    })
                }

                // 执行专家调用（带重试）
                expertResult := e.executeStepWithRetry(ctx, s, plan, result.StepResults)

                mu.Lock()
                result.StepResults[s.ID] = expertResult
                if !expertResult.Success {
                    result.Errors = append(result.Errors, StepError{
                        StepID: s.ID,
                        Expert: s.Expert,
                        Error:  expertResult.Error,
                    })
                }
                mu.Unlock()

                // 发送步骤完成事件
                if emit != nil {
                    emit("expert_progress", map[string]any{
                        "expert":     s.Expert,
                        "status":     "done",
                        "duration_ms": expertResult.Duration.Milliseconds(),
                    })
                }
            }(step)
        }

        wg.Wait()
    }

    result.Duration = time.Since(start)
    result.Success = len(result.Errors) == 0

    return result, nil
}

// executeStepWithRetry 带重试的步骤执行
func (e *Executor) executeStepWithRetry(ctx context.Context, step PlanStep, plan *ExecutionPlan, previousResults map[string]ExpertResult) ExpertResult {
    var lastErr error

    for retry := 0; retry <= e.maxRetry; retry++ {
        // 构建专家输入
        expertInput := e.buildExpertInput(step, plan, previousResults)

        // 调用专家
        result := e.callExpert(ctx, step.Expert, expertInput)

        if result.Success {
            return result
        }

        lastErr = fmt.Errorf(result.Error)

        // 如果是可重试的错误，等待后重试
        if e.isRetryable(result.Error) {
            time.Sleep(time.Second * time.Duration(retry+1))
            continue
        }

        break
    }

    return ExpertResult{
        ExpertName: step.Expert,
        Success:    false,
        Error:      lastErr.Error(),
    }
}

// buildDAG 构建依赖图
func (e *Executor) buildDAG(steps []PlanStep) *DAG {
    dag := NewDAG()

    for _, step := range steps {
        dag.AddNode(step.ID, step)

        for _, dep := range step.DependsOn {
            dag.AddEdge(dep, step.ID)
        }
    }

    return dag
}
```

Executor 的核心不是“理解计划”，而是“确定性执行计划”。它应实现一个代码驱动的 DAG 调度器：

1. 根据 `PlanStep.ID` 和 `DependsOn` 构建依赖图
2. 将所有入度为 0 的步骤并发执行
3. 某一步完成后释放其后继节点
4. 某一步失败后根据 `RetryPolicy` 决定重试或阻断下游
5. 将所有 `StepResult` 汇总给 Summarizer

专家调用输入统一为：

```go
type ExpertTaskInput struct {
    Goal        string         `json:"goal"`
    Task        string         `json:"task"`
    Resources   map[string]any `json:"resources,omitempty"`
    Context     map[string]any `json:"context,omitempty"`
    Constraints []string       `json:"constraints,omitempty"`
}
```

专家返回统一为：

```go
type StepResult struct {
    StepID     string         `json:"step_id"`
    Expert     string         `json:"expert"`
    OK         bool           `json:"ok"`
    Summary    string         `json:"summary"`
    Evidence   []Evidence     `json:"evidence,omitempty"`
    RawOutput  map[string]any `json:"raw_output,omitempty"`
    Error      *StepError     `json:"error,omitempty"`
}

type Evidence struct {
    Type    string `json:"type"`
    Ref     string `json:"ref,omitempty"`
    Content string `json:"content"`
}
```

建议补充统一错误模型：

```go
type StepError struct {
    Code        string `json:"code"`
    Message     string `json:"message"`
    Recoverable bool   `json:"recoverable"`
    Retryable   bool   `json:"retryable"`
    Cause       string `json:"cause,omitempty"`
    Suggestion  string `json:"suggestion,omitempty"`
}
```

推荐错误码集合：

- `invalid_input`
- `resource_not_found`
- `permission_denied`
- `approval_required`
- `tool_execution_failed`
- `dependency_blocked`
- `timeout`
- `incomplete_evidence`

### 2.3.1 Executor 状态机

Executor 除了 DAG 调度外，还需要统一的 step 状态机：

```go
type StepState string

const (
    StepPending         StepState = "pending"
    StepReady           StepState = "ready"
    StepRunning         StepState = "running"
    StepWaitingApproval StepState = "waiting_approval"
    StepRetrying        StepState = "retrying"
    StepBlocked         StepState = "blocked"
    StepFailed          StepState = "failed"
    StepCompleted       StepState = "completed"
    StepCancelled       StepState = "cancelled"
)
```

状态流转约束：

- 初始创建后为 `pending`
- 依赖满足后进入 `ready`
- 被调度执行时进入 `running`
- 命中审批/确认时进入 `waiting_approval`
- 可重试错误进入 `retrying`
- 上游失败且该 step 强依赖上游结果时进入 `blocked`
- 达到成功条件进入 `completed`
- 不可恢复错误进入 `failed`
- 会话取消、总超时、用户中断时进入 `cancelled`

Executor 需要显式定义以下行为：

- `blocked` 的 step 不允许隐式自动恢复，除非 Planner 重新规划
- `waiting_approval` 恢复后，仅重放当前 step，而不是整条计划
- 总超时触发时，所有 `pending/ready/running/retrying` step 必须统一转为 `cancelled`

### 2.3.4 Executor 实现级接口草案

```go
// internal/ai/executor/types.go

package executor

import "context"

type EmitFn func(event string, payload map[string]any) bool

type Request struct {
    SessionID string         `json:"session_id"`
    UserID    uint64         `json:"user_id"`
    Plan      *ExecutionPlan `json:"plan"`
    TraceID   string         `json:"trace_id,omitempty"`
}

type Result struct {
    PlanID      string                `json:"plan_id"`
    Success     bool                  `json:"success"`
    StepStates  map[string]StepState  `json:"step_states"`
    StepResults map[string]StepResult `json:"step_results"`
    Errors      []StepError           `json:"errors,omitempty"`
}

type ResumeRequest struct {
    SessionID string               `json:"session_id"`
    UserID    uint64               `json:"user_id"`
    Approval  *ApprovalResumeState `json:"approval"`
}

type Interface interface {
    Execute(ctx context.Context, req *Request, emit EmitFn) (*Result, error)
    Resume(ctx context.Context, req *ResumeRequest, emit EmitFn) (*Result, error)
}
```

实现约束：

- `Execute(...)` 用于正常计划执行
- `Resume(...)` 仅用于审批/中断恢复
- Executor 自己负责并发、状态流转、审批等待、blocked 依赖和超时
- Orchestrator 不直接控制某个 step 的内部重试和状态变更

### 2.3.2 审批与恢复模型

审批模型建议固定在 Executor 层收口：

- Planner 只做权限预检查，不触发实际审批
- Expert 内部工具调用触发审批时，必须向 Executor 返回结构化 `approval_required`
- Executor 将对应 step 转为 `waiting_approval`
- 审批通过后，Executor 从该 `step_id` 恢复执行
- 审批拒绝后，step 标记为 `failed` 或 `cancelled`，由 Summarizer 负责解释

建议审批恢复上下文最少包含：

```go
type ApprovalResumeState struct {
    PlanID      string         `json:"plan_id"`
    StepID      string         `json:"step_id"`
    Expert      string         `json:"expert"`
    TaskInput   ExpertTaskInput `json:"task_input"`
    LastAttempt int            `json:"last_attempt"`
    ApprovalToken string       `json:"approval_token,omitempty"`
}
```

### 2.3.3 Expert 输出约束

为了避免不同 expert 风格漂移，建议补充以下硬约束：

- 每个 expert 结果必须先返回结构化 `StepResult`
- `Summary` 不得为空，且应可直接给 Summarizer 消费
- `Evidence` 数量建议限制在 `0-10`
- 原始日志、事件列表等大体量数据进入 `RawOutput`，不直接塞入 `Summary`
- 多个 tool 调用的中间结果由 expert 内部归并，Executor 不负责语义合并

### 2.3.5 工具元数据与风险分级

旧工具迁入多专家架构后，不能只靠工具名和 prompt 判断是否需要审批。当前代码里已经存在 `ToolModeReadonly/ToolModeMutating`、`ToolRiskLow/Medium/High` 以及 wrapper 语义：

- `low`：直接执行
- `medium`：进入 review/edit
- `high`：进入 approval

因此新设计应在现有实现口径上扩展，而不是另起一套不兼容的枚举。

建议为所有可暴露给 Expert 的工具补充统一元数据，至少覆盖能力类型、风险级别、幂等性和审批策略。

建议最小元数据结构如下：

```go
type ToolMode string

const (
    ToolModeReadonly ToolMode = "readonly"
    ToolModeMutating ToolMode = "mutating"
)

type ApprovalPolicy string

const (
    ApprovalNever    ApprovalPolicy = "never"
    ApprovalOnDemand ApprovalPolicy = "on_demand"
    ApprovalAlways   ApprovalPolicy = "always"
)

type ToolMeta struct {
    Name             string         `json:"name"`
    Domain           string         `json:"domain"`
    Mode             ToolMode       `json:"mode"`
    Risk             ToolRisk       `json:"risk"`
    Idempotent       bool           `json:"idempotent"`
    ApprovalPolicy   ApprovalPolicy `json:"approval_policy"`
    MutatesResources bool           `json:"mutates_resources"`
    ProducesEvidence bool           `json:"produces_evidence"`
}
```

约束如下：

- `read` 类工具默认应为 `Idempotent=true`
- `write` 类工具必须显式声明 `Risk` 和 `ApprovalPolicy`
- `MutatesResources=true` 的工具不得暴露给 Planner
- `ProducesEvidence=true` 的工具结果允许被 Expert 摘要后进入 `Evidence`

### 2.3.6 工具风险矩阵与审批触发

建议把审批触发逻辑从 prompt 中抽离出来，统一在 Executor/tool middleware 层做确定性判断，并与当前 wrapper 行为对齐：

| 模式 | 风险 | 示例 | 默认审批策略 |
|------|------|------|--------------|
| `readonly` | `low` | `service_status`, `monitor_metric_query`, `deployment_target_detail` | `never` |
| `readonly` | `medium` | `k8s_logs`, `k8s_get_pod_logs`, `host_exec`, `host_ssh_exec_readonly`, `os_get_journal_tail` | `on_demand` |
| `mutating` | `medium` | `job_run`, `host_batch_status_update` | `on_demand` |
| `mutating` | `high` | `service_deploy`, `service_deploy_apply`, `cicd_pipeline_trigger`, `host_batch`, `host_batch_exec_apply` | `always` |

审批触发规则建议定死：

1. `ApprovalAlways` 工具在真正执行前必须进入 `waiting_approval`
2. `ApprovalOnDemand` 工具在命中生产环境、批量目标或高影响面资源时进入 `waiting_approval`
3. `ApprovalNever` 工具不得生成审批 token
4. 同一个 `step_id` 内如果包含多个写操作，按最高风险工具决定审批级别
5. 任何未声明 `ToolMeta` 的 write 类工具不得接入新专家架构

建议补充一个确定性审批判定结果：

```go
type ApprovalDecision struct {
    Required bool      `json:"required"`
    Reason   string    `json:"reason,omitempty"`
    Risk     ToolRisk  `json:"risk,omitempty"`
    Scope    string    `json:"scope,omitempty"`
}
```

其中：

- `Reason` 用于 SSE 与最终解释
- `Scope` 用于表达 `single_resource / batch / cross_env / production`
- `Risk` 必须与工具元数据和运行时输入一致

需要补充一个与现有 wrapper 对齐的约束：

- `ToolRiskMedium` 默认进入 review/edit，而不是直接生成 approval token
- `ToolRiskHigh` 默认进入 approval
- `ApprovalPolicy` 是对现有 `ToolRisk` 的运行时补充，不替代基础风险等级

### 2.3.7 幂等性、副作用与重试约束

Executor 的重试能力只有在工具契约明确时才安全。建议为工具调用补充以下硬规则：

- `Idempotent=true` 的 read/write 工具才允许自动重试
- `Idempotent=false` 的 write 工具在失败后不得由 Executor 自动重放，除非显式获得恢复令牌或工具自己支持去重键
- `waiting_approval` 恢复后，只允许重放当前被阻断的 tool call，不允许重复执行已成功的副作用步骤
- 批量执行类工具必须支持 `preview -> approval -> apply` 三段式，禁止 Planner/Executor 直接把“预览”和“执行”混成一步
- 对外部系统有副作用的工具应支持 `request_id` / `dedupe_key`，以便网络抖动或中断恢复时避免重复执行

建议在工具执行结果中显式暴露副作用信息：

```go
type ToolExecutionReceipt struct {
    ToolName    string   `json:"tool_name"`
    RequestID   string   `json:"request_id,omitempty"`
    Mutated     bool     `json:"mutated"`
    Targets     []string `json:"targets,omitempty"`
    SideEffects []string `json:"side_effects,omitempty"`
}
```

Executor 与 Expert 的责任边界：

- Expert 负责根据工具回执归并最终 `StepResult`
- Executor 负责基于 `Idempotent/MutatesResources/ApprovalPolicy` 控制重试和恢复
- Summarizer 只消费副作用摘要，不直接解释底层重试细节

### 2.3.8 现有工具到风险策略的建议映射

基于当前代码中的 `ToolMode` / `ToolRisk`，建议先按下面这张表收口。实现阶段可以直接据此补 `ToolMeta` 和 expert 装配。

#### HostOpsExpert

| 工具 | 模式 | 风险 | 审批/复核策略 | 备注 |
|------|------|------|---------------|------|
| `os_get_cpu_mem` | `readonly` | `low` | `never` | 证据采集 |
| `os_get_disk_fs` | `readonly` | `low` | `never` | 证据采集 |
| `os_get_net_stat` | `readonly` | `low` | `never` | 证据采集 |
| `os_get_process_top` | `readonly` | `low` | `never` | 证据采集 |
| `os_get_journal_tail` | `readonly` | `medium` | `on_demand` | 日志可能包含敏感信息 |
| `os_get_container_runtime` | `readonly` | `low` | `never` | 证据采集 |
| `host_exec` | `readonly` | `medium` | `on_demand` | 单机只读命令 |
| `host_ssh_exec_readonly` | `readonly` | `medium` | `on_demand` | 远程只读命令 |
| `host_batch_exec_preview` | `readonly` | `medium` | `on_demand` | 批量动作预览 |
| `host_batch_exec_apply` | `mutating` | `high` | `always` | 批量执行 |
| `host_batch_status_update` | `mutating` | `medium` | `on_demand` | 状态变更 |
| `host_batch` | `mutating` | `high` | `always` | 旧兼容工具，建议迁移期后删除 |

#### K8sExpert

| 工具 | 模式 | 风险 | 审批/复核策略 | 备注 |
|------|------|------|---------------|------|
| `k8s_query` | `readonly` | `low` | `never` | 资源查询 |
| `k8s_list_resources` | `readonly` | `low` | `never` | 资源枚举 |
| `k8s_events` | `readonly` | `low` | `never` | 事件查询 |
| `k8s_get_events` | `readonly` | `low` | `never` | 定向事件查询 |
| `k8s_logs` | `readonly` | `medium` | `on_demand` | 可能包含敏感日志 |
| `k8s_get_pod_logs` | `readonly` | `medium` | `on_demand` | 可能包含敏感日志 |

#### ServiceExpert

| 工具 | 模式 | 风险 | 审批/复核策略 | 备注 |
|------|------|------|---------------|------|
| `service_status` | `readonly` | `low` | `never` | 服务状态 |
| `service_get_detail` | `readonly` | `low` | `never` | 服务详情 |
| `service_catalog_list` | `readonly` | `low` | `never` | 目录查询 |
| `service_category_tree` | `readonly` | `low` | `never` | 分类树 |
| `service_visibility_check` | `readonly` | `low` | `never` | 可见性检查 |
| `deployment_target_list` | `readonly` | `low` | `never` | 目标列表 |
| `deployment_target_detail` | `readonly` | `low` | `never` | 目标详情 |
| `deployment_bootstrap_status` | `readonly` | `low` | `never` | 引导状态 |
| `credential_list` | `readonly` | `low` | `never` | 只返回摘要 |
| `credential_test` | `readonly` | `low` | `never` | 连通性测试结果 |
| `config_app_list` | `readonly` | `low` | `never` | 配置应用列表 |
| `config_item_get` | `readonly` | `low` | `never` | 配置读取，需脱敏 |
| `config_diff` | `readonly` | `low` | `never` | 配置差异 |
| `service_deploy_preview` | `readonly` | `medium` | `on_demand` | 变更预览 |
| `service_deploy` | `mutating` | `high` | `always` | 旧入口，建议最终收敛到 preview/apply |
| `service_deploy_apply` | `mutating` | `high` | `always` | 正式执行 |

#### DeliveryExpert

| 工具 | 模式 | 风险 | 审批/复核策略 | 备注 |
|------|------|------|---------------|------|
| `cicd_pipeline_list` | `readonly` | `low` | `never` | 流水线列表 |
| `cicd_pipeline_status` | `readonly` | `low` | `never` | 流水线状态 |
| `job_list` | `readonly` | `low` | `never` | 任务列表 |
| `job_execution_status` | `readonly` | `low` | `never` | 执行状态 |
| `job_run` | `mutating` | `medium` | `on_demand` | 任务触发 |
| `cicd_pipeline_trigger` | `mutating` | `high` | `always` | 流水线触发 |

#### ObservabilityExpert

| 工具 | 模式 | 风险 | 审批/复核策略 | 备注 |
|------|------|------|---------------|------|
| `monitor_alert_rule_list` | `readonly` | `low` | `never` | 告警规则 |
| `monitor_alert` | `readonly` | `low` | `never` | 告警查询 |
| `monitor_alert_active` | `readonly` | `low` | `never` | 活跃告警 |
| `monitor_metric` | `readonly` | `low` | `never` | 指标摘要 |
| `monitor_metric_query` | `readonly` | `low` | `never` | 指标查询 |
| `topology_get` | `readonly` | `low` | `never` | 依赖关系 |
| `audit_log_search` | `readonly` | `low` | `never` | 审计检索 |

#### Planner Support Tools

| 工具 | 模式 | 风险 | 用途 |
|------|------|------|------|
| `host_list_inventory` | `readonly` | `low` | 资源解析 |
| `cluster_list_inventory` | `readonly` | `low` | 资源解析 |
| `service_list_inventory` | `readonly` | `low` | 资源解析 |
| `permission_check` | `readonly` | `low` | 权限预检查 |
| `user_list` | `readonly` | `low` | 用户候选补全 |
| `role_list` | `readonly` | `low` | 角色候选补全 |

补充约束：

- `Planner` 不得直接持有任何 `mutating` 工具
- `medium` 风险的只读工具在生产环境或跨租户查询时可以提升为 `on_demand`
- `service_deploy`、`host_batch` 这类旧单入口 mutating 工具仅作为迁移兼容，最终应被显式的 preview/apply 拆分取代

### 2.4 Domain Expert

推荐采用 5 个 execution experts，而不是 4 个粗粒度专家。原因是现有工具已经自然分成 5 个稳定职责簇：

| Expert | 职责 | 工具前缀/归属 |
|--------|------|--------------|
| `HostOpsExpert` | 主机、OS、SSH、批处理运维 | `os_*`, `host_*` |
| `K8sExpert` | Kubernetes 资源、事件、日志、集群资源查询 | `k8s_*` |
| `ServiceExpert` | 服务对象、部署目标、凭证、配置项 | `service_*`, `deployment_*`, `credential_*`, `config_*` |
| `DeliveryExpert` | 流水线与任务执行链路 | `cicd_*`, `job_*` |
| `ObservabilityExpert` | 监控、告警、拓扑、审计证据 | `monitor_*`, `topology_get`, `audit_log_search` |

Planner 支撑工具不纳入 execution expert：

- `host_list_inventory`
- `service_list_inventory`
- `cluster_list_inventory`
- `user_list`
- `role_list`
- `permission_check`

这些工具更适合用于资源解析、权限预检查和候选项补全，而不是作为领域执行专家的主工具集。`cluster_list_inventory` 虽然服务于 K8s 场景，但在新架构中仍应归 Planner 统一做资源解析，而不是挂在 K8sExpert 下直接执行。

```go
// internal/ai/experts/registry.go

package experts

import (
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/flow/agent/react"
)

// ExpertConfig 专家配置
type ExpertConfig struct {
    Name        string
    Description string
    Tools       []tool.BaseTool
    Prompt      string
    Model       model.ToolCallingChatModel
    MaxStep     int
}

// Expert 领域专家
type Expert struct {
    Name        string
    Description string
    Agent       *react.Agent
    Tools       []tool.BaseTool
}

// ExpertRegistry 专家注册表
type ExpertRegistry struct {
    experts map[string]*Expert
}

// NewExpertRegistry 创建专家注册表
func NewExpertRegistry() *ExpertRegistry {
    return &ExpertRegistry{
        experts: make(map[string]*Expert),
    }
}

// Register 注册专家
func (r *ExpertRegistry) Register(expert *Expert) {
    r.experts[expert.Name] = expert
}

// Get 获取专家
func (r *ExpertRegistry) Get(name string) (*Expert, bool) {
    exp, ok := r.experts[name]
    return exp, ok
}

// List 列出所有专家
func (r *ExpertRegistry) List() []*Expert {
    result := make([]*Expert, 0, len(r.experts))
    for _, exp := range r.experts {
        result = append(result, exp)
    }
    return result
}

// ExpertCatalog 专家目录（注入 Planner prompt）
func (r *ExpertRegistry) ExpertCatalog() string {
    var b strings.Builder
    b.WriteString("## 可用专家\n\n")

    for name, exp := range r.experts {
        b.WriteString(fmt.Sprintf("### %s\n", name))
        b.WriteString(fmt.Sprintf("%s\n\n", exp.Description))
    }

    return b.String()
}
```

领域专家应统一抽象为可被 `AgentAsTool` 导出的能力单元。建议补充如下接口：

```go
type ExpertAgent interface {
    Name() string
    Description() string
    AsTool(ctx context.Context) tool.BaseTool
}
```

工具隔离原则调整为：

- 领域执行工具只属于对应 Expert
- 公共能力可以通过 helper、middleware、RAG 检索器、审计组件共享
- 公共能力默认不直接暴露给 Planner/Executor，避免重新出现大工具池

推荐的职责边界如下：

#### HostOpsExpert

- 聚焦主机资源、进程、系统日志、容器运行时、SSH 只读命令、批处理执行
- 不负责 Kubernetes 资源语义判断
- 当用户问题明显落在节点 OS 层时优先由该专家处理

#### K8sExpert

- 聚焦 Pod/Deployment/Service/Node 查询、事件、日志
- 接收已解析的 `cluster_id/namespace/label/pod_name`
- 不承担服务发布、流水线、凭证、配置差异分析

#### ServiceExpert

- 聚焦服务对象本身及其交付目标
- 负责服务详情、状态、部署目标、凭证、配置项和配置差异
- 当问题围绕“这个服务是什么、部署到哪、依赖什么配置和凭证”时由该专家主导

#### DeliveryExpert

- 聚焦发布链路、流水线、任务执行与触发
- 负责回答“为什么这次发布没成功”“最近谁触发了哪个任务”“流水线卡在哪”
- 不直接承载监控与拓扑分析

#### ObservabilityExpert

- 聚焦症状与证据收集
- 负责监控指标、告警、拓扑关系、审计记录
- 作为排障主证据专家，常与 HostOpsExpert、K8sExpert、ServiceExpert 协同

建议的协同模式：

```text
服务异常排查:
Planner
  -> ServiceExpert        # 服务对象、部署目标、配置/凭证
  -> K8sExpert            # Pod/事件/日志
  -> ObservabilityExpert  # 指标/告警/拓扑/审计
  -> HostOpsExpert        # 如怀疑节点/主机层故障

发布失败排查:
Planner
  -> DeliveryExpert       # 流水线/任务
  -> ServiceExpert        # 服务目标、凭证、配置
  -> K8sExpert            # 发布后资源状态
  -> ObservabilityExpert  # 告警与审计侧证据
```

### 2.4.1 Expert Prompt Contract

所有 execution experts 的 prompt 建议遵循统一 contract，避免风格漂移和职责串线：

```text
你是 {ExpertName}。

你的职责：
- ...

你只能处理：
- ...

你不能处理：
- ...

你可用工具：
- ...

工作原则：
1. ...
2. ...
3. ...

输出要求：
- 先返回结构化结论
- 再列关键证据
- 不输出与职责无关的猜测
```

每个 expert 的 prompt 至少要明确以下 6 段：

1. 角色定位
2. 能力范围
3. 禁止事项
4. 可用工具
5. 决策原则
6. 输出格式

#### HostOpsExpert Prompt Contract

- 角色：主机与 OS 运维专家
- 只能处理：CPU/内存/磁盘/网络/进程/systemd 日志/容器运行时/SSH 只读命令/批处理
- 不能处理：K8s workload 语义、服务配置、流水线分析
- 决策原则：
  - 先查状态，再查日志，再给建议
  - 涉及批量操作必须等待审批
  - 怀疑 K8s 资源层故障时应建议联动 `K8sExpert`

#### K8sExpert Prompt Contract

- 角色：Kubernetes 资源与运行态专家
- 只能处理：Pod/Deployment/Service/Node 查询、events、pod logs、resource 状态
- 不能处理：主机 OS 诊断、配置差异、流水线执行
- 决策原则：
  - 先看 workload 状态
  - 再看 events
  - 再看 logs
  - 怀疑节点问题时建议联动 `HostOpsExpert`
  - 怀疑服务配置问题时建议联动 `ServiceExpert`

#### ServiceExpert Prompt Contract

- 角色：服务对象与交付目标专家
- 只能处理：服务详情、状态、部署目标、凭证、配置项、配置差异
- 不能处理：流水线执行细节、监控指标分析、Pod 运行态根因
- 决策原则：
  - 先确认服务对象和目标环境
  - 再确认部署目标、凭证、配置
  - 发现交付链路问题时建议联动 `DeliveryExpert`
  - 发现运行态问题时建议联动 `K8sExpert` 或 `ObservabilityExpert`

#### DeliveryExpert Prompt Contract

- 角色：交付与流水线专家
- 只能处理：`cicd_*`、`job_*`、发布流程、任务执行状态
- 不能处理：服务配置内容、监控根因、Pod/Node 运行态诊断
- 决策原则：
  - 先确定最近一次 pipeline/job
  - 再查看状态、失败点和执行记录
  - 输入资源错误时建议回到 `ServiceExpert`
  - 发布后运行失败时建议联动 `K8sExpert` / `ObservabilityExpert`

#### ObservabilityExpert Prompt Contract

- 角色：观测与证据专家
- 只能处理：监控指标、告警、拓扑、审计记录、异常时间线
- 不能处理：直接执行变更、配置内容校验、流水线分析
- 决策原则：
  - 先确定时间窗口
  - 再查 metrics / alerts
  - 再补 topology / audit
  - 如果只有症状没有根因证据，必须明确标记“不足以定因”

### 2.4.2 Expert 输出 Contract

建议每个 expert 的最终结论统一收敛为：

```go
type ExpertConclusion struct {
    Summary     string     `json:"summary"`
    Findings    []string   `json:"findings,omitempty"`
    Evidence    []Evidence `json:"evidence,omitempty"`
    Risks       []string   `json:"risks,omitempty"`
    NextActions []string   `json:"next_actions,omitempty"`
    EscalateTo  []string   `json:"escalate_to,omitempty"`
}
```

其中：

- `EscalateTo` 用于显式表达建议联动的专家
- `Summary` 应可直接被 Summarizer 消费
- `Findings` 应面向结构化归纳，不应简单复制原始日志

### 2.4.3 Planner 的 Expert Selection Policy

Planner 不应每次从零自由选择专家，建议遵循基础选择策略：

```text
单服务运行态异常:
  ServiceExpert + K8sExpert + ObservabilityExpert

主机或节点异常:
  HostOpsExpert + ObservabilityExpert

发布失败:
  DeliveryExpert + ServiceExpert + K8sExpert

权限/审计/影响面问题:
  ObservabilityExpert

配置/凭证/目标环境问题:
  ServiceExpert
```

补充约束：

- `mutate` 类 step 默认只允许 `ServiceExpert` 或 `DeliveryExpert` 产出
- `collect_evidence` 类 step 优先使用 `ObservabilityExpert`、`K8sExpert`、`HostOpsExpert`
- 当 Planner 无法在单个 expert 内闭合任务时，应显式拆分多个 steps，而不是把复合任务塞给一个 expert

### 2.4.4 Expert 实现级骨架

建议所有 execution experts 采用统一骨架，避免实现风格发散：

```go
// internal/ai/experts/shared/types.go

package experts

import (
    "context"

    "github.com/cloudwego/eino/components/tool"
)

type TaskInput struct {
    Goal        string         `json:"goal"`
    Task        string         `json:"task"`
    Resources   map[string]any `json:"resources,omitempty"`
    Context     map[string]any `json:"context,omitempty"`
    Constraints []string       `json:"constraints,omitempty"`
}

type TaskOutput struct {
    Conclusion *ExpertConclusion `json:"conclusion"`
}

type Interface interface {
    Name() string
    Description() string
    AsTool(ctx context.Context) tool.BaseTool
}
```

每个 expert 的 `expert.go` 最小模板：

```go
// internal/ai/experts/<name>/expert.go

package <name>

import (
    "context"

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cy77cc/k8s-manage/internal/ai/experts"
)

type Expert struct {
    agent adk.Agent
    tools []tool.BaseTool
}

func New(ctx context.Context, m model.ToolCallingChatModel, tools []tool.BaseTool) (*Expert, error) {
    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        Name,
        Description: Description,
        Instruction: Prompt,
        Model:       m,
        ToolsConfig: adk.ToolsConfig{
            Tools: tools,
        },
    })
    if err != nil {
        return nil, err
    }
    return &Expert{agent: agent, tools: tools}, nil
}

func (e *Expert) Name() string        { return Name }
func (e *Expert) Description() string { return Description }
func (e *Expert) AsTool(ctx context.Context) tool.BaseTool {
    return adk.NewAgentTool(ctx, e.agent)
}

var _ experts.Interface = (*Expert)(nil)
```

每个 expert 的 `prompt.go` 最小模板：

```go
// internal/ai/experts/<name>/prompt.go

package <name>

const (
    Name        = "<ExpertName>"
    Description = "<一句话职责描述>"
)

const Prompt = `
你是 <ExpertName>。

你的职责：
- ...

你只能处理：
- ...

你不能处理：
- ...

你可用工具：
- ...

工作原则：
1. ...
2. ...
3. ...

输出要求：
- 先返回结构化结论
- 再列关键证据
- 不输出与职责无关的猜测
`
```

建议目录结构统一为：

```text
internal/ai/experts/
  hostops/
    expert.go
    prompt.go
    tools.go
  k8s/
    expert.go
    prompt.go
    tools.go
  service/
    expert.go
    prompt.go
    tools.go
  delivery/
    expert.go
    prompt.go
    tools.go
  observability/
    expert.go
    prompt.go
    tools.go
```

### 2.5 Summarizer Agent

Summarizer 负责消费结构化计划与执行结果，并输出最终回答及完整性判断：

```go
type SummarizerInput struct {
    UserGoal        string           `json:"user_goal"`
    Plan            *ExecutionPlan   `json:"plan"`
    Results         []StepResult     `json:"results"`
    Iteration       int              `json:"iteration"`
    PriorConclusion string           `json:"prior_conclusion,omitempty"`
}

type SummarizerDecision struct {
    NeedMoreInvestigation bool     `json:"need_more_investigation"`
    MissingFacts          []string `json:"missing_facts,omitempty"`
    NextPlanHints         []string `json:"next_plan_hints,omitempty"`
    CurrentConclusion     string   `json:"current_conclusion"`
    Confidence            float64  `json:"confidence"`
}
```

当 `NeedMoreInvestigation=true` 且未达到最大迭代次数时，Summarizer 将缺失事实和下一步建议交还 Planner 进行重新规划，而不是直接自己触发额外工具调用。

### 2.5.2 Summarizer 实现级接口草案

```go
// internal/ai/summarizer/types.go

package summarizer

import "context"

type EmitFn func(event string, payload map[string]any) bool

type Request struct {
    SessionID string           `json:"session_id"`
    UserID    uint64           `json:"user_id"`
    Input     *SummarizerInput `json:"input"`
}

type Response struct {
    Decision *SummarizerDecision `json:"decision"`
    Content  string              `json:"content"`
}

type Interface interface {
    Summarize(ctx context.Context, req *Request, emit EmitFn) (*Response, error)
}
```

实现约束：

- `Summarize(...)` 是 Summarizer 唯一对外入口
- `Content` 是面向用户的最终自然语言回答
- `Decision` 是面向系统的循环控制依据
- Summarizer 不直接调平台工具，也不直接操作审批状态

### 2.5.1 Summarizer 判定规则

Summarizer 的判断不应完全依赖 prompt 自由发挥，建议明确以下判定维度：

1. 关键问题是否已有直接证据支撑
2. 不同 expert 结果是否存在冲突
3. 是否仍有高优先级缺失事实
4. 当前结论是否足以支撑用户下一步操作
5. 是否存在高风险操作尚未确认

推荐规则：

- 若缺失核心资源状态、核心错误证据、核心审计线索之一，则 `NeedMoreInvestigation=true`
- 若 expert 结论互相矛盾，则 `NeedMoreInvestigation=true`
- 若仅缺少次要佐证，但已能支撑可执行结论，则允许直接输出结论，并降低 `Confidence`

## 3. 上下文与状态管理

### 3.1 上下文分层

采用 Eino ADK 的 `History` 与 `SessionValues` 分层管理上下文：

- `History`: 用户对话、Planner 摘要、Summarizer 结论
- `SessionValues`: 已解析资源、权限状态、当前 iteration、trace_id、user_context、plan_id
- `ExecutionState`: 原始工具结果、步骤状态、审批状态、checkpoint 信息

约束：

- 原始工具返回不得无上限写入 History
- Executor 使用 `ExecutionState` 承载中间状态
- Summarizer 只消费结果摘要和证据，不直接消费所有原始工具输出

### 3.2 中断恢复

需要延续现有审批与 checkpoint 机制：

- 长会话、审批中断、网络中断后可以基于 `plan_id` + `step_id` 恢复
- 被审批阻断的步骤应持久化为 `waiting_approval`
- 恢复时 Executor 从未完成步骤继续，而不是重跑整个计划

### 3.3 Gateway -> Orchestrator -> Planner 上下文注入边界

为了避免运行时上下文在多个层级重复组装或被污染，明确以下责任分层：

```text
GatewayRuntime
  -> 组装外部请求上下文
  -> 调用 Orchestrator.Run

Orchestrator
  -> 标准化 RunRequest
  -> 注入 session_id / user_id / trace_id / iteration
  -> 管理 ExecutionState 生命周期

Planner
  -> 读取 RuntimeContext
  -> 调用 get_user_context 做标准化补充
  -> 不直接写 SessionState / ExecutionState
```

推荐流向：

```text
HTTP Request
  -> GatewayRuntime.buildRuntimeContext(...)
  -> ai.RunRequest{SessionID, UserID, Message, RuntimeContext}
  -> Orchestrator.Run(...)
  -> Planner.Plan(...)
```

约束：

- `GatewayRuntime` 是唯一允许读取 HTTP/UI 原始上下文的位置
- `Orchestrator` 是唯一允许写入 `ExecutionState` 的协调层
- `Planner` 只读 `RuntimeContext` 和 `UserContext`，不直接持久化状态
- `Executor` 只写 step 级状态，不改写原始 `RuntimeContext`

### 3.4 RuntimeContext 标准结构

`RunRequest.RuntimeContext` 不应是无约束 map，建议收敛为统一结构后再注入：

```go
type RuntimeContext struct {
    Scene            string         `json:"scene,omitempty"`
    CurrentPage      string         `json:"current_page,omitempty"`
    SelectedResource map[string]any `json:"selected_resource,omitempty"`
    QueryParams      map[string]any `json:"query_params,omitempty"`
    RequestMeta      map[string]any `json:"request_meta,omitempty"`
}
```

建议来源：

- `Scene`: 当前会话或页面场景
- `CurrentPage`: 当前前端路由或功能页
- `SelectedResource`: 页面已选中的 service/cluster/host 等
- `QueryParams`: 页面筛选条件
- `RequestMeta`: trace、来源、时间戳等元信息

约束：

- `GatewayRuntime` 负责从前端请求提取并标准化 `RuntimeContext`
- `Planner` 只消费标准化后的 `RuntimeContext`
- 业务层不得把任意未审查字段直接传入 Planner prompt

### 3.5 状态读写边界

建议把状态分成三类，并明确读写者：

| 状态 | 主要内容 | 写入者 | 读取者 |
|------|----------|--------|--------|
| `SessionSnapshot` | 对话消息、基础上下文 | GatewayRuntime / Orchestrator | Planner / Summarizer |
| `ExecutionState` | plan、step state、step result、approval 状态 | Orchestrator / Executor | Orchestrator / Resume 流程 |
| `SessionValues` | trace_id、resolved resources、iteration | Orchestrator | Planner / Executor / Summarizer |

边界约束：

- `GatewayRuntime` 维护会话层存取，不直接操作 step 级状态
- `Orchestrator` 负责初始化 `ExecutionState`
- `Executor` 负责更新 `StepStates` / `StepResults` / `PendingApproval`
- `Planner` 和 `Summarizer` 默认只读，不写持久化状态

### 3.6 Gateway 接口收敛建议

建议在 `GatewayRuntime` 中新增专门面向新架构的入口，而不是继续直接耦合旧 `AIAgent`：

```go
type AIOrchestratorRuntime interface {
    Run(ctx context.Context, req *RunRequest, emit EmitFn) (*RunResult, error)
    Resume(ctx context.Context, req *ResumeRequest, emit EmitFn) (*RunResult, error)
}
```

推荐 `GatewayRuntime` 的职责收敛为：

1. 校验 session/user 基础信息
2. 组装标准化 `RuntimeContext`
3. 调用 `Orchestrator.Run(...)`
4. 将 emit 结果转发给 SSE 层
5. 处理审批后的 `Resume(...)`

不再建议由 `GatewayRuntime` 继续承载旧单体 agent 的推理逻辑。

## 4. SSE 事件设计补充

在现有事件基础上，新增更强语义的状态事件：

| 事件 | 说明 |
|------|------|
| `planner_state` | `clarifying/planning/replanning` |
| `plan_created` | 结构化计划创建完成 |
| `step_start` | 某个 step 开始 |
| `step_result` | 某个 step 的结构化结果 |
| `expert_progress` | 专家执行状态 |

其中：

- `plan_created` 必须携带完整 `ExecutionPlan`
- `step_result` 必须携带 `step_id`、`expert`、`ok`、`summary`
- 前端以 `planner_state` 和 `step_result` 作为主要 UI 驱动事件，`delta` 继续用于文本流

### 4.1 SSE Payload Schema

推荐所有事件共享统一元字段：

```go
type EventMeta struct {
    SessionID string `json:"session_id"`
    PlanID    string `json:"plan_id,omitempty"`
    TraceID   string `json:"trace_id,omitempty"`
    Iteration int    `json:"iteration,omitempty"`
    Timestamp int64  `json:"timestamp"`
}
```

关键事件 payload 建议固定如下：

```go
type PlannerStateEvent struct {
    EventMeta
    State string `json:"state"` // clarifying | planning | replanning
}

type PlanCreatedEvent struct {
    EventMeta
    Plan *ExecutionPlan `json:"plan"`
}

type StepStartEvent struct {
    EventMeta
    StepID string `json:"step_id"`
    Expert string `json:"expert"`
    Intent string `json:"intent,omitempty"`
}

type StepResultEvent struct {
    EventMeta
    StepID   string     `json:"step_id"`
    Expert   string     `json:"expert"`
    OK       bool       `json:"ok"`
    Summary  string     `json:"summary"`
    Evidence []Evidence `json:"evidence,omitempty"`
    Error    *StepError `json:"error,omitempty"`
}

type ApprovalRequiredEvent struct {
    EventMeta
    StepID  string         `json:"step_id"`
    Expert  string         `json:"expert"`
    Tool    string         `json:"tool"`
    Preview map[string]any `json:"preview,omitempty"`
    Message string         `json:"message"`
}

type ErrorEvent struct {
    EventMeta
    Scope   string     `json:"scope"` // global | step | planner | executor | summarizer
    StepID  string     `json:"step_id,omitempty"`
    Message string     `json:"message"`
    Error   *StepError `json:"error,omitempty"`
}

type DoneEvent struct {
    EventMeta
    Conclusion string  `json:"conclusion"`
    Confidence float64 `json:"confidence,omitempty"`
}
```

补充约束：

- `approval_required` 必须带 `step_id`，否则恢复链路不可靠
- `error` 事件必须区分全局错误和 step 级错误
- `done` 事件只能发送一次
- 同一个 `step_id` 必须满足 `step_start -> expert_progress -> step_result` 的顺序约束

## 4.2 Orchestrator 实现级接口草案

Orchestrator 是唯一负责串联 Planner / Executor / Summarizer 的顶层入口。它不拥有工具执行逻辑，也不拥有 expert 内部决策逻辑。

```go
// internal/ai/orchestrator/types.go

package ai

import "context"

type EmitFn func(event string, payload map[string]any) bool

type RunRequest struct {
    SessionID      string         `json:"session_id"`
    UserID         uint64         `json:"user_id"`
    Message        string         `json:"message"`
    RuntimeContext map[string]any `json:"runtime_context,omitempty"`
}

type RunResult struct {
    SessionID          string  `json:"session_id"`
    PlanID             string  `json:"plan_id,omitempty"`
    Content            string  `json:"content"`
    Success            bool    `json:"success"`
    NeedsClarification bool    `json:"needs_clarification,omitempty"`
    Iterations         int     `json:"iterations"`
    Confidence         float64 `json:"confidence,omitempty"`
}

type ResumeRequest struct {
    SessionID string `json:"session_id"`
    UserID    uint64 `json:"user_id"`
    Token     string `json:"token"`
}

type Orchestrator interface {
    Run(ctx context.Context, req *RunRequest, emit EmitFn) (*RunResult, error)
    Resume(ctx context.Context, req *ResumeRequest, emit EmitFn) (*RunResult, error)
}
```

Orchestrator 行为约束：

- `Run(...)` 只负责新会话或普通继续执行
- `Resume(...)` 只负责审批/中断后的恢复
- Orchestrator 只协调 iteration 循环，不直接操作 step 状态
- Orchestrator 统一发出 `planner_state`、`plan_created`、`done` 等全局事件

推荐时序：

```text
Run
  -> Planner.Plan
  -> if clarify/reject/direct_reply: 直接结束
  -> Executor.Execute
  -> Summarizer.Summarize
  -> if need_more_investigation: 回到 Planner.Plan
  -> done
```

### 4.2.1 Orchestrator 正常时序

```text
User
  -> GatewayRuntime
  -> Orchestrator.Run
      -> emit(meta)
      -> emit(planner_state=planning)
      -> Planner.Plan
      -> if decision=clarify:
           emit(done)
           return NeedsClarification
      -> if decision=reject/direct_reply:
           emit(done)
           return final response
      -> emit(plan_created)
      -> Executor.Execute
      -> Summarizer.Summarize
      -> if decision.need_more_investigation:
           iteration += 1
           emit(planner_state=replanning)
           goto Planner.Plan
      -> emit(done)
      -> return final response
```

### 4.2.2 澄清分支时序

```text
Run
  -> Planner.Plan
  -> decision=clarify
  -> emit(planner_state=clarifying)
  -> emit(delta/clarification text)
  -> emit(done)
  -> return {needs_clarification=true}
```

约束：

- 澄清分支不进入 Executor
- 澄清消息必须带候选项或明确补充要求
- 前端再次提交用户补充信息后，视为新的 `Run(...)`，但沿用同一 `session_id`

### 4.2.3 审批与恢复时序

```text
Run
  -> Executor.Execute
      -> Expert.AsTool
          -> tool call
          -> approval_required
      -> step_state=waiting_approval
      -> persist ExecutionState
      -> emit(approval_required)
      -> return interrupted result

Resume
  -> Orchestrator.Resume
      -> load ExecutionState by session/token
      -> Executor.Resume
      -> continue current step
      -> Summarizer.Summarize
      -> emit(done)
```

约束：

- `Resume(...)` 只能恢复一个已进入 `waiting_approval` 的 step
- 恢复后不得重跑已完成 steps
- 审批 token 无效、过期或与 `session_id/step_id` 不匹配时必须终止恢复

### 4.2.4 重规划时序

```text
Run
  -> Planner.Plan(plan v1)
  -> Executor.Execute(v1)
  -> Summarizer.Summarize
  -> decision.need_more_investigation=true
  -> carry forward:
       - prior conclusion
       - missing facts
       - next plan hints
       - existing evidence summary
  -> Planner.Plan(plan v2)
  -> Executor.Execute(v2)
  -> Summarizer.Summarize
```

约束：

- 重规划必须显式增加 `iteration`
- Planner 在重规划时应消费 `MissingFacts` 和 `NextPlanHints`
- 已有 `StepResult` 不应原样复制到新 plan 中，但应作为 Planner 的参考上下文

### 4.2.5 失败终止时序

```text
Run
  -> Planner / Executor / Summarizer
  -> if fatal error:
       emit(error)
       emit(done)
       return failure result
```

fatal error 包括但不限于：

- 总超时
- 超过最大迭代次数
- 无可用 expert
- 持久化状态损坏且无法恢复
- 必要审批信息缺失

约束：

- fatal error 必须带 `scope`
- 若错误发生在 step 级别且可归因，应同时带 `step_id`
- `done` 必须在最终 `error` 之后发出一次，用于结束前端流

## 4.3 运行时配置模型

为支持灰度、按专家调参和多模型策略，建议配置模型补充为：

```go
type RuntimeConfig struct {
    Execution ExecutionConfig `json:"execution"`
    Planner   PlannerRuntime  `json:"planner"`
    Executor  ExecutorRuntime `json:"executor"`
    Experts   ExpertsRuntime  `json:"experts"`
    Summary   SummaryRuntime  `json:"summary"`
    Rollout   RolloutConfig   `json:"rollout"`
}

type ExecutionConfig struct {
    MaxIterations int           `json:"max_iterations"`
    TotalTimeout  time.Duration `json:"total_timeout"`
}

type PlannerRuntime struct {
    MaxStep int    `json:"max_step"`
    Model   string `json:"model,omitempty"`
}

type ExecutorRuntime struct {
    MaxRetry      int           `json:"max_retry"`
    ExpertTimeout time.Duration `json:"expert_timeout"`
}

type ExpertsRuntime struct {
    HostOps       ExpertRuntime `json:"hostops"`
    K8s           ExpertRuntime `json:"k8s"`
    Service       ExpertRuntime `json:"service"`
    Delivery      ExpertRuntime `json:"delivery"`
    Observability ExpertRuntime `json:"observability"`
}

type ExpertRuntime struct {
    Enabled bool          `json:"enabled"`
    MaxStep int           `json:"max_step"`
    Timeout time.Duration `json:"timeout"`
    Model   string        `json:"model,omitempty"`
}

type SummaryRuntime struct {
    MaxStep int    `json:"max_step"`
    Model   string `json:"model,omitempty"`
}

type RolloutConfig struct {
    EnableNewOrchestrator bool    `json:"enable_new_orchestrator"`
    TrafficPercent        int     `json:"traffic_percent"`
    SessionAllowList      []string `json:"session_allow_list,omitempty"`
    UserAllowList         []uint64 `json:"user_allow_list,omitempty"`
}
```

### 4.3.1 运行时遥测与排障约束

新架构上线后，问题通常不会出在“有没有 plan”，而是出在“为什么选了这个 expert、为什么某个 step 卡住、为什么恢复后重复执行”。因此需要把可观测性定义为运行时契约，而不是调试时临时加日志。

建议统一补充三层遥测：

1. 编排层遥测
2. 模型调用遥测
3. 工具调用遥测

建议最小追踪结构如下：

```go
type TraceContext struct {
    TraceID     string `json:"trace_id"`
    SessionID   string `json:"session_id"`
    PlanID      string `json:"plan_id,omitempty"`
    StepID      string `json:"step_id,omitempty"`
    Expert      string `json:"expert,omitempty"`
    Iteration   int    `json:"iteration"`
    ParentSpan  string `json:"parent_span,omitempty"`
    CurrentSpan string `json:"current_span,omitempty"`
}
```

约束：

- `GatewayRuntime` 生成或透传 `trace_id`
- `Planner / Executor / Summarizer / Expert` 共享同一条 `trace_id`
- 每个 `step_id` 必须派生独立 span，便于串联 SSE、日志、审批和工具执行
- 审批恢复时不得生成新的 `trace_id`，只允许追加新的 span

建议记录的关键埋点：

| 层级 | 埋点 | 最小字段 |
|------|------|----------|
| Orchestrator | `run_started/run_finished/run_failed` | `session_id/trace_id/iteration/result` |
| Planner | `plan_started/plan_decided` | `decision_type/clarify_count/expert_count` |
| Executor | `step_scheduled/step_started/step_finished/step_blocked` | `step_id/expert/state/attempt/duration_ms` |
| Expert | `expert_invoked/expert_returned` | `expert/tool_count/duration_ms` |
| Tool | `tool_started/tool_finished/tool_failed` | `tool_name/mode/risk/approval_required/duration_ms` |
| Approval | `approval_waiting/approval_resumed/approval_rejected` | `step_id/token_scope/risk` |

### 4.3.2 指标与告警基线

为了支持灰度和替换旧链路，建议从一开始就为新 orchestrator 建立一组可比较指标：

```text
ai_orchestrator_run_total{result}
ai_orchestrator_run_duration_ms
ai_planner_decision_total{type}
ai_executor_step_total{expert,state}
ai_executor_step_duration_ms{expert}
ai_tool_call_total{tool,mode,risk,result}
ai_tool_call_duration_ms{tool}
ai_approval_total{result,risk}
ai_replan_total{reason}
```

建议的告警关注面：

- `run_failed` 突增
- `waiting_approval` 长时间堆积
- 同一 tool 的 `tool_failed` 明显上升
- `replan_total` 异常增高，说明 Planner/Expert 输出质量下降
- 某个 expert 平均 step 时长异常拉长

### 4.3.3 日志与调试载荷收敛

为了避免模型原文、工具原始结果和大段日志无序写入存储，建议明确日志分层：

- 业务日志：只写流程结论、状态变化、错误码、摘要
- 调试日志：写工具输入摘要、模型决策摘要、评分解释、审批原因
- 原始载荷：放入 `ExecutionState` 或独立对象存储，日志中只保留引用

建议统一调试记录结构：

```go
type DebugRecord struct {
    Scope      string         `json:"scope"`
    TraceID    string         `json:"trace_id"`
    SessionID  string         `json:"session_id"`
    PlanID     string         `json:"plan_id,omitempty"`
    StepID     string         `json:"step_id,omitempty"`
    Component  string         `json:"component"`
    Summary    string         `json:"summary"`
    Attributes map[string]any `json:"attributes,omitempty"`
    PayloadRef string         `json:"payload_ref,omitempty"`
}
```

约束：

- 不在普通应用日志中直接打印完整 prompt、完整模型回复、完整原始日志
- `ResolveScore`、`ApprovalDecision`、`StepError` 应以摘要形式进入 `DebugRecord`
- `PayloadRef` 指向原始输入输出的持久化位置，便于离线排障

### 4.3.4 隐私与审计边界

由于新架构会显式保存更多执行状态，建议同步定义最小脱敏规则：

- 凭证内容、敏感配置值不得进入 `Evidence`、SSE、普通日志
- `RawOutput` 中涉及密钥、token、连接串的字段在持久化前必须脱敏
- `DebugRecord` 只记录参数摘要，不记录高敏原文
- 审批与副作用回执必须可审计，但不得泄露不必要的资源明细

## 4.4 上下文预算与压缩策略

多专家架构最大的运行时风险之一不是单次回答质量，而是上下文不断膨胀后导致 Planner、Summarizer 和 Expert 逐步退化。因此建议在设计阶段就明确 token 预算、摘要层级和裁剪规则。

### 4.4.1 上下文分层

建议把可进入模型上下文的内容固定分为四层：

1. `ConversationHistory`
   - 用户问题、必要澄清、最终总结摘要
2. `PlanningContext`
   - `RuntimeContext`、`ResolvedResources`、`MissingFacts`、`NextPlanHints`
3. `ExecutionSummary`
   - 每个 step 的 `Summary`、`Findings`、证据引用
4. `RawArtifactsRef`
   - 原始日志、完整 tool output、审计结果的引用，不直接拼进 prompt

约束：

- Planner 默认只读 `ConversationHistory + PlanningContext + 历史 step 摘要`
- Expert 默认只读当前 `PlanStep.Input`、相关资源上下文和必要的上游摘要
- Summarizer 读取 `ExecutionSummary`，必要时通过引用读取少量原始证据
- 原始大载荷只能按需提取，不能作为默认上下文注入

### 4.4.2 Token 预算建议

建议按角色分别给出软硬预算：

| 组件 | 软预算 | 硬预算 | 超限策略 |
|------|--------|--------|----------|
| Planner | 8k | 12k | 优先压缩历史 step 摘要，再裁剪对话历史 |
| Expert | 6k | 10k | 仅保留当前 step 相关证据，丢弃无关步骤 |
| Summarizer | 10k | 14k | 先合并重复 findings，再压缩 evidence 内容 |

说明：

- 这里是建议口径，不绑定某个具体模型的窗口上限
- 实际模型窗口更大时，也不建议无限放大上下文
- 超限裁剪必须确定性，不能让不同运行轮次得到完全不同的输入形态

### 4.4.3 历史压缩与证据归档

建议引入一层显式压缩对象，而不是每次重新总结：

```go
type StepDigest struct {
    StepID       string   `json:"step_id"`
    Expert       string   `json:"expert"`
    Summary      string   `json:"summary"`
    Findings     []string `json:"findings,omitempty"`
    EvidenceRefs []string `json:"evidence_refs,omitempty"`
    Risks        []string `json:"risks,omitempty"`
}

type IterationDigest struct {
    Iteration    int          `json:"iteration"`
    Goal         string       `json:"goal"`
    StepDigests  []StepDigest `json:"step_digests"`
    MissingFacts []string     `json:"missing_facts,omitempty"`
    Conclusion   string       `json:"conclusion,omitempty"`
}
```

约束：

- Executor 完成 step 后生成 `StepDigest`
- Summarizer 完成一轮后生成 `IterationDigest`
- Planner 在 replanning 时优先消费 `IterationDigest`，而不是整轮原始 `StepResult`
- 相同证据的多次引用应在 digest 层去重

### 4.4.4 裁剪优先级

当任一组件即将超出硬预算时，按固定顺序裁剪：

1. 删除与当前任务无关的旧 `delta`
2. 将旧 `Evidence.Content` 替换为 `Evidence.Ref`
3. 合并重复 `Findings`
4. 将完整 `StepResult` 压缩为 `StepDigest`
5. 仅保留最近一轮 `IterationDigest` + 当前所需的少量历史结论

禁止事项：

- 不能在裁剪时丢失 `step_id/expert/ref` 这类可追溯字段
- 不能把审批原因、失败错误码、关键风险从上下文里裁掉
- 不能把用户最近一次澄清内容裁掉

配置约束：

- 每个 expert 必须支持独立启停
- 每个 expert 必须支持单独 timeout 和 max_step
- 必须支持通过 feature flag 将新 orchestrator 限定在灰度会话或灰度用户上

## 4.4 状态持久化模型

建议将 `ExecutionState` 明确为可持久化结构，而不是隐式散落在 checkpoint 中：

```go
type ExecutionState struct {
    SessionID   string                  `json:"session_id"`
    PlanID      string                  `json:"plan_id"`
    TraceID     string                  `json:"trace_id"`
    Iteration   int                     `json:"iteration"`
    Planner     *PlannerDecision        `json:"planner,omitempty"`
    StepStates  map[string]StepState    `json:"step_states"`
    StepResults map[string]StepResult   `json:"step_results,omitempty"`
    PendingApproval *ApprovalResumeState `json:"pending_approval,omitempty"`
    LastUpdated int64                   `json:"last_updated"`
}
```

持久化约束：

- 每次 `step_state` 变化后必须刷新持久化状态
- 每次 `step_result` 完成后必须持久化结果摘要
- `waiting_approval` 时必须持久化 `PendingApproval`
- `done` 后应将状态标记为终态，只保留只读查询能力

## 4.5 测试矩阵

建议把测试从“列表”提升为矩阵，避免只测 happy path：

| 层级 | 重点场景 |
|------|----------|
| Planner 单测 | resolve 精确命中、歧义澄清、权限拒绝、direct reply、reject |
| Executor 单测 | DAG 并发、重试、blocked 依赖、超时、审批恢复 |
| Expert 单测 | 工具隔离、prompt contract、结构化输出、风险工具审批 |
| Summarizer 单测 | 证据充分、证据不足、证据冲突、需重规划 |
| Orchestrator 集测 | 多轮 iteration、clarify -> continue、approval -> resume、done |
| SSE 集测 | 事件顺序、事件 payload 完整性、异常中断事件 |

推荐最小端到端用例：

1. 单服务运行态异常排查
2. 发布失败诊断
3. 模糊资源名触发澄清
4. 高风险变更触发审批并恢复
5. 证据不足触发 replanning

## 5. 迁移策略

### 5.1 迁移目标

迁移目标不是“在旧单体 Agent 旁边再堆一套新 Agent”，而是完成以下替换：

- 用 Planner 替代旧 `AIAgent` 中的意图理解、资源补全、权限预检查
- 用 Executor 替代旧单体 `react.Agent` 的统一调度职责
- 用多个 execution experts 替代旧全量工具池
- 用 Summarizer 替代旧模式下模型直接拼接最终答案

因此迁移完成后的目标状态是：

- `internal/ai/agent.go` 不再承载主编排所有权
- 旧“单模型 + 全工具池”执行链路不再作为主路径
- 工具仍可保留原实现，但按 Planner/Expert 重新挂载

### 5.2 迁移原则

迁移遵循以下原则：

1. **先切职责，再删代码**
   先把职责迁出到新模块，再删除旧模块，而不是边改边删导致链路中断。
2. **保留稳定工具实现，重建挂载方式**
   `internal/ai/tools` 下的大部分工具函数保留，只调整其归属与调用入口。
3. **兼容入口短期保留，内部路由逐步切换**
   对外 API、SSE 协议、审批流入口优先保持兼容。
4. **删除必须以后置验收为前提**
   只有在新链路通过端到端验证后，才允许删除旧编排实现。

### 5.3 分阶段迁移路径

#### Phase A: 建立新骨架，不动现网入口

- 新建 `planner/`, `executor/`, `experts/`, `summarizer/`, `orchestrator.go`
- 保留 `internal/ai/agent.go` 作为当前默认执行入口
- 复用现有 `tools.BuildRegisteredTools(...)`、审批中间件、checkpoint store、SSE publisher

目标：

- 新架构能在离线或测试链路上跑通
- 不影响现有用户流量

#### Phase B: 工具归位，建立双轨执行

- 将 `*_list_inventory`、`permission_check`、`user_list`、`role_list` 归为 Planner 支撑工具
- 将执行型工具按 5 个 experts 重新挂载
- 新建 orchestrator，通过 feature flag 或配置切换是否启用新链路

目标：

- 同一套工具实现，同时支持旧入口和新入口
- 新链路可覆盖核心场景，但旧链路仍可回退

#### Phase C: 切换主路径

- 让 `internal/service/ai` / gateway 层优先调用新 orchestrator
- 旧 `AIAgent` 降级为 fallback/compat runner
- 审批恢复、SSE 事件、会话恢复全部以新 orchestrator 为主

目标：

- 新架构成为默认路径
- 旧单体 agent 仅承担兜底或短期兼容

#### Phase D: 删除旧编排

- 删除旧单体 `react.Agent + 全量工具池` 主链路
- 删除已无调用方的兼容包装
- 删除过时的设计文档、映射表、无效测试

目标：

- 代码库中只保留一套主编排模型
- 不再维护旧链路的行为一致性

### 5.4 模块迁移映射

| 旧模块/职责 | 新归属 | 迁移动作 |
|-------------|--------|----------|
| `internal/ai/agent.go` 中的统一推理入口 | `internal/ai/orchestrator.go` | 替换主编排职责，旧入口短期兼容 |
| `internal/ai/agent.go` 中的全量工具挂载 | Planner + Experts | 拆分挂载，不复用全量工具池 |
| `internal/ai/tools` 下工具实现 | Planner/Experts | 保留实现，迁移注册方式 |
| 审批与安全中间件 | Executor / tools middleware | 原样复用，接入新链路 |
| checkpoint / session 恢复 | Executor / Orchestrator | 保留能力，迁移状态模型 |
| SSE 输出 | Orchestrator | 统一由新编排产生 |

### 5.5 逐文件迁移表

下面的表用于指导实际实施时的文件级动作，避免只讨论模块名而不落到代码。

| 文件/目录 | 当前角色 | 迁移后角色 | 动作 |
|-----------|----------|------------|------|
| [internal/ai/agent.go](/root/project/k8s-manage/internal/ai/agent.go) | 单体 `react.Agent` 主入口、全量工具池绑定 | compat runner / fallback | 先降级，后删除旧主链路代码 |
| [internal/ai/orchestrator.go](/root/project/k8s-manage/internal/ai/orchestrator.go) | 现有占位/旧实现 | 新主编排入口 | 重写为 Planner -> Executor -> Experts -> Summarizer 主链路 |
| [internal/ai/gateway.go](/root/project/k8s-manage/internal/ai/gateway.go) | 对外网关、会话/审批/工具执行入口 | 继续保留 | 改为优先调用新 orchestrator，保留会话/审批/SSE 管理职责 |
| [internal/ai/control.go](/root/project/k8s-manage/internal/ai/control.go) | 控制平面与权限支撑 | 继续保留 | 提供 Planner 权限预检查和元信息查询支撑 |
| [internal/ai/model.go](/root/project/k8s-manage/internal/ai/model.go) | 模型构建/配置 | 继续保留 | 作为 Planner/Experts/Summarizer 共用模型装配层 |
| [internal/ai/callbacks.go](/root/project/k8s-manage/internal/ai/callbacks.go) | 回调与事件支撑 | 继续保留 | 对接新 orchestrator 和 Experts 执行事件 |
| [internal/ai/router/](/root/project/k8s-manage/internal/ai/router) | 旧 domain classifier/router | 待评估 | 若 Planner 完全接管路由则删除，否则降级为轻量预分类器 |
| [internal/ai/graph/](/root/project/k8s-manage/internal/ai/graph) | 旧 ActionGraph 执行编排 | 待评估 | 若其校验/执行能力被 Executor 吸收则删除，否则抽取可复用校验能力 |
| [internal/ai/approval/](/root/project/k8s-manage/internal/ai/approval) | 审批路由、任务生成、审批执行 | 继续保留 | 接入新 orchestrator / Executor，复用审批能力 |
| [internal/ai/aspect/](/root/project/k8s-manage/internal/ai/aspect) | 工具安全中间件 | 继续保留 | 继续作为 tools middleware 使用 |
| [internal/ai/state/](/root/project/k8s-manage/internal/ai/state) | 会话/状态模型 | 继续保留并扩展 | 增加 plan/execution/summarizer 所需状态 |
| [internal/ai/rag/](/root/project/k8s-manage/internal/ai/rag) | 检索增强 | 继续保留 | 视需要供 Planner 和 Experts 共享 |
| [internal/ai/tools/](/root/project/k8s-manage/internal/ai/tools) | 工具实现与注册 | 继续保留 | 保留实现层，重构注册与挂载入口 |
| [internal/ai/integration_test.go](/root/project/k8s-manage/internal/ai/integration_test.go) | router + graph 旧链路集成测试 | 重写 | 替换为 orchestrator + AgentAsTool 新链路测试 |
| [internal/ai/router/classifier_test.go](/root/project/k8s-manage/internal/ai/router/classifier_test.go) | 路由分类测试 | 待评估 | 若 router 保留则继续维护，否则删除 |
| [internal/ai/graph/graph_test.go](/root/project/k8s-manage/internal/ai/graph/graph_test.go) | 旧 graph 测试 | 待评估 | 随 graph 去留决定保留或删除 |
| [internal/ai/callbacks_test.go](/root/project/k8s-manage/internal/ai/callbacks_test.go) | 回调测试 | 继续保留 | 适配新 orchestrator 事件流 |
| [internal/ai/approval/*_test.go](/root/project/k8s-manage/internal/ai/approval/executor_test.go) | 审批能力测试 | 继续保留 | 仅修正依赖的 runner/orchestrator 接口 |

### 5.6 删除与保留清单

#### 必须保留

- `internal/ai/tools/...`
  原因：这是能力实现层，不是旧架构问题本身
- 审批、安全、中断恢复相关能力
- 现有 SSE 基础设施和网关入口

#### 需要降级为兼容层

- [internal/ai/agent.go](/root/project/k8s-manage/internal/ai/agent.go)
  迁移中期保留，但仅作为 fallback/compat runner，不再继续承载新功能

#### 在完成切换后应删除

- `internal/ai/agent.go` 中“全量工具注册 + 单体 react.Agent 编排”相关代码
- 旧单体 Agent 专属的 prompt 拼接、工具全集绑定、单链路推理逻辑
- 与旧主链路强耦合、且已被新 orchestrator 替代的测试

#### 需要评估后删除

- `internal/ai/graph/*`
  只有在确认其校验/执行能力已被新 Executor 完整吸收，且没有其他调用方后才删除
- `internal/ai/router/*`
  如果新 Planner 完全接管 domain routing，则可删除；如果仍用于轻量预分类，可保留
- 旧 design/proposal 中关于 4 expert 与单体迁移的过时段落

### 5.7 删除门槛

删除旧实现前，至少满足以下条件：

1. 新 orchestrator 已覆盖主对话入口
2. 五个 execution experts 已全部接入
3. Planner 资源解析、权限预检查、用户澄清已跑通
4. 审批中断恢复在新链路可用
5. SSE 事件已兼容前端
6. 至少一轮端到端回归验证通过

在未满足以上条件前，不应直接删除旧入口，只允许将其降级为 fallback。

### 5.8 推荐执行顺序

推荐按下面顺序迁移，减少来回返工：

1. 抽 Planner 工具和结构化计划
2. 建 Executor 与 `AgentAsTool` 链路
3. 迁移 `HostOpsExpert`、`K8sExpert`
4. 迁移 `ServiceExpert`
5. 迁移 `DeliveryExpert`、`ObservabilityExpert`
6. 接入 Summarizer 和 replanning
7. 切换 gateway 主路径
8. 删除旧单体编排代码

## 6. 关键决策

为避免后续实现反复返工，明确以下技术决策：

1. Planner 输出采用严格结构化对象，而不是依赖自然语言计划解析
2. Executor 采用 Go 代码驱动的 DAG 调度，而不是 LLM 自治调度
3. Expert 返回“结构化结果优先，文本摘要为辅”
4. Summarizer 不直接调用平台工具，补充调查必须经 Planner 重新规划
5. 阶段型 Agent 与能力型 Agent 明确分离，避免再次形成单体大 Agent
