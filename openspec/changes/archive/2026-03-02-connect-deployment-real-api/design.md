# Design: 对接部署管理真实后端 API

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              Frontend Layer                                          │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                     │
│   DeploymentTopologyPage ─────────┐                                                │
│   MetricsDashboardPage ───────────┼──▶ Api.deployment.* / Api.aiops.*              │
│   AuditLogsPage ──────────────────┤                                                │
│   PolicyManagementPage ───────────┤                                                │
│   AIOpsInsightsPage ──────────────┘                                                │
│                                                                                     │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              Backend Layer                                           │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                     │
│   internal/service/                                                                 │
│   ├── deployment/                                                                   │
│   │   ├── topology.go       # 拓扑数据聚合                                          │
│   │   ├── metrics.go        # 指标统计                                              │
│   │   ├── audit.go          # 审计日志                                              │
│   │   └── policy.go         # 策略管理                                              │
│   └── aiops/                                                                        │
│       ├── risk.go           # 风险发现                                              │
│       ├── anomaly.go        # 异常检测                                              │
│       └── suggestion.go     # 优化建议                                              │
│                                                                                     │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              Database Layer                                          │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                     │
│   MySQL/PostgreSQL                                                                  │
│   ├── audit_logs           # 审计日志表                                             │
│   ├── policies             # 策略配置表                                             │
│   ├── risk_findings        # 风险发现表                                             │
│   ├── anomalies            # 异常记录表                                             │
│   └── suggestions          # 优化建议表                                             │
│                                                                                     │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

## API Design

### 1. 部署拓扑 API

```
GET /api/v1/deploy/topology
```

**Response:**
```json
{
  "code": 1000,
  "data": {
    "services": [
      {
        "id": 1,
        "name": "api-gateway",
        "environment": "production",
        "status": "healthy",
        "last_deployment": "2026-03-02T10:00:00Z",
        "target_id": 1
      }
    ],
    "connections": [
      {
        "source_id": 1,
        "target_id": 2,
        "type": "http"
      }
    ]
  }
}
```

**实现逻辑:**
- 从 `deploy_targets` 聚合服务部署信息
- 从 `deploy_releases` 获取最新部署状态
- 可选：从服务发现获取服务依赖关系

### 2. 部署指标 API

```
GET /api/v1/deploy/metrics/summary
GET /api/v1/deploy/metrics/trends?range=daily|weekly|monthly
```

**Response:**
```json
{
  "code": 1000,
  "data": {
    "total_releases": 150,
    "success_rate": 94.5,
    "avg_duration_seconds": 120,
    "by_environment": {
      "production": { "total": 50, "success_rate": 98.0 },
      "staging": { "total": 60, "success_rate": 92.0 },
      "development": { "total": 40, "success_rate": 95.0 }
    }
  }
}
```

**实现逻辑:**
- 从 `deploy_releases` 聚合统计数据
- 按 `state` 字段计算成功率
- 按时间范围分组

### 3. 审计日志 API

```
GET /api/v1/deploy/audit-logs?page=1&page_size=20&action_type=&resource_type=
```

**Response:**
```json
{
  "code": 1000,
  "data": {
    "list": [
      {
        "id": 1,
        "action_type": "release_apply",
        "resource_type": "release",
        "resource_id": 123,
        "actor_id": 1,
        "actor_name": "admin",
        "detail": { "target_name": "prod-cluster" },
        "created_at": "2026-03-02T10:00:00Z"
      }
    ],
    "total": 100
  }
}
```

**实现逻辑:**
- 从 `audit_logs` 表查询
- 支持按操作类型、资源类型过滤
- 关联用户信息获取 actor_name

### 4. 策略管理 API

```
GET    /api/v1/deploy/policies
POST   /api/v1/deploy/policies
PUT    /api/v1/deploy/policies/:id
DELETE /api/v1/deploy/policies/:id
```

**Request Body (POST/PUT):**
```json
{
  "name": "production-traffic-policy",
  "type": "traffic",
  "target_id": 1,
  "config": {
    "max_replicas": 10,
    "min_replicas": 2,
    "target_cpu_utilization": 70
  },
  "enabled": true
}
```

**实现逻辑:**
- CRUD 操作 `policies` 表
- 策略类型: traffic, resilience, access, slo
- 可选：同步到 Kubernetes HPA/Istio 等

### 5. AIOps API

```
GET /api/v1/aiops/risk-findings
GET /api/v1/aiops/anomalies
GET /api/v1/aiops/suggestions
```

**实现逻辑:**
- Phase 3 实现
- 初期可返回基于规则的分析结果
- 后期接入 ML 模型或外部监控系统

## Data Model Design

### audit_logs 表

