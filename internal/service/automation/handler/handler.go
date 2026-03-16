package handler

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	automationlogic "github.com/cy77cc/OpsPilot/internal/service/automation/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logic  *automationlogic.Logic
	svcCtx *svc.ServiceContext
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: automationlogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

func (h *Handler) ListInventories(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "automation:read", "automation:*") {
		return
	}
	rows, err := h.logic.ListInventories(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}

func (h *Handler) CreateInventory(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "automation:write", "automation:*") {
		return
	}
	var req automationlogic.CreateInventoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.CreateInventory(c.Request.Context(), uint(httpx.UIDFromCtx(c)), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) ListPlaybooks(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "automation:read", "automation:*") {
		return
	}
	rows, err := h.logic.ListPlaybooks(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}

func (h *Handler) CreatePlaybook(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "automation:write", "automation:*") {
		return
	}
	var req automationlogic.CreatePlaybookReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.CreatePlaybook(c.Request.Context(), uint(httpx.UIDFromCtx(c)), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) PreviewRun(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "automation:read", "automation:*") {
		return
	}
	var req automationlogic.PreviewRunReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	out, err := h.logic.PreviewRun(c.Request.Context(), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, out)
}

func (h *Handler) ExecuteRun(c *gin.Context) {
	// Mutating action gate.
	if !httpx.Authorize(c, h.svcCtx.DB, "automation:execute", "automation:write", "automation:*") {
		return
	}
	var req automationlogic.ExecuteRunReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.ExecuteRun(c.Request.Context(), uint(httpx.UIDFromCtx(c)), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) GetRun(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "automation:read", "automation:*") {
		return
	}
	row, err := h.logic.GetRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "run not found")
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) GetRunLogs(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "automation:read", "automation:*") {
		return
	}
	rows, err := h.logic.ListRunLogs(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}
