# AI 模型配置管理实现计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现管理员通过数据库配置多个 LLM 供应商和模型，统一管理全局默认模型，支持 OpenClaw 格式 JSON 导入。

**Architecture:** 数据层 (model → dao → logic) + 注册模式 chatmodel 工厂 + 管理员 API。优先读取数据库配置，无配置时回退到 config.yaml。

**Tech Stack:** Go 1.26, Gin, GORM, CloudWeGo Eino

## Execution Guardrails (Must-Have)

- 必须将 AI 运行时选模入口从 `NewChatModel` 统一切换到 DB-aware 入口（`GetDefaultChatModel`），否则“数据库优先”不会生效。
- 必须实现 API Key 加密落库与解密使用链路（`internal/utils/secret.go` + `config.CFG.Security.EncryptionKey`），禁止明文持久化。
- 管理员路由必须接入仓库真实注册入口 `internal/service/service.go`，不能使用不存在的 `server/server.go`。
- 鉴权必须使用仓库已有中间件能力（`JWTAuth` + `CasbinAuth`），不要引用不存在的 `RequireAdmin`。

---

## File Structure

```
internal/
├── model/
│   └── llm_provider.go           # 新增：AILLMProvider 模型定义（扁平结构）
├── dao/ai/
│   └── llm_provider_dao.go       # 新增：模型配置 DAO
├── ai/chatmodel/
│   ├── registry.go               # 新增：注册模式核心
│   ├── qwen.go                   # 新增：Qwen 工厂
│   ├── ark.go                    # 新增：Ark 工厂
│   └── ollama.go                 # 新增：Ollama 工厂
├── service/ai/
│   ├── handler/
│   │   └── llm_provider_handler.go  # 新增：模型管理 API Handler
│   ├── logic/
│   │   ├── llm_provider_logic.go    # 新增：模型配置业务逻辑
│   │   └── llm_provider_crypto.go   # 新增：API Key 加解密封装
│   └── routes.go                 # 修改：注册新路由
└── xcode/
    └── code.go                   # 修改：新增错误码 4010-4014

internal/service/service.go         # 修改：接入 RegisterAdminAIHandlers
internal/ai/agents/**/*.go          # 修改：统一使用 GetDefaultChatModel

storage/migrations/
└── 20260324_0001_create_ai_llm_providers.sql  # 新增：数据库迁移文件
```

---

## Chunk 1: 数据层

### Task 1: 数据库迁移文件

**Files:**
- Create: `storage/migrations/20260324_0001_create_ai_llm_providers.sql`

- [ ] **Step 1: 创建数据库迁移文件**

```sql
-- storage/migrations/20260324_0001_create_ai_llm_providers.sql
-- +migrate Up
CREATE TABLE ai_llm_providers (
  id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  name                VARCHAR(64) NOT NULL COMMENT '显示名称，如 "Qwen3.5-Plus"',
  provider            VARCHAR(32) NOT NULL COMMENT '供应商类型: qwen/ark/ollama',
  model               VARCHAR(128) NOT NULL COMMENT '模型标识，如 "qwen3.5-plus"',
  base_url            VARCHAR(512) NOT NULL COMMENT 'API 端点',
  api_key             VARCHAR(256) NOT NULL COMMENT 'API 密钥（加密存储）',
  api_key_version     INT DEFAULT 1 COMMENT '密钥加密版本，用于密钥轮换',
  temperature         DECIMAL(3,2) DEFAULT 0.70 COMMENT '默认温度参数',
  thinking            TINYINT(1) DEFAULT 0 COMMENT '是否启用思考模式: 0-否 1-是',
  is_default          TINYINT(1) DEFAULT 0 COMMENT '是否为默认模型: 0-否 1-是',
  is_enabled          TINYINT(1) DEFAULT 1 COMMENT '是否启用: 0-禁用 1-启用',
  sort_order          INT DEFAULT 0 COMMENT '排序权重，越大越靠前',
  config_version      INT DEFAULT 1 COMMENT '配置版本号，用于缓存失效',
  created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at          TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
  UNIQUE KEY uk_provider_model (provider, model),
  INDEX idx_enabled_sort (is_enabled, sort_order DESC),
  INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='LLM模型配置表';

-- +migrate Down
DROP TABLE IF EXISTS ai_llm_providers;
```

- [ ] **Step 2: Commit**

```bash
git add storage/migrations/20260324_0001_create_ai_llm_providers.sql
git commit -m "feat(ai): add migration for ai_llm_providers table"
```

### Task 2: Model 定义

**Files:**
- Create: `internal/model/llm_provider.go`
- Create: `internal/model/llm_provider_test.go`

- [ ] **Step 1: 写失败测试**

```go
// internal/model/llm_provider_test.go
package model

import (
    "testing"
    "time"
    "gorm.io/gorm"
)

func TestAILLMProvider_TableName(t *testing.T) {
    p := AILLMProvider{}
    if p.TableName() != "ai_llm_providers" {
        t.Errorf("expected table name 'ai_llm_providers', got '%s'", p.TableName())
    }
}

func TestAILLMProvider_Fields(t *testing.T) {
    p := AILLMProvider{
        ID:            1,
        Name:          "Qwen3.5-Plus",
        Provider:      "qwen",
        Model:         "qwen3.5-plus",
        BaseURL:       "https://api.example.com/v1",
        APIKey:        "sk-test",
        Temperature:   0.7,
        Thinking:      true,
        IsDefault:     true,
        IsEnabled:     true,
        SortOrder:     100,
        ConfigVersion: 1,
    }
    if p.Name != "Qwen3.5-Plus" {
        t.Errorf("expected Name 'Qwen3.5-Plus', got '%s'", p.Name)
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/model/... -v -run TestAILLMProvider`
Expected: FAIL (undefined: AILLMProvider)

- [ ] **Step 3: 实现 Model**

```go
// internal/model/llm_provider.go
// Package model 定义数据库模型。
package model

import (
    "time"

    "gorm.io/gorm"
)

// AILLMProvider 存储 LLM 模型配置。
//
// 支持多供应商管理，管理员可配置多个模型并设置全局默认模型。
// API Key 采用加密存储，确保敏感信息安全。
type AILLMProvider struct {
    ID            uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Name          string         `gorm:"column:name;type:varchar(64);not null" json:"name"`
    Provider      string         `gorm:"column:provider;type:varchar(32);not null;uniqueIndex:uk_provider_model,priority:1" json:"provider"`
    Model         string         `gorm:"column:model;type:varchar(128);not null;uniqueIndex:uk_provider_model,priority:2" json:"model"`
    BaseURL       string         `gorm:"column:base_url;type:varchar(512);not null" json:"base_url"`
    APIKey        string         `gorm:"column:api_key;type:varchar(256);not null" json:"-"` // 不返回给前端
    APIKeyVersion int            `gorm:"column:api_key_version;default:1" json:"api_key_version"`
    Temperature   float64        `gorm:"column:temperature;type:decimal(3,2);default:0.70" json:"temperature"`
    Thinking      bool           `gorm:"column:thinking;default:false" json:"thinking"`
    IsDefault     bool           `gorm:"column:is_default;default:false" json:"is_default"`
    IsEnabled     bool           `gorm:"column:is_enabled;default:true" json:"is_enabled"`
    SortOrder     int            `gorm:"column:sort_order;default:0" json:"sort_order"`
    ConfigVersion int            `gorm:"column:config_version;default:1" json:"config_version"`
    CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
    DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回表名。
func (AILLMProvider) TableName() string { return "ai_llm_providers" }
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/model/... -v -run TestAILLMProvider`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/model/llm_provider.go internal/model/llm_provider_test.go
git commit -m "feat(ai): add AILLMProvider model for LLM configuration"
```

### Task 3: DAO 实现

**Files:**
- Create: `internal/dao/ai/llm_provider_dao.go`

- [ ] **Step 1: 写失败测试**

```go
// internal/dao/ai/llm_provider_dao_test.go
package ai

