# Design: Hybrid MOE Agent 架构

## 架构概览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Hybrid MOE Agent 架构                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                         配置层 (YAML)                                │  │
│   │   configs/experts.yaml        configs/scene_mappings.yaml           │  │
│   └───────────────────────────────┬─────────────────────────────────────┘  │
│                                   │                                         │
│                                   ▼                                         │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                         专家注册表 (ExpertRegistry)                  │  │
│   │   - 加载配置，构建Expert实例                                          │  │
│   │   - 管理专家生命周期                                                   │  │
│   │   - 支持热重载                                                        │  │
│   └───────────────────────────────┬─────────────────────────────────────┘  │
│                                   │                                         │
│                                   ▼                                         │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                         混合路由器 (HybridRouter)                    │  │
│   │   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐            │  │
│   │   │ Scene Router│───▶│Keyword Router│───▶│Domain Router│            │  │
│   │   │  (精确匹配)  │    │ (规则匹配)   │    │ (语义匹配)  │            │  │
│   │   └─────────────┘    └─────────────┘    └─────────────┘            │  │
│   │                              │                                       │  │
│   │                              ▼                                       │  │
│   │                    RouteDecision {primary, helpers, strategy}       │  │
│   └───────────────────────────────┬─────────────────────────────────────┘  │
│                                   │                                         │
│                                   ▼                                         │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                         调度器 (Orchestrator)                        │  │
│   │   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐            │  │
│   │   │Plan Builder │───▶│  Executor   │───▶│ Aggregator  │            │  │
│   │   └─────────────┘    └─────────────┘    └─────────────┘            │  │
│   │                              │                                       │  │
│   │                              ▼                                       │  │
│   │                     ExecutionResult {response, traces}              │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 组件详细设计

### 1. 专家注册表 (ExpertRegistry)

**职责**：加载YAML配置，管理专家实例，支持热重载

```go
// internal/ai/experts/registry.go

package experts

type Expert struct {
    Name         string
    DisplayName  string
    Persona      string
    ToolPatterns []string
    Domains      []DomainWeight
    Keywords     []string
    Capabilities []string
    RiskLevel    string

    // 运行时
    agent       *react.Agent
    tools       map[string]tool.InvokableTool
}

type DomainWeight struct {
    Name   string
    Weight float64
}

type ExpertConfig struct {
    Name         string   `yaml:"name"`
    DisplayName  string   `yaml:"display_name"`
    Persona      string   `yaml:"persona"`
    ToolPatterns []string `yaml:"tool_patterns"`
    Domains      []struct {
        Name   string  `yaml:"name"`
        Weight float64 `yaml:"weight"`
    } `yaml:"domains"`
    Keywords     []string `yaml:"keywords"`
    Capabilities []string `yaml:"capabilities"`
    RiskLevel    string   `yaml:"risk_level"`
}

type ExpertRegistry struct {
    configPath string
    experts    map[string]*Expert
    allTools   map[string]tool.InvokableTool
    mu         sync.RWMutex
}

func NewExpertRegistry(configPath string, allTools map[string]tool.InvokableTool, chatModel model.ToolCallingChatModel) (*ExpertRegistry, error)

func (r *ExpertRegistry) GetExpert(name string) (*Expert, bool)

func (r *ExpertRegistry) ListExperts() []*Expert

func (r *ExpertRegistry) Reload() error

func (r *ExpertRegistry) MatchByKeywords(content string) []*RankedExpert

func (r *ExpertRegistry) MatchByDomain(domain string) []*RankedExpert
```

**配置文件格式** (`configs/experts.yaml`)：

