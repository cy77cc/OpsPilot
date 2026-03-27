// Package handler 提供用户模块的 HTTP 处理器。
//
// 本文件实现认证相关的 HTTP 处理器，包括登录、注册、Token 刷新和登出。
package handler

import (
	"errors"
	"io"

	v1 "github.com/cy77cc/OpsPilot/api/user/v1"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	userLogic "github.com/cy77cc/OpsPilot/internal/service/user/logic"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// Login 用户登录。
//
// @Summary 用户登录
// @Description 用户登录获取 Token
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param request body v1.LoginReq true "登录请求"
// @Success 200 {object} httpx.Response{data=v1.TokenResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /auth/login [post]
func (u *UserHandler) Login(c *gin.Context) {
	var req v1.LoginReq
	if err := c.ShouldBind(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := userLogic.NewUserLogic(u.svcCtx).Login(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
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
	resp, err := userLogic.NewUserLogic(u.svcCtx).Register(c.Request.Context(), req)
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
// @Param request body v1.RefreshReq true "刷新请求"
// @Success 200 {object} httpx.Response{data=v1.TokenResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /auth/refresh [post]
func (u *UserHandler) Refresh(c *gin.Context) {
	var req v1.RefreshReq
	if err := c.ShouldBind(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := userLogic.NewUserLogic(u.svcCtx).Refresh(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

// Logout 用户登出。
//
// @Summary 用户登出
// @Description 将 Refresh Token 从白名单移除
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param request body v1.LogoutReq false "登出请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Router /auth/logout [post]
func (u *UserHandler) Logout(c *gin.Context) {
	var req v1.LogoutReq
	err := c.ShouldBindJSON(&req)
	if err != nil && !errors.Is(err, io.EOF) {
		httpx.BindErr(c, err)
		return
	}
	if err = userLogic.NewUserLogic(u.svcCtx).Logout(c.Request.Context(), req); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
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
	resp, err := userLogic.NewUserLogic(u.svcCtx).GetMe(c.Request.Context(), uid)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}
