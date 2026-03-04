# Spec: Hybrid MOE Agent API 规范

## 配置文件规范

### experts.yaml

**位置**: `configs/experts.yaml`

```yaml
# 专家注册表配置
version: "1.0"

experts:
  - name: string                    # 专家唯一标识，必填
    display_name: string            # 显示名称，必填
    persona: string                 # 专家人设prompt，必填
    tool_patterns: [string]         # 工具名称匹配模式，支持通配符
    domains:                        # 领域权重
      - name: string                # 领域名称
        weight: number              # 权重 0.0-1.0
    keywords: [string]              # 触发关键词
    capabilities: [string]          # 能力描述（用于展示）
    risk_level: string              # 风险等级: low | medium | high
```

**验证规则**:
- `name`: 只允许小写字母、数字、下划线
- `weight`: 范围 [0.0, 1.0]
- `tool_patterns`: 至少一个
- `persona`: 非空字符串

### scene_mappings.yaml

**位置**: `configs/scene_mappings.yaml`

```yaml
# 场景映射配置
version: "1.0"

mappings:
  "<scene_key>":
    primary_expert: string          # 主专家名称，必填
    helper_experts: [string]        # 辅助专家列表
    strategy: string                # 执行策略: single | sequential | parallel
    context_hints: [string]         # 上下文提示字段
```

**验证规则**:
- `primary_expert`: 必须存在于experts.yaml
- `helper_experts`: 每个都必须存在于experts.yaml
- `strategy`: 枚举值

## Go 接口规范

### ExpertRegistry

```go
// Expert 专家实例
type Expert struct {
    Name         string
    DisplayName  string
    Persona      string
    ToolPatterns []string
    Domains      []DomainWeight
    Keywords     []string
    Capabilities []string
    RiskLevel    string
}

// ExpertRegistry 专家注册表接口
type ExpertRegistry interface {
    // GetExpert 获取专家实例
    GetExpert(name string) (*Expert, bool)

    // ListExperts 列出所有专家
    ListExperts() []*Expert

    // Reload 重新加载配置
    Reload() error

    // MatchByKeywords 根据关键词匹配专家
    MatchByKeywords(content string) []*RankedExpert

    // MatchByDomain 根据领域匹配专家
    MatchByDomain(domain string) []*RankedExpert
}

// RankedExpert 带权重的专家
type RankedExpert struct {
    Expert *Expert
    Score  float64
}
```

### HybridRouter

```go
// RouteRequest 路由请求
type RouteRequest struct {
    Message        string
    Scene          string
    History        []*schema.Message
    RuntimeContext map[string]any
}

// RouteDecision 路由决策
type RouteDecision struct {
    PrimaryExpert string            // 主专家
    HelperExperts []string          // 辅助专家
    Strategy      ExecutionStrategy // 执行策略
    Confidence    float64           // 决策置信度
    Source        string            // 决策来源
}

// ExecutionStrategy 执行策略
type ExecutionStrategy string

const (
    StrategySingle     ExecutionStrategy = "single"
    StrategySequential ExecutionStrategy = "sequential"
    StrategyParallel   ExecutionStrategy = "parallel"
)

// HybridRouter 混合路由器接口
type HybridRouter interface {
    // Route 执行路由决策
    Route(ctx context.Context, req *RouteRequest) *RouteDecision
}
```

### Orchestrator

```go
// ExecuteRequest 执行请求
type ExecuteRequest struct {
    Message        string
    Decision       *RouteDecision
    RuntimeContext map[string]any
    History        []*schema.Message
}

// ExecuteResult 执行结果
type ExecuteResult struct {
    Response string        // 最终响应
    Traces   []ExpertTrace // 执行追踪
    Metadata map[string]any
}

// ExpertTrace 专家执行追踪
type ExpertTrace struct {
    ExpertName string
    Input      string
    Output     string
    Duration   time.Duration
    Status     string // "success" | "error" | "skipped"
}

// Orchestrator 调度器接口
type Orchestrator interface {
    // Execute 执行请求
    Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error)
}
```

### ExpertExecutor

```go
// ExecutionPlan 执行计划
type ExecutionPlan struct {
    Steps []ExecutionStep
}

// ExecutionStep 执行步骤
type ExecutionStep struct {
    ExpertName  string
    Task        string
    DependsOn   []int
    ContextFrom []int
}

// ExpertResult 专家执行结果
type ExpertResult struct {
    ExpertName string
    Output     string
    Error      error
    Duration   time.Duration
}

// ExpertExecutor 专家执行器接口
type ExpertExecutor interface {
    // ExecuteStep 执行单步
    ExecuteStep(ctx context.Context, step *ExecutionStep, priorResults []ExpertResult, baseMessage string) (*ExpertResult, error)
}
```

