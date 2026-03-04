# Design: 主控台实时监控增强

## Context

### 当前状态

主控台 (`web/src/pages/Dashboard/Dashboard.tsx`) 是一个静态展示页面，每 30 秒轮询一次数据：
- 调用 5 个独立的 API 获取数据 (`getHostList`, `getTaskList`, `getClusterList`, `getAlertList`, `getReleases`)
- 仅展示数字卡片 (Statistic) 和简单的表格
- 无时序图表，无趋势分析
- SLO 等指标为硬编码值

### 已有基础设施

```
┌─────────────────────────────────────────────────────────────┐
│ 可复用的现有能力                                             │
├─────────────────────────────────────────────────────────────┤
│ @ant-design/charts ─── 已在 MetricsDashboardPage 使用       │
│ monitoring.GetMetrics ─ 可查询 MetricPoint 时序数据          │
│ WebSocket Hub ──────── 现有推送基础设施（通知）              │
│ MetricPoint 表 ─────── 每 60s 采集全局指标                   │
│ AlertEvent 表 ──────── 告警事件存储                          │
└─────────────────────────────────────────────────────────────┘
```

### 约束

- 不引入新的依赖库（使用现有 @ant-design/charts）
- 不修改数据库 Schema（复用现有表）
- 保持单进程交付模式
- MVP 阶段使用轮询，预留 WebSocket 扩展点

## Goals / Non-Goals

**Goals:**
- 运维人员可一眼判断系统整体健康状态
- 支持分钟级趋势可视化（CPU/内存使用率）
- 支持点击卡片下钻到详情页
- 单次 API 请求获取所有概览数据，减少网络开销

**Non-Goals:**
- 不实现秒级实时推送（MVP 阶段）
- 不实现用户自定义仪表盘布局
- 不实现告警规则配置（使用现有告警管理页面）
- 不实现多租户数据隔离（统一视图）

## Decisions

### D1: 后端聚合 API vs 前端多请求

**决策**: 新增 `/api/v1/dashboard/overview` 聚合 API

**理由**:
- 单次请求减少网络往返，提升首屏加载速度
- 后端可统一做缓存优化
- 前端代码更简洁，减少并发请求管理复杂度

**备选方案**:
- ❌ 前端并发请求多个 API: 请求多，首屏慢，难以统一缓存

### D2: 数据聚合策略

**决策**: 后端逻辑层聚合，复用现有 Service 方法

