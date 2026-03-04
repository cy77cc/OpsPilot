# Spec: Dashboard Overview API

后端聚合 API，返回系统概览数据。

## Requirements

### Requirement: Overview API endpoint

系统 SHALL 提供 `/api/v1/dashboard/overview` API 端点，返回系统概览数据。

#### Scenario: Successful overview request
- **WHEN** 用户发起 `GET /api/v1/dashboard/overview` 请求
- **THEN** 系统返回 200 状态码和完整的概览数据

#### Scenario: Unauthorized request
- **WHEN** 未认证用户发起请求
- **THEN** 系统返回 401 状态码

### Requirement: Time range parameter

系统 SHALL 支持 `time_range` 查询参数，值为 `1h`、`6h` 或 `24h`。

#### Scenario: Default time range
- **WHEN** 用户未指定 `time_range` 参数
- **THEN** 系统使用默认值 `1h`

#### Scenario: Custom time range
- **WHEN** 用户指定 `time_range=6h`
- **THEN** 系统返回最近 6 小时的时序数据

### Requirement: Host statistics

系统 SHALL 返回主机健康状态统计，包含 `total`、`healthy`、`degraded`、`offline` 数量。

#### Scenario: Host statistics calculation
- **WHEN** 系统计算主机统计
- **THEN** `healthy` 为 `status=online` 且 `health_state=healthy` 的主机数量
- **AND** `degraded` 为 `status=online` 且 `health_state=degraded` 的主机数量
- **AND** `offline` 为 `status!=online` 的主机数量

### Requirement: Cluster statistics

系统 SHALL 返回集群健康状态统计，包含 `total`、`healthy`、`unhealthy` 数量。

#### Scenario: Cluster statistics calculation
- **WHEN** 系统计算集群统计
- **THEN** `healthy` 为 `status IN (connected, ready, active)` 的集群数量
- **AND** `unhealthy` 为其他状态的集群数量

### Requirement: Service statistics

系统 SHALL 返回服务健康状态统计，基于最近 24 小时发布成功率计算。

#### Scenario: Service healthy calculation
- **WHEN** 系统计算服务统计
- **THEN** `healthy` 为最近 24h 发布成功率 >= 95% 的服务数量
- **AND** `degraded` 为成功率 80-95% 的服务数量
- **AND** `unhealthy` 为成功率 < 80% 或有失败发布的服务数量

### Requirement: Recent alerts

系统 SHALL 返回最近 5 条 `status=firing` 的活跃告警。

#### Scenario: Recent alerts list
- **WHEN** 系统返回告警列表
- **THEN** 列表包含最多 5 条告警
- **AND** 每条告警包含 `id`、`title`、`severity`、`source`、`createdAt` 字段

### Requirement: Recent events

系统 SHALL 返回最近 10 条系统事件，合并 NodeEvent 和 AlertEvent 数据源。

#### Scenario: Recent events list
- **WHEN** 系统返回事件列表
- **THEN** 列表包含最多 10 条事件
- **AND** 事件按 `createdAt` 降序排列
- **AND** 每条事件包含 `id`、`type`、`message`、`createdAt` 字段

### Requirement: Metrics series

系统 SHALL 返回 `cpu_usage` 和 `memory_usage` 时序数据。

#### Scenario: CPU metrics series
- **WHEN** 系统返回 CPU 指标
- **THEN** 数据包含指定时间范围内的 `timestamp` 和 `value`
- **AND** 数据点数量不超过 60 个

#### Scenario: Memory metrics series
- **WHEN** 系统返回内存指标
- **THEN** 数据包含指定时间范围内的 `timestamp` 和 `value`
- **AND** 数据点数量不超过 60 个