### ResultAggregator

```go
// AggregationMode 聚合模式
type AggregationMode string

const (
    AggregationTemplate AggregationMode = "template"
    AggregationLLM      AggregationMode = "llm"
)

// ResultAggregator 结果聚合器接口
type ResultAggregator interface {
    // Aggregate 聚合结果
    Aggregate(ctx context.Context, results []ExpertResult, originalQuery string) (string, error)
}
```

## 错误码规范

| 错误码 | 说明 | HTTP状态码 |
|--------|------|-----------|
| `EXPERT_NOT_FOUND` | 专家不存在 | 500 |
| `EXPERT_CONFIG_INVALID` | 专家配置无效 | 500 |
| `ROUTER_NO_MATCH` | 路由无匹配 | 500 |
| `EXECUTOR_TIMEOUT` | 执行超时 | 504 |
| `EXECUTOR_ERROR` | 执行错误 | 500 |
| `AGGREGATOR_ERROR` | 聚合错误 | 500 |

## 配置示例

### 完整专家配置示例

```yaml
version: "1.0"

experts:
  # ─────────────────────────────────────────────────────────────
  # 基础设施层 (Infrastructure)
  # ─────────────────────────────────────────────────────────────

  - name: host_expert
    display_name: "主机运维专家"
    persona: |
      你是主机运维专家。专注于主机资产管理、SSH远程执行、
      主机状态管理。确保操作安全，谨慎执行高危命令。
      执行变更类操作前必须确认用户意图和审批状态。
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
    keywords:
      - host
      - ssh
      - 主机
      - 服务器
      - 批量执行
      - 资产
    capabilities:
      - "主机资产清单查询"
      - "SSH远程命令执行"
      - "批量主机操作"
      - "主机状态管理"
    risk_level: medium

  - name: os_expert
    display_name: "系统诊断专家"
    persona: |
      你是操作系统诊断专家。专注于CPU/内存/磁盘/网络诊断，
      进程分析，系统日志分析。提供精准的性能问题定位。
    tool_patterns:
      - "os_*"
    domains:
      - name: os_diagnostics
        weight: 0.95
      - name: performance_tuning
        weight: 0.80
      - name: process_management
        weight: 0.85
    keywords:
      - cpu
      - memory
      - disk
      - network
      - 进程
      - 系统日志
      - 性能
      - 负载
    capabilities:
      - "系统资源监控"
      - "进程分析"
      - "日志诊断"
      - "性能问题定位"
    risk_level: low

  - name: container_runtime_expert
    display_name: "容器运行时专家"
    persona: |
      你是容器运行时专家。专注于Docker/Containerd运行时诊断，
      容器状态、镜像管理、运行时问题排查。
    tool_patterns:
      - "os_get_container_runtime"
    domains:
      - name: container_runtime
        weight: 0.95
      - name: docker
        weight: 0.90
      - name: containerd
        weight: 0.85
    keywords:
      - docker
      - container
      - 容器
      - 镜像
      - runtime
      - containerd
    capabilities:
      - "容器运行时状态"
      - "容器诊断"
    risk_level: low

  # ─────────────────────────────────────────────────────────────
  # 容器编排层 (Kubernetes)
  # ─────────────────────────────────────────────────────────────

  - name: k8s_expert
    display_name: "Kubernetes诊断专家"
    persona: |
      你是Kubernetes诊断专家。专注于集群状态、资源列表、事件分析，
      提供集群健康状态的快速诊断。
    tool_patterns:
      - "k8s_list_*"
      - "k8s_get_events"
      - "cluster_*"
    domains:
      - name: kubernetes
        weight: 0.95
      - name: cluster_health
        weight: 0.90
      - name: k8s_resources
        weight: 0.88
    keywords:
      - k8s
      - kubernetes
      - cluster
      - 集群
      - namespace
      - node
    capabilities:
      - "集群资源查询"
      - "事件分析"
      - "集群健康诊断"
    risk_level: low

  - name: k8s_workload_expert
    display_name: "K8s工作负载专家"
    persona: |
      你是K8s工作负载专家。专注于Pod诊断、Deployment问题排查、
      日志分析、工作负载故障定位。
    tool_patterns:
      - "k8s_get_pod_logs"
      - "k8s_list_resources"
    domains:
      - name: k8s_workload
        weight: 0.95
      - name: pod_troubleshooting
        weight: 0.92
      - name: deployment_debug
        weight: 0.90
    keywords:
      - pod
      - deployment
      - replica
      - 日志
      - 重启
      - crash
      - container
      - 工作负载
    capabilities:
      - "Pod日志分析"
      - "工作负载故障诊断"
      - "容器重启原因分析"
    risk_level: low

  # ─────────────────────────────────────────────────────────────
  # 服务管理层 (Service Management)
  # ─────────────────────────────────────────────────────────────

  - name: service_expert
    display_name: "服务管理专家"
    persona: |
      你是服务管理专家。专注于服务目录、服务生命周期、服务详情查询，
      服务可见性管理。
    tool_patterns:
      - "service_*"
      - "service_catalog_*"
      - "service_category_*"
    domains:
      - name: service_management
        weight: 0.95
      - name: service_catalog
        weight: 0.90
      - name: service_lifecycle
        weight: 0.85
    keywords:
      - service
      - 服务
      - catalog
      - 目录
      - 可见性
      - 服务管理
    capabilities:
      - "服务目录查询"
      - "服务详情"
      - "服务可见性检查"
    risk_level: low

  - name: deploy_expert
    display_name: "部署发布专家"
    persona: |
      你是部署发布专家。专注于部署目标管理、服务部署预览与执行，
      环境引导，发布流程。执行变更前必须确认审批。
    tool_patterns:
      - "deployment_*"
      - "service_deploy_*"
    domains:
      - name: deployment
        weight: 0.95
      - name: release_management
        weight: 0.90
      - name: environment_bootstrap
        weight: 0.85
    keywords:
      - deploy
      - 部署
      - 发布
      - target
      - 目标
      - release
      - 上线
    capabilities:
      - "部署目标管理"
      - "服务部署预览"
      - "部署执行"
    risk_level: high

  - name: topology_expert
    display_name: "服务拓扑专家"
    persona: |
      你是服务拓扑专家。专注于服务依赖关系、调用链路、服务间关系分析。
    tool_patterns:
      - "topology_*"
    domains:
      - name: service_topology
        weight: 0.95
      - name: dependency_analysis
        weight: 0.90
    keywords:
      - topology
      - 拓扑
      - 依赖
      - 调用链
      - relationship
      - 关系
    capabilities:
      - "服务拓扑查询"
      - "依赖关系分析"
    risk_level: low

  # ─────────────────────────────────────────────────────────────
  # 发布运维层 (CI/CD & Jobs)
  # ─────────────────────────────────────────────────────────────

  - name: cicd_expert
    display_name: "CI/CD流水线专家"
    persona: |
      你是CI/CD流水线专家。专注于流水线管理、构建状态查询、
      发布流程触发。触发发布前确认用户意图。
    tool_patterns:
      - "cicd_*"
      - "pipeline_*"
    domains:
      - name: cicd
        weight: 0.95
      - name: pipeline
        weight: 0.90
      - name: build
        weight: 0.85
    keywords:
      - pipeline
      - 流水线
      - build
      - 构建
      - cicd
      - 发布
      - jenkins
    capabilities:
      - "流水线状态查询"
      - "构建触发"
    risk_level: medium

  - name: job_expert
    display_name: "定时任务专家"
    persona: |
      你是定时任务专家。专注于定时任务管理、任务执行状态、
      手动触发任务。
    tool_patterns:
      - "job_*"
    domains:
      - name: job_management
        weight: 0.95
      - name: scheduled_tasks
        weight: 0.90
    keywords:
      - job
      - 任务
      - 定时
      - 调度
      - cron
    capabilities:
      - "任务列表查询"
      - "任务执行状态"
      - "手动触发任务"
    risk_level: medium

  # ─────────────────────────────────────────────────────────────
  # 可观测性层 (Observability)
  # ─────────────────────────────────────────────────────────────

  - name: monitor_expert
    display_name: "监控告警专家"
    persona: |
      你是监控告警专家。专注于指标查询、告警规则、活跃告警分析，
      SLO监控，可观测性诊断。
    tool_patterns:
      - "monitor_*"
      - "alert_*"
      - "metric_*"
    domains:
      - name: observability
        weight: 0.95
      - name: alerting
        weight: 0.90
      - name: metrics
        weight: 0.88
    keywords:
      - monitor
      - 监控
      - alert
      - 告警
      - metric
      - 指标
      - prometheus
      - grafana
    capabilities:
      - "指标查询"
      - "告警分析"
      - "SLO监控"
    risk_level: low

  - name: audit_expert
    display_name: "审计日志专家"
    persona: |
      你是审计日志专家。专注于操作审计、日志查询、
      安全事件追溯。
    tool_patterns:
      - "audit_*"
    domains:
      - name: audit
        weight: 0.95
      - name: security_logging
        weight: 0.90
    keywords:
      - audit
      - 审计
      - 日志
      - 操作记录
      - 追溯
    capabilities:
      - "审计日志查询"
      - "操作追溯"
    risk_level: low

  # ─────────────────────────────────────────────────────────────
  # 治理安全层 (Governance & Security)
  # ─────────────────────────────────────────────────────────────

  - name: security_expert
    display_name: "安全治理专家"
    persona: |
      你是安全治理专家。专注于权限管理、角色配置、用户管理，
      最小权限原则，安全最佳实践。
    tool_patterns:
      - "permission_*"
      - "role_*"
      - "user_*"
    domains:
      - name: security
        weight: 0.95
      - name: rbac
        weight: 0.90
      - name: iam
        weight: 0.85
    keywords:
      - permission
      - 权限
      - role
      - 角色
      - user
      - 用户
      - rbac
      - 安全
    capabilities:
      - "权限检查"
      - "角色管理"
      - "用户管理"
    risk_level: low

  - name: config_expert
    display_name: "配置管理专家"
    persona: |
      你是配置管理专家。专注于应用配置、配置项查询、
      配置差异对比。
    tool_patterns:
      - "config_*"
    domains:
      - name: config_management
        weight: 0.95
      - name: app_config
        weight: 0.90
    keywords:
      - config
      - 配置
      - 配置项
      - 配置中心
      - apollo
      - nacos
    capabilities:
      - "配置查询"
      - "配置对比"
    risk_level: low
```

