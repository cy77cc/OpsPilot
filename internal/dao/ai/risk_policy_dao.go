package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// AIToolRiskPolicyDAO provides queries for DB-driven AI tool risk policies.
type AIToolRiskPolicyDAO struct {
	db *gorm.DB
}

// NewAIToolRiskPolicyDAO creates a policy DAO instance.
func NewAIToolRiskPolicyDAO(db *gorm.DB) *AIToolRiskPolicyDAO {
	return &AIToolRiskPolicyDAO{db: db}
}

// ListEnabledByToolName returns enabled policies for a tool ordered by priority.
func (d *AIToolRiskPolicyDAO) ListEnabledByToolName(ctx context.Context, toolName string) ([]model.AIToolRiskPolicy, error) {
	var policies []model.AIToolRiskPolicy
	err := d.db.WithContext(ctx).
		Where("tool_name = ? AND enabled = ?", toolName, true).
		Order("priority DESC, id ASC").
		Find(&policies).Error
	return policies, err
}