import (
    "context"
    "testing"

    "github.com/cy77cc/OpsPilot/internal/model"
    "github.com/stretchr/testify/assert"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to connect database: %v", err)
    }
    if err := db.AutoMigrate(&model.AILLMProvider{}); err != nil {
        t.Fatalf("failed to migrate: %v", err)
    }
    return db
}

func TestLLMProviderDAO_Create(t *testing.T) {
    db := setupTestDB(t)
    dao := NewLLMProviderDAO(db)
    ctx := context.Background()

    provider := &model.AILLMProvider{
        Name:     "Test Model",
        Provider: "qwen",
        Model:    "test-model",
        BaseURL:  "https://api.test.com",
        APIKey:   "sk-test",
    }

    err := dao.Create(ctx, provider)
    assert.NoError(t, err)
    assert.NotZero(t, provider.ID)
}

func TestLLMProviderDAO_GetDefault(t *testing.T) {
    db := setupTestDB(t)
    dao := NewLLMProviderDAO(db)
    ctx := context.Background()

    // 无数据时返回 nil
    result, err := dao.GetDefault(ctx)
    assert.NoError(t, err)
    assert.Nil(t, result)

    // 创建默认模型
    provider := &model.AILLMProvider{
        Name:      "Default Model",
        Provider:  "qwen",
        Model:     "default-model",
        BaseURL:   "https://api.test.com",
        APIKey:    "sk-test",
        IsDefault: true,
        IsEnabled: true,
    }
    _ = dao.Create(ctx, provider)

    result, err = dao.GetDefault(ctx)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "default-model", result.Model)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/dao/ai/... -v -run TestLLMProviderDAO`
Expected: FAIL (undefined: LLMProviderDAO)

- [ ] **Step 3: 实现 DAO**

```go
// internal/dao/ai/llm_provider_dao.go
// Package ai 提供 AI 模块的数据访问层实现。
package ai

import (
    "context"

    "github.com/cy77cc/OpsPilot/internal/model"
    "gorm.io/gorm"
)

// LLMProviderDAO 提供 LLM 模型配置的数据访问方法。
type LLMProviderDAO struct {
    db *gorm.DB
}

// NewLLMProviderDAO 创建 LLMProviderDAO 实例。
func NewLLMProviderDAO(db *gorm.DB) *LLMProviderDAO {
    return &LLMProviderDAO{db: db}
}

// Create 创建模型配置。
func (d *LLMProviderDAO) Create(ctx context.Context, provider *model.AILLMProvider) error {
    return d.db.WithContext(ctx).Create(provider).Error
}

// GetByID 根据 ID 获取模型配置。
func (d *LLMProviderDAO) GetByID(ctx context.Context, id uint64) (*model.AILLMProvider, error) {
    var provider model.AILLMProvider
    err := d.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&provider).Error
    if err == gorm.ErrRecordNotFound {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &provider, nil
}

