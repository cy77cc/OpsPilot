# OpsPilot 组织与权限中心 (Org & Access Center) 设计文档

## 1. 背景与目标
目前系统的用户认证、用户管理与 RBAC 权限功能相对独立。本项目旨在构建一个统一的“组织与权限中心”，引入“部门/团队 (Department/Team)”概念，实现基于部门的批量授权与个人直接授权相结合的混合模式（Approach C）。

### 核心目标
- **组织结构化**：支持树形部门管理。
- **权限整合**：实现“用户 - 部门 - 角色 - 权限”的完整链路。
- **入口统一**：前端整合用户、部门、角色、权限及个人设置。

## 2. 核心模型设计 (Data Models)

### 2.1 部门模型 (Department)
存储组织架构树。
- `id`: 主键
- `name`: 部门名称 (如: 运维部)
- `parent_id`: 父级部门 ID (根部门为 0)
- `leader_id`: 部门负责人用户 ID
- `status`: 状态 (1=正常, 0=禁用)

### 2.2 部门成员关联 (DepartmentMember)
存储用户与部门的所属关系（支持一人多部门）。
- `user_id`: 用户 ID
- `dept_id`: 部门 ID
- `is_primary`: 是否为主部门 (1=是, 0=否)

### 2.3 部门角色关联 (DepartmentRole)
实现部门级别的权限继承。
- `dept_id`: 部门 ID
- `role_id`: 角色 ID

### 2.4 现有模型调整 (Existing Adjustments)
- **User**: 增加 `dept_id` (冗余字段，指向主部门) 提升查询性能。
- **Permission**: 维持现有 Resource:Action 格式。

## 3. 接口设计 (API Endpoints)

### 3.1 组织管理 (Org Management)
- `GET /api/v1/org/departments/tree`: 获取完整部门树。
- `POST /api/v1/org/departments`: 创建部门。
- `PUT /api/v1/org/departments/:id`: 更新部门信息。
- `DELETE /api/v1/org/departments/:id`: 删除部门 (需检查是否有下级或成员)。

### 3.2 成员管理 (Member Management)
- `POST /api/v1/org/members/transfer`: 跨部门调动人员。
- `GET /api/v1/org/departments/:id/members`: 获取部门下所有成员。

### 3.3 混合权限接口 (Access Control)
- `GET /api/v1/auth/me/permissions`: 
    - **逻辑**: 合并 `User -> Role` 和 `User -> Dept -> Role` 的所有权限。
- `POST /api/v1/org/departments/:id/roles`: 为部门绑定角色。

## 4. 前端整合规划 (Frontend Integration)

### 4.1 统一导航结构
- **管理侧 (Admin)**:
    - 组织视图: 左侧树形部门，右侧成员列表与角色配置。
    - 全局用户管理: 扁平化用户列表。
    - 权限中心: 角色定义与权限点维护。
- **个人侧 (Personal)**:
    - 个人中心: 显示所属部门路径及继承的权限摘要。

## 5. 权限判定逻辑 (Approach C)
```text
UserPermissions = Filter(
    User.Roles.Permissions + 
    Department(User.Dept).Roles.Permissions
)
```

## 6. 实施路线图 (Implementation Phases)
1. **Phase 1**: 数据库迁移，增加 Department 相关表。
2. **Phase 2**: 后端逻辑开发 (部门树、混合权限计算)。
3. **Phase 3**: 前端整合开发 (部门树组件、用户列表重构)。
4. **Phase 4**: 数据清理与旧功能迁移。
