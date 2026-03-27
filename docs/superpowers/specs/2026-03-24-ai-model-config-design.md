# AI 模型配置管理设计

- Date: 2026-03-24
- Goal: 支持管理员配置多个 LLM 供应商和模型，统一管理全局默认模型

## 1. 背景与目标

### 当前问题

- 模型配置硬编码在 `config.yaml`，全局单一模型
- 无法动态切换模型，更换模型需要修改配置并重启服务

### 目标

1. 管理员可通过数据库配置多个供应商和模型
2. 管理员设置全局默认模型，所有用户统一使用
3. 默认使用配置文件中的模型配置，保证最小可用
4. 优先读取数据库配置，无配置时回退到配置文件
5. 支持 OpenClaw 格式 JSON 配置导入

### 非目标

- 用户级别模型选择
- 会话级别模型切换
- 模型负载均衡/路由

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  AI Assistant                                           ││
│  │  使用模型: Qwen3.5-Plus (由管理员设置)                    ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        Backend API                           │
│  GET  /api/v1/admin/ai/models          管理员：获取模型列表   │
│  POST /api/v1/admin/ai/models          管理员：创建模型配置   │
│  PUT  /api/v1/admin/ai/models/:id      管理员：更新模型配置   │
│  DEL  /api/v1/admin/ai/models/:id      管理员：删除模型配置   │
│  PUT  /api/v1/admin/ai/models/:id/default  管理员：设置默认   │
│  POST /api/v1/admin/ai/models/import   管理员：导入配置       │
│  POST /api/v1/admin/ai/models/import/preview  管理员：预览导入│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Model Provider                           │
│  - 从数据库加载默认模型配置                                   │
│  - 无配置时回退到 config.yaml                                │
│  - 创建对应的 ChatModel 实例                                  │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 数据流

**用户发送消息流程：**

```
1. 用户输入消息
2. 后端获取全局默认模型配置
3. 创建/复用 ChatModel 实例
4. 执行 Agent 并返回结果
```

## 3. 数据模型

### 3.1 新增表：ai_llm_providers

```sql
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
```

**字段说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | VARCHAR(64) | ✅ | 用户看到的显示名称 |
| `provider` | VARCHAR(32) | ✅ | 供应商：qwen/ark/ollama |
| `model` | VARCHAR(128) | ✅ | 模型标识符 |
| `base_url` | VARCHAR(512) | ✅ | API 端点 |
| `api_key` | VARCHAR(256) | ✅ | API 密钥，加密存储 |
| `api_key_version` | INT | ❌ | 密钥加密版本，默认 1，用于密钥轮换 |
| `temperature` | DECIMAL(3,2) | ❌ | 默认 0.7 |
| `thinking` | TINYINT(1) | ❌ | 默认关闭 |
| `is_default` | TINYINT(1) | ❌ | 系统默认模型（Service 层保证唯一性） |
| `is_enabled` | TINYINT(1) | ❌ | 禁用后用户不可选 |
| `sort_order` | INT | ❌ | 列表排序 |
| `config_version` | INT | ❌ | 配置版本号，更新时递增，用于缓存失效 |
| `deleted_at` | TIMESTAMP | ❌ | 软删除时间，保持历史会话引用完整性 |

**约束：**
- `is_default = true` 全局唯一，由 Service 层保证
- `uk_provider_model` 唯一键确保同一供应商下模型标识唯一
- 软删除保留历史配置

### 3.2 Go Model 定义

```go
// Package model 定义数据库模型。
//
// AILLMProvider 存储 LLM 模型配置，支持多供应商管理。

package model

import (
    "time"
    "gorm.io/gorm"
)

// AILLMProvider 存储 LLM 模型配置。
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

## 4. 后端实现

### 4.1 目录结构

```
internal/
├── model/
│   └── ai/llm_provider.go         # 新增 AILLMProvider model
├── dao/ai/
│   └── llm_provider_dao.go        # 新增：模型配置 DAO
├── ai/
│   └── chatmodel/
│       ├── model.go               # 修改：注册模式入口
│       ├── qwen.go                # 新增：Qwen 工厂
│       ├── ark.go                 # 新增：Ark 工厂
│       └── ollama.go              # 新增：Ollama 工厂
├── service/ai/
│   ├── handler/
│   │   └── llm_provider_handler.go # 新增：模型管理 API
│   ├── logic/
│   │   └── llm_provider_logic.go   # 新增：模型配置服务实现
│   └── routes.go                  # 修改：注册新路由
└── xcode/
    └── code.go                    # 新增错误码：4010-4014
