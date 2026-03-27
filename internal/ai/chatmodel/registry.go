package chatmodel

import (
	"context"
	"fmt"
	"strings"
	"sync"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cy77cc/OpsPilot/internal/model"
)

// ModelFactory 根据数据库配置构建聊天模型。
type ModelFactory interface {
	Create(ctx context.Context, provider *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error)
}

var registry = struct {
	sync.RWMutex
	factories map[string]ModelFactory
}{
	factories: make(map[string]ModelFactory),
}

func normalizeProviderName(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

// Register 注册供应商工厂。
func Register(provider string, factory ModelFactory) {
	if factory == nil {
		return
	}

	key := normalizeProviderName(provider)
	if key == "" {
		return
	}

	registry.Lock()
	defer registry.Unlock()
	registry.factories[key] = factory
}

// GetFactory 获取已注册的供应商工厂。
func GetFactory(provider string) (ModelFactory, bool) {
	registry.RLock()
	defer registry.RUnlock()
	factory, ok := registry.factories[normalizeProviderName(provider)]
	return factory, ok
}

// ResetRegistryForTest 清空注册表，供测试使用。
func ResetRegistryForTest() {
	registry.Lock()
	defer registry.Unlock()
	registry.factories = make(map[string]ModelFactory)
}

// NewChatModelFromProvider 根据数据库配置创建聊天模型实例。
func NewChatModelFromProvider(ctx context.Context, provider *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if provider == nil {
		return nil, fmt.Errorf("llm provider is nil")
	}

	factory, ok := GetFactory(provider.Provider)
	if !ok {
		return nil, fmt.Errorf("unsupported llm provider %q", provider.Provider)
	}
	return factory.Create(ctx, provider, opts)
}
