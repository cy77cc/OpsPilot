package handler

import (
	"errors"
	"io"

	v1 "github.com/cy77cc/k8s-manage/api/user/v1"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	userLogic "github.com/cy77cc/k8s-manage/internal/service/user/logic"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

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
