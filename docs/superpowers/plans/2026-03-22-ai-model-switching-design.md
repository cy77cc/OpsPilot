# AI 模型切换功能实现计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 支持管理员配置多个 LLM 供应商和模型，用户可在会话级别切换模型

**Architecture:**
- 新增 `ai_llm_providers` 表存储模型配置，`ai_chat_sessions` 增加 `model_id` 字段
- 使用注册模式(Registry Pattern) 改造 chatmodel 创建逻辑
- AgentManager 按 model_id 缓存 Agent 实例，LRU 淘汰策略
- 默认使用 config.yaml 配置，数据库有配置时优先读取

**Tech Stack:** Go 1.26, Gin, GORM, React 19, Ant Design 6

---

## Chunk 1: 数据层 - Model 与 DAO

### Task 1: 创建数据库迁移文件

**Files:**
- Create: `storage/migrations/20260322_0008_create_ai_llm_providers.sql`

- [ ] **Step 1: 创建迁移文件**

```sql
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

ALTER TABLE ai_chat_sessions
ADD COLUMN model_id BIGINT UNSIGNED DEFAULT NULL COMMENT '关联模型ID，NULL表示使用默认模型' AFTER scene,
ADD INDEX idx_ai_chat_sessions_model_id (model_id);

-- +migrate Down
ALTER TABLE ai_chat_sessions DROP COLUMN model_id;
DROP TABLE IF EXISTS ai_llm_providers;
```

- [ ] **Step 2: 验证迁移文件格式**

Run: `cat storage/migrations/20260322_0008_create_ai_llm_providers.sql`
Expected: SQL 文件内容正确显示

---

### Task 2: 实现 AILLMProvider Model

**Files:**
- Modify: `internal/model/ai.go`

- [ ] **Step 1: 添加 AILLMProvider 结构体**

在 `internal/model/ai.go` 文件末尾添加:

```go
// AILLMProvider 存储 LLM 模型配置。
//
// 管理员可配置多个供应商和模型，用户可在会话级别选择。
// API Key 使用加密存储，通过 Security.EncryptionKey 配置密钥。
type AILLMProvider struct {
	ID             uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name           string         `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Provider       string         `gorm:"column:provider;type:varchar(32);not null;uniqueIndex:uk_provider_model,priority:1" json:"provider"`
	Model          string         `gorm:"column:model;type:varchar(128);not null;uniqueIndex:uk_provider_model,priority:2" json:"model"`
	BaseURL        string         `gorm:"column:base_url;type:varchar(512);not null" json:"base_url"`
	APIKey         string         `gorm:"column:api_key;type:varchar(256);not null" json:"-"` // 不返回给前端
	APIKeyVersion  int            `gorm:"column:api_key_version;default:1" json:"api_key_version"`
	Temperature    float64        `gorm:"column:temperature;type:decimal(3,2);default:0.70" json:"temperature"`
	Thinking       bool           `gorm:"column:thinking;default:false" json:"thinking"`
	IsDefault      bool           `gorm:"column:is_default;default:false" json:"is_default"`
	IsEnabled      bool           `gorm:"column:is_enabled;default:true" json:"is_enabled"`
	SortOrder      int            `gorm:"column:sort_order;default:0" json:"sort_order"`
	ConfigVersion  int            `gorm:"column:config_version;default:1" json:"config_version"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AILLMProvider) TableName() string { return "ai_llm_providers" }
```

- [ ] **Step 2: 修改 AIChatSession 添加 ModelID 字段**

修改 `internal/model/ai.go` 中的 `AIChatSession` 结构体，在 `Scene` 字段后添加:

```go
type AIChatSession struct {
	ID        string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	UserID    uint64         `gorm:"column:user_id;not null;index:idx_ai_chat_sessions_user_scene_updated,priority:1;index:idx_ai_chat_sessions_user_id" json:"user_id"`
	Scene     string         `gorm:"column:scene;type:varchar(32);not null;default:'ai';index:idx_ai_chat_sessions_user_scene_updated,priority:2" json:"scene"`
	ModelID   *uint64        `gorm:"column:model_id;index:idx_ai_chat_sessions_model_id" json:"model_id"` // 新增：关联模型ID
	Title     string         `gorm:"column:title;type:varchar(255);not null;default:''" json:"title"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime;index:idx_ai_chat_sessions_user_scene_updated,priority:3,sort:desc" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}
```

- [ ] **Step 3: 运行测试确保编译通过**

Run: `cd /root/project/k8s-manage && go build ./internal/model/...`
Expected: 编译成功，无错误

---

### Task 3: 实现 LLMProviderDAO

**Files:**
- Create: `internal/dao/ai/llm_provider_dao.go`
- Create: `internal/dao/ai/llm_provider_dao_test.go`

- [ ] **Step 1: 编写 DAO 测试（TDD）**

创建 `internal/dao/ai/llm_provider_dao_test.go`:

```go
package ai

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&model.AILLMProvider{})
	require.NoError(t, err)
	return db
}

func TestLLMProviderDAO_Create(t *testing.T) {
	db := setupTestDB(t)
	dao := NewLLMProviderDAO(db)

	provider := &model.AILLMProvider{
		Name:        "Qwen3.5-Plus",
		Provider:    "qwen",
		Model:       "qwen3.5-plus",
		BaseURL:     "https://dashscope.aliyuncs.com/v1",
		APIKey:      "test-key",
		Temperature: 0.7,
		IsDefault:   true,
		IsEnabled:   true,
	}

	err := dao.Create(context.Background(), provider)
	assert.NoError(t, err)
	assert.Greater(t, provider.ID, uint64(0))
}

func TestLLMProviderDAO_GetByID(t *testing.T) {
	db := setupTestDB(t)
	dao := NewLLMProviderDAO(db)

	provider := &model.AILLMProvider{
		Name:     "Test",
		Provider: "qwen",
		Model:    "test-model",
		BaseURL:  "http://test",
		APIKey:   "key",
	}
	_ = dao.Create(context.Background(), provider)

	got, err := dao.GetByID(context.Background(), provider.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Test", got.Name)
}

func TestLLMProviderDAO_GetDefault(t *testing.T) {
	db := setupTestDB(t)
	dao := NewLLMProviderDAO(db)

	// 无数据时应返回 nil
	got, err := dao.GetDefault(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, got)

	// 插入默认模型
	provider := &model.AILLMProvider{
		Name:      "Default",
		Provider:  "qwen",
		Model:     "default-model",
		BaseURL:   "http://test",
		APIKey:    "key",
		IsDefault: true,
		IsEnabled: true,
	}
	_ = dao.Create(context.Background(), provider)

	got, err = dao.GetDefault(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "Default", got.Name)
}

func TestLLMProviderDAO_ListEnabled(t *testing.T) {
	db := setupTestDB(t)
	dao := NewLLMProviderDAO(db)

	_ = dao.Create(context.Background(), &model.AILLMProvider{
		Name: "Enabled", Provider: "qwen", Model: "enabled", BaseURL: "http://test", APIKey: "key", IsEnabled: true, SortOrder: 100,
	})
	_ = dao.Create(context.Background(), &model.AILLMProvider{
		Name: "Disabled", Provider: "ark", Model: "disabled", BaseURL: "http://test", APIKey: "key", IsEnabled: false,
	})

	list, err := dao.ListEnabled(context.Background())
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "Enabled", list[0].Name)
}

