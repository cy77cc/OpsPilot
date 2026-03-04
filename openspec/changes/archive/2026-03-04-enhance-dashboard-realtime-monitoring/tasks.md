# Tasks: 主控台实时监控增强

## 1. 后端 API 类型定义

- [x] 1.1 创建 `api/dashboard/v1/dashboard.go`，定义请求/响应类型
  - `OverviewRequest`: time_range 参数
  - `OverviewResponse`: hosts/clusters/services/alerts/events/metrics 结构
  - `HealthStats`: total/healthy/degraded/unhealthy/offline 字段
  - `AlertItem`: id/title/severity/source/createdAt
  - `EventItem`: id/type/message/createdAt
  - `MetricPoint`: timestamp/value

## 2. 后端业务逻辑

- [x] 2.1 创建 `internal/service/dashboard/logic.go`
  - 实现 `GetOverview(ctx, timeRange)` 方法
  - 实现 `aggregateHostStats()` - 查询 Node 表计算健康统计
  - 实现 `aggregateClusterStats()` - 查询 Cluster 表计算健康统计
  - 实现 `aggregateServiceStats()` - 查询 ServiceReleaseRecord 计算成功率
  - 实现 `getRecentAlerts()` - 查询 AlertEvent 表 (status=firing, limit=5)
  - 实现 `getRecentEvents()` - 合并 NodeEvent 和 AlertEvent (limit=10)
  - 实现 `getMetricsSeries()` - 查询 MetricPoint 表 (cpu_usage, memory_usage)

- [x] 2.2 使用 errgroup 并行执行各聚合方法，优化响应时间

## 3. 后端 HTTP Handler

- [x] 3.1 创建 `internal/service/dashboard/handler.go`
  - 实现 `GetOverview` Handler
  - 解析 time_range 查询参数
  - 调用 Logic 层方法
  - 返回 JSON 响应

- [x] 3.2 创建 `internal/service/dashboard/routes.go`
  - 注册 `GET /api/v1/dashboard/overview` 路由
  - 应用 JWT 认证中间件

## 4. 前端 API 模块

- [x] 4.1 创建 `web/src/api/modules/dashboard.ts`
  - 定义 TypeScript 类型 (`OverviewResponse`, `HealthStats`, etc.)
  - 实现 `getOverview(timeRange)` API 调用方法

## 5. 前端组件 - 基础组件

- [x] 5.1 创建 `web/src/components/Dashboard/TimeRangeSelector.tsx`
  - 1h / 6h / 24h 选择器
  - 自动刷新按钮
  - 回调 onChange(timeRange)

## 6. 前端组件 - 健康卡片

- [x] 6.1 创建 `web/src/components/Dashboard/HealthCard.tsx`
  - 接收 title/data/onClick 属性
  - 显示总量、健康数量、健康百分比
  - 使用 Progress 组件展示进度条
  - 根据健康百分比显示不同颜色和图标
  - 支持点击跳转

- [x] 6.2 添加悬停阴影效果，提示可点击

## 7. 前端组件 - 时序图表

- [x] 7.1 创建 `web/src/components/Dashboard/TimeseriesChart.tsx`
  - 使用 @ant-design/charts Line 组件
  - 接收 title/data 属性
  - X 轴时间，Y 轴数值
  - 空状态显示"暂无数据"
  - 加载状态显示骨架屏

- [x] 7.2 添加工具提示 (tooltip) 配置

## 8. 前端组件 - 告警面板

- [x] 8.1 创建 `web/src/components/Dashboard/AlertPanel.tsx`
  - 显示活跃告警数量徽章
  - 列表展示最近 5 条告警
  - 根据严重级别显示不同颜色标签
  - 空状态显示"暂无告警"

- [x] 8.2 添加"查看全部"链接跳转到告警管理页

## 9. 前端组件 - 事件流

- [x] 9.1 创建 `web/src/components/Dashboard/EventStream.tsx`
  - 列表展示最近 10 条事件
  - 根据事件类型显示不同图标
  - 使用相对时间格式（如"5 分钟前"）
  - 空状态显示"暂无事件"

- [x] 9.2 添加"查看全部"链接

## 10. 前端页面重构

- [x] 10.1 重构 `web/src/pages/Dashboard/Dashboard.tsx`
  - 移除旧的 5 个 API 调用，改用 `getOverview`
  - 添加时间范围选择器
  - 添加 60 秒自动刷新 (useInterval)
  - 组合使用新的组件 (HealthCard, TimeseriesChart, AlertPanel, EventStream)

- [x] 10.2 实现面板化布局
  - 第一行: 主机/集群/服务健康卡片
  - 第二行: CPU/内存时序图表
  - 第三行: 告警面板 + 事件流

## 11. 测试

- [x] 11.1 后端单元测试 `internal/service/dashboard/logic_test.go`
  - 测试各聚合方法
  - 测试健康状态计算逻辑

- [x] 11.2 前端组件测试
  - HealthCard 组件测试
  - TimeseriesChart 组件测试

## 12. 文档更新

- [x] 12.1 更新 API 文档 (如有)
- [x] 12.2 运行 `openspec validate` 确保变更规范
