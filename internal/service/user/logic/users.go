// Package logic 提供用户模块的业务逻辑层。
//
// 本文件实现用户信息相关的业务逻辑，包括用户查询和权限加载。
package logic

import (
	"context"
	"fmt"

	v1 "github.com/cy77cc/OpsPilot/api/user/v1"
	dao "github.com/cy77cc/OpsPilot/internal/dao/user"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// UserLogic 是用户模块的业务逻辑层。
//
// 职责:
//   - 处理用户相关的业务逻辑
//   - 调用 DAO 层进行数据访问
//   - 加载用户的角色和权限信息
type UserLogic struct {
	svcCtx       *svc.ServiceContext    // 服务上下文
	userDAO      *dao.UserDAO           // 用户数据访问对象
	whiteListDao *dao.WhiteListDao      // JWT 白名单数据访问对象
}

// NewUserLogic 创建用户业务逻辑实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库、缓存等依赖
//
// 返回: 用户业务逻辑实例
func NewUserLogic(svcCtx *svc.ServiceContext) *UserLogic {
	return &UserLogic{
		svcCtx:       svcCtx,
		userDAO:      dao.NewUserDAO(svcCtx.DB, svcCtx.Cache, svcCtx.Rdb),
		whiteListDao: dao.NewWhiteListDao(svcCtx.DB, svcCtx.Cache, svcCtx.Rdb),
	}
}

// GetUser 根据用户 ID 获取用户信息。
//
// 参数:
//   - ctx: 上下文
//   - id: 用户 ID
//
// 返回: 用户响应信息，失败返回错误
func (l *UserLogic) GetUser(ctx context.Context, id model.UserID) (v1.UserResp, error) {
	user, err := l.userDAO.FindOneById(ctx, id)
	if err != nil {
		return v1.UserResp{}, err
	}
	return v1.UserResp{
		Id:            uint64(user.ID),
		Username:      user.Username,
		Email:         user.Email,
		Phone:         user.Phone,
		Avatar:        user.Avatar,
		Status:        int32(user.Status),
		CreateTime:    user.CreateTime,
		UpdateTime:    user.UpdateTime,
		LastLoginTime: user.LastLoginTime,
	}, nil
}

// GetMe 获取当前登录用户的完整信息。
//
// 参数:
//   - ctx: 上下文
//   - uid: 用户 ID (支持多种类型: uint, uint64, int, int64, float64)
//
// 返回: 包含用户信息和权限的 map，失败返回错误
//
// 返回字段:
//   - id: 用户 ID
//   - username: 用户名
//   - name: 显示名称
//   - email: 邮箱
//   - status: 状态
//   - roles: 角色列表
//   - permissions: 权限列表
func (l *UserLogic) GetMe(ctx context.Context, uid any) (map[string]any, error) {
	var userID model.UserID
	switch v := uid.(type) {
	case uint:
		userID = model.UserID(v)
	case uint64:
		userID = model.UserID(v)
	case int:
		userID = model.UserID(v)
	case int64:
		userID = model.UserID(v)
	case float64:
		userID = model.UserID(v)
	default:
		return nil, fmt.Errorf("invalid uid type")
	}

	user, err := l.userDAO.FindOneById(ctx, userID)
	if err != nil {
		return nil, err
	}
	roles, permissions, err := l.loadRolesAndPermissions(ctx, uint64(user.ID))
	if err != nil {
		roles = []string{}
		permissions = []string{}
	}
	return map[string]any{
		"id":          user.ID,
		"username":    user.Username,
		"name":        user.Username,
		"email":       user.Email,
		"status":      "active",
		"roles":       roles,
		"permissions": permissions,
	}, nil
}
