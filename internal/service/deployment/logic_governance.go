// Package deployment 提供服务治理策略的业务逻辑实现。
//
// 本文件包含服务治理策略的查询和更新逻辑。
package deployment

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
)

// GetGovernance 获取服务治理策略。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//   - env: 环境
//
// 返回: 服务治理策略
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

// UpsertGovernance 创建或更新服务治理策略。
//
// 参数:
//   - ctx: 上下文
//   - uid: 用户 ID
//   - serviceID: 服务 ID
//   - req: 治理策略请求
//
// 返回: 服务治理策略
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
