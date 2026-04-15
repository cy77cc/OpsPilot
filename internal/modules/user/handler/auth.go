// Package handler 提供用户模块的 HTTP 处理器。
//
// 本文件实现认证相关的 HTTP 处理器，包括登录、注册、Token 刷新和登出。
package handler

import (
	"context"
	"net/http"
	"time"

	v1 "github.com/cy77cc/OpsPilot/api/user/v1"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	userLogic "github.com/cy77cc/OpsPilot/internal/modules/user/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

type authLogic interface {
	Login(ctx context.Context, req v1.LoginReq) (v1.TokenResp, error)
	Register(ctx context.Context, req v1.UserCreateReq) (v1.TokenResp, error)
	Refresh(ctx context.Context, req v1.RefreshReq) (v1.TokenResp, error)
	Logout(ctx context.Context, req v1.LogoutReq) error
	GetMe(ctx context.Context, uid any) (map[string]any, error)
}

var newAuthLogic = func(svcCtx *svc.ServiceContext) authLogic {
	return userLogic.NewUserLogic(svcCtx)
}

const (
	authAccessCookieName  = "opspilot_at"
	authRefreshCookieName = "opspilot_rt"
)

func authCookieMaxAge(ttl time.Duration) int {
	if ttl <= 0 {
		return 0
	}
	return int(ttl.Seconds())
}

func setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(authAccessCookieName, accessToken, authCookieMaxAge(config.CFG.JWT.Expire), "/", "", true, true)
	c.SetCookie(authRefreshCookieName, refreshToken, authCookieMaxAge(config.CFG.JWT.RefreshExpire), "/", "", true, true)
}

func clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(authAccessCookieName, "", -1, "/", "", true, true)
	c.SetCookie(authRefreshCookieName, "", -1, "/", "", true, true)
}

// AuthPublicResp 是认证接口对外返回的数据（不包含 token 字段）。
type AuthPublicResp struct {
	User        *v1.AuthUser `json:"user,omitempty"`
	Roles       []string     `json:"roles"`
	Permissions []string     `json:"permissions,omitempty"`
}

func authPublicResp(resp v1.TokenResp) AuthPublicResp {
	return AuthPublicResp{
		User:        resp.User,
		Roles:       resp.Roles,
		Permissions: resp.Permissions,
	}
}

// Login 用户登录。
//
// @Summary 用户登录
// @Description 用户登录获取 Token
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param request body v1.LoginReq true "登录请求"
// @Success 200 {object} httpx.Response{data=AuthPublicResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /auth/login [post]
func (u *UserHandler) Login(c *gin.Context) {
	var req v1.LoginReq
	if err := c.ShouldBind(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := newAuthLogic(u.svcCtx).Login(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	setAuthCookies(c, resp.AccessToken, resp.RefreshToken)
	httpx.OK(c, authPublicResp(resp))
}

// Register 用户注册。
//
// @Summary 用户注册
// @Description 注册新用户并返回 Token
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param request body v1.UserCreateReq true "注册请求"
// @Success 200 {object} httpx.Response{data=v1.TokenResp}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /auth/register [post]
func (u *UserHandler) Register(c *gin.Context) {
	var req v1.UserCreateReq
	if err := c.ShouldBind(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := newAuthLogic(u.svcCtx).Register(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

// Refresh 刷新 Token。
//
// @Summary 刷新 Token
// @Description 使用 Refresh Token 获取新的 Access Token
// @Tags 用户认证
// @Accept json
// @Produce json
// @Success 200 {object} httpx.Response{data=AuthPublicResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /auth/refresh [post]
func (u *UserHandler) Refresh(c *gin.Context) {
	req := v1.RefreshReq{}
	if refreshToken, err := c.Cookie(authRefreshCookieName); err == nil {
		req.RefreshToken = refreshToken
	}

	resp, err := newAuthLogic(u.svcCtx).Refresh(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	setAuthCookies(c, resp.AccessToken, resp.RefreshToken)
	httpx.OK(c, authPublicResp(resp))
}

// Logout 用户登出。
//
// @Summary 用户登出
// @Description 将 Refresh Token 从白名单移除
// @Tags 用户认证
// @Accept json
// @Produce json
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Router /auth/logout [post]
func (u *UserHandler) Logout(c *gin.Context) {
	req := v1.LogoutReq{}
	if refreshToken, err := c.Cookie(authRefreshCookieName); err == nil {
		req.RefreshToken = refreshToken
	}

	if err := newAuthLogic(u.svcCtx).Logout(c.Request.Context(), req); err != nil {
		clearAuthCookies(c)
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	clearAuthCookies(c)
	httpx.OK(c, nil)
}

// Me 获取当前用户信息。
//
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息，包括角色和权限
// @Tags 用户认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /auth/me [get]
func (u *UserHandler) Me(c *gin.Context) {
	uid := httpx.UIDFromCtx(c)
	if uid == 0 {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	resp, err := newAuthLogic(u.svcCtx).GetMe(c.Request.Context(), uid)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}