// GetDefault 获取默认模型配置。
//
// 查询优先级：
//  1. is_default = true 且 is_enabled = true
//  2. 返回 nil 表示无默认模型
func (d *LLMProviderDAO) GetDefault(ctx context.Context) (*model.AILLMProvider, error) {
    var provider model.AILLMProvider
    err := d.db.WithContext(ctx).
        Where("is_default = ? AND is_enabled = ? AND deleted_at IS NULL", true, true).
        First(&provider).Error
    if err == gorm.ErrRecordNotFound {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &provider, nil
}

// GetFirstEnabled 获取第一个启用的模型（作为备选默认）。
func (d *LLMProviderDAO) GetFirstEnabled(ctx context.Context) (*model.AILLMProvider, error) {
    var provider model.AILLMProvider
    err := d.db.WithContext(ctx).
        Where("is_enabled = ? AND deleted_at IS NULL", true).
        Order("id ASC").
        First(&provider).Error
    if err == gorm.ErrRecordNotFound {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &provider, nil
}

// ListAll 获取所有模型配置（管理员）。
func (d *LLMProviderDAO) ListAll(ctx context.Context) ([]*model.AILLMProvider, error) {
    var providers []*model.AILLMProvider
    err := d.db.WithContext(ctx).
        Where("deleted_at IS NULL").
        Order("sort_order DESC, id ASC").
        Find(&providers).Error
    return providers, err
}

// ListEnabled 获取所有启用的模型配置。
func (d *LLMProviderDAO) ListEnabled(ctx context.Context) ([]*model.AILLMProvider, error) {
    var providers []*model.AILLMProvider
    err := d.db.WithContext(ctx).
        Where("is_enabled = ? AND deleted_at IS NULL", true).
        Order("sort_order DESC, id ASC").
        Find(&providers).Error
    return providers, err
}

// Update 更新模型配置。
func (d *LLMProviderDAO) Update(ctx context.Context, provider *model.AILLMProvider) error {
    return d.db.WithContext(ctx).Save(provider).Error
}

// UpdateFields 更新指定字段。
func (d *LLMProviderDAO) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
    return d.db.WithContext(ctx).
        Model(&model.AILLMProvider{}).
        Where("id = ? AND deleted_at IS NULL", id).
        Updates(fields).Error
}

// ClearDefault 清除所有默认标记。
func (d *LLMProviderDAO) ClearDefault(ctx context.Context) error {
    return d.db.WithContext(ctx).
        Model(&model.AILLMProvider{}).
        Where("is_default = ? AND deleted_at IS NULL", true).
        Update("is_default", false).Error
}

// SoftDelete 软删除模型配置。
func (d *LLMProviderDAO) SoftDelete(ctx context.Context, id uint64) error {
    return d.db.WithContext(ctx).
        Where("id = ?", id).
        Delete(&model.AILLMProvider{}).Error
}

// FindByProviderModel 根据供应商和模型标识查找。
func (d *LLMProviderDAO) FindByProviderModel(ctx context.Context, provider, model string) (*model.AILLMProvider, error) {
    var p model.AILLMProvider
    err := d.db.WithContext(ctx).
        Where("provider = ? AND model = ? AND deleted_at IS NULL", provider, model).
        First(&p).Error
    if err == gorm.ErrRecordNotFound {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &p, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/dao/ai/... -v -run TestLLMProviderDAO`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/dao/ai/llm_provider_dao.go internal/dao/ai/llm_provider_dao_test.go
git commit -m "feat(ai): add LLMProviderDAO for model config persistence"
```

### Task 4: 错误码定义

**Files:**
- Modify: `internal/xcode/code.go`

- [ ] **Step 1: 添加错误码常量**

在 `internal/xcode/code.go` 的业务错误码区域添加：

```go
// 在 LoginFailed Xcode = 4009 后添加
LLMProviderNotFound      Xcode = 4010 // 模型配置不存在
LLMProviderDisabled      Xcode = 4011 // 模型已禁用
LLMProviderInUse         Xcode = 4012 // 模型正在使用中
LLMImportInvalidJSON     Xcode = 4013 // JSON 格式无效
LLMImportValidationFail  Xcode = 4014 // 导入配置验证失败
```

- [ ] **Step 2: 添加错误消息**

在 `Msg()` 方法的 switch 语句中，追加以下 case（不要重复定义已存在的 `LoginFailed`）：

```go
// 在业务错误 case 区域追加
case LLMProviderNotFound:
    return "模型配置不存在"
case LLMProviderDisabled:
    return "模型已禁用"
case LLMProviderInUse:
    return "模型正在使用中"
case LLMImportInvalidJSON:
    return "JSON 格式无效"
case LLMImportValidationFail:
    return "导入配置验证失败"
```

- [ ] **Step 3: 添加 HTTP 状态码映射**

在 `HttpStatus()` 方法的 switch 语句中添加：

```go
// 在 default 前添加
case LLMProviderNotFound:
    return http.StatusNotFound
case LLMProviderDisabled, LLMProviderInUse, LLMImportInvalidJSON, LLMImportValidationFail:
    return http.StatusBadRequest
```

- [ ] **Step 4: 运行测试确认编译通过**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/xcode/code.go
git commit -m "feat(ai): add error codes 4010-4014 for LLM provider management"
```

---

## Chunk 2: ChatModel 注册模式

### Task 5: 注册模式核心

**Files:**
- Create: `internal/ai/chatmodel/registry.go`

- [ ] **Step 1: 写失败测试**

```go
// internal/ai/chatmodel/registry_test.go
package chatmodel

import (
    "context"
    "testing"

    einomodel "github.com/cloudwego/eino/components/model"
    "github.com/cy77cc/OpsPilot/internal/model"
    "github.com/stretchr/testify/assert"
)

func TestRegistry_Register(t *testing.T) {
    // 清空注册表
    registry.Lock()
    registry.factories = make(map[string]ModelFactory)
    registry.Unlock()

    // 注册一个 mock 工厂
    Register("test_provider", &mockFactory{})

    factory, ok := GetFactory("test_provider")
    assert.True(t, ok)
    assert.NotNil(t, factory)
}

func TestRegistry_GetFactory_NotFound(t *testing.T) {
    registry.Lock()
    registry.factories = make(map[string]ModelFactory)
    registry.Unlock()

    factory, ok := GetFactory("nonexistent")
    assert.False(t, ok)
    assert.Nil(t, factory)
}

type mockFactory struct{}

func (f *mockFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
    return nil, nil
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/chatmodel/... -v -run TestRegistry`
Expected: FAIL (undefined: ModelFactory, Register, GetFactory)

- [ ] **Step 3: 实现注册模式核心**

```go
// internal/ai/chatmodel/registry.go
// Package chatmodel 提供 AI 模型的初始化和健康检查功能。
//
// 本文件实现注册模式，支持动态注册不同的 LLM 供应商工厂。
// 新增供应商只需实现 ModelFactory 接口并调用 Register 注册。
package chatmodel

import (
    "context"
    "fmt"
    "sync"

    einomodel "github.com/cloudwego/eino/components/model"
    "github.com/cy77cc/OpsPilot/internal/model"
)

// ModelFactory 定义模型工厂接口。
//
// 每个供应商需要实现此接口，根据数据库配置创建对应的 ChatModel 实例。
type ModelFactory interface {
    // Create 根据配置创建聊天模型实例。
    //
    // 参数:
    //   - ctx: 上下文
    //   - provider: 数据库中的模型配置
    //   - opts: 运行时选项（超时、温度覆盖等）
    //
    // 返回: ToolCallingChatModel 实例或错误
    Create(ctx context.Context, provider *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error)
}

// registry 全局供应商注册表。
var registry = struct {
    sync.RWMutex
    factories map[string]ModelFactory
}{
    factories: make(map[string]ModelFactory),
}

// Register 注册供应商工厂。
//
// 通常在 init() 函数中调用，实现供应商的自动注册。
func Register(provider string, factory ModelFactory) {
    registry.Lock()
    defer registry.Unlock()
    registry.factories[provider] = factory
}

// GetFactory 获取供应商工厂。
func GetFactory(provider string) (ModelFactory, bool) {
    registry.RLock()
    defer registry.RUnlock()
    f, ok := registry.factories[provider]
    return f, ok
}

// NewChatModelFromProvider 根据数据库配置创建聊天模型实例。
//
// 参数:
//   - ctx: 上下文
//   - provider: 数据库中的模型配置
//   - opts: 运行时选项
//
// 返回: ToolCallingChatModel 实例或错误
func NewChatModelFromProvider(ctx context.Context, provider *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
    factory, ok := GetFactory(provider.Provider)
    if !ok {
        return nil, fmt.Errorf("unsupported llm provider %q", provider.Provider)
    }
    return factory.Create(ctx, provider, opts)
}

// ListRegisteredProviders 返回已注册的供应商列表。
func ListRegisteredProviders() []string {
    registry.RLock()
    defer registry.RUnlock()
    providers := make([]string, 0, len(registry.factories))
    for p := range registry.factories {
        providers = append(providers, p)
    }
    return providers
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/chatmodel/... -v -run TestRegistry`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/chatmodel/registry.go internal/ai/chatmodel/registry_test.go
git commit -m "feat(ai): add registry pattern for chatmodel factories"
```

### Task 6: Qwen 工厂

**Files:**
- Create: `internal/ai/chatmodel/qwen.go`

- [ ] **Step 1: 实现 Qwen 工厂**

```go
// internal/ai/chatmodel/qwen.go
// Package chatmodel 提供 AI 模型的初始化和健康检查功能。
package chatmodel

import (
    "context"

    qwenmodel "github.com/cloudwego/eino-ext/components/model/qwen"
    einomodel "github.com/cloudwego/eino/components/model"
    "github.com/cy77cc/OpsPilot/internal/model"
)

func init() {
    Register("qwen", &qwenFactory{})
}

// qwenFactory 实现 Qwen 模型工厂。
type qwenFactory struct{}

// Create 根据 AILLMProvider 配置创建 Qwen ChatModel 实例。
func (f *qwenFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
    temp := float32(p.Temperature)
    thinking := p.Thinking
    return qwenmodel.NewChatModel(ctx, &qwenmodel.ChatModelConfig{
        APIKey:         p.APIKey,
        BaseURL:        p.BaseURL,
        Model:          p.Model,
        Temperature:    &temp,
        Timeout:        opts.Timeout,
        EnableThinking: &thinking,
    })
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/ai/chatmodel/qwen.go
git commit -m "feat(ai): add Qwen chatmodel factory"
```

### Task 7: Ark 工厂

**Files:**
- Create: `internal/ai/chatmodel/ark.go`

- [ ] **Step 1: 实现 Ark 工厂**

```go
// internal/ai/chatmodel/ark.go
// Package chatmodel 提供 AI 模型的初始化和健康检查功能。
package chatmodel

import (
    "context"

    arkmodel "github.com/cloudwego/eino-ext/components/model/ark"
    einomodel "github.com/cloudwego/eino/components/model"
    "github.com/cy77cc/OpsPilot/internal/model"
    "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

func init() {
    Register("ark", &arkFactory{})
}

// arkFactory 实现 Ark (火山引擎) 模型工厂。
type arkFactory struct{}

// Create 根据 AILLMProvider 配置创建 Ark ChatModel 实例。
func (f *arkFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
    temp := float32(p.Temperature)
    return arkmodel.NewChatModel(ctx, &arkmodel.ChatModelConfig{
        APIKey:      p.APIKey,
        BaseURL:     p.BaseURL,
        Model:       p.Model,
        Temperature: &temp,
        Timeout:     &opts.Timeout,
        Thinking: &model.Thinking{
            Type: model.ThinkingTypeDisabled,
        },
    })
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/ai/chatmodel/ark.go
git commit -m "feat(ai): add Ark chatmodel factory"
```

### Task 8: Ollama 工厂

**Files:**
- Create: `internal/ai/chatmodel/ollama.go`

- [ ] **Step 1: 实现 Ollama 工厂**

```go
// internal/ai/chatmodel/ollama.go
// Package chatmodel 提供 AI 模型的初始化和健康检查功能。
package chatmodel

import (
    "context"

    ollamamodel "github.com/cloudwego/eino-ext/components/model/ollama"
    einomodel "github.com/cloudwego/eino/components/model"
    "github.com/cy77cc/OpsPilot/internal/model"
)

func init() {
    Register("ollama", &ollamaFactory{})
}

// ollamaFactory 实现 Ollama 模型工厂。
type ollamaFactory struct{}

// Create 根据 AILLMProvider 配置创建 Ollama ChatModel 实例。
func (f *ollamaFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
    return ollamamodel.NewChatModel(ctx, &ollamamodel.ChatModelConfig{
        BaseURL: p.BaseURL,
        Model:   p.Model,
        Timeout: opts.Timeout,
    })
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/ai/chatmodel/ollama.go
git commit -m "feat(ai): add Ollama chatmodel factory"
```

### Task 9: 改造现有 model.go

**Files:**
- Modify: `internal/ai/chatmodel/model.go`

- [ ] **Step 1: 添加 GetDefaultChatModel 函数**

在 `internal/ai/chatmodel/model.go` 文件末尾添加：

```go
// GetDefaultChatModel 获取默认模型并创建 ChatModel 实例。
//
// 回退优先级：
//  1. 数据库 is_default = true 的启用模型
//  2. 数据库 ID 最小的启用模型
//  3. config.yaml 中的配置
//
// 参数:
//   - ctx: 上下文
//   - db: 数据库连接（可为 nil，此时使用配置文件）
//   - opts: 运行时选项
//
// 返回: ToolCallingChatModel 实例或错误
func GetDefaultChatModel(ctx context.Context, db *gorm.DB, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
    if !config.CFG.LLM.Enable {
        return nil, fmt.Errorf("llm disabled")
    }

    // 尝试从数据库获取默认模型
    if db != nil {
        dao := aidao.NewLLMProviderDAO(db)

        // 优先获取数据库中的默认模型
        provider, err := dao.GetDefault(ctx)
        if err != nil {
            return nil, fmt.Errorf("failed to get default provider: %w", err)
        }

        // 无默认模型时，尝试获取第一个启用的模型
        if provider == nil {
            provider, err = dao.GetFirstEnabled(ctx)
            if err != nil {
                return nil, fmt.Errorf("failed to get first enabled provider: %w", err)
            }
        }

        // 数据库有配置时，使用注册模式创建
        if provider != nil {
            return NewChatModelFromProvider(ctx, provider, opts)
        }
    }

    // 回退到配置文件，使用旧逻辑
    return NewChatModel(ctx, opts)
}
```

- [ ] **Step 2: 添加必要导入**

在文件顶部添加导入：

```go
import (
    // ... 现有导入 ...
    aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
    "gorm.io/gorm"
)
```

- [ ] **Step 3: 运行测试确认编译通过**

Run: `go build ./internal/ai/chatmodel/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/ai/chatmodel/model.go
git commit -m "feat(ai): add GetDefaultChatModel with database fallback"
```

### Task 9B: 切换 AI 运行时调用入口（关键）

**Files:**
- Modify: `internal/ai/agents/router.go`
- Modify: `internal/ai/agents/diagnosis/agent.go`
- Modify: `internal/ai/agents/inspection/agent.go`
- Modify: `internal/ai/agents/change/agent.go`
- Modify: `internal/ai/agents/qa/qa.go`

- [ ] **Step 1: 写失败测试（或最小编译用例）**

目标：确保运行时不再直接调用 `chatmodel.NewChatModel(...)`，统一经由 `chatmodel.GetDefaultChatModel(ctx, db, opts)`。

- [ ] **Step 2: 最小实现**

将以上 Agent 创建模型的调用统一切换到 `GetDefaultChatModel`，并通过 `svcCtx.DB` 透传数据库连接。

- [ ] **Step 3: 运行验证**

Run: `go test ./internal/ai/agents/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/ai/agents/router.go internal/ai/agents/diagnosis/agent.go internal/ai/agents/inspection/agent.go internal/ai/agents/change/agent.go internal/ai/agents/qa/qa.go
git commit -m "refactor(ai): route all agent model selection through database-aware default resolver"
```

---

## Chunk 3: 业务逻辑层

### Task 10A: API Key 加密与解密链路（关键）

**Files:**
- Create: `internal/service/ai/logic/llm_provider_crypto.go`
- Create: `internal/service/ai/logic/llm_provider_crypto_test.go`
- Modify: `internal/service/ai/logic/llm_provider_logic.go`

- [ ] **Step 1: 写失败测试**

覆盖以下场景：
- 创建/导入模型时，写入数据库前加密 `api_key`
- 读取用于模型实例化时，解密 `api_key`
- `config.CFG.Security.EncryptionKey` 为空时返回明确错误，不允许明文落库

- [ ] **Step 2: 最小实现**

实现 `encryptAPIKey` / `decryptAPIKey` 封装，内部统一调用：
- `utils.EncryptText(plain, config.CFG.Security.EncryptionKey)`
- `utils.DecryptText(cipher, config.CFG.Security.EncryptionKey)`

并在 `Create` / `Update` / `ImportConfig` 路径接入。

- [ ] **Step 3: 运行测试确认通过**

Run: `go test ./internal/service/ai/logic/... -v -run \"APIKey|Encrypt|Decrypt|LLMProviderLogic\"`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/service/ai/logic/llm_provider_crypto.go internal/service/ai/logic/llm_provider_crypto_test.go internal/service/ai/logic/llm_provider_logic.go
git commit -m "feat(ai): encrypt llm provider api keys at rest and decrypt on use"
```

### Task 10: LLM Provider Logic

**Files:**
- Create: `internal/service/ai/logic/llm_provider_logic.go`

- [ ] **Step 1: 写失败测试**

```go
// internal/service/ai/logic/llm_provider_logic_test.go
package logic

import (
    "context"
    "testing"

    "github.com/cy77cc/OpsPilot/internal/model"
    "github.com/stretchr/testify/assert"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func setupLogicTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to connect database: %v", err)
    }
    if err := db.AutoMigrate(&model.AILLMProvider{}); err != nil {
        t.Fatalf("failed to migrate: %v", err)
    }
    return db
}

func TestLLMProviderLogic_GetDefault_FallbackToConfig(t *testing.T) {
    db := setupLogicTestDB(t)
    logic := NewLLMProviderLogic(db)

    ctx := context.Background()
    provider, err := logic.GetDefault(ctx)

    assert.NoError(t, err)
    assert.NotNil(t, provider)
    // ID=0 表示来自配置文件
    assert.Equal(t, uint64(0), provider.ID)
}

func TestLLMProviderLogic_Create_WithDefault(t *testing.T) {
    db := setupLogicTestDB(t)
    logic := NewLLMProviderLogic(db)

    ctx := context.Background()
    provider := &model.AILLMProvider{
        Name:      "Test Model",
        Provider:  "qwen",
        Model:     "test-model",
        BaseURL:   "https://api.test.com",
        APIKey:    "sk-test",
        IsDefault: true,
    }

    err := logic.Create(ctx, provider)
    assert.NoError(t, err)

    // 验证默认模型唯一性
    provider2 := &model.AILLMProvider{
        Name:      "Another Model",
        Provider:  "ark",
        Model:     "another-model",
        BaseURL:   "https://api.test2.com",
        APIKey:    "sk-test2",
        IsDefault: true,
    }
    err = logic.Create(ctx, provider2)
    assert.NoError(t, err)

    // 验证只有一个默认模型
    result, _ := logic.GetDefault(ctx)
    assert.Equal(t, "another-model", result.Model)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/service/ai/logic/... -v -run TestLLMProviderLogic`
Expected: FAIL (undefined: LLMProviderLogic)

- [ ] **Step 3: 实现 Logic**

```go
// internal/service/ai/logic/llm_provider_logic.go
// Package logic 实现 AI 模块的业务逻辑层。
package logic

import (
    "context"
    "strings"

    aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
    aimodel "github.com/cy77cc/OpsPilot/internal/model"
    "github.com/cy77cc/OpsPilot/internal/config"
    "github.com/cy77cc/OpsPilot/internal/xcode"
    "gorm.io/gorm"
)

// LLMProviderLogic 提供 LLM 模型配置的业务逻辑。
type LLMProviderLogic struct {
    dao *aidao.LLMProviderDAO
    db  *gorm.DB
}

// NewLLMProviderLogic 创建 LLMProviderLogic 实例。
func NewLLMProviderLogic(db *gorm.DB) *LLMProviderLogic {
    return &LLMProviderLogic{
        dao: aidao.NewLLMProviderDAO(db),
        db:  db,
    }
}

// GetDefault 获取全局默认模型配置。
//
// 回退优先级：
//  1. 数据库 is_default = true 的启用模型
//  2. 数据库 ID 最小的启用模型
//  3. config.yaml 中的配置（构造虚拟 Provider）
func (l *LLMProviderLogic) GetDefault(ctx context.Context) (*aimodel.AILLMProvider, error) {
    // 1. 尝试获取数据库中的默认模型
    provider, err := l.dao.GetDefault(ctx)
    if err != nil {
        return nil, err
    }
    if provider != nil {
        return provider, nil
    }

    // 2. 尝试获取 ID 最小的启用模型
    provider, err = l.dao.GetFirstEnabled(ctx)
    if err != nil {
        return nil, err
    }
    if provider != nil {
        return provider, nil
    }

    // 3. 回退到 config.yaml
    return l.buildProviderFromConfig(), nil
}

// buildProviderFromConfig 从配置文件构造虚拟 Provider。
func (l *LLMProviderLogic) buildProviderFromConfig() *aimodel.AILLMProvider {
    return &aimodel.AILLMProvider{
        ID:          0, // 0 表示来自配置文件
        Name:        config.CFG.LLM.Model,
        Provider:    config.CFG.LLM.Provider,
        Model:       config.CFG.LLM.Model,
        BaseURL:     config.CFG.LLM.BaseURL,
        APIKey:      config.CFG.LLM.APIKey,
        Temperature: config.CFG.LLM.Temperature,
        Thinking:    false,
        IsDefault:   true,
        IsEnabled:   true,
        ConfigVersion: 0, // 0 表示配置文件版本
    }
}

// ListAll 获取所有模型配置（管理员）。
func (l *LLMProviderLogic) ListAll(ctx context.Context) ([]*aimodel.AILLMProvider, error) {
    return l.dao.ListAll(ctx)
}

// ListEnabled 获取所有启用的模型配置。
func (l *LLMProviderLogic) ListEnabled(ctx context.Context) ([]*aimodel.AILLMProvider, error) {
    return l.dao.ListEnabled(ctx)
}

// Create 创建模型配置。
//
// 如果 is_default = true，自动将其他模型的 is_default 置为 false。
func (l *LLMProviderLogic) Create(ctx context.Context, provider *aimodel.AILLMProvider) error {
    cipherKey, err := l.encryptAPIKey(provider.APIKey)
    if err != nil {
        return err
    }
    provider.APIKey = cipherKey

    return l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        dao := aidao.NewLLMProviderDAO(tx)

        // 如果设置为默认，先清除其他默认标记
        if provider.IsDefault {
            if err := dao.ClearDefault(ctx); err != nil {
                return err
            }
        }

        return dao.Create(ctx, provider)
    })
}

// Update 更新模型配置。
//
// 支持部分更新：
//   - api_key 为空时保持原值
//   - 未传递的字段保持原值
func (l *LLMProviderLogic) Update(ctx context.Context, id uint64, updates map[string]any) error {
    // 检查模型是否存在
    provider, err := l.dao.GetByID(ctx, id)
    if err != nil {
        return err
    }
    if provider == nil {
        return xcode.NewErrCode(xcode.LLMProviderNotFound)
    }

    // 如果设置为默认，先清除其他默认标记
    if isDefault, ok := updates["is_default"].(bool); ok && isDefault {
        if err := l.dao.ClearDefault(ctx); err != nil {
            return err
        }
    }

    // api_key 为空时不更新；非空时先加密再落库
    if apiKey, ok := updates["api_key"].(string); ok && apiKey == "" {
        delete(updates, "api_key")
    } else if ok {
        encrypted, encErr := l.encryptAPIKey(apiKey)
        if encErr != nil {
            return encErr
        }
        updates["api_key"] = encrypted
    }

    // 递增 config_version
    updates["config_version"] = gorm.Expr("config_version + 1")

    return l.dao.UpdateFields(ctx, id, updates)
}

// SetDefault 设置默认模型。
//
// 将指定模型设为默认，其他模型的 is_default 置为 false。
func (l *LLMProviderLogic) SetDefault(ctx context.Context, id uint64) error {
    // 检查模型是否存在
    provider, err := l.dao.GetByID(ctx, id)
    if err != nil {
        return err
    }
    if provider == nil {
        return xcode.NewErrCode(xcode.LLMProviderNotFound)
    }

    return l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        dao := aidao.NewLLMProviderDAO(tx)

        // 清除其他默认标记
        if err := dao.ClearDefault(ctx); err != nil {
            return err
        }

        // 设置新的默认模型
        return dao.UpdateFields(ctx, id, map[string]any{
            "is_default":     true,
            "config_version": gorm.Expr("config_version + 1"),
        })
    })
}

// SoftDelete 软删除模型配置。
func (l *LLMProviderLogic) SoftDelete(ctx context.Context, id uint64) error {
    // 检查模型是否存在
    provider, err := l.dao.GetByID(ctx, id)
    if err != nil {
        return err
    }
    if provider == nil {
        return xcode.NewErrCode(xcode.LLMProviderNotFound)
    }

    return l.dao.SoftDelete(ctx, id)
}

// MaskAPIKey 对 API Key 进行脱敏处理。
//
// 格式：前缀(3字符) + *** + 后缀(3字符)
// 例如：sk-abc123xyz -> sk-***xyz
func MaskAPIKey(apiKey string) string {
    if len(apiKey) <= 6 {
        return "***"
    }
    return apiKey[:3] + "***" + apiKey[len(apiKey)-3:]
}

// ============== 导入相关类型和逻辑 ==============

// ImportRequest 导入请求。
type ImportRequest struct {
    Mode            string                     `json:"mode"`             // merge/replace
    ProviderMapping map[string]string          `json:"provider_mapping"` // 手动映射
    Config          *OpenClawConfig            `json:"config"`           // OpenClaw 格式配置
}

// OpenClawConfig OpenClaw 格式配置。
type OpenClawConfig struct {
    Models struct {
        Mode      string                    `json:"mode"`
        Providers map[string]ProviderConfig `json:"providers"`
    } `json:"models"`
    Agents struct {
        Defaults struct {
            Model struct {
                Primary string `json:"primary"` // 格式：provider/model
            } `json:"model"`
        } `json:"defaults"`
    } `json:"agents"`
}

// ProviderConfig 供应商配置。
type ProviderConfig struct {
    BaseURL string       `json:"baseUrl"`
    APIKey  string       `json:"apiKey"`
    API     string       `json:"api"`
    Models  []ModelEntry `json:"models"`
}

// ModelEntry 模型条目。
type ModelEntry struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Compat struct {
        ThinkingFormat string `json:"thinkingFormat"`
    } `json:"compat"`
}

