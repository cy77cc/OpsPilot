// Package service 提供服务环境匹配校验相关的业务逻辑。
//
// 本文件实现服务环境与集群环境类型的匹配校验。
package service

import (
	"context"
	"errors"

	"github.com/cy77cc/OpsPilot/internal/model"
)

// ValidateEnvMatch 校验服务环境与集群环境类型是否匹配。
//
// 校验规则:
//   - 集群 env_type 为空或 'development' 时跳过校验（兼容现有数据）
//   - 服务环境必须与集群 env_type 匹配
//
// 参数:
//   - ctx: 上下文
//   - serviceEnv: 服务环境
//   - clusterID: 集群 ID
//
// 返回: 不匹配时返回错误，匹配返回 nil
func (l *Logic) ValidateEnvMatch(ctx context.Context, serviceEnv string, clusterID uint) error {
	var cluster model.Cluster
	if err := l.svcCtx.DB.WithContext(ctx).Select("env_type").First(&cluster, clusterID).Error; err != nil {
		return errors.New("cluster not found")
	}

	// 如果集群 env_type 是默认值 'development'，跳过校验（兼容现有数据）
	// 这样新创建的集群在没有明确设置环境类型时，可以部署任何环境的服务
	if cluster.EnvType == "" || cluster.EnvType == "development" {
		return nil
	}

	// 校验服务环境与集群环境类型是否匹配
	if serviceEnv != cluster.EnvType {
		return errors.New("ENV_MISMATCH: service env '" + serviceEnv + "' does not match cluster env_type '" + cluster.EnvType + "'")
	}

	return nil
}