```

### 4.2 模型配置服务

```go
// Package logic 提供 AI 模型配置的业务逻辑实现。
//
// LLMProviderService 负责：
//   - 模型配置的 CRUD 操作
//   - 默认模型的唯一性保证
//   - 配置文件回退机制
//   - JSON 导入解析

package logic

import (
    "context"
    "github.com/cy77cc/OpsPilot/internal/model"
)

// LLMProviderService 提供模型配置查询能力。
type LLMProviderService interface {
    // GetDefault 获取全局默认模型配置。
    //
    // 回退优先级：
    //  1. 数据库 is_default = true 的启用模型
    //  2. 数据库 ID 最小的启用模型
    //  3. config.yaml 中的配置（构造虚拟 Provider）
    GetDefault(ctx context.Context) (*model.AILLMProvider, error)

    // ListEnabled 获取所有启用的模型配置（排除软删除）。
    ListEnabled(ctx context.Context) ([]*model.AILLMProvider, error)

    // ListAll 获取所有模型配置（管理员）。
    ListAll(ctx context.Context) ([]*model.AILLMProvider, error)

    // Create 创建模型配置（管理员）。
    // 如果 is_default = true，自动将其他模型的 is_default 置为 false。
    Create(ctx context.Context, provider *model.AILLMProvider) error

    // Update 更新模型配置（管理员）。
    // 支持部分更新：api_key 为空时保持原值。
    Update(ctx context.Context, provider *model.AILLMProvider) error

    // SetDefault 设置默认模型（管理员）。
    // 将指定模型设为默认，其他模型的 is_default 置为 false。
    SetDefault(ctx context.Context, id uint64) error

    // SoftDelete 软删除模型配置（管理员）。
    SoftDelete(ctx context.Context, id uint64) error

    // ImportConfig 导入 OpenClaw 格式的模型配置（管理员）。
    ImportConfig(ctx context.Context, req *ImportRequest) (*ImportResult, error)

    // PreviewImport 预览导入结果（管理员）。
    PreviewImport(ctx context.Context, req *ImportRequest) (*ImportPreview, error)
}
```

**默认模型回退逻辑实现：**

```go
func (s *llmProviderService) GetDefault(ctx context.Context) (*model.AILLMProvider, error) {
    // 1. 尝试获取数据库中的默认模型
    var provider model.AILLMProvider
    err := s.db.Where("is_default = ? AND is_enabled = ? AND deleted_at IS NULL", true, true).
        First(&provider).Error
    if err == nil {
        return &provider, nil
    }

    // 2. 尝试获取 ID 最小的启用模型
    err = s.db.Where("is_enabled = ? AND deleted_at IS NULL", true).
        Order("id ASC").
        First(&provider).Error
    if err == nil {
        return &provider, nil
    }

    // 3. 回退到 config.yaml
    return s.buildProviderFromConfig(), nil
}

func (s *llmProviderService) buildProviderFromConfig() *model.AILLMProvider {
    return &model.AILLMProvider{
        ID:             0, // 0 表示来自配置文件
        Name:           config.CFG.LLM.Model,
        Provider:       config.CFG.LLM.Provider,
        Model:          config.CFG.LLM.Model,
        BaseURL:        config.CFG.LLM.BaseURL,
        APIKey:         config.CFG.LLM.APIKey,
        Temperature:    0.7,
        Thinking:       false,
        IsDefault:      true,
        IsEnabled:      true,
        ConfigVersion:  0, // 0 表示配置文件版本
    }
}
```

### 4.3 Chat Model 创建改造（注册模式）

使用注册模式提升扩展性，新增供应商只需添加注册文件，无需修改核心逻辑。

```go
// Package chatmodel 提供 AI 模型的初始化和健康检查功能。

package chatmodel

import (
    "context"
    "fmt"
    "sync"

    einomodel "github.com/cloudwego/eino/components/model"
    "github.com/cy77cc/OpsPilot/internal/model"
)

