# 组织与权限中心 (Org Center) 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建统一的组织架构与权限管理系统，支持部门层级、成员划拨及“部门+个人”混合授权模式。

**Architecture:** 采用 Approach C (混合模式)，通过新增 Department、DepartmentMember 和 DepartmentRole 模型，重构权限加载逻辑，整合前端管理入口。

**Tech Stack:** Go, GIN, GORM, Casbin (适配器层级), TailwindCSS (前端).

---

### Task 1: 数据库模型定义 (Models)

**Files:**
- Modify: `internal/modules/user/model/model.go`
- Modify: `internal/core/storage/migration/dev_auto.go`

- [ ] **Step 1: 在 `internal/modules/user/model/model.go` 中增加部门相关模型**

```go
// Department 是部门表模型。
type Department struct {
	ID         UserID `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"column:name;type:varchar(64);not null" json:"name"`
	ParentID   int64  `gorm:"column:parent_id;not null;default:0" json:"parent_id"`
	LeaderID   int64  `gorm:"column:leader_id;not null;default:0" json:"leader_id"`
	Status     int8   `gorm:"column:status;not null;default:1" json:"status"`
	CreateTime int64  `gorm:"column:create_time;not null;default:0;autoCreateTime" json:"create_time"`
	UpdateTime int64  `gorm:"column:update_time;not null;default:0;autoUpdateTime" json:"update_time"`
}

func (Department) TableName() string { return "departments" }

// DepartmentMember 存储用户与部门关联。
type DepartmentMember struct {
	ID     UserID `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID int64  `gorm:"column:user_id;not null;index" json:"user_id"`
	DeptID int64  `gorm:"column:dept_id;not null;index" json:"dept_id"`
}

func (DepartmentMember) TableName() string { return "department_members" }

// DepartmentRole 存储部门与角色关联。
type DepartmentRole struct {
	ID     UserID `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DeptID int64  `gorm:"column:dept_id;not null;index" json:"dept_id"`
	RoleID int64  `gorm:"column:role_id;not null;index" json:"role_id"`
}

func (DepartmentRole) TableName() string { return "department_roles" }
```

- [ ] **Step 2: 更新 `internal/core/storage/migration/dev_auto.go` 注册新模型**
- [ ] **Step 3: 运行并验证迁移**
Run: `go run cmd/opspilot/main.go migrate`

---

### Task 2: 组织架构 DAO 实现 (DAO)

**Files:**
- Create: `internal/modules/user/dao/dept.go`

- [ ] **Step 1: 实现 DepartmentDAO，包含 Tree 查询和基本 CRUD**
- [ ] **Step 2: 编写测试用例验证部门树递归查询**

---

### Task 3: 核心逻辑：混合权限加载 (Business Logic)

**Files:**
- Modify: `internal/modules/user/logic/auth.go`

- [ ] **Step 1: 重构 `loadRolesAndPermissions` 逻辑**
    - 输入: userID
    - 步骤 1: 加载用户直接关联的角色。
    - 步骤 2: 加载用户所属部门关联的角色。
    - 步骤 3: 合并去重角色和权限。
- [ ] **Step 2: 编写单元测试验证混合权限计算的准确性**

---

### Task 4: 组织管理逻辑与处理器 (Logic & Handler)

**Files:**
- Create: `internal/modules/user/logic/org.go`
- Create: `internal/modules/user/handler/org.go`
- Modify: `internal/modules/user/api/routes.go`

- [ ] **Step 1: 实现 OrgLogic (部门树、成员划拨逻辑)**
- [ ] **Step 2: 实现 OrgHandler 并注册路由 `/api/v1/org/*`**

---

### Task 5: 前端整合入口 (Frontend)

**Files:**
- Modify: `web/src/layout/index.tsx` (假设路径)
- Create: `web/src/views/org/index.tsx`

- [ ] **Step 1: 在导航栏整合“组织中心”入口**
- [ ] **Step 2: 实现部门树与成员列表的联动展示**
