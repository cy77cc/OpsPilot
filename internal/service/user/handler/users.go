// Package handler 提供用户模块的 HTTP 处理器。
//
// 本文件实现用户信息相关的 HTTP 处理器，包括用户查询等功能。
package handler

import (
	"strconv"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	userLogic "github.com/cy77cc/OpsPilot/internal/service/user/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// UserHandler 是用户模块的 HTTP 处理器。
//
// 职责:
//   - 处理用户相关的 HTTP 请求
//   - 调用 Logic 层执行业务逻辑
//   - 返回统一的 HTTP 响应
type UserHandler struct {
	svcCtx *svc.ServiceContext
}

// NewUserHandler 创建用户处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库、缓存等依赖
//
// 返回: 用户处理器实例
func NewUserHandler(svcCtx *svc.ServiceContext) *UserHandler {
	return &UserHandler{
		svcCtx: svcCtx,
	}
}

// GetUserInfo 获取用户信息。
//
// @Summary 获取用户信息
// @Description 根据用户 ID 获取用户详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户 ID"
// @Success 200 {object} httpx.Response{data=v1.UserResp}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /user/{id} [get]
func (u *UserHandler) GetUserInfo(c *gin.Context) {
	idStr := c.Param("id")
	var id model.UserID

	if idInt, err := strconv.Atoi(idStr); err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	} else {
		id = model.UserID(idInt)
	}
	resp, err := userLogic.NewUserLogic(u.svcCtx).GetUser(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}