### 场景映射配置示例

```yaml
version: "1.0"

mappings:
  # 部署管理
  deployment:clusters:
    primary_expert: k8s_expert
    helper_experts: [monitor_expert]
    strategy: sequential
    context_hints: [cluster_id, namespace]

  deployment:credentials:
    primary_expert: host_expert
    helper_experts: []
    strategy: single

  deployment:hosts:
    primary_expert: host_expert
    helper_experts: [os_expert]
    strategy: sequential

  deployment:targets:
    primary_expert: deploy_expert
    helper_experts: [k8s_expert]
    strategy: sequential
    context_hints: [target_id, env]

  deployment:releases:
    primary_expert: deploy_expert
    helper_experts: [cicd_expert, k8s_expert]
    strategy: sequential
    context_hints: [service_id, cluster_id, env]

  deployment:approvals:
    primary_expert: cicd_expert
    helper_experts: [audit_expert]
    strategy: sequential

  deployment:topology:
    primary_expert: topology_expert
    helper_experts: [service_expert]
    strategy: sequential
    context_hints: [service_id]

  deployment:metrics:
    primary_expert: monitor_expert
    helper_experts: [k8s_expert]
    strategy: sequential
    context_hints: [service_id]

  deployment:audit:
    primary_expert: audit_expert
    helper_experts: []
    strategy: single
    context_hints: [user_id]

  deployment:aiops:
    primary_expert: monitor_expert
    helper_experts: [k8s_expert, os_expert]
    strategy: sequential
    context_hints: [cluster_id, service_id]

  # 服务管理
  services:list:
    primary_expert: service_expert
    helper_experts: []
    strategy: single
    context_hints: [service_id]

  services:detail:
    primary_expert: service_expert
    helper_experts: [k8s_expert, topology_expert]
    strategy: sequential
    context_hints: [service_id]

  services:provision:
    primary_expert: service_expert
    helper_experts: [deploy_expert]
    strategy: sequential
    context_hints: [project_id]

  services:deploy:
    primary_expert: deploy_expert
    helper_experts: [k8s_expert, cicd_expert]
    strategy: sequential
    context_hints: [service_id, target_id]

  services:catalog:
    primary_expert: service_expert
    helper_experts: []
    strategy: single
    context_hints: [category_id]

  # 治理管理
  governance:users:
    primary_expert: security_expert
    helper_experts: [audit_expert]
    strategy: sequential
    context_hints: [user_id]

  governance:roles:
    primary_expert: security_expert
    helper_experts: []
    strategy: single
    context_hints: [role_id]

  governance:permissions:
    primary_expert: security_expert
    helper_experts: [audit_expert]
    strategy: sequential
    context_hints: [user_id, resource, action]
```
