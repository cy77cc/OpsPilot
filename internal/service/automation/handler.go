package automation

import (
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logic  *Logic
	svcCtx *svc.ServiceContext
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: NewLogic(svcCtx), svcCtx: svcCtx}
}

func (h *Handler) ListInventories(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "automation:read", "automation:*") {
		return
	}
	rows, err := h.logic.listInventories(c.Request.Context())
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
	var req createInventoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.createInventory(c.Request.Context(), uint(httpx.UIDFromCtx(c)), req)
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
	rows, err := h.logic.listPlaybooks(c.Request.Context())
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
	var req createPlaybookReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.createPlaybook(c.Request.Context(), uint(httpx.UIDFromCtx(c)), req)
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
	var req previewRunReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	out, err := h.logic.previewRun(c.Request.Context(), req)
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
	var req executeRunReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.executeRun(c.Request.Context(), uint(httpx.UIDFromCtx(c)), req)
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
	row, err := h.logic.getRun(c.Request.Context(), c.Param("id"))
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
	rows, err := h.logic.listRunLogs(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}