func TestLLMProviderDAO_UpdateSessionModelID(t *testing.T) {
	db := setupTestDB(t)
	dao := NewLLMProviderDAO(db)

	// 创建会话表
	err := db.AutoMigrate(&model.AIChatSession{})
	require.NoError(t, err)

	// 创建测试会话
	session := &model.AIChatSession{
		ID:     "test-session-1",
		UserID: 1,
		Scene:  "ai",
		Title:  "Test",
	}
	err = db.Create(session).Error
	require.NoError(t, err)

	// 创建模型
	provider := &model.AILLMProvider{
		Name: "Test", Provider: "qwen", Model: "test", BaseURL: "http://test", APIKey: "key",
	}
	_ = dao.Create(context.Background(), provider)

	// 更新会话模型
	err = dao.UpdateSessionModelID(context.Background(), "test-session-1", provider.ID)
	assert.NoError(t, err)

	// 验证更新
	var updated model.AIChatSession
	db.First(&updated, "id = ?", "test-session-1")
	assert.NotNil(t, updated.ModelID)
	assert.Equal(t, provider.ID, *updated.ModelID)
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /root/project/k8s-manage && go test ./internal/dao/ai/ -run TestLLMProvider -v`
Expected: 测试失败，提示 `NewLLMProviderDAO` 未定义

- [ ] **Step 3: 实现 DAO**

创建 `internal/dao/ai/llm_provider_dao.go`:

```go
// Package ai 提供 AI 模块的数据访问层。
//
// LLMProviderDAO 负责 LLM 模型配置的 CRUD 操作。

package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// LLMProviderDAO 提供 LLM 模型配置的数据访问能力。
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

// GetByID 根据ID获取模型配置。
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
// 优先级：is_default=true 且 is_enabled=true > ID 最小的启用模型。
func (d *LLMProviderDAO) GetDefault(ctx context.Context) (*model.AILLMProvider, error) {
	var provider model.AILLMProvider

	// 优先查找标记为默认的启用模型
	err := d.db.WithContext(ctx).
		Where("is_default = ? AND is_enabled = ? AND deleted_at IS NULL", true, true).
		First(&provider).Error
	if err == nil {
		return &provider, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 回退到 ID 最小的启用模型
	err = d.db.WithContext(ctx).
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

// ListEnabled 获取所有启用的模型配置。
func (d *LLMProviderDAO) ListEnabled(ctx context.Context) ([]*model.AILLMProvider, error) {
	var providers []*model.AILLMProvider
	err := d.db.WithContext(ctx).
		Where("is_enabled = ? AND deleted_at IS NULL", true).
		Order("sort_order DESC, id ASC").
		Find(&providers).Error
	return providers, err
}

// ListAll 获取所有模型配置（含禁用，管理员用）。
func (d *LLMProviderDAO) ListAll(ctx context.Context) ([]*model.AILLMProvider, error) {
	var providers []*model.AILLMProvider
	err := d.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("sort_order DESC, id ASC").
		Find(&providers).Error
	return providers, err
}

// Update 更新模型配置。
func (d *LLMProviderDAO) Update(ctx context.Context, provider *model.AILLMProvider) error {
	return d.db.WithContext(ctx).Save(provider).Error
}

// SoftDelete 软删除模型配置。
func (d *LLMProviderDAO) SoftDelete(ctx context.Context, id uint64) error {
	return d.db.WithContext(ctx).
		Model(&model.AILLMProvider{}).
		Where("id = ?", id).
		Update("deleted_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}

// ClearOtherDefaults 清除其他模型的默认标记。
//
// 设置新默认模型时调用，保证只有一个 is_default=true。
func (d *LLMProviderDAO) ClearOtherDefaults(ctx context.Context, excludeID uint64) error {
	return d.db.WithContext(ctx).
		Model(&model.AILLMProvider{}).
		Where("id != ? AND is_default = ? AND deleted_at IS NULL", excludeID, true).
		Update("is_default", false).Error
}

// HasSessionReferences 检查模型是否有关联会话。
func (d *LLMProviderDAO) HasSessionReferences(ctx context.Context, modelID uint64) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.AIChatSession{}).
		Where("model_id = ?", modelID).
		Count(&count).Error
	return count > 0, err
}

// UpdateSessionModelID 更新会话的模型ID。
//
// 用于用户切换会话模型时更新关联。
func (d *LLMProviderDAO) UpdateSessionModelID(ctx context.Context, sessionID string, modelID uint64) error {
	return d.db.WithContext(ctx).
		Model(&model.AIChatSession{}).
		Where("id = ?", sessionID).
		Update("model_id", modelID).Error
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /root/project/k8s-manage && go test ./internal/dao/ai/ -run TestLLMProvider -v`
Expected: 所有测试通过

- [ ] **Step 5: 提交**

```bash
git add storage/migrations/20260322_0008_create_ai_llm_providers.sql
git add internal/model/ai.go
git add internal/dao/ai/llm_provider_dao.go
git add internal/dao/ai/llm_provider_dao_test.go
git commit -m "feat(ai): add AILLMProvider model and DAO for multi-model support"
```

---

## Chunk 2: ChatModel 注册模式改造

### Task 4: 实现模型工厂注册模式

**Files:**
- Modify: `internal/ai/chatmodel/model.go`
- Create: `internal/ai/chatmodel/qwen.go`
- Create: `internal/ai/chatmodel/ark.go`
- Create: `internal/ai/chatmodel/ollama.go`

- [ ] **Step 1: 编写工厂注册测试**

创建 `internal/ai/chatmodel/model_test.go`:

```go
package chatmodel

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	// 测试获取已注册的工厂
	factory, ok := GetFactory("qwen")
	assert.True(t, ok)
	assert.NotNil(t, factory)

	// 测试获取未注册的工厂
	_, ok = GetFactory("unknown")
	assert.False(t, ok)
}

func TestNewChatModelFromProvider(t *testing.T) {
	ctx := context.Background()

	provider := &model.AILLMProvider{
		Name:        "Test Qwen",
		Provider:    "qwen",
		Model:       "qwen3.5-plus",
		BaseURL:     "https://dashscope.aliyuncs.com/v1",
		APIKey:      "test-key",
		Temperature: 0.7,
		Thinking:    false,
	}

	_, err := NewChatModelFromProvider(ctx, provider, ChatModelConfig{
		Timeout:  30,
		Thinking: false,
	})
	// 注意：实际调用会因 API Key 无效失败，这里只测试工厂能否创建
	// 如果返回 unsupported provider 错误说明注册失败
	require.NotNil(t, err) // 预期有错误（API Key 无效等）
	assert.NotContains(t, err.Error(), "unsupported")
}

func TestUnsupportedProvider(t *testing.T) {
	ctx := context.Background()

	provider := &model.AILLMProvider{
		Provider: "unknown_provider",
		Model:    "test",
	}

	_, err := NewChatModelFromProvider(ctx, provider, ChatModelConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported llm provider")
}
```

- [ ] **Step 2: 改造 model.go 实现注册模式**

修改 `internal/ai/chatmodel/model.go`:

```go
// Package chatmodel 提供 AI 模型的初始化和健康检查功能。
//
// 本文件负责根据配置创建不同类型的聊天模型，使用注册模式支持多供应商扩展。
//
// 重要：所有供应商工厂通过 init() 注册，必须通过 Import() 函数强制导入，
// 否则 Go 编译器会优化掉未使用的包，导致 init() 不执行。
package chatmodel

import (
	"context"
	"fmt"
	"strings"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
)

// ChatModelConfig 定义聊天模型配置选项。
type ChatModelConfig struct {
	// Timeout 模型调用的超时时间。
	Timeout time.Duration
	// Thinking 是否启用模型的思考模式。
	Thinking bool
	// Temp 模型生成文本的温度参数。
	Temp float32
}

// ModelFactory 定义模型工厂接口。
//
// 每个供应商实现此接口并通过 init() 注册到全局注册表。
type ModelFactory interface {
	// Create 根据配置创建聊天模型实例。
	Create(ctx context.Context, provider *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error)
}

// registry 全局供应商注册表。
var registry = struct {
	factories map[string]ModelFactory
}{
	factories: make(map[string]ModelFactory),
}

// Register 注册供应商工厂。
//
// 在各供应商文件的 init() 函数中调用。
func Register(provider string, factory ModelFactory) {
	registry.factories[provider] = factory
}

// GetFactory 获取供应商工厂。
func GetFactory(provider string) (ModelFactory, bool) {
	f, ok := registry.factories[provider]
	return f, ok
}

// Import 强制导入所有供应商工厂。
//
// 必须在程序启动时调用，确保所有 init() 函数执行。
// Go 编译器会优化掉未显式引用的包，导致 init() 不执行。
//
// 使用示例：
//
//	func main() {
//	    chatmodel.Import() // 强制注册所有工厂
//	    // ...
//	}
func Import() {
	// 通过匿名导入触发各工厂的 init()
	// 这里显式引用所有供应商包
	import(
	_ "github.com/cy77cc/OpsPilot/internal/ai/chatmodel" // 触发本包的 init 依赖链
)
	// 各供应商通过 init() 自行注册
	// qwen.go:   init() { Register("qwen", ...) }
	// ark.go:    init() { Register("ark", ...) }
	// ollama.go: init() { Register("ollama", ...) }
}

// NewChatModel 根据配置文件创建聊天模型实例（向后兼容）。
//
// 支持 Ollama、Qwen、Ark 三种 Provider。
func NewChatModel(ctx context.Context, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if !config.CFG.LLM.Enable {
		return nil, fmt.Errorf("llm disabled")
	}

	// 构造虚拟 Provider，从配置文件读取
	provider := &model.AILLMProvider{
		Provider:    strings.TrimSpace(strings.ToLower(config.CFG.LLM.Provider)),
		Model:       config.CFG.LLM.Model,
		BaseURL:     config.CFG.LLM.BaseURL,
		APIKey:      config.CFG.LLM.APIKey,
		Temperature: config.CFG.LLM.Temperature,
		Thinking:    opts.Thinking,
	}

	return NewChatModelFromProvider(ctx, provider, opts)
}

// NewChatModelFromProvider 根据数据库配置创建聊天模型实例。
//
// 从注册表中查找对应供应商的工厂并创建模型。
func NewChatModelFromProvider(ctx context.Context, provider *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	factory, ok := GetFactory(provider.Provider)
	if !ok {
		return nil, fmt.Errorf("unsupported llm provider %q", provider.Provider)
	}
	return factory.Create(ctx, provider, opts)
}

// CheckModelHealth 检查模型健康状态。
//
// 发送简单的 ping 消息验证模型是否正常响应。
func CheckModelHealth(ctx context.Context) error {
	model, err := NewChatModel(ctx, ChatModelConfig{
		Timeout:  10 * time.Second,
		Thinking: false,
		Temp:     0,
	})
	if err != nil {
		return err
	}
	_, err = model.Generate(ctx, []*schema.Message{schema.UserMessage("ping")})
	return err
}
```

**重要：确保工厂注册生效**

在 `internal/ai/chatmodel/doc.go` 中添加包文档：

```go
// Package chatmodel 提供 AI 模型工厂实现。
//
// 使用方法：
//
//	import chatmodel "github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
//
//	func main() {
//	    // 确保 init() 执行，注册所有工厂
//	    _ = chatmodel.Import
//	}
package chatmodel
```

- [ ] **Step 3: 实现 Qwen 工厂**

创建 `internal/ai/chatmodel/qwen.go`:

```go
// Package chatmodel 提供 AI 模型工厂实现。
//
// 本文件实现 Qwen 供应商的模型工厂。

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

type qwenFactory struct{}

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

- [ ] **Step 4: 实现 Ark 工厂**

创建 `internal/ai/chatmodel/ark.go`:

```go
// Package chatmodel 提供 AI 模型工厂实现。
//
// 本文件实现 Ark (火山引擎) 供应商的模型工厂。

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

type arkFactory struct{}

func (f *arkFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	temp := float32(p.Temperature)
	thinking := model.ThinkingTypeDisabled
	if p.Thinking {
		thinking = model.ThinkingTypeEnabled
	}
	return arkmodel.NewChatModel(ctx, &arkmodel.ChatModelConfig{
		APIKey:      p.APIKey,
		BaseURL:     p.BaseURL,
		Model:       p.Model,
		Temperature: &temp,
		Timeout:     &opts.Timeout,
		Thinking: &model.Thinking{
			Type: thinking,
		},
	})
}
```

- [ ] **Step 5: 实现 Ollama 工厂**

创建 `internal/ai/chatmodel/ollama.go`:

```go
// Package chatmodel 提供 AI 模型工厂实现。
//
// 本文件实现 Ollama 供应商的模型工厂。

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

type ollamaFactory struct{}

func (f *ollamaFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	return ollamamodel.NewChatModel(ctx, &ollamamodel.ChatModelConfig{
		BaseURL: p.BaseURL,
		Model:   p.Model,
		Timeout: opts.Timeout,
	})
}
```

- [ ] **Step 6: 运行测试验证**

Run: `cd /root/project/k8s-manage && go test ./internal/ai/chatmodel/ -v`
Expected: 测试通过

- [ ] **Step 7: 提交**

```bash
git add internal/ai/chatmodel/model.go
git add internal/ai/chatmodel/model_test.go
git add internal/ai/chatmodel/qwen.go
git add internal/ai/chatmodel/ark.go
git add internal/ai/chatmodel/ollama.go
git commit -m "refactor(ai): implement registry pattern for chat model factories"
```

---

### Task 5: 实现 AgentManager 缓存管理

**Files:**
- Create: `internal/ai/agent_manager.go`
- Create: `internal/ai/agent_manager_test.go`

- [ ] **Step 1: 编写 AgentManager 测试**

创建 `internal/ai/agent_manager_test.go`:

```go
package ai

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLLMProviderService struct {
	providers map[uint64]*model.AILLMProvider
	defaultID uint64
}

func (m *mockLLMProviderService) GetByID(ctx context.Context, id uint64) (*model.AILLMProvider, error) {
	return m.providers[id], nil
}

func (m *mockLLMProviderService) GetDefault(ctx context.Context) (*model.AILLMProvider, error) {
	if m.defaultID == 0 {
		return &model.AILLMProvider{
			ID:       0,
			Provider: "qwen",
			Model:    "default-from-config",
		}, nil
	}
	return m.providers[m.defaultID], nil
}

func TestAgentManager_GetOrCreateAgent(t *testing.T) {
	mock := &mockLLMProviderService{
		providers: map[uint64]*model.AILLMProvider{
			1: {ID: 1, Provider: "qwen", Model: "model-1", BaseURL: "http://test", APIKey: "key1", ConfigVersion: 1},
			2: {ID: 2, Provider: "qwen", Model: "model-2", BaseURL: "http://test", APIKey: "key2", ConfigVersion: 1},
		},
		defaultID: 1,
	}

	mgr := NewAgentManager(mock)
	ctx := context.Background()

	// 第一次创建
	agent1, err := mgr.GetOrCreateAgent(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, agent1)

	// 缓存命中
	agent1Again, err := mgr.GetOrCreateAgent(ctx, 1)
	require.NoError(t, err)
	assert.Same(t, agent1, agent1Again, "should return cached agent")

	// 创建另一个模型
	agent2, err := mgr.GetOrCreateAgent(ctx, 2)
	require.NoError(t, err)
	assert.NotNil(t, agent2)

	// 缓存统计
	stats := mgr.Stats()
	assert.Equal(t, 2, stats.Size)
}

func TestAgentManager_Invalidate(t *testing.T) {
	mock := &mockLLMProviderService{
		providers: map[uint64]*model.AILLMProvider{
			1: {ID: 1, Provider: "qwen", Model: "model-1", BaseURL: "http://test", APIKey: "key1", ConfigVersion: 1},
		},
	}

	mgr := NewAgentManager(mock)
	ctx := context.Background()

	_, _ = mgr.GetOrCreateAgent(ctx, 1)
	assert.Equal(t, 1, mgr.Stats().Size)

	mgr.Invalidate(1)
	assert.Equal(t, 0, mgr.Stats().Size)
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /root/project/k8s-manage && go test ./internal/ai/ -run TestAgentManager -v`
Expected: 测试失败，提示类型未定义

- [ ] **Step 3: 实现 AgentManager**

创建 `internal/ai/agent_manager.go`:

```go
// Package ai 提供 Agent 实例管理能力。
//
// AgentManager 负责：
//   - 按模型配置缓存 Agent 实例
//   - LRU 淘汰策略（使用 container/list 实现双向链表）
//   - 同步校验配置版本，确保强一致性

package ai

import (
	"container/list"
	"context"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/agents"
	"github.com/cy77cc/OpsPilot/internal/model"
)

const maxAgentCacheSize = 10

// LLMProviderService 定义模型配置服务接口。
//
// AgentManager 通过此接口获取模型配置，避免直接依赖 DAO。
type LLMProviderService interface {
	// GetByID 根据ID获取模型配置。
	GetByID(ctx context.Context, id uint64) (*model.AILLMProvider, error)
	// GetDefault 获取默认模型配置。
	GetDefault(ctx context.Context) (*model.AILLMProvider, error)
}

// AgentManager 管理 Agent 实例的生命周期和缓存。
type AgentManager struct {
	mu       sync.RWMutex
	cache    map[uint64]*cacheEntry   // model_id -> entry
	lruList  *list.List               // 双向链表，element.Value = modelID
	lruIndex map[uint64]*list.Element // model_id -> 链表节点，O(1) 查找
	provider LLMProviderService
}

type cacheEntry struct {
	agent         adk.ResumableAgent
	configVersion int // 创建时的配置版本，用于校验是否过期
}

// AgentManagerStats 缓存统计信息。
type AgentManagerStats struct {
	Size     int
	Capacity int
}

// NewAgentManager 创建 AgentManager 实例。
func NewAgentManager(provider LLMProviderService) *AgentManager {
	return &AgentManager{
		cache:    make(map[uint64]*cacheEntry),
		lruList:  list.New(),
		lruIndex: make(map[uint64]*list.Element),
		provider: provider,
	}
}

// GetOrCreateAgent 获取或创建指定模型的 Agent 实例。
//
// 缓存校验逻辑（同步版本，保证强一致性）：
//  1. 缓存命中时，同步检查 configVersion 是否与数据库一致
//  2. 版本不一致则失效旧缓存，重新创建
//
// 同步校验的优势：
//   - 模型调用失败代价高（API Key 失效、配置错误等）
//   - 读取 configVersion 通常是一次极快的查询
//   - 保证管理员更新配置后立即生效
func (m *AgentManager) GetOrCreateAgent(ctx context.Context, modelID uint64) (adk.ResumableAgent, error) {
	// 1. 检查缓存
	m.mu.RLock()
	if entry, ok := m.cache[modelID]; ok {
		m.mu.RUnlock()
		// 同步校验版本（保证强一致性）
		config, err := m.provider.GetByID(ctx, modelID)
		if err != nil {
			return nil, err
		}
		if config.ConfigVersion == entry.configVersion {
			m.touchLRU(modelID)
			return entry.agent, nil
		}
		// 版本不一致，失效旧缓存
		m.Invalidate(modelID)
	} else {
		m.mu.RUnlock()
	}

	// 2. 加载模型配置
	config, err := m.provider.GetByID(ctx, modelID)
	if err != nil {
		return nil, err
	}

	// 3. 创建新 Agent
	agent, err := agents.NewRouterWithModelConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	// 4. 存入缓存
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if entry, ok := m.cache[modelID]; ok {
		return entry.agent, nil
	}

	m.cache[modelID] = &cacheEntry{
		agent:         agent,
		configVersion: config.ConfigVersion,
	}
	element := m.lruList.PushFront(modelID)
	m.lruIndex[modelID] = element

	// 5. LRU 淘汰
	if len(m.cache) > maxAgentCacheSize {
		m.evictOldest()
	}

	return agent, nil
}

// Invalidate 清除指定模型的缓存。
func (m *AgentManager) Invalidate(modelID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, modelID)
	if element, ok := m.lruIndex[modelID]; ok {
		m.lruList.Remove(element)
		delete(m.lruIndex, modelID)
	}
}

// touchLRU 将指定模型移到 LRU 链表头部（最近使用）。
func (m *AgentManager) touchLRU(modelID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if element, ok := m.lruIndex[modelID]; ok {
		m.lruList.MoveToFront(element)
	}
}

// evictOldest 淘汰最久未使用的 Agent 实例。
func (m *AgentManager) evictOldest() {
	element := m.lruList.Back()
	if element == nil {
		return
	}
	modelID := element.Value.(uint64)
	delete(m.cache, modelID)
	m.lruList.Remove(element)
	delete(m.lruIndex, modelID)
}

// Stats 返回缓存统计信息。
func (m *AgentManager) Stats() AgentManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return AgentManagerStats{
		Size:     len(m.cache),
		Capacity: maxAgentCacheSize,
	}
}
```

- [ ] **Step 4: 修改 agents/router.go 添加 NewRouterWithModelConfig**

修改 `internal/ai/agents/router.go`，添加新函数:

```go
// NewRouterWithModelConfig 创建使用指定模型配置的路由。
//
// 用于 AgentManager 按 model_id 创建不同配置的 Agent。
func NewRouterWithModelConfig(ctx context.Context, provider *model.AILLMProvider) (adk.ResumableAgent, error) {
	// 使用 provider 配置创建模型
	model, err := chatmodel.NewChatModelFromProvider(ctx, provider, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: provider.Thinking,
		Temp:     float32(provider.Temperature),
	})
	if err != nil {
		return nil, err
	}

	routerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "OpsPilotAgent",
		Description:   "OpsPilot infrastructure operations assistant",
		Instruction:   prompt.ROUTERPROMPT,
		Model:         model,
		MaxIterations: 3,
	})
	if err != nil {
		return nil, err
	}

	qaAgent, err := qa.NewQAAgentWithModel(ctx, model)
	if err != nil {
		return nil, err
	}

	changeAgent, err := change.NewChangeAgentWithModel(ctx, model)
	if err != nil {
		return nil, err
	}

	diagnosisAgent, err := diagnosis.NewDiagnosisAgentWithModel(ctx, model)
	if err != nil {
		return nil, err
	}

	subagents := []adk.Agent{qaAgent, diagnosisAgent, changeAgent}
	return adk.SetSubAgents(ctx, routerAgent, subagents)
}
```

- [ ] **Step 5: 运行测试验证**

Run: `cd /root/project/k8s-manage && go test ./internal/ai/ -run TestAgentManager -v`
Expected: 测试通过

- [ ] **Step 6: 提交**

```bash
git add internal/ai/agent_manager.go
git add internal/ai/agent_manager_test.go
git add internal/ai/agents/router.go
git commit -m "feat(ai): add AgentManager with LRU cache for multi-model support"
```

---

## Chunk 3: Service 层实现

### Task 6: 实现 LLMProviderService

**Files:**
- Create: `internal/service/ai/logic/llm_provider_logic.go`
- Create: `internal/service/ai/logic/llm_provider_logic_test.go`

- [ ] **Step 1: 编写 Service 测试**

创建 `internal/service/ai/logic/llm_provider_logic_test.go`:

```go
package logic

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestService(t *testing.T) *LLMProviderLogic {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&model.AILLMProvider{}, &model.AIChatSession{})
	require.NoError(t, err)
	return NewLLMProviderLogic(db)
}

func TestLLMProviderLogic_GetDefault_FallbackToConfig(t *testing.T) {
	logic := setupTestService(t)

	// 无数据库配置时，应返回虚拟 Provider
	provider, err := logic.GetDefault(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, uint64(0), provider.ID, "should be virtual provider from config")
}

func TestLLMProviderLogic_Create(t *testing.T) {
	logic := setupTestService(t)

	provider := &model.AILLMProvider{
		Name:      "Test Model",
		Provider:  "qwen",
		Model:     "test-model",
		BaseURL:   "http://test",
		APIKey:    "test-key",
		IsDefault: true,
	}

	err := logic.Create(context.Background(), provider)
	require.NoError(t, err)
	assert.Greater(t, provider.ID, uint64(0))

	// 验证默认标记唯一
	provider2 := &model.AILLMProvider{
		Name:      "Another",
		Provider:  "ark",
		Model:     "another-model",
		BaseURL:   "http://test",
		APIKey:    "key",
		IsDefault: true,
	}
	err = logic.Create(context.Background(), provider2)
	require.NoError(t, err)

	// 查询验证只有一个默认
	list, _ := logic.ListEnabled(context.Background())
	defaultCount := 0
	for _, p := range list {
		if p.IsDefault {
			defaultCount++
		}
	}
	assert.Equal(t, 1, defaultCount, "should have only one default model")
}

func TestLLMProviderLogic_Update_ConfigVersion(t *testing.T) {
	logic := setupTestService(t)

	provider := &model.AILLMProvider{
		Name:     "Test",
		Provider: "qwen",
		Model:    "test",
		BaseURL:  "http://test",
		APIKey:   "key",
	}
	_ = logic.Create(context.Background(), provider)
	originalVersion := provider.ConfigVersion

	// 更新
	provider.Name = "Updated"
	err := logic.Update(context.Background(), provider)
	require.NoError(t, err)
	assert.Greater(t, provider.ConfigVersion, originalVersion, "config_version should be incremented")
}

func TestLLMProviderLogic_RejectVirtualProvider(t *testing.T) {
	logic := setupTestService(t)

	// 尝试创建 ID=0 的虚拟 Provider
	virtualProvider := &model.AILLMProvider{
		ID:       0,
		Name:     "Virtual",
		Provider: "qwen",
		Model:    "virtual",
		BaseURL:  "http://virtual",
		APIKey:   "key",
	}

	err := logic.Create(context.Background(), virtualProvider)
	assert.Error(t, err, "should reject creating ID=0 provider")
	assert.Contains(t, err.Error(), "ID=0")

	// 尝试更新 ID=0 的虚拟 Provider
	err = logic.Update(context.Background(), virtualProvider)
	assert.Error(t, err, "should reject updating ID=0 provider")
}

func TestLLMProviderLogic_UpdateSessionModel(t *testing.T) {
	logic := setupTestService(t)

	// 创建模型
	provider := &model.AILLMProvider{
		Name:      "Test Model",
		Provider:  "qwen",
		Model:     "test",
		BaseURL:   "http://test",
		APIKey:    "key",
		IsEnabled: true,
	}
	_ = logic.Create(context.Background(), provider)

	// 创建会话
	session := &model.AIChatSession{
		ID:     "test-session",
		UserID: 1,
		Scene:  "ai",
		Title:  "Test",
	}
	_ = logic.db.Create(session)

	// 更新会话模型
	err := logic.UpdateSessionModel(context.Background(), "test-session", provider.ID)
	require.NoError(t, err)

	// 验证更新
	var updated model.AIChatSession
	logic.db.First(&updated, "id = ?", "test-session")
	assert.NotNil(t, updated.ModelID)
	assert.Equal(t, provider.ID, *updated.ModelID)
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /root/project/k8s-manage && go test ./internal/service/ai/logic/ -run TestLLMProvider -v`
Expected: 测试失败，提示类型未定义

- [ ] **Step 3: 实现 LLMProviderLogic**

创建 `internal/service/ai/logic/llm_provider_logic.go`:

```go
// Package logic 实现 AI 模块的业务逻辑层。
//
// LLMProviderLogic 提供 LLM 模型配置的业务逻辑实现。

package logic

import (
	"context"
	"errors"

	"github.com/cy77cc/OpsPilot/internal/config"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
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

// GetByID 根据ID获取模型配置。
func (l *LLMProviderLogic) GetByID(ctx context.Context, id uint64) (*model.AILLMProvider, error) {
	provider, err := l.dao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, xcode.New(xcode.NotFound, "模型不存在")
	}
	return provider, nil
}

// GetDefault 获取默认模型配置。
//
// 回退优先级：
//  1. 数据库 is_default = true 的启用模型
//  2. 数据库 ID 最小的启用模型
//  3. config.yaml 中的配置（构造虚拟 Provider）
func (l *LLMProviderLogic) GetDefault(ctx context.Context) (*model.AILLMProvider, error) {
	// 1. 尝试从数据库获取默认模型
	provider, err := l.dao.GetDefault(ctx)
	if err != nil {
		return nil, err
	}
	if provider != nil {
		return provider, nil
	}

	// 2. 回退到配置文件
	return l.buildProviderFromConfig(), nil
}

// buildProviderFromConfig 从配置文件构造虚拟 Provider。
//
// ID = 0 表示来自配置文件，不受数据库管理。
func (l *LLMProviderLogic) buildProviderFromConfig() *model.AILLMProvider {
	return &model.AILLMProvider{
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

// ListEnabled 获取所有启用的模型配置（用户端）。
func (l *LLMProviderLogic) ListEnabled(ctx context.Context) ([]*model.AILLMProvider, error) {
	return l.dao.ListEnabled(ctx)
}

// ListAll 获取所有模型配置（管理员）。
func (l *LLMProviderLogic) ListAll(ctx context.Context) ([]*model.AILLMProvider, error) {
	return l.dao.ListAll(ctx)
}

// Create 创建模型配置（管理员）。
//
// 如果 is_default = true，自动将其他模型的 is_default 置为 false。
// 注意：不允许创建 ID=0 的 Provider，ID=0 是保留给配置文件虚拟 Provider 的。
func (l *LLMProviderLogic) Create(ctx context.Context, provider *model.AILLMProvider) error {
	// 防护：拒绝 ID=0（虚拟 Provider）
	if provider.ID == 0 {
		return errors.New("不能创建 ID=0 的模型配置，该 ID 保留给配置文件")
	}

	return l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := aidao.NewLLMProviderDAO(tx)

		// 创建配置
		if err := dao.Create(ctx, provider); err != nil {
			return err
		}

		// 如果是默认模型，清除其他默认标记
		if provider.IsDefault {
			if err := dao.ClearOtherDefaults(ctx, provider.ID); err != nil {
				return err
			}
		}

		return nil
	})
}

// Update 更新模型配置（管理员）。
//
// 更新时递增 config_version，支持部分更新：
//   - api_key 为空时保持原值
//   - 未传递的字段保持原值
//
// 注意：不允许更新 ID=0 的虚拟 Provider。
func (l *LLMProviderLogic) Update(ctx context.Context, provider *model.AILLMProvider) error {
	// 防护：拒绝更新 ID=0（虚拟 Provider）
	if provider.ID == 0 {
		return errors.New("不能更新 ID=0 的模型配置，该 ID 保留给配置文件")
	}

	return l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := aidao.NewLLMProviderDAO(tx)

		// 获取现有配置
		existing, err := dao.GetByID(ctx, provider.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			return xcode.New(xcode.NotFound, "模型不存在")
		}

		// 部分更新逻辑
		if provider.APIKey == "" {
			provider.APIKey = existing.APIKey
		}

		// 递增版本号
		provider.ConfigVersion = existing.ConfigVersion + 1

		// 更新配置
		if err := dao.Update(ctx, provider); err != nil {
			return err
		}

		// 如果是默认模型，清除其他默认标记
		if provider.IsDefault {
			if err := dao.ClearOtherDefaults(ctx, provider.ID); err != nil {
				return err
			}
		}

		return nil
	})
}

// SoftDelete 软删除模型配置（管理员）。
//
// 检查是否有关联会话，有则拒绝删除。
func (l *LLMProviderLogic) SoftDelete(ctx context.Context, id uint64) error {
	// 检查关联会话
	hasRefs, err := l.dao.HasSessionReferences(ctx, id)
	if err != nil {
		return err
	}
	if hasRefs {
		return errors.New("模型有关联会话，无法删除")
	}

	return l.dao.SoftDelete(ctx, id)
}

// UpdateSessionModel 更新会话的模型ID。
//
// 用户切换模型时调用，更新 ai_chat_sessions.model_id。
func (l *LLMProviderLogic) UpdateSessionModel(ctx context.Context, sessionID string, modelID uint64) error {
	// 验证模型存在且启用
	provider, err := l.dao.GetByID(ctx, modelID)
	if err != nil {
		return err
	}
	if provider == nil {
		return xcode.New(xcode.ModelNotFound, "模型不存在")
	}
	if !provider.IsEnabled {
		return xcode.New(xcode.ModelDisabled, "模型已禁用")
	}

	return l.dao.UpdateSessionModelID(ctx, sessionID, modelID)
}
```

- [ ] **Step 4: 运行测试验证**

Run: `cd /root/project/k8s-manage && go test ./internal/service/ai/logic/ -run TestLLMProvider -v`
Expected: 测试通过

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/logic/llm_provider_logic.go
git add internal/service/ai/logic/llm_provider_logic_test.go
git commit -m "feat(ai): add LLMProviderLogic for model config management"
```

---

### Task 7: 实现模型管理 API Handler

**Files:**
- Create: `internal/service/ai/handler/model_handler.go`
- Modify: `internal/service/ai/routes.go`
- Modify: `internal/xcode/code.go`

- [ ] **Step 1: 添加错误码**

在 `internal/xcode/code.go` 中添加新的错误码:

```go
// 在业务错误码 const 块中添加
	ModelNotFound      Xcode = 4010 // 模型不存在
	ModelDisabled      Xcode = 4011 // 模型已禁用
	ModelHasReferences Xcode = 4012 // 模型有关联会话
```

并在 `Msg()` 方法的 switch 中添加对应消息:

```go
	case ModelNotFound:
		return "模型不存在"
	case ModelDisabled:
		return "模型已禁用"
	case ModelHasReferences:
		return "模型有关联会话"
```

- [ ] **Step 2: 实现 ModelHandler**

创建 `internal/service/ai/handler/model_handler.go`:

```go
// Package handler 实现 AI 模块的 HTTP 处理器。
//
// ModelHandler 处理模型管理相关的 API 请求。

package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/xcode"
)

// ModelHandler 处理模型管理 API。
type ModelHandler struct {
	logic *logic.LLMProviderLogic
}

// NewModelHandler 创建 ModelHandler 实例。
func NewModelHandler(logic *logic.LLMProviderLogic) *ModelHandler {
	return &ModelHandler{logic: logic}
}

// ListModelsRequest 获取模型列表请求。
type ListModelsRequest struct {
	Scene string `form:"scene"`
}

// ModelSummary 模型摘要信息（用户端）。
type ModelSummary struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	IsDefault bool   `json:"is_default"`
}

// ListModelsResponse 获取模型列表响应。
type ListModelsResponse struct {
	Models         []ModelSummary `json:"models"`
	DefaultModelID uint64         `json:"default_model_id"`
}

// ListModels 获取可用模型列表（用户端）。
//
// GET /api/v1/ai/models
func (h *ModelHandler) ListModels(c *gin.Context) {
	ctx := c.Request.Context()

	providers, err := h.logic.ListEnabled(ctx)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	// 获取默认模型
	defaultProvider, err := h.logic.GetDefault(ctx)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	models := make([]ModelSummary, 0, len(providers))
	for _, p := range providers {
		models = append(models, ModelSummary{
			ID:        p.ID,
			Name:      p.Name,
			Provider:  p.Provider,
			IsDefault: p.IsDefault,
		})
	}

	httpx.OK(c, ListModelsResponse{
		Models:         models,
		DefaultModelID: defaultProvider.ID,
	})
}

// SwitchModelRequest 切换模型请求。
type SwitchModelRequest struct {
	ModelID uint64 `json:"model_id" binding:"required"`
}

// SwitchModel 切换会话模型。
//
// PUT /api/v1/ai/sessions/:id/model
func (h *ModelHandler) SwitchModel(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		httpx.Fail(c, xcode.MissingParam, "缺少会话ID")
		return
	}

	var req SwitchModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	ctx := c.Request.Context()

	// 调用 Logic 层更新会话模型
	err := h.logic.UpdateSessionModel(ctx, sessionID, req.ModelID)
	if err != nil {
		if ce, ok := err.(*xcode.CodeError); ok {
			httpx.Fail(c, ce.Code, ce.Msg)
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	// 获取模型名称用于响应
	provider, _ := h.logic.GetByID(ctx, req.ModelID)
	modelName := ""
	if provider != nil {
		modelName = provider.Name
	}

	httpx.OK(c, gin.H{
		"session_id": sessionID,
		"model_id":   req.ModelID,
		"model_name": modelName,
	})
}

// AdminModelDetail 管理员模型详情。
type AdminModelDetail struct {
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
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// ListAdminModels 获取所有模型配置（管理员）。
//
// GET /api/v1/admin/ai/models
func (h *ModelHandler) ListAdminModels(c *gin.Context) {
	ctx := c.Request.Context()

	providers, err := h.logic.ListAll(ctx)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	models := make([]AdminModelDetail, 0, len(providers))
	for _, p := range providers {
		models = append(models, AdminModelDetail{
			ID:           p.ID,
			Name:         p.Name,
			Provider:     p.Provider,
			Model:        p.Model,
			BaseURL:      p.BaseURL,
			APIKeyMasked: maskAPIKey(p.APIKey),
			Temperature:  p.Temperature,
			Thinking:     p.Thinking,
			IsDefault:    p.IsDefault,
			IsEnabled:    p.IsEnabled,
			SortOrder:    p.SortOrder,
			ConfigVersion: p.ConfigVersion,
			CreatedAt:    p.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    p.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"models": models})
}

// CreateModelRequest 创建模型请求。
type CreateModelRequest struct {
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

// CreateModel 创建模型配置（管理员）。
//
// POST /api/v1/admin/ai/models
func (h *ModelHandler) CreateModel(c *gin.Context) {
	var req CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
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

	if err := h.logic.Create(c.Request.Context(), provider); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{"id": provider.ID})
}

// UpdateModelRequest 更新模型请求。
type UpdateModelRequest struct {
	Name        *string  `json:"name"`
	Provider    *string  `json:"provider"`
	Model       *string  `json:"model"`
	BaseURL     *string  `json:"base_url"`
	APIKey      *string  `json:"api_key"`
	Temperature *float64 `json:"temperature"`
	Thinking    *bool    `json:"thinking"`
	IsDefault   *bool    `json:"is_default"`
	IsEnabled   *bool    `json:"is_enabled"`
	SortOrder   *int     `json:"sort_order"`
}

// UpdateModel 更新模型配置（管理员）。
//
// PUT /api/v1/admin/ai/models/:id
func (h *ModelHandler) UpdateModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "无效的模型ID")
		return
	}

	var req UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	// 获取现有配置
	existing, err := h.logic.GetByID(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	// 部分更新
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Provider != nil {
		existing.Provider = *req.Provider
	}
	if req.Model != nil {
		existing.Model = *req.Model
	}
	if req.BaseURL != nil {
		existing.BaseURL = *req.BaseURL
	}
	if req.APIKey != nil && *req.APIKey != "" {
		existing.APIKey = *req.APIKey
	}
	if req.Temperature != nil {
		existing.Temperature = *req.Temperature
	}
	if req.Thinking != nil {
		existing.Thinking = *req.Thinking
	}
	if req.IsDefault != nil {
		existing.IsDefault = *req.IsDefault
	}
	if req.IsEnabled != nil {
		existing.IsEnabled = *req.IsEnabled
	}
	if req.SortOrder != nil {
		existing.SortOrder = *req.SortOrder
	}

	if err := h.logic.Update(c.Request.Context(), existing); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{"config_version": existing.ConfigVersion})
}

// DeleteModel 删除模型配置（管理员）。
//
// DELETE /api/v1/admin/ai/models/:id
func (h *ModelHandler) DeleteModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "无效的模型ID")
		return
	}

	if err := h.logic.SoftDelete(c.Request.Context(), id); err != nil {
		if err.Error() == "模型有关联会话，无法删除" {
			httpx.Fail(c, xcode.ModelHasReferences, err.Error())
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, nil)
}

