# CMDB 增强版模块实施计划 (CMDB Enhanced Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现一个具备身份对齐、权属治理、四层分层及树状延迟展开能力的生产级 CMDB。

**Architecture:** 采用 PostgreSQL + JSONB 作为存储，引入身份映射表实现多源合并。业务逻辑层解耦接入契约（Ingestion DTO），并提供高性能的树状和局部拓扑查询 API。

**Tech Stack:** Go (Gin, GORM), PostgreSQL, JSONB.

---

### Task 1: 数据库模型升级 (Database Schema Upgrade)

**Files:**
- Modify: `internal/modules/cmdb/model/model.go`

- [ ] **Step 1: 更新 `CMDBCI` 模型**
添加环境、地域、权属元数据等字段。
```go
type CMDBCI struct {
	ID           uint           `gorm:"primaryKey;column:id" json:"id"`
	CIUID        string         `gorm:"column:ci_uid;type:varchar(160);not null;uniqueIndex" json:"ci_uid"`
	CIType       string         `gorm:"column:ci_type;type:varchar(64);not null;index" json:"ci_type"`
	Name         string         `gorm:"column:name;type:varchar(128);not null;index" json:"name"`
	Status       string         `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
	Env          string         `gorm:"column:env;type:varchar(32);index" json:"env"`
	Region       string         `gorm:"column:region;type:varchar(64);index" json:"region"`
	ProjectID    uint           `gorm:"column:project_id;default:0;index" json:"project_id"`
	TeamID       uint           `gorm:"column:team_id;default:0;index" json:"team_id"`
	OwnerID      uint           `gorm:"column:owner_id;default:0;index" json:"owner_id"`
	SourceMain   string         `gorm:"column:source_main;type:varchar(64);index" json:"source_main"`
	LastSeenAt   *time.Time     `gorm:"column:last_seen_at" json:"last_seen_at,omitempty"`
	FirstSeenAt  time.Time      `gorm:"column:first_seen_at;autoCreateTime" json:"first_seen_at"`
	AttrsJSON    string         `gorm:"column:attrs_json;type:text" json:"attrs_json"`
	AttrMetaJSON string         `gorm:"column:attr_meta_json;type:text" json:"attr_meta_json"` // 存储属性权属
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}
```

- [ ] **Step 2: 添加 `CMDBIdentity` 模型**
用于多源身份映射。
```go
type CMDBIdentity struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`
	CIID       uint      `gorm:"column:ci_id;not null;index" json:"ci_id"`
	Source     string    `gorm:"column:source;type:varchar(64);not null;index:idx_source_extid,priority:1" json:"source"`
	ExternalID string    `gorm:"column:external_id;type:varchar(160);not null;index:idx_source_extid,priority:2" json:"external_id"`
	Confidence float64   `gorm:"column:confidence;default:1.0" json:"confidence"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}
```

- [ ] **Step 3: 更新 `CMDBRelation` 模型**
添加来源、状态、置信度。
```go
type CMDBRelation struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`
	FromCIID     uint      `gorm:"column:from_ci_id;not null;index:idx_rel_from_to,priority:1" json:"from_ci_id"`
	ToCIID       uint      `gorm:"column:to_ci_id;not null;index:idx_rel_from_to,priority:2" json:"to_ci_id"`
	RelationType string    `gorm:"column:relation_type;type:varchar(64);not null;index:idx_rel_from_to,priority:3" json:"relation_type"`
	Source       string    `gorm:"column:source;type:varchar(64);default:'discovery'" json:"source"`
	Status       string    `gorm:"column:status;type:varchar(32);default:'active'" json:"status"`
	Confidence   float64   `gorm:"column:confidence;default:1.0" json:"confidence"`
	LastSeenAt   *time.Time `gorm:"column:last_seen_at" json:"last_seen_at"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}
```

- [ ] **Step 4: 执行数据库迁移测试**
确保 GORM AutoMigrate 正常工作。
Run: `go test ./internal/modules/cmdb/model/...`

- [ ] **Step 5: Commit**
```bash
git add internal/modules/cmdb/model/model.go
git commit -m "feat(cmdb): update database models for v2.0"
```

---

### Task 2: 标准接入契约与 Match & Merge 逻辑 (Ingestion Pipeline)

**Files:**
- Modify: `internal/modules/cmdb/logic/logic.go`

- [ ] **Step 1: 定义 `IngestionDTO` 结构**
在 `logic.go` 中定义通用的接入数据结构。
```go
type IngestionDTO struct {
	Source     string            `json:"source"`
	CIType     string            `json:"ci_type"`
	ExternalID string            `json:"external_id"`
	Name       string            `json:"name"`
	Env        string            `json:"env"`
	Region     string            `json:"region"`
	Attributes map[string]any    `json:"attributes"`
	Relations  []RelationDTO     `json:"relations"`
}

type RelationDTO struct {
	TargetType  string `json:"target_type"`
	TargetExtID string `json:"target_ext_id"`
	Type        string `json:"type"`
}
```

- [ ] **Step 2: 实现 `Ingest` 核心方法**
实现 Match (身份对齐) ➔ Merge (属性合并) ➔ Relation Sync。
需要处理 `attr_meta` 中的 `Managed` 属性。

- [ ] **Step 3: 编写 TDD 测试用例**
测试多源身份合并和属性保护。
Test: `internal/modules/cmdb/logic/ingest_test.go`

- [ ] **Step 4: Commit**
```bash
git add internal/modules/cmdb/logic/logic.go
git commit -m "feat(cmdb): implement ingestion pipeline and match & merge logic"
```

---

### Task 3: 树状导航与局部拓扑 API (Topology APIs)

**Files:**
- Modify: `api/cmdb/v1/cmdb.go`
- Modify: `internal/modules/cmdb/logic/logic.go`
- Modify: `internal/modules/cmdb/handler/handler.go`
- Modify: `internal/modules/cmdb/api/routes.go`

- [ ] **Step 1: 更新 API 模型定义**
添加 `TreeResp` 和 `SubgraphResp`。

- [ ] **Step 2: 实现 `GetTree` 逻辑**
支持 `parent_id` 逐级展开，按四层模型逻辑返回节点。

- [ ] **Step 3: 实现 `GetSubgraph` 逻辑**
支持给定 `root_id` 展开 `depth` 层的局部拓扑。

- [ ] **Step 4: 注册路由并测试接口**
注册 `/api/v1/cmdb/tree` 和 `/api/v1/cmdb/topology/subgraph`。
Run: `curl http://localhost:8080/api/v1/cmdb/tree`

- [ ] **Step 5: Commit**
```bash
git add api/cmdb/v1/cmdb.go internal/modules/cmdb/...
git commit -m "feat(cmdb): add tree-navigation and subgraph topology APIs"
```

---

### Task 4: 生命周期管理与清理逻辑 (LCM & Cleanup)

**Files:**
- Modify: `internal/modules/cmdb/logic/logic.go`

- [ ] **Step 1: 实现 `Mark & Sweep` 清理逻辑**
基于 `last_seen_at` 标记过期资产。

- [ ] **Step 2: 编写测试验证老化规则**
验证不同类型的资源（Host vs Service）按不同频率老化。

- [ ] **Step 3: Commit**
```bash
git add internal/modules/cmdb/logic/logic.go
git commit -m "feat(cmdb): implement life cycle management sweep logic"
```
