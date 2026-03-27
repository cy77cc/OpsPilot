package chatmodel

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	arkmodel "github.com/cloudwego/eino-ext/components/model/ark"
	ollamamodel "github.com/cloudwego/eino-ext/components/model/ollama"
	qwenmodel "github.com/cloudwego/eino-ext/components/model/qwen"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/config"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/utils"
	arkruntime "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"gorm.io/gorm"
)

// ChatModelConfig 描述聊天模型初始化时的运行时选项。
type ChatModelConfig struct {
	Timeout  time.Duration
	Thinking bool
	Temp     float32
}

// NewChatModel 根据配置创建聊天模型实例。
func NewChatModel(ctx context.Context, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	return newConfiguredChatModel(ctx, opts)
}

// GetDefaultChatModel 获取默认模型并创建聊天模型实例。
//
// 回退优先级：
//  1. 数据库 is_default = true 的启用模型
//  2. 数据库 ID 最小的启用模型
//  3. config.yaml 中的配置
func GetDefaultChatModel(ctx context.Context, db *gorm.DB, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !config.CFG.LLM.Enable {
		return nil, fmt.Errorf("llm disabled")
	}

	if db == nil {
		db = dbFromRuntimeContext(ctx)
	}
	if db != nil {
		dao := aidao.NewLLMProviderDAO(db)
		provider, err := dao.GetDefault(ctx)
		if err != nil {
			return nil, fmt.Errorf("get default llm provider: %w", err)
		}
		if provider == nil {
			provider, err = dao.GetFirstEnabled(ctx)
			if err != nil {
				return nil, fmt.Errorf("get first enabled llm provider: %w", err)
			}
		}
		if provider != nil {
			providerForUse, decErr := decryptProviderAPIKey(provider)
			if decErr != nil {
				return nil, fmt.Errorf("decrypt llm provider api key: %w", decErr)
			}
			return NewChatModelFromProvider(ctx, providerForUse, opts)
		}
	}

	return newConfiguredChatModel(ctx, opts)
}

func newConfiguredChatModel(ctx context.Context, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !config.CFG.LLM.Enable {
		return nil, fmt.Errorf("llm disabled")
	}

	switch strings.TrimSpace(strings.ToLower(config.CFG.LLM.Provider)) {
	case "ollama":
		return ollamamodel.NewChatModel(ctx, &ollamamodel.ChatModelConfig{
			BaseURL: config.CFG.LLM.BaseURL,
			Model:   config.CFG.LLM.Model,
			Timeout: opts.Timeout,
		})
	case "qwen":
		thinking := opts.Thinking
		temp := opts.Temp
		return qwenmodel.NewChatModel(ctx, &qwenmodel.ChatModelConfig{
			APIKey:         config.CFG.LLM.APIKey,
			BaseURL:        config.CFG.LLM.BaseURL,
			Model:          config.CFG.LLM.Model,
			Temperature:    &temp,
			Timeout:        opts.Timeout,
			EnableThinking: &thinking,
		})
	case "ark":
		temp := opts.Temp
		return arkmodel.NewChatModel(ctx, &arkmodel.ChatModelConfig{
			APIKey:      config.CFG.LLM.APIKey,
			BaseURL:     config.CFG.LLM.BaseURL,
			Model:       config.CFG.LLM.Model,
			Temperature: &temp,
			Timeout:     &opts.Timeout,
			Thinking: &arkruntime.Thinking{
				Type: arkruntime.ThinkingTypeDisabled,
			},
		})
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", config.CFG.LLM.Provider)
	}
}

func decryptProviderAPIKey(provider *model.AILLMProvider) (*model.AILLMProvider, error) {
	if provider == nil {
		return nil, nil
	}

	out := *provider
	cipherText := strings.TrimSpace(out.APIKey)
	if out.ID == 0 || cipherText == "" {
		return &out, nil
	}

	plain, err := utils.DecryptText(cipherText, strings.TrimSpace(config.CFG.Security.EncryptionKey))
	if err != nil {
		return nil, err
	}
	out.APIKey = plain
	return &out, nil
}

// CheckModelHealth 检查模型健康状态。
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

func dbFromRuntimeContext(ctx context.Context) *gorm.DB {
	services := runtimectx.Services(ctx)
	if services == nil {
		return nil
	}

	value := reflect.ValueOf(services)
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}

	field := value.FieldByName("DB")
	if !field.IsValid() || !field.CanInterface() {
		return nil
	}
	db, _ := field.Interface().(*gorm.DB)
	return db
}
