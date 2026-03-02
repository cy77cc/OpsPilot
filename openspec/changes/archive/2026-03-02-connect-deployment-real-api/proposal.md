# Proposal: 对接部署管理真实后端 API

## Why

部署管理模块的可观测性子模块目前使用硬编码的 mock 数据，导致：
1. 用户无法看到真实的部署拓扑、指标、审计日志等数据
2. 策略管理和 AIOps 功能无法实际使用
3. 前后端数据流不完整，影响用户体验

## What Changes

### 新增后端 API（5个模块）

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        新增 API 端点                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  1. 部署拓扑 API                                                         │
│     GET /deploy/topology              获取服务部署拓扑                   │
│     GET /deploy/topology/:env         按环境获取拓扑                     │
│                                                                         │
│  2. 部署指标 API                                                         │
│     GET /deploy/metrics/summary       获取指标汇总                       │
│     GET /deploy/metrics/trends        获取趋势数据                       │
│                                                                         │
│  3. 审计日志 API                                                         │
│     GET /deploy/audit-logs            获取审计日志列表                   │
│     GET /deploy/audit-logs/:id        获取日志详情                       │
│                                                                         │
│  4. 策略管理 API                                                         │
│     GET /deploy/policies              获取策略列表                       │
│     POST /deploy/policies             创建策略                          │
│     PUT /deploy/policies/:id          更新策略                          │
│     DELETE /deploy/policies/:id       删除策略                          │
│                                                                         │
│  5. AIOps API                                                            │
│     GET /aiops/risk-findings          获取风险发现                       │
│     GET /aiops/anomalies              获取异常检测                       │
│     GET /aiops/suggestions            获取优化建议                       │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 前端页面改造（5个页面）

| 页面 | 改动内容 |
|------|----------|
| DeploymentTopologyPage | 替换 mock 数据，调用 `/deploy/topology` |
| MetricsDashboardPage | 替换 mock 数据，调用 `/deploy/metrics/*` |
| AuditLogsPage | 替换 mock 数据，调用 `/deploy/audit-logs` |
| PolicyManagementPage | 替换 mock 数据，调用 `/deploy/policies` |
| AIOpsInsightsPage | 替换 mock 数据，调用 `/aiops/*` |

### 新增数据模型

```go
// DeploymentTopology 部署拓扑
type DeploymentTopology struct {
    Services    []TopologyService `json:"services"`
    Connections []TopologyConnection `json:"connections"`
}

// TopologyService 拓扑服务节点
type TopologyService struct {
    ID             uint   `json:"id"`
    Name           string `json:"name"`
    Environment    string `json:"environment"`
    Status         string `json:"status"`
    LastDeployment string `json:"last_deployment"`
    TargetID       uint   `json:"target_id"`
}

// DeploymentMetric 部署指标
type DeploymentMetric struct {
    Date                 string  `json:"date"`
    DeploymentCount      int     `json:"deployment_count"`
    SuccessCount         int     `json:"success_count"`
    FailureCount         int     `json:"failure_count"`
    SuccessRate          float64 `json:"success_rate"`
    AvgDeploymentTime    float64 `json:"avg_deployment_time"`
}

// AuditLog 审计日志
type AuditLog struct {
    ID          uint                   `json:"id"`
    ActionType  string                 `json:"action_type"`
    ResourceType string                 `json:"resource_type"`
    ResourceID  uint                   `json:"resource_id"`
    ActorID     uint                   `json:"actor_id"`
    ActorName   string                 `json:"actor_name"`
    Detail      map[string]interface{} `json:"detail"`
    CreatedAt   time.Time              `json:"created_at"`
}

// Policy 策略
type Policy struct {
    ID          uint                   `json:"id"`
    Name        string                 `json:"name"`
    Type        string                 `json:"type"` // traffic, resilience, access, slo
    TargetID    uint                   `json:"target_id"`
    Config      map[string]interface{} `json:"config"`
    Enabled     bool                   `json:"enabled"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

// RiskFinding 风险发现
type RiskFinding struct {
    ID          uint     `json:"id"`
    Type        string   `json:"type"`
    Severity    string   `json:"severity"`
    Title       string   `json:"title"`
    Description string   `json:"description"`
    ServiceID   uint     `json:"service_id"`
    ServiceName string   `json:"service_name"`
    CreatedAt   time.Time `json:"created_at"`
}

// Anomaly 异常检测
type Anomaly struct {
    ID          uint     `json:"id"`
    Type        string   `json:"type"`
    Metric      string   `json:"metric"`
    Value       float64  `json:"value"`
    Threshold   float64  `json:"threshold"`
    ServiceID   uint     `json:"service_id"`
    ServiceName string   `json:"service_name"`
    CreatedAt   time.Time `json:"created_at"`
}

// Suggestion 优化建议
type Suggestion struct {
    ID          uint     `json:"id"`
    Type        string   `json:"type"`
    Title       string   `json:"title"`
    Description string   `json:"description"`
    Impact      string   `json:"impact"` // high, medium, low
    ServiceID   uint     `json:"service_id"`
    ServiceName string   `json:"service_name"`
    CreatedAt   time.Time `json:"created_at"`
}
```

## Capabilities

### New Capabilities
- `deployment-topology`: 可视化服务部署拓扑和依赖关系
- `deployment-metrics`: 部署指标统计和趋势分析
- `deployment-audit`: 部署操作审计日志
- `deployment-policies`: 部署策略管理（流量、弹性、访问、SLO）
- `aiops-insights`: AIOps 风险发现、异常检测、优化建议

## Impact

### Frontend
- 修改 5 个页面组件，替换 mock 数据为真实 API 调用
- 新增 API 模块函数
- 更新类型定义

### Backend
- 新增 5 个服务模块（topology, metrics, audit, policies, aiops）
- 新增数据模型和数据库迁移
- 新增路由注册

### Database
- 新增表: `audit_logs`, `policies`, `risk_findings`, `anomalies`, `suggestions`

## Risks

1. **数据量问题** - 审计日志可能数据量大，需要分页和归档策略
2. **AIOps 复杂度** - 风险发现和异常检测可能需要接入监控系统
3. **策略执行** - 策略管理需要与 Kubernetes/Compose 集成

## Phasing

### Phase 1: 基础 API（优先）
- 审计日志 API
- 指标统计 API

### Phase 2: 拓扑和策略
- 部署拓扑 API
- 策略管理 API

### Phase 3: AIOps
- 风险发现 API
- 异常检测 API
- 优化建议 API