// ImportResult 导入结果。
type ImportResult struct {
    Imported int            `json:"imported"`
    Updated  int            `json:"updated"`
    Skipped  int            `json:"skipped"`
    Details  []ImportDetail `json:"details"`
}

// ImportDetail 导入详情。
type ImportDetail struct {
    Provider string `json:"provider"`
    Model    string `json:"model"`
    Action   string `json:"action"` // created/updated/skipped
}

// ImportPreview 导入预览。
type ImportPreview struct {
    Providers    []ProviderPreview `json:"providers"`
    DefaultModel string            `json:"default_model"`
    Summary      PreviewSummary    `json:"summary"`
}

// ProviderPreview 供应商预览。
type ProviderPreview struct {
    ImportKey       string          `json:"import_key"`
    InferredProvider string         `json:"inferred_provider"`
    BaseURL         string          `json:"base_url"`
    Models          []ModelPreview  `json:"models"`
}

// ModelPreview 模型预览。
type ModelPreview struct {
    ID            string `json:"id"`
    Name          string `json:"name"`
    Action        string `json:"action"` // create/update
    WillBeDefault bool   `json:"will_be_default"`
}

// PreviewSummary 预览统计。
type PreviewSummary struct {
    ToCreate int `json:"to_create"`
    ToUpdate int `json:"to_update"`
    ToSkip   int `json:"to_skip"`
}