```
┌─────────────────────────────────────────────────────────────┐
│                    Dashboard Logic                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  GetOverview(ctx, timeRange)                                │
│  ├── aggregateHostStats() ──▶ Node 表查询                   │
│  ├── aggregateClusterStats() ─▶ Cluster 表查询              │
│  ├── aggregateServiceStats() ▶ ServiceReleaseRecord 表查询  │
│  ├── getRecentAlerts() ──────▶ AlertEvent 表查询 (firing)   │
│  ├── getRecentEvents() ──────▶ NodeEvent/AlertEvent 聚合    │
│  └── getMetricsSeries() ─────▶ MetricPoint 表查询           │
│       ├── cpu_usage (最近 1h/6h/24h)                        │
│       └── memory_usage (最近 1h/6h/24h)                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**理由**:
- 复用现有表结构，无需 Migration
- 逻辑集中在 `internal/service/dashboard/logic.go`
- 后续可扩展缓存层

### D3: 健康状态判定规则

**决策**: 基于现有字段计算健康状态

| 资源类型 | 健康 | 降级 | 异常 |
|---------|------|------|------|
| 主机 | `status=online` 且 `health_state=healthy` | `status=online` 且 `health_state=degraded` | `status!=online` 或 `health_state=critical` |
| 集群 | `status IN (connected,ready,active)` | `status=degraded` | `status NOT IN (connected,ready,active,degraded)` |
| 服务 | 最近 24h 发布成功率 ≥ 95% | 成功率 80-95% | 成功率 < 80% 或有失败发布 |

**理由**:
- 不引入新字段，基于现有数据计算
- 服务健康状态基于发布成功率（简化方案）

### D4: 前端组件架构

**决策**: 面板化组件，每个面板独立可测试

```
web/src/components/Dashboard/
├── HealthCard.tsx        # 健康状态卡片（可点击下钻）
├── TimeseriesChart.tsx   # 时序图表（复用 @ant-design/charts Line）
├── AlertPanel.tsx        # 告警面板（显示最近 5 条）
├── EventStream.tsx       # 事件流（显示最近 10 条）
└── TimeRangeSelector.tsx # 时间范围选择器
```

**理由**:
- 组件独立，便于测试和复用
- 符合项目现有组件组织方式
- 参考 `MetricsDashboardPage.tsx` 的图表实现

### D5: 刷新策略

**决策**: 60 秒轮询 + 手动刷新

**理由**:
- MVP 阶段保持简单
- 后端 MetricPoint 每 60s 采集一次，对齐刷新间隔
- 预留 WebSocket 推送扩展点（后续可升级）

**备选方案**:
- ❌ WebSocket 实时推送: 需要扩展 Hub，增加复杂度，MVP 暂不需要

## Risks / Trade-offs

### R1: 聚合 API 性能

**风险**: 聚合多个数据源可能导致响应慢

**缓解**:
- 各查询并行执行 (`errgroup.Group`)
- 限制时序数据点数量（最多 60 个点）
- 后续可引入 Redis 缓存（5 分钟 TTL）

### R2: 服务健康状态准确性

**风险**: 基于发布成功率判断服务健康，不够精确

**缓解**:
- MVP 阶段简化方案
- 后续可接入 K8s Pod 状态、健康检查探针等

### R3: 事件数据来源

**风险**: 系统事件分散在多个表（NodeEvent, AlertEvent 等）

**缓解**:
- MVP 阶段合并 NodeEvent 和 AlertEvent（按时间排序取前 10）
- 后续可考虑统一的事件总线

## API Contract

### GET /api/v1/dashboard/overview

**Query Parameters**:
- `time_range`: `1h` | `6h` | `24h` (默认 `1h`)

**Response**:
```json
{
  "hosts": {
    "total": 15,
    "healthy": 12,
    "degraded": 2,
    "offline": 1
  },
  "clusters": {
    "total": 5,
    "healthy": 4,
    "unhealthy": 1
  },
  "services": {
    "total": 32,
    "healthy": 28,
    "degraded": 2,
    "unhealthy": 2
  },
  "alerts": {
    "firing": 3,
    "recent": [
      { "id": "1", "title": "CPU 超过 85%", "severity": "warning", "source": "node-01", "createdAt": "..." }
    ]
  },
  "events": [
    { "id": "1", "type": "host_online", "message": "node-05 上线", "createdAt": "..." }
  ],
  "metrics": {
    "cpu_usage": [
      { "timestamp": "2024-01-15T10:00:00Z", "value": 45.2 },
      { "timestamp": "2024-01-15T10:01:00Z", "value": 48.1 }
    ],
    "memory_usage": [
      { "timestamp": "2024-01-15T10:00:00Z", "value": 62.5 }
    ]
  }
}
```

## File Changes Summary

```
后端新增:
├── api/dashboard/v1/dashboard.go      # 类型定义
└── internal/service/dashboard/
    ├── routes.go                       # 路由注册
    ├── handler.go                      # HTTP Handler
    └── logic.go                        # 业务逻辑

前端修改/新增:
├── web/src/api/modules/dashboard.ts   # API 调用
├── web/src/pages/Dashboard/Dashboard.tsx  # 重构
└── web/src/components/Dashboard/
    ├── HealthCard.tsx
    ├── TimeseriesChart.tsx
    ├── AlertPanel.tsx
    ├── EventStream.tsx
    └── TimeRangeSelector.tsx
```
