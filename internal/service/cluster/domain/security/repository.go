package security

import (
	"context"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// Phase3Repository 是 Phase 3 领域仓储入口，后续任务中逐步补齐具体读写方法。
type Phase3Repository struct {
	db *gorm.DB
}

func NewPhase3Repository(db *gorm.DB) *Phase3Repository {
	return &Phase3Repository{db: db}
}

func (r *Phase3Repository) UpsertAdmissionPolicy(ctx context.Context, policy model.AdmissionPolicy) (*model.AdmissionPolicy, error) {
	var existing model.AdmissionPolicy
	err := r.db.WithContext(ctx).
		Where("cluster_id = ? AND policy_name = ?", policy.ClusterID, policy.PolicyName).
		First(&existing).Error
	if err == nil {
		updates := map[string]any{
			"version":      policy.Version,
			"status":       policy.Status,
			"content_json": policy.ContentJSON,
			"updated_at":   time.Now().UTC(),
		}
		if uerr := r.db.WithContext(ctx).Model(&model.AdmissionPolicy{}).
			Where("id = ?", existing.ID).
			Updates(updates).Error; uerr != nil {
			return nil, uerr
		}
		if ferr := r.db.WithContext(ctx).First(&existing, existing.ID).Error; ferr != nil {
			return nil, ferr
		}
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	policy.CreatedAt = time.Now().UTC()
	policy.UpdatedAt = policy.CreatedAt
	if err := r.db.WithContext(ctx).Create(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *Phase3Repository) CreateAdmissionExemption(ctx context.Context, rec model.AdmissionExemption) (*model.AdmissionExemption, error) {
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Phase3Repository) GetAdmissionExemption(ctx context.Context, clusterID uint, id uint) (*model.AdmissionExemption, error) {
	var rec model.AdmissionExemption
	if err := r.db.WithContext(ctx).
		Where("cluster_id = ? AND id = ?", clusterID, id).
		First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Phase3Repository) UpdateAdmissionExemptionStatus(ctx context.Context, clusterID uint, id uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.AdmissionExemption{}).
		Where("cluster_id = ? AND id = ?", clusterID, id).
		Update("status", status).Error
}