// PreviewImport 预览导入结果。
func (l *LLMProviderLogic) PreviewImport(ctx context.Context, req *ImportRequest) (*ImportPreview, error) {
    if req.Config == nil {
        return nil, xcode.NewErrCode(xcode.LLMImportInvalidJSON)
    }

    preview := &ImportPreview{}

    // 解析默认模型
    defaultModel := parseDefaultModel(req.Config)
    preview.DefaultModel = defaultModel

    // 遍历供应商
    for providerName, providerConfig := range req.Config.Models.Providers {
        systemProvider := l.resolveProvider(providerName, providerConfig.BaseURL, req.ProviderMapping)

        pp := ProviderPreview{
            ImportKey:        providerName,
            InferredProvider: systemProvider,
            BaseURL:          providerConfig.BaseURL,
        }

        for _, m := range providerConfig.Models {
            mp := ModelPreview{
                ID:   m.ID,
                Name: m.Name,
            }

            // 检查是否已存在
            existing, _ := l.dao.FindByProviderModel(ctx, systemProvider, m.ID)
            if existing != nil {
                mp.Action = "update"
                preview.Summary.ToUpdate++
            } else {
                mp.Action = "create"
                preview.Summary.ToCreate++
            }

            // 检查是否为默认模型
            mp.WillBeDefault = (defaultModel == providerName+"/"+m.ID)

            pp.Models = append(pp.Models, mp)
        }

        preview.Providers = append(preview.Providers, pp)
    }

    return preview, nil
}