// maskAPIKey 脱敏 API Key。
//
// 格式: 前缀 + *** + 后缀3位。
func maskAPIKey(key string) string {
	if len(key) <= 6 {
		return "***"
	}
	prefix := key[:3]
	suffix := key[len(key)-3:]
	return prefix + "***" + suffix
}
```

- [ ] **Step 3: 注册路由**

修改 `internal/service/ai/routes.go`:

```go
package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/middleware"
	aiHandler "github.com/cy77cc/OpsPilot/internal/service/ai/handler"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

func RegisterAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := aiHandler.NewAIHandler(svcCtx)
	h.StartApprovalWorker(context.Background())

	// 初始化模型管理 Handler
	modelHandler := aiHandler.NewModelHandler(logic.NewLLMProviderLogic(svcCtx.DB))

	g := v1.Group("/ai", middleware.JWTAuth())
	{
		// 对话相关
		g.POST("/chat", h.Chat)
		g.GET("/sessions", h.ListSessions)
		g.POST("/sessions", h.CreateSession)
		g.GET("/sessions/:id", h.GetSession)
		g.DELETE("/sessions/:id", h.DeleteSession)
		g.GET("/runs/:runId", h.GetRun)
		g.GET("/runs/:runId/projection", h.GetRunProjection)
		g.GET("/run-contents/:id", h.GetRunContent)
		g.GET("/diagnosis/:reportId", h.GetDiagnosisReport)

		// 审批相关 (Human-in-the-Loop)
		g.GET("/approvals/pending", h.ListPendingApprovals)
		g.GET("/approvals/:id", h.GetApproval)
		g.POST("/approvals/:id/submit", h.SubmitApproval)

		// 模型管理（用户端）
		g.GET("/models", modelHandler.ListModels)
		g.PUT("/sessions/:id/model", modelHandler.SwitchModel)
	}

	// 管理员路由
	admin := v1.Group("/admin/ai", middleware.JWTAuth(), middleware.RequireAdmin())
	{
		admin.GET("/models", modelHandler.ListAdminModels)
		admin.POST("/models", modelHandler.CreateModel)
		admin.PUT("/models/:id", modelHandler.UpdateModel)
		admin.DELETE("/models/:id", modelHandler.DeleteModel)
	}
}
```

- [ ] **Step 4: 编译验证**

Run: `cd /root/project/k8s-manage && go build ./...`
Expected: 编译成功

- [ ] **Step 5: 提交**

```bash
git add internal/xcode/code.go
git add internal/service/ai/handler/model_handler.go
git add internal/service/ai/routes.go
git commit -m "feat(ai): add model management APIs for user and admin"
```

---

## Chunk 4: 前端实现

### Task 8: 实现前端 API 和 Hook

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Create: `web/src/components/AI/hooks/useModels.ts`

- [ ] **Step 1: 添加模型相关 API**

在 `web/src/api/modules/ai.ts` 中添加:

```typescript
// === 模型管理相关类型 ===

