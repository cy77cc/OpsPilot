package handler

import (
	"strconv"

	v1 "github.com/cy77cc/OpsPilot/api/user/v1"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/modules/user/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

type OrgHandler struct {
	svcCtx *svc.ServiceContext
}

func NewOrgHandler(svcCtx *svc.ServiceContext) *OrgHandler {
	return &OrgHandler{
		svcCtx: svcCtx,
	}
}

func (h *OrgHandler) GetDepartmentTree(c *gin.Context) {
	l := logic.NewOrgLogic(h.svcCtx)
	resp, err := l.GetDepartmentTree(c.Request.Context())
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *OrgHandler) CreateDepartment(c *gin.Context) {
	var req v1.DepartmentCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	l := logic.NewOrgLogic(h.svcCtx)
	if err := l.CreateDepartment(c.Request.Context(), req); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

func (h *OrgHandler) UpdateDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	var req v1.DepartmentUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	req.Id = id

	l := logic.NewOrgLogic(h.svcCtx)
	if err := l.UpdateDepartment(c.Request.Context(), id, req); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

func (h *OrgHandler) DeleteDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	l := logic.NewOrgLogic(h.svcCtx)
	if err := l.DeleteDepartment(c.Request.Context(), id); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

func (h *OrgHandler) TransferMember(c *gin.Context) {
	var req v1.MemberTransferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	err := logic.NewOrgLogic(h.svcCtx).TransferMember(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

func (h *OrgHandler) GetDepartmentMembers(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resp, err := logic.NewOrgLogic(h.svcCtx).GetDepartmentMembers(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *OrgHandler) GetDepartmentRoles(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resp, err := logic.NewOrgLogic(h.svcCtx).GetDepartmentRoles(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *OrgHandler) UpdateDepartmentRoles(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	var req v1.DepartmentRolesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	err := logic.NewOrgLogic(h.svcCtx).UpdateDepartmentRoles(c.Request.Context(), id, req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}