```yaml
version: "1.0"

experts:
  - name: host_expert
    display_name: "主机运维专家"
    persona: |
      你是主机运维专家。专注于主机资产管理、SSH远程执行、
      主机状态管理。确保操作安全，谨慎执行高危命令。
    tool_patterns:
      - "host_*"
      - "credential_*"
    domains:
      - name: host_management
        weight: 0.95
      - name: ssh_operations
        weight: 0.90
      - name: asset_inventory
        weight: 0.85
    keywords: [host, ssh, 主机, 服务器, 批量执行]
    capabilities:
      - "主机资产清单查询"
      - "SSH远程命令执行"
      - "批量主机操作"
    risk_level: medium
```

### 2. 混合路由器 (HybridRouter)

**职责**：三层fallback路由策略

```go
// internal/ai/experts/router.go

package experts

type RouteRequest struct {
    Message       string
    Scene         string
    History       []*schema.Message
    RuntimeContext map[string]any
}

type RouteDecision struct {
    PrimaryExpert string
    HelperExperts []string
    Strategy      ExecutionStrategy
    Confidence    float64
    Source        string // "scene" | "keyword" | "domain" | "default"
}

type ExecutionStrategy string

const (
    StrategySingle     ExecutionStrategy = "single"     // 单专家
    StrategySequential ExecutionStrategy = "sequential" // 串行协作
    StrategyParallel   ExecutionStrategy = "parallel"   // 并行协作
)

type HybridRouter struct {
    registry      *ExpertRegistry
    sceneMappings *SceneMappings
    domainMatcher *DomainMatcher // 可选：语义匹配器
}

func NewHybridRouter(registry *ExpertRegistry, sceneMappingsPath string) (*HybridRouter, error)

func (r *HybridRouter) Route(ctx context.Context, req *RouteRequest) *RouteDecision

// 内部方法
func (r *HybridRouter) routeByScene(scene string) *RouteDecision

func (r *HybridRouter) routeByKeywords(content string) *RouteDecision

func (r *HybridRouter) routeByDomain(content string) *RouteDecision

func (r *HybridRouter) routeDefault() *RouteDecision
```

**路由流程**：

```
                    RouteRequest
                         │
                         ▼
              ┌─────────────────────┐
              │   Scene Router      │
              │   (精确匹配)         │
              └──────────┬──────────┘
                         │
            ┌────────────┼────────────┐
            │匹配        │未匹配       │
            ▼            │            ▼
     ┌─────────────┐     │     ┌─────────────────────┐
     │返回Decision │     │     │   Keyword Router    │
     │source=scene │     │     │   (规则匹配)         │
     └─────────────┘     │     └──────────┬──────────┘
                         │                │
                         │   ┌────────────┼────────────┐
                         │   │匹配        │未匹配       │
                         │   ▼            │            ▼
                         │ ┌─────────────┐│     ┌─────────────────────┐
                         │ │返回Decision ││     │   Domain Router    │
                         │ │source=keyword│    │   (语义匹配)         │
                         │ └─────────────┘│     └──────────┬──────────┘
                         │                │                │
                         │                │   ┌────────────┼────────────┐
                         │                │   │匹配        │未匹配       │
                         │                │   ▼            │            ▼
                         │                │ ┌─────────────┐│     ┌─────────────┐
                         │                │ │返回Decision ││     │ Default     │
                         │                │ │source=domain││     │ primary=general│
                         │                │ └─────────────┘│     └─────────────┘
                         │                │                │
                         └────────────────┴────────────────┘
```

**场景映射配置** (`configs/scene_mappings.yaml`)：

```yaml
version: "1.0"

mappings:
  deployment:clusters:
    primary_expert: k8s_expert
    helper_experts: [monitor_expert]
    strategy: sequential
    context_hints: [cluster_id, namespace]

  deployment:hosts:
    primary_expert: host_expert
    helper_experts: [os_expert]
    strategy: sequential

  services:detail:
    primary_expert: service_expert
    helper_experts: [k8s_expert, topology_expert]
    strategy: sequential

  services:deploy:
    primary_expert: deploy_expert
    helper_experts: [k8s_expert, cicd_expert]
    strategy: sequential

  governance:permissions:
    primary_expert: security_expert
    helper_experts: [audit_expert]
    strategy: sequential
```

