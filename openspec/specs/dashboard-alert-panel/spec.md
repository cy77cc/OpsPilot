# Spec: Dashboard Alert Panel

告警面板组件，显示最近活跃告警。

## Requirements

### Requirement: Alert panel display

系统 SHALL 显示活跃告警面板，展示最近的告警事件。

#### Scenario: Alert panel rendering
- **WHEN** 用户访问主控台页面
- **THEN** 页面显示活跃告警面板
- **AND** 面板标题为"活跃告警"
- **AND** 面板显示告警总数徽章

### Requirement: Alert severity visualization

系统 SHALL 使用颜色编码区分告警严重级别。

#### Scenario: Critical alert display
- **WHEN** 告警严重级别为 `critical`
- **THEN** 告警条目显示红色图标和背景

#### Scenario: Warning alert display
- **WHEN** 告警严重级别为 `warning`
- **THEN** 告警条目显示黄色图标和背景

#### Scenario: Info alert display
- **WHEN** 告警严重级别为 `info`
- **THEN** 告警条目显示蓝色图标和背景

### Requirement: Alert list display

系统 SHALL 显示最多 5 条最近告警，包含标题、严重级别、来源。

#### Scenario: Alert list content
- **WHEN** 有活跃告警
- **THEN** 列表显示最近 5 条告警
- **AND** 每条告警显示标题、严重级别标签、来源信息

#### Scenario: No alerts
- **WHEN** 无活跃告警
- **THEN** 面板显示"暂无告警"的空状态

### Requirement: Alert drill-down

系统 SHALL 支持点击告警条目跳转到告警详情。

#### Scenario: Alert item click
- **WHEN** 用户点击告警条目
- **THEN** 系统导航到告警详情页 (`/monitoring/alerts`)

### Requirement: View all alerts

系统 SHALL 提供"查看全部"链接跳转到告警管理页面。

#### Scenario: View all link
- **WHEN** 用户点击"查看全部"链接
- **THEN** 系统导航到告警管理页面 (`/monitoring/alerts`)
