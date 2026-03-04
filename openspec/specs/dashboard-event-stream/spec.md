# Spec: Dashboard Event Stream

事件流组件，显示最近系统事件。

## Requirements

### Requirement: Event stream display

系统 SHALL 显示事件流组件，展示最近的系统事件。

#### Scenario: Event stream rendering
- **WHEN** 用户访问主控台页面
- **THEN** 页面显示事件流组件
- **AND** 组件标题为"最近事件"
- **AND** 事件按时间降序排列

### Requirement: Event type visualization

系统 SHALL 使用图标区分不同类型的事件。

#### Scenario: Host event display
- **WHEN** 事件类型与主机相关 (如 `host_online`, `host_offline`)
- **THEN** 事件条目显示主机图标

#### Scenario: Cluster event display
- **WHEN** 事件类型与集群相关
- **THEN** 事件条目显示集群图标

#### Scenario: Deployment event display
- **WHEN** 事件类型与部署相关
- **THEN** 事件条目显示部署图标

### Requirement: Event list display

系统 SHALL 显示最多 10 条最近事件，包含类型、消息、时间。

#### Scenario: Event list content
- **WHEN** 有系统事件
- **THEN** 列表显示最近 10 条事件
- **AND** 每条事件显示类型图标、消息内容、相对时间（如"5 分钟前"）

#### Scenario: No events
- **WHEN** 无系统事件
- **THEN** 组件显示"暂无事件"的空状态

### Requirement: Event relative time

系统 SHALL 使用相对时间格式显示事件发生时间。

#### Scenario: Recent event time
- **WHEN** 事件发生在 1 小时内
- **THEN** 显示"X 分钟前"

#### Scenario: Today event time
- **WHEN** 事件发生在今天但超过 1 小时
- **THEN** 显示"X 小时前"

#### Scenario: Older event time
- **WHEN** 事件发生在昨天或更早
- **THEN** 显示具体日期时间

### Requirement: View all events

系统 SHALL 提供"查看全部"链接。

#### Scenario: View all link
- **WHEN** 用户点击"查看全部"链接
- **THEN** 系统导航到事件/审计日志页面
