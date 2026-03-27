// Package service 提供服务目录管理的业务逻辑层。
//
// 本文件定义服务管理模块的核心业务逻辑结构。
package service

import "github.com/cy77cc/OpsPilot/internal/svc"

// Logic 封装服务管理模块的业务逻辑。
//
// 提供服务 CRUD、渲染、部署、版本管理等核心业务功能。
type Logic struct {
	svcCtx *svc.ServiceContext
}

// NewLogic 创建服务管理模块的业务逻辑实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//
// 返回: Logic 实例指针
func NewLogic(svcCtx *svc.ServiceContext) *Logic { return &Logic{svcCtx: svcCtx} }