### 3. 调度器 (Orchestrator)

**职责**：协调多专家执行，管理上下文传递

```go
// internal/ai/experts/orchestrator.go

package experts

type ExecuteRequest struct {
    Message        string
    Decision       *RouteDecision
    RuntimeContext map[string]any
    History        []*schema.Message
}

type ExecuteResult struct {
    Response string
    Traces   []ExpertTrace
    Metadata map[string]any
}

type ExpertTrace struct {
    ExpertName string
    Input      string
    Output     string
    Duration   time.Duration
    Status     string
}

type Orchestrator struct {
    registry   *ExpertRegistry
    executor   *ExpertExecutor
    aggregator *ResultAggregator
}

func NewOrchestrator(registry *ExpertRegistry, chatModel model.ToolCallingChatModel) *Orchestrator

func (o *Orchestrator) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error)

func (o *Orchestrator) buildPlan(decision *RouteDecision) *ExecutionPlan

func (o *Orchestrator) executePlan(ctx context.Context, plan *ExecutionPlan, req *ExecuteRequest) ([]ExpertResult, error)

func (o *Orchestrator) aggregateResults(results []ExpertResult, req *ExecuteRequest) (string, error)
```

### 4. 执行器 (ExpertExecutor)

**职责**：执行单个专家，传递上下文

```go
// internal/ai/experts/executor.go

package experts

type ExecutionPlan struct {
    Steps []ExecutionStep
}

type ExecutionStep struct {
    ExpertName  string
    Task        string
    DependsOn   []int // step indices
    ContextFrom []int // step indices to pull context from
}

type ExpertResult struct {
    ExpertName string
    Output     string
    Error      error
    Duration   time.Duration
}

type ExpertExecutor struct {
    registry *ExpertRegistry
}

func NewExpertExecutor(registry *ExpertRegistry) *ExpertExecutor

func (e *ExpertExecutor) ExecuteStep(ctx context.Context, step *ExecutionStep, priorResults []ExpertResult, baseMessage string) (*ExpertResult, error)

func (e *ExpertExecutor) buildExpertMessage(step *ExecutionStep, priorResults []ExpertResult, baseMessage string) string
```

### 5. 聚合器 (ResultAggregator)

**职责**：合并专家输出，生成最终响应

```go
// internal/ai/experts/aggregator.go

package experts

type AggregationMode string

const (
    AggregationTemplate AggregationMode = "template" // 无LLM
    AggregationLLM      AggregationMode = "llm"      // LLM总结
)

type ResultAggregator struct {
    mode      AggregationMode
    chatModel model.ToolCallingChatModel
}

func NewResultAggregator(mode AggregationMode, chatModel model.ToolCallingChatModel) *ResultAggregator

func (a *ResultAggregator) Aggregate(results []ExpertResult, originalQuery string) (string, error)

func (a *ResultAggregator) aggregateByTemplate(results []ExpertResult) string

func (a *ResultAggregator) aggregateByLLM(ctx context.Context, results []ExpertResult, originalQuery string) (string, error)
```

**模板聚合示例**：

```
┌──────────────────────────────────────────────────────────────────┐
│ 诊断报告                                                         │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│ 【K8s诊断结果】                                                   │
│ {k8s_expert.output}                                              │
│                                                                  │
│ 【监控指标分析】                                                   │
│ {monitor_expert.output}                                          │
│                                                                  │
│ 【建议】                                                          │
│ 根据以上分析，建议...                                              │
└──────────────────────────────────────────────────────────────────┘
```