// ModelFactory 定义模型工厂接口。
type ModelFactory interface {
    // Create 根据配置创建聊天模型实例。
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
func NewChatModelFromProvider(ctx context.Context, provider *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
    factory, ok := GetFactory(provider.Provider)
    if !ok {
        return nil, fmt.Errorf("unsupported llm provider %q", provider.Provider)
    }
    return factory.Create(ctx, provider, opts)
}
```

**供应商工厂实现示例：**

```go
// File: internal/ai/chatmodel/qwen.go

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

## 5. API 设计

### 5.1 管理员 API

所有 API 均需管理员权限。

#### GET /api/v1/admin/ai/models

获取所有模型配置（含完整信息）。

**Response:**
```json
{
  "code": 1000,
  "msg": "请求成功",
  "data": {
    "models": [
      {
        "id": 1,
        "name": "Qwen3.5-Plus",
        "provider": "qwen",
        "model": "qwen3.5-plus",
        "base_url": "https://coding.dashscope.aliyuncs.com/v1",
        "api_key_masked": "sk-***abc",
        "temperature": 0.7,
        "thinking": false,
        "is_default": true,
        "is_enabled": true,
        "sort_order": 100
      }
    ]
  }
}
```

**注意：** `api_key_masked` 为脱敏后的 API Key，格式为前缀 + `***` + 后缀 3 位。

#### POST /api/v1/admin/ai/models

创建模型配置。

**Request:**
```json
{
  "name": "Doubao-Pro",
  "provider": "ark",
  "model": "doubao-pro-32k",
  "base_url": "https://ark.cn-beijing.volces.com/api/v3",
  "api_key": "xxx",
  "temperature": 0.7,
  "thinking": false,
  "is_default": false,
  "sort_order": 50
}
```

#### PUT /api/v1/admin/ai/models/:id

更新模型配置。**支持部分更新**：

- 未传递的字段保持原值
- `api_key` 为空或 `null` 时保持原值

**Request（部分更新示例）：**
```json
{
  "name": "Doubao-Pro-32K",
  "temperature": 0.5
}
```

#### PUT /api/v1/admin/ai/models/:id/default

设置指定模型为默认模型。

**Response:**
```json
{
  "code": 1000,
  "msg": "请求成功"
}
```

#### DELETE /api/v1/admin/ai/models/:id

软删除模型配置。

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 4010 | 模型不存在 |

#### POST /api/v1/admin/ai/models/import

导入模型配置。

**Request:**
```json
{
  "mode": "merge",
  "provider_mapping": {
    "bailian": "qwen",
    "volcengine": "ark"
  },
  "config": {
    "models": {
      "providers": {
        "bailian": {
          "baseUrl": "https://coding.dashscope.aliyuncs.com/v1",
          "apiKey": "YOUR_API_KEY",
          "models": [
            {
              "id": "qwen3.5-plus",
              "name": "qwen3.5-plus",
              "compat": { "thinkingFormat": "qwen" }
            }
          ]
        }
      }
    },
    "agents": {
      "defaults": {
        "model": {
          "primary": "bailian/qwen3.5-plus"
        }
      }
    }
  }
}
```

**字段说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `mode` | string | 否 | 导入模式：`merge`（默认）/ `replace` |
| `provider_mapping` | object | 否 | 手动指定供应商映射，未指定的使用自动推断 |
| `config` | object | 是 | OpenClaw 格式配置 |

**供应商映射优先级：**

1. `provider_mapping` 中手动指定的映射
2. 根据 `baseUrl` 自动推断
3. 默认使用 `qwen`

**Response:**
```json
{
  "code": 1000,
  "msg": "请求成功",
  "data": {
    "imported": 7,
    "updated": 1,
    "skipped": 0,
    "details": [
      { "provider": "bailian", "model": "qwen3.5-plus", "action": "created" },
      { "provider": "bailian", "model": "MiniMax-M2.5", "action": "updated" }
    ]
  }
}
```

#### POST /api/v1/admin/ai/models/import/preview

预览导入结果（不实际执行导入）。

**Request:**
```json
{
  "mode": "merge",
  "provider_mapping": {
    "bailian": "qwen"
  },
  "config": { ... }
}
```

**Response:**
```json
{
  "code": 1000,
  "msg": "请求成功",
  "data": {
    "providers": [
      {
        "import_key": "bailian",
        "inferred_provider": "qwen",
        "base_url": "https://coding.dashscope.aliyuncs.com/v1",
        "models": [
          {
            "id": "qwen3.5-plus",
            "name": "qwen3.5-plus",
            "action": "create",
            "will_be_default": true
          },
          {
            "id": "MiniMax-M2.5",
            "name": "MiniMax-M2.5",
            "action": "update",
            "will_be_default": false
          }
        ]
      }
    ],
    "default_model": "bailian/qwen3.5-plus",
    "summary": {
      "to_create": 7,
      "to_update": 1,
      "to_skip": 0
    }
  }
}
```

### 5.2 错误码

| 错误码 | 说明 |
|--------|------|
| 4010 | 模型不存在 |
| 4011 | 模型已禁用 |
| 4012 | 模型有关联会话，无法删除 |
| 4013 | JSON 格式无效 |
| 4014 | 导入配置验证失败 |

## 6. JSON 导入功能

### 6.1 字段映射

支持 OpenClaw 格式的 JSON 配置导入，字段映射如下：

| OpenClaw 字段 | 系统字段 | 说明 |
|--------------|---------|------|
| `providers[key].baseUrl` | `base_url` | API 端点 |
| `providers[key].apiKey` | `api_key` | API 密钥 |
| `models[].id` | `model` | 模型标识 |
| `models[].name` | `name` | 显示名称 |
| `models[].compat.thinkingFormat` | `thinking` | 非空则为 true |
| `agents.defaults.model.primary` | `is_default` | 匹配的模型设为默认 |

### 6.2 供应商推断逻辑

```go
// mapProviderType 根据 baseUrl 推断供应商类型。
func mapProviderType(providerName, baseURL string) string {
    // 1. 已知供应商名称映射
    knownProviders := map[string]string{
        "bailian": "qwen",
        "volcengine": "ark",
        "ollama": "ollama",
    }
    if p, ok := knownProviders[providerName]; ok {
        return p
    }

    // 2. 根据 URL 推断
    if strings.Contains(baseURL, "dashscope.aliyuncs.com") {
        return "qwen"
    }
    if strings.Contains(baseURL, "ark.cn-beijing.volces.com") {
        return "ark"
    }
    if strings.Contains(baseURL, "localhost:11434") {
        return "ollama"
    }

    // 3. 默认使用 qwen（兼容 OpenAI API）
    return "qwen"
}
```

### 6.3 导入逻辑实现

```go
// ImportConfig 导入模型配置。
//
// 参数:
//   - ctx: 上下文
//   - req: 导入请求（含模式、映射、配置）
//
// 返回: 导入结果统计
func (s *llmProviderService) ImportConfig(ctx context.Context, req *ImportRequest) (*ImportResult, error) {
    // 1. replace 模式：清除现有配置
    if req.Mode == "replace" {
        s.db.Where("1 = 1").Delete(&model.AILLMProvider{})
    }

    result := &ImportResult{}

    // 2. 解析默认模型
    defaultModel := parseDefaultModel(req.Config)

    // 3. 遍历供应商
    for providerName, providerConfig := range req.Config.Models.Providers {
        systemProvider := s.resolveProvider(providerName, providerConfig.BaseURL, req.ProviderMapping)

        for _, m := range providerConfig.Models {
            // 4. 检查是否已存在
            existing, exists := s.findExisting(systemProvider, m.ID)

            if req.Mode == "merge" && exists {
                // 更新已存在的模型
                s.updateFromImport(existing, providerConfig, m)
                s.db.Save(existing)
                result.Updated++
                result.Details = append(result.Details, ImportDetail{
                    Provider: providerName,
                    Model:    m.ID,
                    Action:   "updated",
                })
            } else {
                // 创建新模型
                newProvider := s.createFromImport(providerConfig, m, systemProvider, defaultModel)
                s.db.Create(newProvider)
                result.Imported++
                result.Details = append(result.Details, ImportDetail{
                    Provider: providerName,
                    Model:    m.ID,
                    Action:   "created",
                })
            }
        }
    }

    return result, nil
}

// resolveProvider 解析供应商类型，优先使用手动映射。
func (s *llmProviderService) resolveProvider(importKey, baseURL string, mapping map[string]string) string {
    // 1. 手动映射优先
    if mapped, ok := mapping[importKey]; ok {
        return mapped
    }
    // 2. 自动推断
    return mapProviderType(importKey, baseURL)
}
```

### 6.4 前端交互流程

1. 管理员上传 JSON 文件或粘贴 JSON 内容
2. 调用预览 API，展示推断结果
3. 管理员可修改供应商映射（下拉选择 qwen/ark/ollama）
4. 确认后调用导入 API 执行导入

## 7. 前端实现

### 7.1 组件结构

```
web/src/
├── pages/Admin/
│   └── AIModelConfig/
│       ├── index.tsx               # 新增：模型配置管理页面
│       ├── ModelForm.tsx           # 新增：模型编辑表单
│       ├── ImportModal.tsx         # 新增：导入配置弹窗
│       └── hooks/
│           └── useLLMProviders.ts  # 新增：模型管理 Hook
└── api/modules/ai/
    └── llmProvider.ts              # 新增：模型配置 API
```

### 7.2 管理员模型配置页面

**功能：**
1. 模型列表展示（名称、供应商、状态、是否默认）
2. 新增模型
3. 编辑模型
4. 删除模型（软删除）
5. 设置默认模型
6. JSON 配置导入

### 7.3 用户端 UI

用户端**无任何模型相关 UI**。所有用户统一使用管理员设置的默认模型，用户感知不到模型存在。

## 8. 安全考虑

### 8.1 API Key 安全

- API Key 在数据库中加密存储（使用现有 `SECURITY_ENCRYPTION_KEY`）
- 管理员 API 返回时脱敏显示（如 `sk-***xxx`）

### 8.2 权限控制

- 所有模型管理 API 需管理员权限
- 用户端无任何模型相关 API

### 8.3 输入验证

- `provider` 限制为枚举值：qwen/ark/ollama
- `base_url` 验证为合法 URL
- `temperature` 范围：0.0 - 2.0
- `model` 长度限制

## 9. 测试策略

### 9.1 单元测试

- `LLMProviderService`：CRUD 操作、默认模型回退逻辑、SetDefault 唯一性保证
- `NewChatModelFromProvider`：各供应商模型创建
- `ImportConfig`：导入逻辑、供应商推断、字段映射

### 9.2 集成测试

- 创建模型配置 → 设为默认 → AI 对话使用新模型
- 删除默认模型 → 回退到配置文件配置
- JSON 导入 → 预览 → 执行导入

### 9.3 E2E 测试

- 管理员添加/编辑/删除模型
- 管理员设置默认模型
- 管理员导入 JSON 配置

## 10. 迁移与兼容

### 10.1 数据迁移

```sql
-- 创建新表
CREATE TABLE ai_llm_providers (...);

-- 无需修改 ai_chat_sessions 表
```

### 10.2 向后兼容

- 数据库无模型配置时，自动使用 `config.yaml` 配置
- 已有会话无需迁移

## 11. 实施计划

### Phase 1: 数据层（1 天）

1. 创建数据库迁移文件
2. 实现 `AILLMProvider` model
3. 实现 `LLMProviderDAO`
4. 实现 `LLMProviderService`（基础 CRUD）

### Phase 2: 后端核心（1 天）

1. 改造 `chatmodel/model.go` 注册模式
2. 实现供应商工厂（qwen/ark/ollama）
3. 修改 AI Logic 获取默认模型

### Phase 3: API 层（0.5 天）

1. 实现管理员 API
2. 权限控制集成

### Phase 4: 导入功能（1 天）

1. 实现导入/预览 API
2. OpenClaw 格式解析
3. 供应商推断逻辑

### Phase 5: 前端（1 天）

1. 实现管理员模型配置页面
2. 实现导入配置弹窗
3. API 模块封装

### Phase 6: 测试（0.5 天）

1. 单元测试
2. 集成测试

## 12. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 模型配置错误导致服务不可用 | 高 | 配置验证 + 健康检查 + 回退机制 |
| API Key 泄露 | 高 | 加密存储 + 脱敏返回 + 权限控制 + api_key_version 支持轮换 |
| 多个 is_default 导致行为不一致 | 中 | Service 层保证唯一性 + 明确回退优先级 |
| 导入格式解析错误 | 低 | 预览机制 + 详细错误信息 |
| 供应商推断错误 | 低 | 支持手动映射覆盖 |