export interface AIModel {
  id: number;
  name: string;
  provider: string;
  is_default: boolean;
}

export interface AIModelsResponse {
  models: AIModel[];
  default_model_id: number;
}

export interface AIAdminModel {
  id: number;
  name: string;
  provider: string;
  model: string;
  base_url: string;
  api_key_masked: string;
  temperature: number;
  thinking: boolean;
  is_default: boolean;
  is_enabled: boolean;
  sort_order: number;
  config_version: number;
  created_at: string;
  updated_at: string;
}

export interface CreateModelRequest {
  name: string;
  provider: string;
  model: string;
  base_url: string;
  api_key: string;
  temperature?: number;
  thinking?: boolean;
  is_default?: boolean;
  sort_order?: number;
}

export interface UpdateModelRequest {
  name?: string;
  provider?: string;
  model?: string;
  base_url?: string;
  api_key?: string;
  temperature?: number;
  thinking?: boolean;
  is_default?: boolean;
  is_enabled?: boolean;
  sort_order?: number;
}

// 在 aiApi 对象中添加以下方法:

  // 获取可用模型列表
  async getModels(): Promise<ApiResponse<AIModelsResponse>> {
    return apiService.get('/ai/models');
  },

  // 切换会话模型
  async switchSessionModel(sessionId: string, modelId: number): Promise<ApiResponse<{ session_id: string; model_id: number; model_name: string }>> {
    return apiService.put(`/ai/sessions/${sessionId}/model`, { model_id: modelId });
  },

  // === 管理员 API ===

  // 获取所有模型配置（管理员）
  async listAdminModels(): Promise<ApiResponse<{ models: AIAdminModel[] }>> {
    return apiService.get('/admin/ai/models');
  },

  // 创建模型配置
  async createModel(params: CreateModelRequest): Promise<ApiResponse<{ id: number }>> {
    return apiService.post('/admin/ai/models', params);
  },

  // 更新模型配置
  async updateModel(id: number, params: UpdateModelRequest): Promise<ApiResponse<{ config_version: number }>> {
    return apiService.put(`/admin/ai/models/${id}`, params);
  },

  // 删除模型配置
  async deleteModel(id: number): Promise<ApiResponse<void>> {
    return apiService.delete(`/admin/ai/models/${id}`);
  },
