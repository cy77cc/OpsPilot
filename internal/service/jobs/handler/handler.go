// Package handler 提供任务管理的 HTTP 处理器。
//
// 本文件包含任务管理相关的所有 HTTP Handler 实现：
//   - ListJobs: 获取任务列表
//   - GetJob: 获取任务详情
//   - CreateJob: 创建任务
//   - UpdateJob: 更新任务
//   - DeleteJob: 删除任务
//   - StartJob: 启动任务
//   - StopJob: 停止任务
//   - GetJobExecutions: 获取执行记录
//   - GetJobLogs: 获取任务日志
package handler

import (
	"strconv"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	jobslogic "github.com/cy77cc/OpsPilot/internal/service/jobs/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// Handler 是任务管理的 HTTP 处理器。
//
// 职责:
//   - 接收 HTTP 请求并解析参数
//   - 调用 logic 层执行业务逻辑
//   - 返回统一格式的 HTTP 响应
//
// 依赖:
//   - logic: 任务业务逻辑层
//   - svcCtx: 服务上下文，包含数据库连接等资源
type Handler struct {
	logic  *jobslogic.Logic
	svcCtx *svc.ServiceContext
}

// NewHandler 创建任务管理 Handler 实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等资源
//
// 返回: 初始化完成的 Handler 实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: jobslogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

// ListJobs 获取任务列表。
//
// @Summary 获取任务列表
// @Description 分页获取所有任务信息
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /jobs [get]
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

// GetJob 获取任务详情。
//
// @Summary 获取任务详情
// @Description 根据 ID 获取单个任务的详细信息
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "任务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /jobs/{id} [get]
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

// CreateJob 创建任务。
//
// @Summary 创建任务
// @Description 创建新的定时任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body jobslogic.CreateJobReq true "任务创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /jobs [post]
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

// UpdateJob 更新任务。
//
// @Summary 更新任务
// @Description 更新指定任务的信息
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "任务 ID"
// @Param request body jobslogic.UpdateJobReq true "任务更新请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /jobs/{id} [put]
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

// DeleteJob 删除任务。
//
// @Summary 删除任务
// @Description 删除指定的任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "任务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /jobs/{id} [delete]
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

// StartJob 启动任务。
//
// @Summary 启动任务
// @Description 启动指定的任务执行
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "任务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /jobs/{id}/start [post]
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

// StopJob 停止任务。
//
// @Summary 停止任务
// @Description 停止正在运行的任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "任务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /jobs/{id}/stop [post]
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

// GetJobExecutions 获取任务执行记录。
//
// @Summary 获取任务执行记录
// @Description 分页获取指定任务的执行记录列表
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "任务 ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /jobs/{id}/executions [get]
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

// GetJobLogs 获取任务日志。
//
// @Summary 获取任务日志
// @Description 分页获取指定任务的执行日志列表
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "任务 ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /jobs/{id}/logs [get]
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