// ImportConfig 导入模型配置。
func (l *LLMProviderLogic) ImportConfig(ctx context.Context, req *ImportRequest) (*ImportResult, error) {
    if req.Config == nil {
        return nil, xcode.NewErrCode(xcode.LLMImportInvalidJSON)
    }

    result := &ImportResult{}

    // 解析默认模型
    defaultModel := parseDefaultModel(req.Config)

    // replace 模式：清除现有配置
    if req.Mode == "replace" {
        l.db.Where("1 = 1").Unscoped().Delete(&aimodel.AILLMProvider{})
    }

    // 遍历供应商
    for providerName, providerConfig := range req.Config.Models.Providers {
        systemProvider := l.resolveProvider(providerName, providerConfig.BaseURL, req.ProviderMapping)

        for _, m := range providerConfig.Models {
            // 检查是否已存在
            existing, _ := l.dao.FindByProviderModel(ctx, systemProvider, m.ID)

            isDefault := (defaultModel == providerName+"/"+m.ID)

            if req.Mode == "merge" && existing != nil {
                // 更新已存在的模型
                existing.Name = m.Name
                existing.BaseURL = providerConfig.BaseURL
                if providerConfig.APIKey != "" {
                    encrypted, encErr := l.encryptAPIKey(providerConfig.APIKey)
                    if encErr != nil {
                        return nil, encErr
                    }
                    existing.APIKey = encrypted
                }
                existing.Thinking = m.Compat.ThinkingFormat != ""
                if isDefault {
                    existing.IsDefault = true
                }
                l.dao.Update(ctx, existing)
                result.Updated++
                result.Details = append(result.Details, ImportDetail{
                    Provider: providerName,
                    Model:    m.ID,
                    Action:   "updated",
                })
            } else {
                // 创建新模型
                newProvider := &aimodel.AILLMProvider{
                    Name:        m.Name,
                    Provider:    systemProvider,
                    Model:       m.ID,
                    BaseURL:     providerConfig.BaseURL,
                    APIKey:      "", // 下方通过 encryptAPIKey 赋值
                    Temperature: 0.7,
                    Thinking:    m.Compat.ThinkingFormat != "",
                    IsDefault:   isDefault,
                    IsEnabled:   true,
                }
                encrypted, encErr := l.encryptAPIKey(providerConfig.APIKey)
                if encErr != nil {
                    return nil, encErr
                }
                newProvider.APIKey = encrypted
                l.dao.Create(ctx, newProvider)
                result.Imported++
                result.Details = append(result.Details, ImportDetail{
                    Provider: providerName,
                    Model:    m.ID,
                    Action:   "created",
                })
            }
        }
    }

    // 确保只有一个默认模型
    if result.Imported > 0 || result.Updated > 0 {
        providers, _ := l.dao.ListEnabled(ctx)
        defaultCount := 0
        for _, p := range providers {
            if p.IsDefault {
                defaultCount++
            }
        }
        if defaultCount > 1 {
            // 清除所有默认标记，设置第一个为默认
            l.dao.ClearDefault(ctx)
            for _, p := range providers {
                if p.IsDefault {
                    l.dao.UpdateFields(ctx, p.ID, map[string]any{"is_default": true})
                    break
                }
            }
        }
    }

    return result, nil
}

// resolveProvider 解析供应商类型，优先使用手动映射。
func (l *LLMProviderLogic) resolveProvider(importKey, baseURL string, mapping map[string]string) string {
    // 1. 手动映射优先
    if mapped, ok := mapping[importKey]; ok {
        return mapped
    }
    // 2. 自动推断
    return mapProviderType(importKey, baseURL)
}

