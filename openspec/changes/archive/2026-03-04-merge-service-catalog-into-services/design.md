# Design: 合并服务目录到服务管理

## Context

### 当前状态

系统存在两套服务相关模块：

```
服务管理 (Services)                    服务目录 (Catalog)
────────────────────────              ────────────────────────
Service 表                            ServiceTemplate 表
├── ServiceRevision                   ServiceCategory 表
├── ServiceVariableSet                ├── visibility/status
├── ServiceDeployTarget               ├── 审核流程
└── ServiceReleaseRecord              └── 分类管理

API: /services/*                      API: /catalog/*
前端: ServiceListPage                 前端: CatalogListPage
      ServiceDetailPage                     TemplateListPage
      ServiceProvisionPage                  CatalogDeployPage
                                           ReviewListPage
```

**问题**:
1. 功能重叠: 配置管理、变量系统、部署能力重复
2. 概念混淆: 用户需要理解"模板"和"实例"的区别
3. 维护成本: 两套代码、两套 API、两套测试

### 约束

- 不使用数据库外键，关联关系由业务层处理
- 保持单进程交付模式
- 兼容现有部署流程和发布记录

## Goals / Non-Goals

**Goals:**
- 简化数据模型，删除冗余表
- 统一服务入口，降低用户认知负担
- 支持两类服务: 中间件服务 + 业务服务
- 支持团队可见性控制和跨团队授权

**Non-Goals:**
- 不保留模板审核流程 (简化为直接发布)
- 不保留独立分类表 (改为 service_kind 字段)
- 不迁移现有发布历史数据

## Decisions

### D1: 服务模型设计

**决策**: 扩展现有 Service 表，合并模板能力

**数据模型**:

```sql
-- services 表扩展字段
ALTER TABLE services ADD COLUMN service_kind VARCHAR(16) DEFAULT 'business';
ALTER TABLE services ADD COLUMN visibility VARCHAR(16) DEFAULT 'team';
ALTER TABLE services ADD COLUMN granted_teams JSON;
ALTER TABLE services ADD COLUMN icon VARCHAR(256);
ALTER TABLE services ADD COLUMN tags JSON;
ALTER TABLE services ADD COLUMN deploy_count INT DEFAULT 0;

-- service_kind: middleware / business
-- visibility: private / team / team-granted / public
-- granted_teams: [team_id, ...]
```

**字段说明**:

| 字段 | 类型 | 说明 |
|------|------|------|
| service_kind | ENUM | `middleware` (中间件) / `business` (业务服务) |
| visibility | ENUM | 可见性级别 |
| granted_teams | JSON | 被授权可见的团队 ID 列表 |
| icon | VARCHAR | 服务图标 |
| tags | JSON | 标签 ["mysql", "database"] |
| deploy_count | INT | 部署次数统计 |

### D2: 可见性控制

**决策**: 四级可见性 + 团队授权

