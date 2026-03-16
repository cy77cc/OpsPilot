package handler

import (
	"strconv"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	jobslogic "github.com/cy77cc/OpsPilot/internal/service/jobs/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logic  *jobslogic.Logic
	svcCtx *svc.ServiceContext
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: jobslogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

// ListJobs 获取任务列表
func (h *Handler) ListJobs(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "task:read", "task:*") {
		return
	}

	var req jobslogic.ListJobsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	jobs, total, err := h.logic.ListJobs(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{"list": jobs, "total": total})
}

// GetJob 获取任务详情
func (h *Handler) GetJob(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "task:read", "task:*") {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	job, err := h.logic.GetJob(c.Request.Context(), uint(id))
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "job not found")
		return
	}

	httpx.OK(c, job)
}

// CreateJob 创建任务
func (h *Handler) CreateJob(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "task:write", "task:*") {
		return
	}

	var req jobslogic.CreateJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	job, err := h.logic.CreateJob(c.Request.Context(), uint(httpx.UIDFromCtx(c)), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, job)
}

// UpdateJob 更新任务
func (h *Handler) UpdateJob(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "task:write", "task:*") {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	var req jobslogic.UpdateJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	job, err := h.logic.UpdateJob(c.Request.Context(), uint(id), req)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "job not found")
		return
	}

	httpx.OK(c, job)
}

// DeleteJob 删除任务
func (h *Handler) DeleteJob(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "task:write", "task:*") {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	if err := h.logic.DeleteJob(c.Request.Context(), uint(id)); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{"message": "deleted"})
}

// StartJob 启动任务
func (h *Handler) StartJob(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "task:write", "task:*") {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	if err := h.logic.StartJob(c.Request.Context(), uint(id)); err != nil {
		httpx.Fail(c, xcode.NotFound, "job not found")
		return
	}

	httpx.OK(c, gin.H{"message": "started"})
}

// StopJob 停止任务
func (h *Handler) StopJob(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "task:write", "task:*") {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	if err := h.logic.StopJob(c.Request.Context(), uint(id)); err != nil {
		httpx.Fail(c, xcode.NotFound, "job not found")
		return
	}

	httpx.OK(c, gin.H{"message": "stopped"})
}

// GetJobExecutions 获取任务执行记录
func (h *Handler) GetJobExecutions(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "task:read", "task:*") {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	executions, total, err := h.logic.GetJobExecutions(c.Request.Context(), uint(id), page, pageSize)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{"list": executions, "total": total})
}

// GetJobLogs 获取任务日志
func (h *Handler) GetJobLogs(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "task:read", "task:*") {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, total, err := h.logic.GetJobLogs(c.Request.Context(), uint(id), page, pageSize)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{"list": logs, "total": total})
}
