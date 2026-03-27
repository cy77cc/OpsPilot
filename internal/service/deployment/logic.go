// Package deployment 提供部署管理服务的业务逻辑层。
//
// 本文件定义部署服务的核心业务逻辑结构。
package deployment

import (
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// Logic 是部署服务的业务逻辑层，封装数据库操作和业务规则。
type Logic struct {
	svcCtx *svc.ServiceContext
}

// NewLogic 创建业务逻辑层实例。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: Logic 实例
func NewLogic(svcCtx *svc.ServiceContext) *Logic { return &Logic{svcCtx: svcCtx} }
