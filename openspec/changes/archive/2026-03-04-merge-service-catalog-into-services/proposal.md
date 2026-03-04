# Proposal: 合并服务目录到服务管理

## Why

当前服务目录 (Catalog) 和服务管理 (Services) 是两个独立的模块，存在功能重叠和概念混淆：
- 两者都有配置管理、变量系统、部署能力
- 用户需要理解"模板"和"实例"的区别，增加认知负担
- 维护两套代码和 API 增加系统复杂度

设计理念：**服务即制品** - 每个服务都是可复用、可部署的制品，无需区分模板和实例。

## What Changes

### 数据模型合并

- **删除表**: `service_templates`, `service_categories`
- **扩展 `services` 表**:
  - 新增 `service_kind` 字段: `middleware` (中间件) / `business` (业务服务)
  - 新增 `visibility` 字段: `private` / `team` / `team-granted` / `public`
  - 新增 `granted_teams` 字段: JSON 数组，存储被授权可见的团队 ID
  - 新增 `icon`, `tags`, `deploy_count` 字段

### 服务分类简化

- 中间件服务 (`middleware`): MySQL/Redis/Kafka 等，运维团队维护，全员可见
- 业务服务 (`business`): api-gateway/user-service 等，业务团队维护，团队可见

### 可见性控制

- `private`: 仅创建者可见
- `team`: 团队成员可见
- `team-granted`: 团队 + 被授权团队可见
- `public`: 全员可见 (middleware 默认)

### API 合并

- 删除 `/api/v1/catalog/*` 路由
- 扩展 `/api/v1/services/*` 统一入口
- 新增 `PUT /services/:id/visibility` 设置可见性
- 新增 `PUT /services/:id/grant-teams` 授权团队

### 前端合并

- 删除 `web/src/pages/Catalog/` 目录
- 扩展 `web/src/pages/Services/` 页面
- 服务列表支持 `service_kind` 筛选
- 服务详情页整合配置、部署、历史功能

### 后端模块合并

- 删除 `internal/service/catalog/` 模块
- 扩展 `internal/service/service/` 模块
- 删除 `api/catalog/v1/` 类型定义

## Capabilities

### New Capabilities

- `service-visibility-control`: 服务可见性控制，支持团队授权

### Modified Capabilities

- `service-management`: 扩展服务模型，新增 service_kind、visibility、granted_teams
- `service-catalog-browse`: 合并到服务列表，删除独立能力
- `service-catalog-deploy`: 合并到服务部署，删除独立能力
- `service-template-management`: 合并到服务管理，删除独立能力
- `service-template-review`: 删除，不再需要审核流程
- `service-category-management`: 删除，改为 service_kind 字段

## Impact

### 数据迁移

- 新增迁移脚本: 扩展 services 表字段
- 数据迁移: 将 service_templates 数据迁移到 services 表
- 清理: 删除 service_templates 和 service_categories 表

### 后端变更

- 删除 `internal/service/catalog/` 目录
- 扩展 `internal/model/service_studio.go` (或创建新的 service model)
- 扩展 `api/service/v1/service.go` 类型定义
- 更新 `internal/service/service/routes.go` 路由

### 前端变更

- 删除 `web/src/pages/Catalog/` 目录
- 删除 `web/src/api/modules/catalog.ts`
- 扩展 `web/src/pages/Services/` 页面
- 更新 `web/src/api/modules/services.ts`
- 更新路由和导航菜单

### OpenSpec 更新

- 删除 `specs/service-catalog-*` 系列规范
- 扩展 `specs/service-*` 相关规范
- 归档 `add-service-catalog` 变更
