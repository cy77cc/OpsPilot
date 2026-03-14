# 主控台可观测性增强设计

## 概述

本设计文档描述 OpsPilot 主控台可观测性增强方案，包括数据源修正、新增数据模型、API 设计和前端布局调整。

## 目标

1. 修正 AI 活动卡片数据源错误
2. 增加集群资源使用监控（CPU/内存/Pod）
3. 增加运行状态统计（部署/CI/K8s 异常）
4. 增加工作负载健康概览
5. 丰富事件流（增加 K8s 事件、部署事件）

## 约束

- 避免频繁调用 K8s API，使用缓存层
- 数据延迟控制在 1-5 分钟内
- 保持现有 API 兼容性

## 架构

```
┌────────────────────────────────────────────────────────────────┐
│                     展示层 (Dashboard.tsx)                      │
└────────────────────────────────────────────────────────────────┘
                              ▲
┌────────────────────────────────────────────────────────────────┐
│                     聚合层 (dashboard/logic.go)                 │
└────────────────────────────────────────────────────────────────┘
                              ▲
┌──────────────┬──────────────┬──────────────┬──────────────────┐
│   主机数据    │   集群数据    │   应用数据    │    AI 数据       │
└──────────────┴──────────────┴──────────────┴──────────────────┘
                              ▲
┌──────────────┬──────────────┬──────────────┬──────────────────┐
│  数据库表     │  缓存表       │  Prometheus  │    K8s API       │
│  (现有)      │  (新增)       │  (现有)      │   (定时同步)     │
└──────────────┴──────────────┴──────────────┴──────────────────┘
```

## 数据模型

### 新增表：cluster_resource_snapshots

存储集群资源使用快照，每 5 分钟采集一次。