## 数据流

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              完整数据流                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   1. 初始化阶段                                                              │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │                                                                   │    │
│   │   configs/experts.yaml ────┐                                      │    │
│   │                            ├──▶ ExpertRegistry ──▶ Expert实例们   │    │
│   │   configs/scene_mappings   │                                      │    │
│   │                   ─────────┘                                      │    │
│   │                                                                   │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│   2. 请求处理阶段                                                            │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │                                                                   │    │
│   │   HTTP Request                                                    │    │
│   │       │                                                           │    │
│   │       ▼                                                           │    │
│   │   PlatformAgent.Stream(ctx, messages)                            │    │
│   │       │                                                           │    │
│   │       ├──▶ HybridRouter.Route() ──▶ RouteDecision                │    │
│   │       │                                     │                     │    │
│   │       │                                     ▼                     │    │
│   │       ├──▶ Orchestrator.Execute() ◀──── ExecutionPlan            │    │
│   │       │         │                                                 │    │
│   │       │         ├──▶ Executor.ExecuteStep() × N                   │    │
│   │       │         │         │                                       │    │
│   │       │         │         ▼                                       │    │
│   │       │         │   Expert.agent.Generate()                       │    │
│   │       │         │         │                                       │    │
│   │       │         │         ▼                                       │    │
│   │       │         │   ExpertResult                                  │    │
│   │       │         │                                                 │    │
│   │       │         ├──▶ Aggregator.Aggregate()                       │    │
│   │       │         │         │                                       │    │
│   │       │         │         ▼                                       │    │
│   │       │         │   Final Response                                │    │
│   │       │         │                                                 │    │
│   │       │         ▼                                                 │    │
│   │       │   ExecuteResult                                           │    │
│   │       │                                                           │    │
│   │       ▼                                                           │    │
│   │   Stream Response                                                 │    │
│   │                                                                   │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 专家列表（12个）

| 层级 | 专家名 | 工具范围 | 风险等级 |
|------|--------|----------|----------|
| **基础设施** | host_expert | host_*, credential_* | medium |
| | os_expert | os_* | low |
| | container_runtime_expert | container runtime | low |
| **Kubernetes** | k8s_expert | k8s_list, k8s_events, cluster_* | low |
| | k8s_workload_expert | k8s_pod_logs, deployment诊断 | low |
| **服务管理** | service_expert | service_*, catalog_* | low |
| | deploy_expert | deployment_*, service_deploy_* | high |
| | topology_expert | topology_* | low |
| **发布运维** | cicd_expert | pipeline_* | medium |
| | job_expert | job_* | medium |
| **可观测性** | monitor_expert | monitor_*, alert_*, metric_* | low |
| | audit_expert | audit_* | low |
| **治理安全** | security_expert | permission_*, role_*, user_* | low |
| | config_expert | config_* | low |

## 错误处理

### 配置加载失败

```go
func (r *ExpertRegistry) loadWithFallback() error {
    if err := r.load(); err != nil {
        log.Warn("failed to load expert config, using defaults", "error", err)
        return r.loadDefaults()
    }
    return nil
}
```

### 路由失败

```go
func (r *HybridRouter) Route(ctx context.Context, req *RouteRequest) *RouteDecision {
    // 每层都有fallback，最终保证返回有效decision
    if decision := r.routeByScene(req.Scene); decision != nil {
        return decision
    }
    if decision := r.routeByKeywords(req.Message); decision != nil {
        return decision
    }
    return r.routeDefault()
}
```

### 专家执行失败

```go
func (o *Orchestrator) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
    results, err := o.executePlan(ctx, plan, req)
    if err != nil {
        // 部分失败：返回已完成的结果 + 错误信息
        return &ExecuteResult{
            Response: o.partialResponse(results, err),
            Traces:   traces,
            Metadata: map[string]any{"error": err.Error()},
        }, nil
    }
    // ...
}
```

## 性能考虑

1. **配置缓存**：ExpertRegistry在启动时加载配置，运行时只读
2. **路由轻量**：路由决策只做字符串匹配，无网络调用
3. **并行执行**：当strategy=parallel时，Executor可并行调用专家
4. **流式输出**：支持streaming response，减少首字节延迟
