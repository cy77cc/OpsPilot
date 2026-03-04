# Spec: Dashboard Timeseries Charts

时序图表组件，展示 CPU/内存使用率趋势。

## ADDED Requirements

### Requirement: CPU usage chart

系统 SHALL 显示 CPU 使用率时序图表。

#### Scenario: CPU chart rendering
- **WHEN** 用户访问主控台页面
- **THEN** 页面显示 CPU 使用率时序图表
- **AND** 图表使用 @ant-design/charts Line 组件渲染
- **AND** X 轴为时间，Y 轴为使用率百分比

#### Scenario: CPU chart data display
- **WHEN** CPU 数据成功加载
- **THEN** 图表显示指定时间范围内的数据点
- **AND** 图表显示平均值的标注线

### Requirement: Memory usage chart

系统 SHALL 显示内存使用率时序图表。

#### Scenario: Memory chart rendering
- **WHEN** 用户访问主控台页面
- **THEN** 页面显示内存使用率时序图表
- **AND** 图表使用 @ant-design/charts Line 组件渲染
- **AND** X 轴为时间，Y 轴为使用率百分比

#### Scenario: Memory chart data display
- **WHEN** 内存数据成功加载
- **THEN** 图表显示指定时间范围内的数据点
- **AND** 图表显示平均值的标注线

### Requirement: Chart empty state

系统 SHALL 在无数据时显示空状态提示。

#### Scenario: No data available
- **WHEN** 指定时间范围内无数据
- **THEN** 图表区域显示"暂无数据"提示

### Requirement: Chart loading state

系统 SHALL 在数据加载时显示加载状态。

#### Scenario: Data loading
- **WHEN** 数据正在加载中
- **THEN** 图表区域显示骨架屏或加载动画

### Requirement: Chart tooltip

系统 SHALL 在鼠标悬停数据点时显示详细信息。

#### Scenario: Tooltip display
- **WHEN** 用户将鼠标悬停在图表数据点上
- **THEN** 显示工具提示，包含时间和具体数值