```sql
CREATE TABLE cluster_resource_snapshots (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cluster_id INT UNSIGNED NOT NULL,
  cpu_allocatable_cores DECIMAL(10,2) NOT NULL DEFAULT 0,
  cpu_requested_cores DECIMAL(10,2) NOT NULL DEFAULT 0,
  cpu_limit_cores DECIMAL(10,2) NOT NULL DEFAULT 0,
  cpu_usage_cores DECIMAL(10,2) NOT NULL DEFAULT 0,
  memory_allocatable_mb BIGINT NOT NULL DEFAULT 0,
  memory_requested_mb BIGINT NOT NULL DEFAULT 0,
  memory_limit_mb BIGINT NOT NULL DEFAULT 0,
  memory_usage_mb BIGINT NOT NULL DEFAULT 0,
  pod_total INT NOT NULL DEFAULT 0,
  pod_running INT NOT NULL DEFAULT 0,
  pod_pending INT NOT NULL DEFAULT 0,
  pod_failed INT NOT NULL DEFAULT 0,
  pv_count INT NOT NULL DEFAULT 0,
  pvc_count INT NOT NULL DEFAULT 0,
  storage_used_gb DECIMAL(10,2) NOT NULL DEFAULT 0,
  collected_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_cluster_collected (cluster_id, collected_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 新增表：k8s_workload_stats

存储 K8s 工作负载统计，每 1 分钟采集一次。

```sql
CREATE TABLE k8s_workload_stats (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cluster_id INT UNSIGNED NOT NULL,
  namespace VARCHAR(128) NOT NULL DEFAULT '',
  deployment_total INT NOT NULL DEFAULT 0,
  deployment_healthy INT NOT NULL DEFAULT 0,
  statefulset_total INT NOT NULL DEFAULT 0,
  statefulset_healthy INT NOT NULL DEFAULT 0,
  daemonset_total INT NOT NULL DEFAULT 0,
  daemonset_healthy INT NOT NULL DEFAULT 0,
  service_count INT NOT NULL DEFAULT 0,
  ingress_count INT NOT NULL DEFAULT 0,
  collected_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_cluster_ns_collected (cluster_id, namespace, collected_at),
  INDEX idx_collected (collected_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 新增表：k8s_issue_pods

缓存异常 Pod 列表，用于快速展示问题 Pod。

```sql
CREATE TABLE k8s_issue_pods (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cluster_id INT UNSIGNED NOT NULL,
  namespace VARCHAR(128) NOT NULL,
  pod_name VARCHAR(256) NOT NULL,
  issue_type VARCHAR(64) NOT NULL,
  issue_reason VARCHAR(256) NOT NULL,
  message TEXT,
  first_seen_at TIMESTAMP NOT NULL,
  last_seen_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_cluster_ns_pod (cluster_id, namespace, pod_name),
  INDEX idx_issue_type (issue_type),
  INDEX idx_last_seen (last_seen_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## API 设计

### GET /api/v1/dashboard/overview

返回主控台所有数据的聚合响应。

#### 响应结构

```typescript
interface OverviewResponse {
  // 健康概览
  health: {
    hosts: HealthStats;        // 现有
    clusters: HealthStats;     // 增强：结合 cluster_nodes 状态
    applications: HealthStats; // 修改：基于 deployment_releases
    workloads: WorkloadStats;  // 新增：K8s 工作负载
  };

  // 资源使用
  resources: {
    hosts: TimeseriesData[];      // 现有：主机 CPU/内存时序
    clusters: ClusterResource[];  // 新增：集群资源概览
  };

  // 运行状态（新增）
  operations: {
    deployments: DeploymentStats;
    cicd: CICDStats;
    issue_pods: IssuePodStats;
  };

  // 告警事件
  alerts: {
    firing: number;
    recent: AlertItem[];
  };

  // 事件流（增强）
  events: EventItem[];

  // AI 活动（修正数据源）
  ai: AIActivity;
}

interface HealthStats {
  total: number;
  healthy: number;
  degraded: number;
  unhealthy: number;
  offline?: number;
}

interface WorkloadStats {
  deployments: { total: number; healthy: number };
  statefulsets: { total: number; healthy: number };
  daemonsets: { total: number; healthy: number };
  services: number;
  ingresses: number;
}

interface ClusterResource {
  cluster_id: number;
  cluster_name: string;
  cpu: {
    allocatable: number;
    requested: number;
    usage: number;
    usage_percent: number;
  };
  memory: {
    allocatable: number;
    requested: number;
    usage: number;
    usage_percent: number;
  };
  pods: {
    total: number;
    running: number;
    pending: number;
    failed: number;
  };
}

interface DeploymentStats {
  running: number;      // 正在部署
  pending_approval: number;
  today_total: number;
  today_success: number;
  today_failed: number;
}

interface CICDStats {
  running: number;
  queued: number;
  today_total: number;
  today_success: number;
  today_failed: number;
}

interface IssuePodStats {
  total: number;
  by_type: Record<string, number>;  // CrashLoopBackOff, ImagePullBackOff, etc.
}

interface EventItem {
  id: string;
  type: string;      // host_alert / k8s_event / deployment_event / ci_event
  severity: string;  // info / warning / error
  source: string;    // 来源标识
  message: string;
  created_at: string;
}

interface AIActivity {
  stats: {
    session_count: number;
    token_count: number;
    avg_duration_ms: number;
    success_rate: number;
  };
  by_scene: Record<string, number>;
  recent_sessions: AISessionItem[];
}
```

## 数据采集

### 采集任务设计

| 任务名称 | 采集频率 | 数据源 | 目标表 |
|----------|----------|--------|--------|
| ClusterResourceCollector | 每 5 分钟 | K8s API + Prometheus | cluster_resource_snapshots |
| WorkloadStatsCollector | 每 1 分钟 | K8s API | k8s_workload_stats |
| IssuePodCollector | 每 30 秒 | K8s API | k8s_issue_pods |

### 采集器实现位置

```
internal/service/dashboard/
├── collector/
│   ├── cluster_resource.go    # 集群资源采集
│   ├── workload_stats.go      # 工作负载统计采集
│   ├── issue_pods.go          # 异常 Pod 采集
│   └── scheduler.go           # 定时调度器
```

### 采集逻辑

#### ClusterResourceCollector

```go
func (c *ClusterResourceCollector) Collect(ctx context.Context, clusterID uint) error {
    // 1. 获取集群客户端
    client, err := c.getK8sClient(clusterID)

    // 2. 查询 Node 资源
    nodes, _ := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
    // 计算 allocatable CPU/Memory

    // 3. 查询 Pod 资源请求
    pods, _ := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
    // 累加 requests/limits

    // 4. 从 Prometheus 查询实际使用
    cpuUsage := c.queryPrometheus("sum(rate(container_cpu_usage_seconds_total[5m]))")
    memUsage := c.queryPrometheus("sum(container_memory_working_set_bytes)")

    // 5. 写入快照表
    return c.db.Create(&ClusterResourceSnapshot{...}).Error
}
```

## 后端实现

### 修改文件清单

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `internal/model/dashboard.go` | 新增 | 新增 3 个数据模型 |
| `internal/service/dashboard/logic.go` | 修改 | 重构 GetOverview 方法 |
| `internal/service/dashboard/collector/*.go` | 新增 | 数据采集器 |
| `internal/service/dashboard/routes.go` | 不变 | 路由已存在 |
| `api/dashboard/v1/dashboard.go` | 修改 | 新增响应类型定义 |
| `storage/migrations/*.sql` | 新增 | 新表迁移文件 |

### logic.go 重构要点

```go
func (l *Logic) GetOverview(ctx context.Context, timeRange string) (*OverviewResponse, error) {
    // 使用 errgroup 并行获取各维度数据
    group, gctx := errgroup.WithContext(ctx)

    // 1. 健康概览（并行）
    group.Go(func() error { return l.collectHealth(gctx, out) })

    // 2. 资源使用（并行）
    group.Go(func() error { return l.collectResources(gctx, out) })

    // 3. 运行状态（并行）
    group.Go(func() error { return l.collectOperations(gctx, out) })

    // 4. 告警事件（并行）
    group.Go(func() error { return l.collectAlerts(gctx, out) })

    // 5. 事件流（并行）
    group.Go(func() error { return l.collectEvents(gctx, out) })

    // 6. AI 活动（并行，修正数据源）
    group.Go(func() error { return l.collectAIActivity(gctx, out) })

    return out, group.Wait()
}
```

### AI 活动数据源修正

```go
func (l *Logic) collectAIActivity(ctx context.Context, out *OverviewResponse) error {
    // 修正：从 ai_executions 表读取
    var stats struct {
        SessionCount int64
        TokenCount   int64
        SuccessCount int64
        TotalCount   int64
    }

    // 会话数从 ai_chat_sessions 统计
    l.db.Model(&model.AIChatSession{}).Count(&stats.SessionCount)

    // Token 数从 ai_executions 汇总
    l.db.Model(&model.AIExecution{}).
        Select("COALESCE(SUM(total_tokens), 0)").
        Scan(&stats.TokenCount)

    // 成功率从 ai_executions 计算
    l.db.Model(&model.AIExecution{}).Count(&stats.TotalCount)
    l.db.Model(&model.AIExecution{}).Where("status = ?", "success").Count(&stats.SuccessCount)

    // ...
}
```

## 前端实现

### 修改文件清单

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `web/src/pages/Dashboard/Dashboard.tsx` | 修改 | 布局调整、新增卡片 |
| `web/src/components/Dashboard/*.tsx` | 新增 | 新增组件 |
| `web/src/api/modules/dashboard.ts` | 修改 | 新增类型定义 |
| `web/src/components/Dashboard/WorkloadHealthCard.tsx` | 新增 | 工作负载健康卡片 |
| `web/src/components/Dashboard/ClusterResourceCard.tsx` | 新增 | 集群资源卡片 |
| `web/src/components/Dashboard/OperationsCard.tsx` | 新增 | 运行状态卡片 |

### 布局调整

```tsx
// Dashboard.tsx
<Row gutter={[16, 16]}>
  {/* 健康概览 - 4 列 */}
  <Col xs={24} sm={12} lg={6}>
    <HealthCard title="主机健康" data={health.hosts} />
  </Col>
  <Col xs={24} sm={12} lg={6}>
    <HealthCard title="集群健康" data={health.clusters} />
  </Col>
  <Col xs={24} sm={12} lg={6}>
    <HealthCard title="应用健康" data={health.applications} />
  </Col>
  <Col xs={24} sm={12} lg={6}>
    <WorkloadHealthCard data={health.workloads} />
  </Col>
</Row>

<Row gutter={[16, 16]}>
  {/* 资源使用 - 2 列 */}
  <Col xs={24} xl={12}>
    <TimeseriesChart title="主机资源使用率" series={resources.hosts} />
  </Col>
  <Col xs={24} xl={12}>
    <ClusterResourceCard data={resources.clusters} />
  </Col>
</Row>

<Row gutter={[16, 16]}>
  {/* 运行状态 + 告警 + AI - 3 列 */}
  <Col xs={24} md={8}>
    <OperationsCard data={operations} />
  </Col>
  <Col xs={24} md={8}>
    <AlertPanel alerts={alerts} />
  </Col>
  <Col xs={24} md={8}>
    <AIActivityCard data={ai} />
  </Col>
</Row>

<Row gutter={[16, 16]}>
  {/* 事件流 - 全宽 */}
  <Col xs={24}>
    <EventStream events={events} />
  </Col>
</Row>
```

## 迁移计划

### 阶段 1：数据源修正（优先级高）

1. 修改 `getAIActivity` 方法，从正确的表读取数据
2. 前端无需改动

### 阶段 2：新增数据模型和采集器

1. 创建迁移文件
2. 实现采集器
3. 启动定时任务

### 阶段 3：API 增强

1. 修改 OverviewResponse 结构
2. 实现新的聚合逻辑

### 阶段 4：前端更新

1. 更新组件
2. 调整布局

## 测试要点

1. **采集器测试**：验证定时任务正常执行，数据正确写入
2. **API 测试**：验证各维度数据正确聚合
3. **前端测试**：验证 UI 渲染正确，无数据时优雅降级
4. **性能测试**：验证并发查询不影响响应时间

## 监控指标

- `opspilot_dashboard_overview_duration_seconds`：Overview API 响应时间
- `opspilot_collector_duration_seconds`：采集任务执行时间
- `opspilot_collector_errors_total`：采集任务错误次数
