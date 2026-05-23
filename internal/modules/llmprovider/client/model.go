package client

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/constants"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	llmdao "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/dao"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/model"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
)

// ChatModelConfig 描述聊天模型初始化时的运行时选项。
type ChatModelConfig struct {
	Timeout  time.Duration
	Thinking bool
	Temp     float32
}

var (
	modelCache      sync.Map // map[string]einomodel.ToolCallingChatModel
	healthMap       sync.Map // map[uint64]bool (id -> isHealthy)
	watcherInitOnce sync.Once
)

// InitCacheWatcher 启动 Redis 订阅，当配置变更时清除本地缓存。
func InitCacheWatcher(rdb redis.UniversalClient, db *gorm.DB) {
	watcherInitOnce.Do(func() {
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
			ctx, cancel := context.WithCancel(context.Background())
			go startHealthCheckLoop(ctx, db, cancel)
		}
	})
}

func startHealthCheckLoop(ctx context.Context, db *gorm.DB, shutdown context.CancelFunc) {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()
	defer shutdown()

	const maxConcurrent = 5
	sem := semaphore.NewWeighted(maxConcurrent)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		dao := llmdao.NewLLMProviderDAO(db)
		providers, err := dao.ListEnabled(ctx)
		if err != nil {
			log.Printf("llmprovider: health check list providers failed: %v", err)
			continue
		}

		for _, p := range providers {
			p := p
			if err := sem.Acquire(ctx, 1); err != nil {
				continue
			}
			go func() {
				defer sem.Release(1)
				pForUse, _ := decryptProviderAPIKey(&p)
				if pForUse == nil {
					return
				}
				err := checkProviderHealth(ctx, pForUse)
				if err != nil {
					log.Printf("llmprovider: health check failed for %s (%s): %v", p.Name, p.Model, err)
					healthMap.Store(p.ID, false)
					// Invalidate cache on failure
					prefix := fmt.Sprintf("%d:", p.ID)
					modelCache.Range(func(key, value any) bool {
						keyStr, ok := key.(string)
						if !ok {
							return true
						}
						if strings.HasPrefix(keyStr, prefix) {
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

// GetDefaultChatModel 获取默认模型并创建聊天模型实例。
//
// 回退优先级：
//  1. 数据库 is_default = true 的启用模型
//  2. 数据库 ID 最小的启用模型
//  3. config.yaml 中的配置（通过统一 factory 路径）
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
			logger.L().Warn("list enabled llm providers failed, falling back to config.yaml",
				logger.Error(err))
		} else {
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
				logger.L().Warn("all llm providers appear unhealthy, falling back to first provider",
					logger.String("name", providers[0].Name),
					logger.String("model", providers[0].Model))
				provider = &providers[0]
			}

			if provider != nil {
				logger.L().Debug("using database llm provider",
					logger.String("name", provider.Name),
					logger.String("provider", provider.Provider),
					logger.String("model", provider.Model))

				cacheKey := fmt.Sprintf("%d:%d:%d:%d:%t:%f", provider.ID, provider.ConfigVersion, provider.APIKeyVersion, opts.Timeout.Milliseconds(), opts.Thinking, opts.Temp)
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
	} else {
		logger.L().Warn("no database connection for llm provider lookup, falling back to config.yaml")
	}

	// 回退：使用 config.yaml 配置，通过统一 factory 路径创建模型
	logger.L().Info("using config.yaml llm provider as fallback",
		logger.String("provider", config.CFG.LLM.Provider),
		logger.String("model", config.CFG.LLM.Model))

	return NewChatModelFromProvider(ctx, &model.AILLMProvider{
		Provider:    strings.TrimSpace(strings.ToLower(config.CFG.LLM.Provider)),
		Model:       config.CFG.LLM.Model,
		BaseURL:     config.CFG.LLM.BaseURL,
		APIKey:      config.CFG.LLM.APIKey,
		Temperature: config.CFG.LLM.Temperature,
	}, opts)
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
	if m == nil {
		return fmt.Errorf("no chat model available for health check")
	}
	_, err = m.Generate(ctx, []*schema.Message{schema.UserMessage("ping")})
	return err
}

func dbFromRuntimeContext(ctx context.Context) *gorm.DB {
	if extractor := dbExtractor.Load(); extractor != nil {
		return (*extractor)(ctx)
	}
	return nil
}

// SetDBExtractor registers a function to extract *gorm.DB from context.
// Call this once during application initialization.
func SetDBExtractor(fn func(context.Context) *gorm.DB) {
	wrapped := dbExtractorFunc(fn)
	dbExtractor.Store(&wrapped)
}

type dbExtractorFunc func(context.Context) *gorm.DB

var dbExtractor atomic.Pointer[dbExtractorFunc]
