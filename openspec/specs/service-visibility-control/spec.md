# Spec: Service Visibility Control

服务可见性控制，支持团队授权。

## ADDED Requirements

### Requirement: Service visibility levels

系统 SHALL 支持四级可见性设置。

#### Scenario: Private visibility
- **WHEN** 服务可见性设置为 `private`
- **THEN** 仅创建者可查看和操作该服务

#### Scenario: Team visibility
- **WHEN** 服务可见性设置为 `team`
- **THEN** 团队成员可查看和操作该服务

#### Scenario: Team-granted visibility
- **WHEN** 服务可见性设置为 `team-granted`
- **THEN** 团队成员和被授权团队成员可查看该服务
- **AND** 仅本团队成员可编辑该服务

#### Scenario: Public visibility
- **WHEN** 服务可见性设置为 `public`
- **THEN** 所有用户可查看该服务
- **AND** 仅团队成员可编辑该服务

### Requirement: Grant teams visibility

系统 SHALL 支持授权其他团队可见。

#### Scenario: Grant team access
- **WHEN** 团队管理员授权其他团队可见
- **THEN** 被授权团队的成员可查看该服务

#### Scenario: Revoke team access
- **WHEN** 团队管理员移除团队授权
- **THEN** 被移除团队的成员无法再查看该服务

### Requirement: Service kind default visibility

系统 SHALL 根据服务类型设置默认可见性。

#### Scenario: Middleware service default
- **WHEN** 创建 `middleware` 类型服务
- **THEN** 默认可见性为 `public`

#### Scenario: Business service default
- **WHEN** 创建 `business` 类型服务
- **THEN** 默认可见性为 `team`

### Requirement: Visibility API

系统 SHALL 提供可见性设置 API。

#### Scenario: Update visibility
- **WHEN** 团队管理员调用 `PUT /services/:id/visibility`
- **THEN** 系统更新服务可见性

#### Scenario: Update granted teams
- **WHEN** 团队管理员调用 `PUT /services/:id/grant-teams`
- **THEN** 系统更新授权团队列表

### Requirement: Visibility permission check

系统 SHALL 校验可见性操作权限。

#### Scenario: Non-admin cannot change visibility
- **WHEN** 非团队管理员尝试修改可见性
- **THEN** 系统拒绝并提示"无权限修改可见性"

### Requirement: Service list filtering

系统 SHALL 支持按服务类型筛选。

#### Scenario: Filter by service kind
- **WHEN** 用户在服务列表选择 `middleware`
- **THEN** 系统仅显示 `service_kind=middleware` 的服务

#### Scenario: Filter by visibility
- **WHEN** 用户在服务列表选择可见性筛选
- **THEN** 系统显示用户有权查看的服务