```

- [ ] **Step 2: 实现 useModels Hook**

创建 `web/src/components/AI/hooks/useModels.ts`:

```typescript
import { useState, useEffect, useCallback } from 'react';
import { aiApi, AIModel, AIModelsResponse } from '../../../api/modules/ai';

interface UseModelsResult {
  models: AIModel[];
  defaultModelId: number | null;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

export function useModels(): UseModelsResult {
  const [models, setModels] = useState<AIModel[]>([]);
  const [defaultModelId, setDefaultModelId] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchModels = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await aiApi.getModels();
      const data = response.data as AIModelsResponse;
      setModels(data.models || []);
      setDefaultModelId(data.default_model_id || null);
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取模型列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchModels();
  }, [fetchModels]);

  return {
    models,
    defaultModelId,
    loading,
    error,
    refresh: fetchModels,
  };
}

interface UseSwitchModelResult {
  switchModel: (sessionId: string, modelId: number) => Promise<{ success: boolean; modelName?: string; error?: string }>;
  switching: boolean;
}

export function useSwitchModel(): UseSwitchModelResult {
  const [switching, setSwitching] = useState(false);

  const switchModel = useCallback(async (sessionId: string, modelId: number) => {
    setSwitching(true);
    try {
      const response = await aiApi.switchSessionModel(sessionId, modelId);
      return { success: true, modelName: (response.data as any)?.model_name };
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : '切换模型失败' };
    } finally {
      setSwitching(false);
    }
  }, []);

  return { switchModel, switching };
}
```

- [ ] **Step 3: 实现 ModelSelector 组件**

创建 `web/src/components/AI/ModelSelector.tsx`:

```tsx
import React from 'react';
import { Dropdown, Button, Spin, message } from 'antd';
import { DownOutlined, RobotOutlined } from '@ant-design/icons';
import { useModels, useSwitchModel } from './hooks/useModels';
import type { MenuProps } from 'antd';