```
┌─────────────────────────────────────────────────────────────────┐
│ 可见性级别                                                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ private        仅创建者可见                                     │
│   └─ 适用场景: 草稿阶段，个人测试                               │
│                                                                 │
│ team           团队成员可见                                     │
│   └─ 适用场景: 业务服务默认，团队内部使用                       │
│                                                                 │
│ team-granted   团队 + 被授权团队可见                            │
│   └─ 适用场景: 跨团队共享服务                                   │
│                                                                 │
│ public         全员可见                                         │
│   └─ 适用场景: 中间件服务默认，全员可部署                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**权限矩阵**:

| 操作 | 创建者 | 团队成员 | 被授权团队 | 其他用户 |
|------|--------|----------|-----------|----------|
| 查看配置 | ✅ | ✅ | ✅ | public 时 ✅ |
| 部署服务 | ✅ | ✅ | ✅ | public 时 ✅ |
| 编辑配置 | ✅ | ✅ | ❌ | ❌ |
| 管理可见性 | ✅ | 团队管理员 | ❌ | ❌ |

### D3: 服务类型区分

**决策**: 通过 service_kind 字段区分两类服务

| 服务类型 | service_kind | 默认可见性 | 维护者 |
|---------|--------------|-----------|--------|
| MySQL/Redis/Kafka | middleware | public | 运维团队 |
| Nginx/Prometheus | middleware | public | 运维团队 |
| api-gateway | business | team | 业务团队 |
| user-service | business | team | 业务团队 |

**前端筛选**:
- 服务列表页提供 `service_kind` 筛选器
- 导航菜单提供"中间件服务"和"业务服务"快捷入口

### D4: API 设计

**决策**: 统一到 `/api/v1/services/*`

```
现有 API (保留):
  GET    /services                    -- 服务列表
  GET    /services/:id                -- 服务详情
  POST   /services                    -- 创建服务
  PUT    /services/:id                -- 更新服务
  DELETE /services/:id                -- 删除服务
  POST   /services/:id/deploy         -- 部署服务
  POST   /services/:id/deploy/preview -- 预览渲染
  GET    /services/:id/releases       -- 发布历史
  GET    /services/:id/revisions      -- 版本历史

新增 API:
  PUT    /services/:id/visibility     -- 设置可见性
  PUT    /services/:id/grant-teams    -- 授权团队

删除 API:
  /catalog/*                          -- 整个 catalog 路由组
```

### D5: 数据迁移策略

**决策**: 分步迁移，保证数据完整性

**迁移步骤**:

```
Step 1: 扩展 services 表
  ALTER TABLE services ADD COLUMN service_kind...
  ALTER TABLE services ADD COLUMN visibility...

Step 2: 迁移 service_templates 数据
  INSERT INTO services (name, display_name, ...)
  SELECT name, display_name, ... FROM service_templates;

Step 3: 清理旧表
  DROP TABLE service_templates;
  DROP TABLE service_categories;

Step 4: 清理代码
  删除 catalog 模块代码
```

**回滚策略**:
- 保留 service_templates 和 service_categories 表备份
- 通过 migration down 脚本回滚

### D6: 前端架构

**决策**: 统一到 Services 页面

**页面结构**:

```
web/src/pages/Services/
├── ServiceListPage.tsx       -- 扩展: 支持筛选 service_kind
├── ServiceDetailPage.tsx     -- 扩展: 整合部署、历史功能
├── ServiceProvisionPage.tsx  -- 扩展: 支持设置 service_kind/visibility
├── ServiceDeployPage.tsx     -- 新增: 从 CatalogDeployPage 迁移
└── ServiceVisibilityPage.tsx -- 新增: 可见性设置
```

**删除页面**:
```
web/src/pages/Catalog/  (整个目录删除)
```

## Risks / Trade-offs

### R1: 数据迁移风险

**风险**: 迁移过程中可能丢失数据

**缓解**:
- 迁移前完整备份
- 分步执行，每步验证
- 提供回滚脚本

### R2: API 兼容性

**风险**: 删除 `/catalog/*` API 可能影响现有调用

**缓解**:
- 前端统一使用 `/services/*` API
- 无外部 API 调用者 (内部系统)

### R3: 用户习惯变更

**风险**: 用户习惯了"目录"和"服务"的分离

**缓解**:
- 导航菜单提供平滑过渡
- 文档更新，说明新设计理念

## Migration Plan

### Phase 1: 数据准备

1. 创建数据库迁移脚本
2. 备份 service_templates 和 service_categories 表
3. 在测试环境验证迁移

### Phase 2: 后端变更

1. 扩展 Service model
2. 扩展 Service API
3. 删除 Catalog 模块

### Phase 3: 前端变更

1. 扩展 Services 页面
2. 更新路由和导航
3. 删除 Catalog 页面

### Phase 4: 清理

1. 删除旧数据库表
2. 更新 OpenSpec 规范
3. 更新文档

## File Changes Summary

```
后端变更:
├── 扩展 internal/model/service_studio.go
├── 扩展 api/service/v1/service.go
├── 扩展 internal/service/service/routes.go
├── 扩展 internal/service/service/logic.go
├── 删除 internal/service/catalog/ (整个目录)
└── 删除 api/catalog/ (整个目录)

前端变更:
├── 扩展 web/src/pages/Services/
├── 扩展 web/src/api/modules/services.ts
├── 删除 web/src/pages/Catalog/ (整个目录)
└── 删除 web/src/api/modules/catalog.ts

数据库迁移:
└── storage/migrations/YYYYMMDD_XXXXXX_merge_catalog_to_services.sql

OpenSpec:
├── 删除 specs/service-catalog-* 系列
├── 扩展 specs/service-* 系列
└── 新增 specs/service-visibility-control
```
