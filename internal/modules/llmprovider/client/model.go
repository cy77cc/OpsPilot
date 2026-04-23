package client

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	arkmodel "github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/claude"
	ollamamodel "github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	qwenmodel "github.com/cloudwego/eino-ext/components/model/qwen"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/constants"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	llmdao "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/dao"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/redis/go-redis/v9"
	arkruntime "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"gorm.io/gorm"
)

// ChatModelConfig 描述聊天模型初始化时的运行时选项。
type ChatModelConfig struct {
	Timeout  time.Duration
	Thinking bool
	Temp     float32
}

var (
	modelCache sync.Map // map[string]einomodel.ToolCallingChatModel
	healthMap  sync.Map // map[uint64]bool (id -> isHealthy)
)

// InitCacheWatcher 启动 Redis 订阅，当配置变更时清除本地缓存。
func InitCacheWatcher(rdb redis.UniversalClient, db *gorm.DB) {
	if rdb != nil {
		go func() {
			pubsub := rdb.Subscribe(context.Background(), constants.LLMConfigUpdateChannel)
			defer pubsub.Close()

			ch := pubsub.Channel()
			for range ch {
				modelCache.Range(func(key, value any) bool {
					modelCache.Delete(key)
					return true
				})
			}
		}()
	}

	if db != nil {
		go startHealthCheckLoop(db)
	}
}

func startHealthCheckLoop(db *gorm.DB) {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	ctx := context.Background()
	for range ticker.C {
		dao := llmdao.NewLLMProviderDAO(db)
		providers, err := dao.ListEnabled(ctx)
		if err != nil {
			log.Printf("llmprovider: health check list providers failed: %v", err)
			continue
		}

		for _, p := range providers {
			p := p
			go func() {
				pForUse, _ := decryptProviderAPIKey(&p)
				if pForUse == nil {
					return
				}
				err := checkProviderHealth(ctx, pForUse)
				if err != nil {
					log.Printf("llmprovider: health check failed for %s (%s): %v", p.Name, p.Model, err)
					healthMap.Store(p.ID, false)
					// Invalidate cache on failure
					modelCache.Range(func(key, value any) bool {
						if strings.HasPrefix(key.(string), fmt.Sprintf("%d:", p.ID)) {
							modelCache.Delete(key)
						}
						return true
					})
				} else {
					healthMap.Store(p.ID, true)
				}
			}()
		}
	}
}

func checkProviderHealth(ctx context.Context, p *model.AILLMProvider) error {
	m, err := NewChatModelFromProvider(ctx, p, ChatModelConfig{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return err
	}
	_, err = m.Generate(ctx, []*schema.Message{schema.UserMessage("ping")})
	return err
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
		dao := llmdao.NewLLMProviderDAO(db)
		// 按照 is_default, sort_order 排序获取所有启用的供应商
		providers, err := dao.ListEnabled(ctx)
		if err != nil {
			return nil, fmt.Errorf("list enabled llm providers: %w", err)
		}

		// 找到第一个健康的（或者还未检测过健康的）
		var provider *model.AILLMProvider
		for i := range providers {
			p := &providers[i]
			isHealthy, ok := healthMap.Load(p.ID)
			if !ok || isHealthy.(bool) {
				provider = p
				break
			}
		}

		// 如果都挂了，由于目前没法完全确信 healthMap，还是尝试用第一个（默认的）作为保底
		if provider == nil && len(providers) > 0 {
			provider = &providers[0]
		}

		if provider != nil {
			cacheKey := fmt.Sprintf("%d:%v", provider.ID, opts)
			if val, ok := modelCache.Load(cacheKey); ok {
				return val.(einomodel.ToolCallingChatModel), nil
			}

			providerForUse, decErr := decryptProviderAPIKey(provider)
			if decErr != nil {
				return nil, fmt.Errorf("decrypt llm provider api key: %w", decErr)
			}
			m, err := NewChatModelFromProvider(ctx, providerForUse, opts)
			if err == nil {
				modelCache.Store(cacheKey, m)
			}
			return m, err
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
	case "openai", "deepseek", "moonshot", "zhipu", "google":
		temp := opts.Temp
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:      config.CFG.LLM.APIKey,
			BaseURL:     config.CFG.LLM.BaseURL,
			Model:       config.CFG.LLM.Model,
			Temperature: &temp,
			Timeout:     opts.Timeout,
		})
	case "minimax":
		temp := opts.Temp
		return claude.NewChatModel(ctx, &claude.Config{
			APIKey:      config.CFG.LLM.APIKey,
			BaseURL:     &config.CFG.LLM.BaseURL,
			Model:       config.CFG.LLM.Model,
			Temperature: &temp,
			Thinking: &claude.Thinking{
				Enable: false,
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
func CheckModelHealth(ctx context.Context, db *gorm.DB) error {
	m, err := GetDefaultChatModel(ctx, db, ChatModelConfig{
		Timeout:  10 * time.Second,
		Thinking: false,
		Temp:     0,
	})
	if err != nil {
		return err
	}
	_, err = m.Generate(ctx, []*schema.Message{schema.UserMessage("ping")})
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