```sql
CREATE TABLE audit_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    action_type VARCHAR(64) NOT NULL COMMENT '操作类型: release_apply, release_approve, target_create, etc.',
    resource_type VARCHAR(64) NOT NULL COMMENT '资源类型: release, target, cluster, credential',
    resource_id BIGINT UNSIGNED NOT NULL COMMENT '资源ID',
    actor_id BIGINT UNSIGNED NOT NULL COMMENT '操作者ID',
    detail JSON COMMENT '操作详情',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_action_type (action_type),
    INDEX idx_resource (resource_type, resource_id),
    INDEX idx_actor (actor_id),
    INDEX idx_created_at (created_at)
);
```

### policies 表

```sql
CREATE TABLE policies (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL COMMENT 'traffic, resilience, access, slo',
    target_id BIGINT UNSIGNED COMMENT '关联的部署目标ID',
    config JSON COMMENT '策略配置',
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_type (type),
    INDEX idx_target (target_id)
);
```

### risk_findings 表

```sql
CREATE TABLE risk_findings (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    type VARCHAR(64) NOT NULL COMMENT 'risk type',
    severity VARCHAR(16) NOT NULL COMMENT 'critical, high, medium, low',
    title VARCHAR(255) NOT NULL,
    description TEXT,
    service_id BIGINT UNSIGNED,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP NULL,
    INDEX idx_severity (severity),
    INDEX idx_service (service_id)
);
```

### anomalies 表

```sql
CREATE TABLE anomalies (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    type VARCHAR(64) NOT NULL,
    metric VARCHAR(64) NOT NULL,
    value DOUBLE NOT NULL,
    threshold DOUBLE NOT NULL,
    service_id BIGINT UNSIGNED,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP NULL,
    INDEX idx_type (type),
    INDEX idx_service (service_id)
);
```

### suggestions 表

```sql
CREATE TABLE suggestions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    type VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    impact VARCHAR(16) NOT NULL COMMENT 'high, medium, low',
    service_id BIGINT UNSIGNED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    applied_at TIMESTAMP NULL,
    INDEX idx_impact (impact),
    INDEX idx_service (service_id)
);
```

## Frontend Integration

### API Module Updates

```typescript
// web/src/api/modules/deployment.ts 新增

export const deploymentApi = {
  // ... 现有方法 ...

  // 拓扑
  getTopology(params?: { environment?: string }): Promise<ApiResponse<DeploymentTopology>> {
    return apiService.get('/deploy/topology', { params });
  },

  // 指标
  getMetricsSummary(): Promise<ApiResponse<MetricsSummary>> {
    return apiService.get('/deploy/metrics/summary');
  },

  getMetricsTrends(params?: { range?: 'daily' | 'weekly' | 'monthly' }): Promise<ApiResponse<DeploymentMetric[]>> {
    return apiService.get('/deploy/metrics/trends', { params });
  },

  // 审计日志
  getAuditLogs(params?: { action_type?: string; resource_type?: string }): Promise<ApiResponse<PaginatedResponse<AuditLog>>> {
    return apiService.get('/deploy/audit-logs', { params });
  },

  // 策略
  getPolicies(params?: { type?: string; target_id?: number }): Promise<ApiResponse<PaginatedResponse<Policy>>> {
    return apiService.get('/deploy/policies', { params });
  },

  createPolicy(payload: CreatePolicyRequest): Promise<ApiResponse<Policy>> {
    return apiService.post('/deploy/policies', payload);
  },

  updatePolicy(id: number, payload: Partial<CreatePolicyRequest>): Promise<ApiResponse<Policy>> {
    return apiService.put(`/deploy/policies/${id}`, payload);
  },

  deletePolicy(id: number): Promise<ApiResponse<void>> {
    return apiService.delete(`/deploy/policies/${id}`);
  },
};

// web/src/api/modules/aiops.ts 新增
export const aiopsApi = {
  getRiskFindings(): Promise<ApiResponse<PaginatedResponse<RiskFinding>>> {
    return apiService.get('/aiops/risk-findings');
  },

  getAnomalies(): Promise<ApiResponse<PaginatedResponse<Anomaly>>> {
    return apiService.get('/aiops/anomalies');
  },

  getSuggestions(): Promise<ApiResponse<PaginatedResponse<Suggestion>>> {
    return apiService.get('/aiops/suggestions');
  },
};
```

## Error Handling

所有 API 遵循统一的错误响应格式：

```json
{
  "code": 4001,
  "msg": "参数错误",
  "data": null
}
```

前端统一处理：
- 401: 跳转登录
- 403: 显示无权限提示
- 404: 显示资源不存在
- 500: 显示服务器错误提示