interface ModelSelectorProps {
  sessionId: string;
  currentModelId?: number | null;
  onModelChange?: (modelId: number, modelName: string) => void;
  disabled?: boolean;
}

export function ModelSelector({
  sessionId,
  currentModelId,
  onModelChange,
  disabled,
}: ModelSelectorProps) {
  const { models, defaultModelId, loading, error } = useModels();
  const { switchModel, switching } = useSwitchModel();

  const currentModel = models.find((m) => m.id === currentModelId) ||
    models.find((m) => m.id === defaultModelId) ||
    models[0];

  const handleMenuClick: MenuProps['onClick'] = async (e) => {
    const modelId = Number(e.key);
    if (modelId === currentModelId) {
      return;
    }

    const result = await switchModel(sessionId, modelId);
    if (result.success) {
      const newModel = models.find((m) => m.id === modelId);
      message.success(`已切换至 ${newModel?.name || '新模型'}`);
      onModelChange?.(modelId, newModel?.name || '');
    } else {
      message.error(result.error || '切换失败');
    }
  };

  const menuItems: MenuProps['items'] = models.map((model) => ({
    key: String(model.id),
    label: (
      <span>
        {model.name}
        {model.is_default && <span style={{ color: '#999', marginLeft: 8 }}>(默认)</span>}
      </span>
    ),
  }));

  if (loading) {
    return (
      <Button size="small" disabled>
        <Spin size="small" />
      </Button>
    );
  }

  if (error || models.length === 0) {
    return null;
  }

  return (
    <Dropdown
      menu={{ items: menuItems, onClick: handleMenuClick }}
      trigger={['click']}
      disabled={disabled || switching}
    >
      <Button size="small" style={{ marginLeft: 8 }}>
        <RobotOutlined />
        <span style={{ marginLeft: 4 }}>{currentModel?.name || '选择模型'}</span>
        <DownOutlined style={{ marginLeft: 4, fontSize: 10 }} />
      </Button>
    </Dropdown>
  );
}
```

- [ ] **Step 4: 集成到 CopilotSurface**

修改 `web/src/components/AI/CopilotSurface.tsx`，在 Drawer 标题区域添加模型选择器:

```tsx
// 在文件顶部导入
import { ModelSelector } from './ModelSelector';

