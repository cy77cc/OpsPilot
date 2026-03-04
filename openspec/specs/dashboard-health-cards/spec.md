# Spec: Dashboard Health Cards

主机/集群/服务健康状态卡片组件，支持点击下钻到详情页。

## Requirements

### Requirement: Health card display

系统 SHALL 显示主机、集群、服务三个健康状态卡片。

#### Scenario: Health cards rendering
- **WHEN** 用户访问主控台页面
- **THEN** 页面显示三个健康状态卡片：主机、集群、服务
- **AND** 每个卡片显示总量、健康数量、健康百分比

### Requirement: Health status visualization

系统 SHALL 使用进度条和颜色编码展示健康状态。

#### Scenario: Healthy status display
- **WHEN** 健康百分比 >= 90%
- **THEN** 进度条显示绿色
- **AND** 显示勾选图标

#### Scenario: Degraded status display
- **WHEN** 健康百分比在 70-90%
- **THEN** 进度条显示黄色
- **AND** 显示警告图标

#### Scenario: Unhealthy status display
- **WHEN** 健康百分比 < 70%
- **THEN** 进度条显示红色
- **AND** 显示错误图标

### Requirement: Health card drill-down

系统 SHALL 支持点击健康卡片跳转到对应的详情页。

#### Scenario: Host card click
- **WHEN** 用户点击主机健康卡片
- **THEN** 系统导航到主机列表页 (`/hosts`)

#### Scenario: Cluster card click
- **WHEN** 用户点击集群健康卡片
- **THEN** 系统导航到集群列表页 (`/k8s/clusters`)

#### Scenario: Service card click
- **WHEN** 用户点击服务健康卡片
- **THEN** 系统导航到服务列表页 (`/services`)

### Requirement: Health card hover effect

系统 SHALL 在鼠标悬停时显示卡片阴影效果，提示可点击。

#### Scenario: Hover feedback
- **WHEN** 用户将鼠标悬停在健康卡片上
- **THEN** 卡片显示阴影效果
- **AND** 鼠标指针变为手型
