// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现 AI 工具风险策略的数据访问对象。
package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// AIToolRiskPolicyDAO 提供 AI 工具风险策略的数据访问功能。
type AIToolRiskPolicyDAO struct {
	db *gorm.DB
}

// NewAIToolRiskPolicyDAO 创建 AI 工具风险策略 DAO 实例。
func NewAIToolRiskPolicyDAO(db *gorm.DB) *AIToolRiskPolicyDAO {
	return &AIToolRiskPolicyDAO{db: db}
}

// ListEnabledByToolName 列出工具的已启用策略。
//
// 按优先级降序排列，优先级高的策略优先匹配。
//
// 参数:
//   - ctx: 上下文
//   - toolName: 工具名称
//
// 返回: 策略列表
func (d *AIToolRiskPolicyDAO) ListEnabledByToolName(ctx context.Context, toolName string) ([]model.AIToolRiskPolicy, error) {
	var policies []model.AIToolRiskPolicy
	err := d.db.WithContext(ctx).
		Where("tool_name = ? AND enabled = ?", toolName, true).
		Order("priority DESC, id ASC").
		Find(&policies).Error
	return policies, err
}
