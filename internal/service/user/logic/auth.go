// Package logic 提供用户模块的业务逻辑层。
//
// 本文件实现认证相关的业务逻辑，包括登录、注册、Token 刷新和登出。
package logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "github.com/cy77cc/OpsPilot/api/user/v1"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"gorm.io/gorm"
)

// Login 用户登录。
//
// 流程:
//   1. 检查用户是否存在
//   2. 验证密码
//   3. 生成 Access Token 和 Refresh Token
//   4. 将 Refresh Token 加入白名单
//   5. 更新最后登录时间
//   6. 加载用户角色和权限
//
// 参数:
//   - ctx: 上下文
//   - req: 登录请求，包含用户名和密码
//
// 返回: Token 响应，包含 Access Token、Refresh Token 和用户信息
func (l *UserLogic) Login(ctx context.Context, req v1.LoginReq) (v1.TokenResp, error) {
	// 1. Check user existence
	user, err := l.userDAO.FindOneByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return v1.TokenResp{}, xcode.NewErrCode(xcode.UserNotExist)
		}
		return v1.TokenResp{}, fmt.Errorf("failed to query user: %w", err)
	}

	// 2. Verify password
	if !utils.VerifyPassword(req.Password, user.PasswordHash) {
		return v1.TokenResp{}, xcode.NewErrCode(xcode.PasswordError)
	}

	// 3. Generate Token
	token, err := utils.GenToken(uint(user.ID), false)
	if err != nil {
		return v1.TokenResp{}, fmt.Errorf("failed to generate token: %w", err)
	}

	refreshToken, err := utils.GenToken(uint(user.ID), true)
	if err != nil {
		return v1.TokenResp{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	if err := l.whiteListDao.AddToWhitelist(ctx, refreshToken, time.Now().Add(config.CFG.JWT.RefreshExpire)); err != nil {
		return v1.TokenResp{}, xcode.NewErrCode(xcode.CacheError)
	}

	// 4. Update last login time
	user.LastLoginTime = time.Now().Unix()
	if err := l.userDAO.Update(ctx, user); err != nil {
		// Log error but don't fail login? Or fail?
		// For strict consistency, we might fail or just log.
		// Here we'll just log/ignore for now as we don't have logger injected yet
	}

	roles, permissions, _ := l.loadRolesAndPermissions(ctx, uint64(user.ID))
	return v1.TokenResp{
		AccessToken:  token,
		RefreshToken: refreshToken,
		Expires:      time.Now().Add(config.CFG.JWT.Expire).Unix(), // Should match config
		Uid:          uint64(user.ID),
		Roles:        roles,
		User: &v1.AuthUser{
			Id:          uint64(user.ID),
			Username:    user.Username,
			Name:        user.Username,
			Email:       user.Email,
			Status:      "active",
			Roles:       roles,
			Permissions: permissions,
		},
		Permissions: permissions,
	}, nil
}

// Register 用户注册。
//
// 流程:
//   1. 检查用户名是否已存在
//   2. 对密码进行哈希处理
//   3. 创建用户记录
//   4. 为新用户分配默认 viewer 角色
//   5. 生成 Access Token 和 Refresh Token
//   6. 将 Refresh Token 加入白名单
//   7. 加载用户角色和权限
//
// 参数:
//   - ctx: 上下文
//   - req: 注册请求，包含用户名、密码和邮箱
//
// 返回: Token 响应，包含 Access Token、Refresh Token 和用户信息
func (l *UserLogic) Register(ctx context.Context, req v1.UserCreateReq) (v1.TokenResp, error) {
	// 1. Check if user exists
	_, err := l.userDAO.FindOneByUsername(ctx, req.Username)
	if err == nil {
		return v1.TokenResp{}, xcode.NewErrCode(xcode.UserAlreadyExist)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return v1.TokenResp{}, fmt.Errorf("database error: %w", err)
	}

	// 2. Create User
	encryptPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		return v1.TokenResp{}, fmt.Errorf("failed to hash password: %w", err)
	}

	newUser := &model.User{
		Username:     req.Username,
		PasswordHash: encryptPwd, // Plaintext for demo, should be hashed
		Email:        req.Email,
		CreateTime:   time.Now().Unix(),
		UpdateTime:   time.Now().Unix(),
	}

	if err := l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(newUser).Error; err != nil {
			return err
		}
		var viewerRole model.Role
		if err := tx.Where("LOWER(code) = ?", "viewer").First(&viewerRole).Error; err == nil {
			if err := tx.Create(&model.UserRole{UserID: int64(newUser.ID), RoleID: int64(viewerRole.ID)}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return v1.TokenResp{}, fmt.Errorf("failed to create user: %w", err)
	}

	// 3. Generate Token
	token, err := utils.GenToken(uint(newUser.ID), false)
	if err != nil {
		return v1.TokenResp{}, fmt.Errorf("failed to generate token: %w", err)
	}

	refreshToken, err := utils.GenToken(uint(newUser.ID), true)
	if err != nil {
		return v1.TokenResp{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	if err := l.whiteListDao.AddToWhitelist(ctx, refreshToken, time.Now().Add(config.CFG.JWT.RefreshExpire)); err != nil {
		return v1.TokenResp{}, xcode.NewErrCode(xcode.CacheError)
	}

	roles, permissions, _ := l.loadRolesAndPermissions(ctx, uint64(newUser.ID))
	return v1.TokenResp{
		AccessToken:  token,
		RefreshToken: refreshToken,
		Expires:      time.Now().Add(config.CFG.JWT.Expire).Unix(),
		Uid:          uint64(newUser.ID),
		Roles:        roles,
		User: &v1.AuthUser{
			Id:          uint64(newUser.ID),
			Username:    newUser.Username,
			Name:        newUser.Username,
			Email:       newUser.Email,
			Status:      "active",
			Roles:       roles,
			Permissions: permissions,
		},
		Permissions: permissions,
	}, nil
}

// Refresh 刷新 Token。
//
// 流程:
//   1. 检查 Refresh Token 是否在白名单中
//   2. 解析 Token 获取用户信息
//   3. 生成新的 Access Token 和 Refresh Token
//   4. 从白名单删除旧 Token，添加新 Token
//   5. 加载用户角色和权限
//
// 参数:
//   - ctx: 上下文
//   - req: 刷新请求，包含 Refresh Token
//
// 返回: 新的 Token 响应
func (l *UserLogic) Refresh(ctx context.Context, req v1.RefreshReq) (v1.TokenResp, error) {
	// Verify refresh token (simplified)
	// 第一步判断rtoken在不在白名单
	ok, err := l.whiteListDao.IsWhitelisted(ctx, req.RefreshToken)
	if err != nil || !ok {
		return v1.TokenResp{}, xcode.NewErrCode(xcode.TokenInvalid)
	}
	// 解析token，判断过期时间
	claims, err := utils.ParseToken(req.RefreshToken)
	if err != nil {
		return v1.TokenResp{}, xcode.NewErrCode(xcode.TokenExpired)
	}

	// 生成新的atoken和rtoken
	newToken, err := utils.GenToken(claims.Uid, false)
	if err != nil {
		return v1.TokenResp{}, err
	}

	newRefreshToken, err := utils.GenToken(claims.Uid, true)
	if err != nil {
		return v1.TokenResp{}, err
	}

	// 从缓存中删除旧的rtoken，添加新的rtoken
	if err := l.whiteListDao.DeleteToken(ctx, req.RefreshToken); err != nil {
		return v1.TokenResp{}, xcode.NewErrCode(xcode.CacheError)
	}

	if err := l.whiteListDao.AddToWhitelist(ctx, newRefreshToken, time.Now().Add(config.CFG.JWT.RefreshExpire)); err != nil {
		return v1.TokenResp{}, xcode.NewErrCode(xcode.CacheError)
	}

	roles, permissions, _ := l.loadRolesAndPermissions(ctx, uint64(claims.Uid))
	return v1.TokenResp{
		AccessToken:  newToken,
		RefreshToken: newRefreshToken,
		Expires:      time.Now().Add(config.CFG.JWT.Expire).Unix(),
		Uid:          uint64(claims.Uid),
		Roles:        roles,
		Permissions:  permissions,
	}, nil
}

// Logout 用户登出。
//
// 流程:
//   1. 检查 Refresh Token 是否为空
//   2. 从白名单中删除 Refresh Token
//
// 参数:
//   - ctx: 上下文
//   - req: 登出请求，包含 Refresh Token
//
// 返回: 成功返回 nil，失败返回错误
func (l *UserLogic) Logout(ctx context.Context, req v1.LogoutReq) error {
	if strings.TrimSpace(req.RefreshToken) == "" {
		return nil
	}
	if err := l.whiteListDao.DeleteToken(ctx, req.RefreshToken); err != nil {
		return xcode.FromError(err)
	}
	return nil
}

// loadRolesAndPermissions 加载用户的角色和权限列表。
//
// 流程:
//   1. 通过 user_roles 表查询用户的所有角色
//   2. 通过 role_permissions 表查询角色关联的所有权限
//   3. 如果用户是 admin 角色，添加 "*:*" 全局权限
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户 ID
//
// 返回: 角色列表、权限列表、错误
func (l *UserLogic) loadRolesAndPermissions(ctx context.Context, userID uint64) ([]string, []string, error) {
	roleRows := make([]struct {
		Code string `gorm:"column:code"`
	}, 0)
	if err := l.svcCtx.DB.WithContext(ctx).
		Table("roles").
		Select("roles.code").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Scan(&roleRows).Error; err != nil {
		return nil, nil, err
	}
	roles := make([]string, 0, len(roleRows))
	roleSet := make(map[string]struct{}, len(roleRows))
	for _, row := range roleRows {
		code := strings.TrimSpace(row.Code)
		if code == "" {
			continue
		}
		if _, ok := roleSet[code]; ok {
			continue
		}
		roleSet[code] = struct{}{}
		roles = append(roles, code)
	}

	permRows := make([]struct {
		Code string `gorm:"column:code"`
	}, 0)
	if err := l.svcCtx.DB.WithContext(ctx).
		Table("permissions").
		Select("permissions.code").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Scan(&permRows).Error; err != nil {
		return roles, nil, err
	}
	permissions := make([]string, 0, len(permRows)+1)
	permSet := make(map[string]struct{}, len(permRows)+1)
	for _, row := range permRows {
		code := strings.TrimSpace(row.Code)
		if code == "" {
			continue
		}
		if _, ok := permSet[code]; ok {
			continue
		}
		permSet[code] = struct{}{}
		permissions = append(permissions, code)
	}
	for _, roleCode := range roles {
		if strings.EqualFold(roleCode, "admin") {
			if _, ok := permSet["*:*"]; !ok {
				permissions = append(permissions, "*:*")
			}
			break
		}
	}

	return roles, permissions, nil
}