// parseDefaultModel 解析默认模型标识。
func parseDefaultModel(config *OpenClawConfig) string {
    if config == nil {
        return ""
    }
    return config.Agents.Defaults.Model.Primary
}

// mapProviderType 根据 baseUrl 推断供应商类型。
func mapProviderType(providerName, baseURL string) string {
    // 1. 已知供应商名称映射
    knownProviders := map[string]string{
        "bailian":   "qwen",
        "volcengine": "ark",
        "ollama":    "ollama",
    }
    if p, ok := knownProviders[strings.ToLower(providerName)]; ok {
        return p
    }

    // 2. 根据 URL 推断
    lowerURL := strings.ToLower(baseURL)
    if strings.Contains(lowerURL, "dashscope.aliyuncs.com") {
        return "qwen"
    }
    if strings.Contains(lowerURL, "ark.cn-beijing.volces.com") {
        return "ark"
    }
    if strings.Contains(lowerURL, "localhost:11434") || strings.Contains(lowerURL, "127.0.0.1:11434") {
        return "ollama"
    }

    // 3. 默认使用 qwen（兼容 OpenAI API）
    return "qwen"
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/service/ai/logic/... -v -run TestLLMProviderLogic`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/llm_provider_logic.go internal/service/ai/logic/llm_provider_logic_test.go
git commit -m "feat(ai): add LLMProviderLogic with CRUD and import functionality"
```

---

## Chunk 4: API 层

### Task 11: Handler 实现

**Files:**
- Create: `internal/service/ai/handler/llm_provider_handler.go`

- [ ] **Step 1: 实现 Handler**

```go
// internal/service/ai/handler/llm_provider_handler.go
// Package handler 实现 AI 模块的 HTTP 处理器。
package handler

import (
    "strconv"

    "github.com/cy77cc/OpsPilot/internal/httpx"
    "github.com/cy77cc/OpsPilot/internal/model"
    "github.com/cy77cc/OpsPilot/internal/service/ai/logic"
    "github.com/cy77cc/OpsPilot/internal/svc"
    "github.com/cy77cc/OpsPilot/internal/xcode"
    "github.com/gin-gonic/gin"
)

// LLMProviderHandler 处理 LLM 模型配置相关的 HTTP 请求。
type LLMProviderHandler struct {
    logic *logic.LLMProviderLogic
}

// NewLLMProviderHandler 创建 LLMProviderHandler 实例。
func NewLLMProviderHandler(svcCtx *svc.ServiceContext) *LLMProviderHandler {
    return &LLMProviderHandler{
        logic: logic.NewLLMProviderLogic(svcCtx.DB),
    }
}

// ListModels 获取所有模型配置（管理员）。
// GET /api/v1/admin/ai/models
func (h *LLMProviderHandler) ListModels(c *gin.Context) {
    providers, err := h.logic.ListAll(c.Request.Context())
    if err != nil {
        httpx.ServerErr(c, err)
        return
    }

    // 构建响应，脱敏 API Key
    type ModelResponse struct {
        ID             uint64  `json:"id"`
        Name           string  `json:"name"`
        Provider       string  `json:"provider"`
        Model          string  `json:"model"`
        BaseURL        string  `json:"base_url"`
        APIKeyMasked   string  `json:"api_key_masked"`
        Temperature    float64 `json:"temperature"`
        Thinking       bool    `json:"thinking"`
        IsDefault      bool    `json:"is_default"`
        IsEnabled      bool    `json:"is_enabled"`
        SortOrder      int     `json:"sort_order"`
        ConfigVersion  int     `json:"config_version"`
    }

    models := make([]ModelResponse, len(providers))
    for i, p := range providers {
        models[i] = ModelResponse{
            ID:            p.ID,
            Name:          p.Name,
            Provider:      p.Provider,
            Model:         p.Model,
            BaseURL:       p.BaseURL,
            APIKeyMasked:  logic.MaskAPIKey(p.APIKey),
            Temperature:   p.Temperature,
            Thinking:      p.Thinking,
            IsDefault:     p.IsDefault,
            IsEnabled:     p.IsEnabled,
            SortOrder:     p.SortOrder,
            ConfigVersion: p.ConfigVersion,
        }
    }

    httpx.OK(c, map[string]any{"models": models})
}

// CreateModel 创建模型配置。
// POST /api/v1/admin/ai/models
func (h *LLMProviderHandler) CreateModel(c *gin.Context) {
    var req struct {
        Name        string  `json:"name" binding:"required"`
        Provider    string  `json:"provider" binding:"required"`
        Model       string  `json:"model" binding:"required"`
        BaseURL     string  `json:"base_url" binding:"required"`
        APIKey      string  `json:"api_key" binding:"required"`
        Temperature float64 `json:"temperature"`
        Thinking    bool    `json:"thinking"`
        IsDefault   bool    `json:"is_default"`
        SortOrder   int     `json:"sort_order"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        httpx.BindErr(c, err)
        return
    }

    // 验证 provider
    validProviders := map[string]bool{"qwen": true, "ark": true, "ollama": true}
    if !validProviders[req.Provider] {
        httpx.Fail(c, xcode.ErrInvalidParam, "无效的供应商类型")
        return
    }

    provider := &model.AILLMProvider{
        Name:        req.Name,
        Provider:    req.Provider,
        Model:       req.Model,
        BaseURL:     req.BaseURL,
        APIKey:      req.APIKey,
        Temperature: req.Temperature,
        Thinking:    req.Thinking,
        IsDefault:   req.IsDefault,
        IsEnabled:   true,
        SortOrder:   req.SortOrder,
    }

    if provider.Temperature == 0 {
        provider.Temperature = 0.7
    }

    if err := h.logic.Create(c.Request.Context(), provider); err != nil {
        httpx.ServerErr(c, err)
        return
    }

    httpx.OK(c, map[string]any{"id": provider.ID})
}

// UpdateModel 更新模型配置。
// PUT /api/v1/admin/ai/models/:id
func (h *LLMProviderHandler) UpdateModel(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        httpx.BindErr(c, err)
        return
    }

    var req map[string]any
    if err := c.ShouldBindJSON(&req); err != nil {
        httpx.BindErr(c, err)
        return
    }

    if err := h.logic.Update(c.Request.Context(), id, req); err != nil {
        if ce, ok := err.(*xcode.CodeError); ok {
            httpx.Fail(c, ce.Code, ce.Msg)
            return
        }
        httpx.ServerErr(c, err)
        return
    }

    httpx.OK(c, nil)
}

// SetDefaultModel 设置默认模型。
// PUT /api/v1/admin/ai/models/:id/default
func (h *LLMProviderHandler) SetDefaultModel(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        httpx.BindErr(c, err)
        return
    }

    if err := h.logic.SetDefault(c.Request.Context(), id); err != nil {
        if ce, ok := err.(*xcode.CodeError); ok {
            httpx.Fail(c, ce.Code, ce.Msg)
            return
        }
        httpx.ServerErr(c, err)
        return
    }

    httpx.OK(c, nil)
}

// DeleteModel 删除模型配置。
// DELETE /api/v1/admin/ai/models/:id
func (h *LLMProviderHandler) DeleteModel(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        httpx.BindErr(c, err)
        return
    }

    if err := h.logic.SoftDelete(c.Request.Context(), id); err != nil {
        if ce, ok := err.(*xcode.CodeError); ok {
            httpx.Fail(c, ce.Code, ce.Msg)
            return
        }
        httpx.ServerErr(c, err)
        return
    }

    httpx.OK(c, nil)
}