// 在组件内添加状态
const [currentModelId, setCurrentModelId] = React.useState<number | null>(null);

// 在 Drawer 的 title 或 extra 区域添加
// 找到 Drawer 组件，修改 title 或 extra 部分
// 例如在 extra 的 Space 中添加:
<ModelSelector
  sessionId={activeConversationKey === NEW_SESSION_KEY ? '' : activeConversationKey}
  currentModelId={currentModelId}
  onModelChange={(id, name) => {
    setCurrentModelId(id);
    // 可选：显示切换提示
  }}
  disabled={activeConversationKey === NEW_SESSION_KEY}
/>
```

- [ ] **Step 5: 运行前端测试**

Run: `cd /root/project/k8s-manage/web && npm run build`
Expected: 编译成功

- [ ] **Step 6: 提交**

```bash
git add web/src/api/modules/ai.ts
git add web/src/components/AI/hooks/useModels.ts
git add web/src/components/AI/ModelSelector.tsx
git add web/src/components/AI/CopilotSurface.tsx
git commit -m "feat(web): add ModelSelector component for session-level model switching"
```

---

## Chunk 5: 集成与测试

### Task 9: 集成 AgentManager 到 Logic

**Files:**
- Modify: `internal/service/ai/logic/logic.go`

- [ ] **Step 1: 在 Logic 中集成 AgentManager**

修改 `internal/service/ai/logic/logic.go`，在 `Logic` 结构体中添加 `AgentManager`:

```go
// Logic 封装 AI 模块的核心业务逻辑。
type Logic struct {
	svcCtx             *svc.ServiceContext
	ChatDAO            *aidao.AIChatDAO
	// ... 其他字段保持不变

	// AgentManager 按 model_id 缓存 Agent 实例
	agentManager *ai.AgentManager
}

