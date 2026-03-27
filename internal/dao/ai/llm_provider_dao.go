// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现 LLM Provider 配置的数据访问对象。
package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// LLMProviderDAO 负责 LLM 供应商配置的持久化访问。
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

// Update 更新模型配置。
func (d *LLMProviderDAO) Update(ctx context.Context, provider *model.AILLMProvider) error {
	return d.db.WithContext(ctx).Save(provider).Error
}

// GetByID 根据 ID 获取模型配置。
//
// 参数:
//   - ctx: 上下文
//   - id: 模型 ID
//
// 返回: 模型配置或 nil（不存在时）
func (d *LLMProviderDAO) GetByID(ctx context.Context, id uint64) (*model.AILLMProvider, error) {
	var provider model.AILLMProvider
	err := d.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&provider).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

// GetDefault 获取系统默认启用模型。
//
// 返回: 默认模型配置或 nil（不存在时）
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

// GetFirstEnabled 获取第一个启用的模型，用于无默认模型时回退。
//
// 返回: 第一个启用的模型配置或 nil（不存在时）
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

// ListEnabled 获取所有启用的模型配置。
//
// 返回: 启用的模型配置列表（按 sort_order 降序）
func (d *LLMProviderDAO) ListEnabled(ctx context.Context) ([]model.AILLMProvider, error) {
	var providers []model.AILLMProvider
	err := d.db.WithContext(ctx).
		Where("is_enabled = ? AND deleted_at IS NULL", true).
		Order("sort_order DESC, id ASC").
		Find(&providers).Error
	return providers, err
}

// ListAll 获取所有非删除的模型配置。
//
// 返回: 所有模型配置列表
func (d *LLMProviderDAO) ListAll(ctx context.Context) ([]model.AILLMProvider, error) {
	var providers []model.AILLMProvider
	err := d.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("sort_order DESC, id ASC").
		Find(&providers).Error
	return providers, err
}

// SoftDelete 软删除模型配置。
//
// 参数:
//   - ctx: 上下文
//   - id: 模型 ID
//
// 返回: 错误信息
func (d *LLMProviderDAO) SoftDelete(ctx context.Context, id uint64) error {
	return d.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.AILLMProvider{}).Error
}

// ClearDefault 清除所有默认标记。
//
// 在设置新默认模型前调用，确保只有一个默认模型。
//
// 返回: 错误信息
func (d *LLMProviderDAO) ClearDefault(ctx context.Context) error {
	return d.db.WithContext(ctx).
		Model(&model.AILLMProvider{}).
		Where("is_default = ? AND deleted_at IS NULL", true).
		Update("is_default", false).Error
}
