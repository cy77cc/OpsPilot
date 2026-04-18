# 修复主机添加时的唯一索引冲突 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 允许手动添加多个主机，同时保留云主机导入的唯一性约束。

**Architecture:** 通过修改 GORM 模型标签引入 PostgreSQL 的部分索引 (Partial Index)，并执行必要的数据清洗和索引重建。

**Tech Stack:** Go, GORM, PostgreSQL.

---

### Task 1: 修改主机模型定义

**Files:**
- Modify: `internal/modules/host/model/node.go`

- [ ] **Step 1: 修改 Provider 和 ProviderID 的 GORM 标签**

增加 `where` 子句到 `uniqueIndex:idx_provider_instance`。

```go
// internal/modules/host/model/node.go

// 修改第 45-46 行
Provider   *string `gorm:"column:provider;type:varchar(32);uniqueIndex:idx_provider_instance,where:provider IS NOT NULL AND provider != ''" json:"provider"`
ProviderID *string `gorm:"column:provider_instance_id;type:varchar(128);uniqueIndex:idx_provider_instance,where:provider IS NOT NULL AND provider != ''" json:"provider_instance_id"`
```

- [ ] **Step 2: 提交代码**

```bash
git add internal/modules/host/model/node.go
git commit -m "feat(host): implement partial index for provider unique constraint"
```

---

### Task 2: 实现数据迁移与索引重建

由于 GORM `AutoMigrate` 可能不会自动应用索引的 `where` 条件变更，我们需要手动处理索引重建。

**Files:**
- Modify: `internal/bootstrap/migrations.go`

- [ ] **Step 1: 增加索引重建逻辑**

在 `RunBootstrapMigrations` 中增加对 PostgreSQL 的特殊处理。

```go
// internal/bootstrap/migrations.go

// 在 RunBootstrapMigrations 函数末尾，RunDevAutoMigrate 之后增加：
func fixHostUniqueIndex(db *gorm.DB) error {
	// 1. 清理存量空字符串数据
	if err := db.Exec("UPDATE nodes SET provider = NULL WHERE provider = ''").Error; err != nil {
		return err
	}
	if err := db.Exec("UPDATE nodes SET provider_instance_id = NULL WHERE provider_instance_id = ''").Error; err != nil {
		return err
	}

	// 2. 重建索引
	// 先删除旧索引（如果存在且不是部分索引）
	if err := db.Exec("DROP INDEX IF EXISTS idx_provider_instance").Error; err != nil {
		return err
	}

	// 3. 创建部分索引
	return db.Exec(`
		CREATE UNIQUE INDEX idx_provider_instance 
		ON nodes(provider, provider_instance_id) 
		WHERE provider IS NOT NULL AND provider != ''
	`).Error
}
```

- [ ] **Step 2: 调用修复函数**

```go
// internal/bootstrap/migrations.go

func RunBootstrapMigrations() error {
    // ... 原有逻辑
    if config.CFG.App.AutoMigrate {
        if err := migration.RunDevAutoMigrate(db); err != nil {
            return fmt.Errorf("run dev auto migrate failed: %w", err)
        }
        // 增加此行
        if err := fixHostUniqueIndex(db); err != nil {
            return fmt.Errorf("fix host unique index failed: %w", err)
        }
    }
    return nil
}
```

- [ ] **Step 3: 提交代码**

```bash
git add internal/bootstrap/migrations.go
git commit -m "fix(host): add migration to recreate provider unique index as partial index"
```

---

### Task 3: 验证修复效果

**Files:**
- Create: `internal/modules/host/logic/repro_fix_test.go`

- [ ] **Step 1: 编写验证测试**

```go
package logic

import (
	"context"
	"testing"
	model "github.com/cy77cc/OpsPilot/internal/modules/host/model"
)

func TestFixCreateMultipleManualHosts(t *testing.T) {
	s, db := newHostLogicTestService(t)
    // 模拟迁移逻辑
    if err := db.Exec("DROP INDEX IF EXISTS idx_provider_instance").Error; err != nil {
        t.Fatal(err)
    }
    if err := db.Exec(`CREATE UNIQUE INDEX idx_provider_instance ON nodes(provider, provider_instance_id) WHERE provider IS NOT NULL AND provider != ''`).Error; err != nil {
        t.Fatal(err)
    }

	ctx := context.Background()

	// 1. 创建第一个手动主机
	req1 := CreateReq{
		Name:     "Host1",
		IP:       "192.168.1.1",
		Port:     22,
		Username: "root",
		Source:   "manual_ssh",
	}
	_, err := s.CreateWithProbe(ctx, 1, true, req1)
	if err != nil {
		t.Fatalf("Failed to create first manual host: %v", err)
	}

	// 2. 创建第二个手动主机 (之前会报错)
	req2 := CreateReq{
		Name:     "Host2",
		IP:       "192.168.1.2",
		Port:     22,
		Username: "root",
		Source:   "manual_ssh",
	}
	_, err = s.CreateWithProbe(ctx, 1, true, req2)
	if err != nil {
		t.Fatalf("Failed to create second manual host: %v", err)
	}

    // 3. 验证云主机依然有约束
    provider := "aliyun"
    insID := "i-12345"
    node1 := &model.Node{
        Name: "Cloud1",
        IP: "1.1.1.1",
        Provider: &provider,
        ProviderID: &insID,
        Status: "online",
    }
    if err := db.Create(node1).Error; err != nil {
        t.Fatal(err)
    }

    node2 := &model.Node{
        Name: "Cloud2",
        IP: "1.1.1.2",
        Provider: &provider,
        ProviderID: &insID,
        Status: "online",
    }
    err = db.Create(node2).Error
    if err == nil {
        t.Fatal("Expected unique constraint violation for duplicate cloud hosts")
    }
}
```

- [ ] **Step 2: 运行测试**

Run: `go test -v ./internal/modules/host/logic/ -run TestFixCreateMultipleManualHosts`
Expected: PASS

- [ ] **Step 3: 清理临时测试文件**

```bash
rm internal/modules/host/logic/reproduce_issue_test.go
rm internal/modules/host/logic/repro_fix_test.go
```

- [ ] **Step 4: 提交**

```bash
git add .
git commit -m "test(host): verify fix for host unique constraint"
```
