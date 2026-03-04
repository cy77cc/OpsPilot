# Proposal: 主控台实时监控增强

## Why

当前主控台(Dashboard)仅展示静态数字卡片，缺乏实时监控能力和趋势可视化。运维人员打开主控台无法一眼判断系统健康状态，需要逐个页面查看详情。作为运维平台的核心入口，主控台应当提供 Grafana + Datadog 风格的实时监控视图，支持概览 + 下钻的工作模式。

## What Changes

### 后端
- 新增聚合 API `/api/v1/dashboard/overview`，一次性返回系统概览数据
- 返回主机/集群/服务健康状态统计
- 返回 CPU/内存使用率时序数据（最近 1h/6h/24h）
- 返回活跃告警列表（最近 5 条）
- 返回最近事件流（最近 10 条）

### 前端
- 重构 Dashboard 为面板化布局（Grafana 风格）
- 新增时间范围选择器（1h / 6h / 24h）和自动刷新按钮
- 新增主机/集群/服务健康状态卡片（支持点击下钻）
- 新增 CPU/内存使用率时序图表（使用 @ant-design/charts）
- 新增活跃告警面板（显示最近告警，可点击查看详情）
- 新增事件流组件（显示最近系统事件）

### 非功能性
- 保持 60s 轮询刷新（MVP 阶段）
- 预留 WebSocket 推送扩展点

## Capabilities

### New Capabilities

- `dashboard-overview-api`: 后端聚合 API，返回系统概览数据（健康统计 + 时序数据 + 告警 + 事件）
- `dashboard-health-cards`: 主机/集群/服务健康状态卡片组件，支持点击下钻到详情页
- `dashboard-timeseries-charts`: 时序图表组件，展示 CPU/内存使用率趋势
- `dashboard-alert-panel`: 告警面板组件，显示最近活跃告警
- `dashboard-event-stream`: 事件流组件，显示最近系统事件

### Modified Capabilities

无现有 spec 需要修改。

## Impact

### 后端
- 新增 `internal/service/dashboard/` 模块
  - `routes.go` - 路由注册
  - `handler.go` - HTTP 处理
  - `logic.go` - 业务逻辑（聚合 hosts/clusters/services/alerts/metrics 数据）
- 新增 `api/dashboard/v1/dashboard.go` - API 类型定义

### 前端
- 重构 `web/src/pages/Dashboard/Dashboard.tsx`
- 新增 `web/src/api/modules/dashboard.ts` - API 调用
- 新增组件:
  - `web/src/components/Dashboard/HealthCard.tsx`
  - `web/src/components/Dashboard/TimeseriesChart.tsx`
  - `web/src/components/Dashboard/AlertPanel.tsx`
  - `web/src/components/Dashboard/EventStream.tsx`

### 依赖
- 复用现有 `@ant-design/charts` 图表库
- 复用现有 `monitoring.GetMetrics` API
- 复用现有 WebSocket Hub（预留扩展）
