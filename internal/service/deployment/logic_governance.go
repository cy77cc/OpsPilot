package deployment

import (
	"context"

	"github.com/cy77cc/k8s-manage/internal/model"
)

func (l *Logic) GetGovernance(ctx context.Context, serviceID uint, env string) (*model.ServiceGovernancePolicy, error) {
	var row model.ServiceGovernancePolicy
	err := l.svcCtx.DB.WithContext(ctx).
		Where("service_id = ? AND env = ?", serviceID, defaultIfEmpty(env, "staging")).
		First(&row).Error
	if err != nil {
		return &model.ServiceGovernancePolicy{ServiceID: serviceID, Env: defaultIfEmpty(env, "staging")}, nil
	}
	return &row, nil
}

func (l *Logic) UpsertGovernance(ctx context.Context, uid uint64, serviceID uint, req GovernanceReq) (*model.ServiceGovernancePolicy, error) {
	env := defaultIfEmpty(req.Env, "staging")
	var row model.ServiceGovernancePolicy
	err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ? AND env = ?", serviceID, env).First(&row).Error
	if err != nil {
		row = model.ServiceGovernancePolicy{ServiceID: serviceID, Env: env}
	}
	row.TrafficPolicyJSON = toJSON(req.TrafficPolicy)
	row.ResiliencePolicyJSON = toJSON(req.ResiliencePolicy)
	row.AccessPolicyJSON = toJSON(req.AccessPolicy)
	row.SLOPolicyJSON = toJSON(req.SLOPolicy)
	row.UpdatedBy = uint(uid)
	if row.ID == 0 {
		if err := l.svcCtx.DB.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	} else if err := l.svcCtx.DB.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
