# Tasks: 合并服务目录到服务管理

## 1. 数据库迁移

- [x] 1.1 创建扩展 services 表的迁移脚本
  - 新增 `service_kind` 字段 (ENUM: middleware, business)
  - 新增 `visibility` 字段 (ENUM: private, team, team-granted, public)
  - 新增 `granted_teams` 字段 (JSON)
  - 新增 `icon` 字段 (VARCHAR)
  - 新增 `tags` 字段 (JSON)
  - 新增 `deploy_count` 字段 (INT)

- [x] 1.2 创建数据迁移脚本
  - 迁移 `service_templates` 数据到 `services` 表
  - 设置 `service_kind` (根据分类推断)
  - 设置 `visibility` (根据现有状态推断)

- [x] 1.3 创建清理迁移脚本
  - 删除 `service_templates` 表
  - 删除 `service_categories` 表

## 2. 后端 Model 扩展

- [x] 2.1 扩展 `internal/model/service_studio.go` 或创建新 model
  - 添加 `ServiceKind` 字段
  - 添加 `Visibility` 字段
  - 添加 `GrantedTeams` 字段
  - 添加 `Icon` 字段
  - 添加 `Tags` 字段
  - 添加 `DeployCount` 字段

- [x] 2.2 删除 `internal/model/service_template.go`
- [x] 2.3 删除 `internal/model/service_category.go`

## 3. 后端 API 扩展

- [x] 3.1 扩展 `api/service/v1/service.go`
  - 添加 `ServiceKind` 类型
  - 添加 `Visibility` 类型
  - 扩展 `ServiceCreateReq` 添加新字段
  - 扩展 `ServiceResponse` 添加新字段
  - 新增 `VisibilityUpdateReq`
  - 新增 `GrantTeamsReq`

- [x] 3.2 删除 `api/catalog/v1/` 目录

## 4. 后端 Logic 扩展

- [x] 4.1 扩展 `internal/service/service/logic.go`
  - 添加可见性检查逻辑
  - 添加授权团队检查逻辑
  - 添加服务类型筛选逻辑

- [x] 4.2 新增可见性管理逻辑
  - `UpdateVisibility` 方法
  - `UpdateGrantedTeams` 方法
  - `CheckViewPermission` 方法
  - `CheckEditPermission` 方法

- [x] 4.3 删除 `internal/service/catalog/` 目录

## 5. 后端 Handler 扩展

- [x] 5.1 扩展 `internal/service/service/handler.go`
  - 新增 `UpdateVisibility` Handler
  - 新增 `UpdateGrantedTeams` Handler
  - 扩展 `List` 支持筛选 service_kind

## 6. 后端路由扩展

- [x] 6.1 扩展 `internal/service/service/routes.go`
  - 添加 `PUT /services/:id/visibility` 路由
  - 添加 `PUT /services/:id/grant-teams` 路由

- [x] 6.2 删除 catalog 模块注册
  - 从 `internal/service/service.go` 移除 catalog 注册

## 7. 前端 API 模块

- [x] 7.1 扩展 `web/src/api/modules/services.ts`
  - 添加 `ServiceKind`, `Visibility` 类型
  - 扩展 `ServiceItem` 类型
  - 新增 `updateVisibility` 方法
  - 新增 `updateGrantedTeams` 方法

- [x] 7.2 删除 `web/src/api/modules/catalog.ts`

## 8. 前端页面扩展

- [x] 8.1 扩展 `web/src/pages/Services/ServiceListPage.tsx`
  - 添加 service_kind 筛选器 (中间件服务/业务服务)
  - 更新列表项显示 icon 和 tags

- [x] 8.2 扩展 `web/src/pages/Services/ServiceDetailPage.tsx`
  - 显示 service_kind 和 visibility
  - 添加"可见性设置"入口
  - 整合部署功能

- [x] 8.3 扩展 `web/src/pages/Services/ServiceProvisionPage.tsx`
  - 添加 service_kind 选择
  - 添加 visibility 选择
  - 支持 icon 上传/选择

- [x] 8.4 迁移 `CatalogDeployPage` 功能到 Services
  - 创建或扩展 `ServiceDeployPage.tsx`

- [x] 8.5 新增 `ServiceVisibilityPage.tsx`
  - 可见性设置界面
  - 授权团队管理界面

## 9. 前端页面删除

- [x] 9.1 删除 `web/src/pages/Catalog/` 目录及所有文件
  - CatalogListPage.tsx
  - CatalogDetailPage.tsx
  - CatalogDeployPage.tsx
  - TemplateListPage.tsx
  - TemplateCreatePage.tsx
  - TemplateEditPage.tsx
  - ReviewListPage.tsx
  - CategoryManagePage.tsx
  - 相关测试文件

## 10. 前端路由更新

- [x] 10.1 更新路由配置
  - 删除 `/catalog/*` 路由
  - 添加 `/services/:id/deploy` 路由
  - 添加 `/services/:id/visibility` 路由

- [x] 10.2 更新导航菜单
  - 删除"服务目录"菜单项
  - 更新"服务管理"子菜单
    - 全部服务
    - 中间件服务
    - 业务服务

## 11. 测试更新

- [x] 11.1 更新后端测试
  - 删除 catalog 模块测试
  - 扩展 service 模块测试 (可见性、授权)

- [x] 11.2 更新前端测试
  - 删除 Catalog 页面测试
  - 扩展 Services 页面测试

## 12. OpenSpec 更新

- [x] 12.1 删除旧 specs
  - 删除 `specs/service-catalog-browse`
  - 删除 `specs/service-catalog-deploy`
  - 删除 `specs/service-template-management`
  - 删除 `specs/service-template-review`
  - 删除 `specs/service-category-management`

- [x] 12.2 更新现有 specs
  - 扩展服务管理相关 spec

- [x] 12.3 归档旧变更
  - 归档 `add-service-catalog` 变更

## 13. 文档更新

- [x] 13.1 更新 `docs/platform-vision-gap-analysis.md`
- [x] 13.2 更新 API 文档
- [x] 13.3 运行 `openspec validate` 验证
