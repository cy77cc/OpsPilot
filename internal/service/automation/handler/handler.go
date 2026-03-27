// Package handler 提供自动化运维服务的 HTTP 处理器。
//
// 本包实现自动化运维相关的 HTTP 接口，包括：
//   - 清单管理（Inventory）：主机清单的增删查
//   - Playbook 管理：自动化脚本的增删查
//   - 运行管理：执行预览、执行任务、查询状态和日志
//
// 权限控制：
//   - automation:read - 读取权限
//   - automation:write - 写入权限
//   - automation:execute - 执行权限
//   - automation:* - 完全权限
package handler

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	automationlogic "github.com/cy77cc/OpsPilot/internal/service/automation/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// Handler 是自动化运维服务的 HTTP 处理器。
//
// 封装 Logic 层调用，处理 HTTP 请求解析、权限校验和响应格式化。
type Handler struct {
	logic  *automationlogic.Logic // 业务逻辑层
	svcCtx *svc.ServiceContext    // 服务上下文
}

// NewHandler 创建自动化运维处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库、配置等依赖
//
// 返回: 初始化后的 Handler 实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: automationlogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

// ListInventories 获取主机清单列表。
//
// @Summary 获取主机清单列表
// @Description 获取所有自动化主机清单，按 ID 降序排列
// @Tags 自动化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}} "成功返回清单列表"
// @Failure 401 {object} httpx.Response "未授权"
// @Failure 500 {object} httpx.Response "服务器错误"
// @Router /automation/inventories [get]
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

// CreateInventory 创建主机清单。
//
// @Summary 创建主机清单
// @Description 创建新的自动化主机清单，包含名称和主机配置
// @Tags 自动化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body automationlogic.CreateInventoryReq true "清单创建请求"
// @Success 200 {object} httpx.Response{data=model.AutomationInventory} "成功返回创建的清单"
// @Failure 400 {object} httpx.Response "参数错误"
// @Failure 401 {object} httpx.Response "未授权"
// @Failure 500 {object} httpx.Response "服务器错误"
// @Router /automation/inventories [post]
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

// ListPlaybooks 获取 Playbook 列表。
//
// @Summary 获取 Playbook 列表
// @Description 获取所有自动化 Playbook，按 ID 降序排列
// @Tags 自动化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}} "成功返回 Playbook 列表"
// @Failure 401 {object} httpx.Response "未授权"
// @Failure 500 {object} httpx.Response "服务器错误"
// @Router /automation/playbooks [get]
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

// CreatePlaybook 创建 Playbook。
//
// @Summary 创建 Playbook
// @Description 创建新的自动化 Playbook，包含名称、YAML 内容和风险等级
// @Tags 自动化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body automationlogic.CreatePlaybookReq true "Playbook 创建请求"
// @Success 200 {object} httpx.Response{data=model.AutomationPlaybook} "成功返回创建的 Playbook"
// @Failure 400 {object} httpx.Response "参数错误"
// @Failure 401 {object} httpx.Response "未授权"
// @Failure 500 {object} httpx.Response "服务器错误"
// @Router /automation/playbooks [post]
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

// PreviewRun 预览执行任务。
//
// @Summary 预览执行任务
// @Description 预览自动化任务的执行参数和风险等级，生成预览令牌
// @Tags 自动化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body automationlogic.PreviewRunReq true "预览请求"
// @Success 200 {object} httpx.Response{data=map[string]interface{}} "成功返回预览结果"
// @Failure 400 {object} httpx.Response "参数错误"
// @Failure 401 {object} httpx.Response "未授权"
// @Failure 500 {object} httpx.Response "服务器错误"
// @Router /automation/runs/preview [post]
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

// ExecuteRun 执行自动化任务。
//
// @Summary 执行自动化任务
// @Description 执行自动化任务，需要提供审批令牌，系统将校验主机运行资格后执行
// @Tags 自动化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body automationlogic.ExecuteRunReq true "执行请求"
// @Success 200 {object} httpx.Response{data=model.AutomationRun} "成功返回执行记录"
// @Failure 400 {object} httpx.Response "参数错误"
// @Failure 401 {object} httpx.Response "未授权"
// @Failure 500 {object} httpx.Response "服务器错误"
// @Router /automation/runs/execute [post]
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

// GetRun 获取任务执行详情。
//
// @Summary 获取任务执行详情
// @Description 根据 ID 获取自动化任务的执行状态和结果
// @Tags 自动化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path string true "任务 ID"
// @Success 200 {object} httpx.Response{data=model.AutomationRun} "成功返回执行详情"
// @Failure 401 {object} httpx.Response "未授权"
// @Failure 404 {object} httpx.Response "任务不存在"
// @Router /automation/runs/{id} [get]
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

// GetRunLogs 获取任务执行日志。
//
// @Summary 获取任务执行日志
// @Description 根据 ID 获取自动化任务的执行日志列表
// @Tags 自动化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path string true "任务 ID"
// @Success 200 {object} httpx.Response{data=map[string]interface{}} "成功返回日志列表"
// @Failure 401 {object} httpx.Response "未授权"
// @Failure 500 {object} httpx.Response "服务器错误"
// @Router /automation/runs/{id}/logs [get]
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
