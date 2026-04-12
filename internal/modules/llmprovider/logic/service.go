// Package logic 提供 LLM Provider 的业务逻辑。
package logic

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/dao"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/model"
	"gorm.io/gorm"
)

// Service 提供 LLM Provider 的业务逻辑方法。
type Service struct {
	dao *dao.LLMProviderDAO
}

// NewService 创建 LLM Provider 业务逻辑服务。
func NewService(db *gorm.DB) *Service {
	return &Service{
		dao: dao.NewLLMProviderDAO(db),
	}
}

// Create 创建 LLM Provider 配置。
func (s *Service) Create(ctx context.Context, provider *model.AILLMProvider) error {
	return s.dao.Create(ctx, provider)
}

// Update 更新 LLM Provider 配置。
func (s *Service) Update(ctx context.Context, provider *model.AILLMProvider) error {
	return s.dao.Update(ctx, provider)
}

// GetByID 根据 ID 获取 LLM Provider 配置。
func (s *Service) GetByID(ctx context.Context, id uint64) (*model.AILLMProvider, error) {
	return s.dao.GetByID(ctx, id)
}

// GetDefault 获取默认的 LLM Provider 配置。
func (s *Service) GetDefault(ctx context.Context) (*model.AILLMProvider, error) {
	return s.dao.GetDefault(ctx)
}

// GetFirstEnabled 获取第一个启用的 LLM Provider。
func (s *Service) GetFirstEnabled(ctx context.Context) (*model.AILLMProvider, error) {
	return s.dao.GetFirstEnabled(ctx)
}

// ListEnabled 获取所有启用的 LLM Provider。
func (s *Service) ListEnabled(ctx context.Context) ([]model.AILLMProvider, error) {
	return s.dao.ListEnabled(ctx)
}

// ListAll 获取所有 LLM Provider。
func (s *Service) ListAll(ctx context.Context) ([]model.AILLMProvider, error) {
	return s.dao.ListAll(ctx)
}

// SoftDelete 软删除 LLM Provider 配置。
func (s *Service) SoftDelete(ctx context.Context, id uint64) error {
	return s.dao.SoftDelete(ctx, id)
}

// ClearDefault 清除默认标记。
func (s *Service) ClearDefault(ctx context.Context) error {
	return s.dao.ClearDefault(ctx)
}