// NewLogic 创建 Logic 实例。
func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	llmProviderLogic := NewLLMProviderLogic(svcCtx.DB)
	return &Logic{
		svcCtx:       svcCtx,
		ChatDAO:      aidao.NewAIChatDAO(svcCtx.DB),
		// ...
		agentManager: ai.NewAgentManager(llmProviderLogic),
	}
}
```

- [ ] **Step 2: 修改 Chat 方法使用 AgentManager**

在 `Logic.Chat` 方法中，替换 `agents.NewRouter(ctx)` 为从 `AgentManager` 获取:

```go
// 在 Chat 方法中找到创建 agent 的位置，修改为:

	// 获取会话关联的模型ID
	var modelID uint64
	if shell.Session != nil && shell.Session.ModelID != nil {
		modelID = *shell.Session.ModelID
	}

	// 获取或创建 Agent 实例
	agent, err := l.agentManager.GetOrCreateAgent(ctx, modelID)
	if err != nil {
		// 回退到默认
		agent, err = agents.NewRouter(ctx)
		if err != nil {
			return nil, err
		}
	}
```

- [ ] **Step 3: 运行编译验证**

Run: `cd /root/project/k8s-manage && go build ./...`
Expected: 编译成功

- [ ] **Step 4: 运行完整测试**

Run: `cd /root/project/k8s-manage && go test ./internal/... -v -count=1`
Expected: 所有测试通过

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/logic/logic.go
git commit -m "feat(ai): integrate AgentManager into Logic for model-based agent selection"
```

---

### Task 10: 运行数据库迁移

- [ ] **Step 1: 应用数据库迁移**

Run: `cd /root/project/k8s-manage && make migrate-up`
Expected: 迁移成功

- [ ] **Step 2: 验证表结构**

Run: `mysql -e "DESCRIBE ai_llm_providers;"` (或使用项目配置的数据库)
Expected: 表结构正确创建

---

## 验收清单

- [ ] 数据库迁移成功，`ai_llm_providers` 表和 `ai_chat_sessions.model_id` 字段存在
- [ ] 后端 API `/api/v1/ai/models` 返回可用模型列表
- [ ] 后端 API `/api/v1/ai/sessions/:id/model` 可切换会话模型
- [ ] 管理员 API 可创建/更新/删除模型配置
- [ ] 前端模型选择器显示在会话标题旁
- [ ] 切换模型后下一条消息使用新模型
- [ ] 无模型配置时回退到 config.yaml 配置

---

## 风险缓解

| 风险 | 缓解措施 |
|------|----------|
| 模型配置错误导致服务不可用 | 回退到 config.yaml 配置 |
| Agent 缓存内存泄漏 | LRU 淘汰 + 最大实例数限制 |
| API Key 泄露 | 加密存储 + 脱敏返回 + 权限控制 |
| 配置变更后缓存不更新 | config_version 校验 + Update 时递增版本 |