// ImportModels 导入模型配置。
// POST /api/v1/admin/ai/models/import
func (h *LLMProviderHandler) ImportModels(c *gin.Context) {
    var req logic.ImportRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httpx.BindErr(c, err)
        return
    }

    // 默认 merge 模式
    if req.Mode == "" {
        req.Mode = "merge"
    }

    result, err := h.logic.ImportConfig(c.Request.Context(), &req)
    if err != nil {
        if ce, ok := err.(*xcode.CodeError); ok {
            httpx.Fail(c, ce.Code, ce.Msg)
            return
        }
        httpx.ServerErr(c, err)
        return
    }

    httpx.OK(c, result)
}

// PreviewImport 预览导入结果。
// POST /api/v1/admin/ai/models/import/preview
func (h *LLMProviderHandler) PreviewImport(c *gin.Context) {
    var req logic.ImportRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httpx.BindErr(c, err)
        return
    }

    preview, err := h.logic.PreviewImport(c.Request.Context(), &req)
    if err != nil {
        if ce, ok := err.(*xcode.CodeError); ok {
            httpx.Fail(c, ce.Code, ce.Msg)
            return
        }
        httpx.ServerErr(c, err)
        return
    }

    httpx.OK(c, preview)
}
```

- [ ] **Step 2: 运行测试确认编译通过**

Run: `go build ./internal/service/ai/handler/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/service/ai/handler/llm_provider_handler.go
git commit -m "feat(ai): add LLMProviderHandler for admin APIs"
```

### Task 12: 路由注册

**Files:**
- Modify: `internal/service/ai/routes.go`

- [ ] **Step 1: 添加路由注册函数**

在 `internal/service/ai/routes.go` 文件中添加新函数：

```go
// RegisterAdminAIHandlers 注册管理员 AI 相关路由。
//
// 所有路由需要管理员权限。
func RegisterAdminAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
    h := aiHandler.NewLLMProviderHandler(svcCtx)

    // 使用仓库现有鉴权能力（JWT + Casbin），不要引用不存在的 RequireAdmin
    readOnly := middleware.CasbinAuth(svcCtx.CasbinEnforcer, "ai:model:read")
    writeOnly := middleware.CasbinAuth(svcCtx.CasbinEnforcer, "ai:model:write")
    admin := v1.Group("/admin/ai", middleware.JWTAuth())
    {
        // 模型配置管理
        admin.GET("/models", readOnly, h.ListModels)
        admin.POST("/models", writeOnly, h.CreateModel)
        admin.PUT("/models/:id", writeOnly, h.UpdateModel)
        admin.PUT("/models/:id/default", writeOnly, h.SetDefaultModel)
        admin.DELETE("/models/:id", writeOnly, h.DeleteModel)
        admin.POST("/models/import", writeOnly, h.ImportModels)
        admin.POST("/models/import/preview", writeOnly, h.PreviewImport)
    }
}
```

- [ ] **Step 2: 在主路由中注册**

确保在仓库真实入口 `internal/service/service.go` 调用：

```go
// 在现有的 RegisterAIHandlers 调用后添加
ai.RegisterAdminAIHandlers(v1, svcCtx)
```

- [ ] **Step 3: 运行测试确认编译通过**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/service/ai/routes.go internal/service/service.go
git commit -m "feat(ai): register admin LLM provider routes"
```

---

## Chunk 5: 集成测试

### Task 13: 集成测试

**Files:**
- Create: `internal/service/ai/handler/llm_provider_handler_test.go`

- [ ] **Step 1: 写集成测试**

```go
// internal/service/ai/handler/llm_provider_handler_test.go
package handler

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/cy77cc/OpsPilot/internal/model"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func setupHandlerTest(t *testing.T) (*gin.Engine, *gorm.DB) {
    gin.SetMode(gin.TestMode)

    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to connect database: %v", err)
    }
    if err := db.AutoMigrate(&model.AILLMProvider{}); err != nil {
        t.Fatalf("failed to migrate: %v", err)
    }

    r := gin.New()
    return r, db
}

func TestListModels_Empty(t *testing.T) {
    r, db := setupHandlerTest(t)
    handler := NewLLMProviderHandlerWithDB(db)

    // 集成测试聚焦 handler 行为，路由路径需与生产一致
    r.GET("/api/v1/admin/ai/models", handler.ListModels)

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/admin/ai/models", nil)
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var resp map[string]any
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, float64(1000), resp["code"])
}

func TestCreateModel_Success(t *testing.T) {
    r, db := setupHandlerTest(t)
    handler := NewLLMProviderHandlerWithDB(db)

    r.POST("/api/v1/admin/ai/models", handler.CreateModel)

    body := map[string]any{
        "name":     "Test Model",
        "provider": "qwen",
        "model":    "test-model",
        "base_url": "https://api.test.com",
        "api_key":  "sk-test",
    }
    jsonBody, _ := json.Marshal(body)

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/admin/ai/models", bytes.NewReader(jsonBody))
    req.Header.Set("Content-Type", "application/json")
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var resp map[string]any
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, float64(1000), resp["code"])
    assert.NotNil(t, resp["data"])
}

// NewLLMProviderHandlerWithDB 创建用于测试的 Handler
func NewLLMProviderHandlerWithDB(db *gorm.DB) *LLMProviderHandler {
    return &LLMProviderHandler{
        logic: logic.NewLLMProviderLogic(db),
    }
}
```

- [ ] **Step 2: 补充路由鉴权回归用例**

新增 `internal/service/ai/handler/routes_admin_auth_test.go`，最少覆盖：
- 未携带 JWT 访问 `GET /api/v1/admin/ai/models` 返回 `401`
- 无 Casbin 权限访问写接口返回 `403`

- [ ] **Step 3: 运行测试**

Run: `go test ./internal/service/ai/handler/... -v -run TestListModels`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/service/ai/handler/llm_provider_handler_test.go internal/service/ai/handler/routes_admin_auth_test.go
git commit -m "test(ai): add integration tests for LLMProviderHandler"
```

---

## Summary

实现完成后的文件结构：

```
internal/
├── model/llm_provider.go              # 模型定义（扁平结构）
├── dao/ai/llm_provider_dao.go         # 数据访问层
├── ai/chatmodel/
│   ├── registry.go                    # 注册模式核心
│   ├── qwen.go                        # Qwen 工厂
│   ├── ark.go                         # Ark 工厂
│   └── ollama.go                      # Ollama 工厂
├── service/ai/
│   ├── handler/llm_provider_handler.go # HTTP 处理器
│   ├── logic/llm_provider_logic.go      # 业务逻辑
│   ├── logic/llm_provider_crypto.go     # API Key 加解密
│   └── routes.go                       # 路由注册
└── xcode/code.go                       # 错误码

internal/service/service.go             # 主路由入口接入
storage/migrations/20260324_0001_create_ai_llm_providers.sql  # 数据库迁移
```

**关键 API 端点：**
- `GET /api/v1/admin/ai/models` - 获取模型列表
- `POST /api/v1/admin/ai/models` - 创建模型
- `PUT /api/v1/admin/ai/models/:id` - 更新模型
- `PUT /api/v1/admin/ai/models/:id/default` - 设置默认
- `DELETE /api/v1/admin/ai/models/:id` - 删除模型
- `POST /api/v1/admin/ai/models/import` - 导入配置
- `POST /api/v1/admin/ai/models/import/preview` - 预览导入
