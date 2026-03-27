// Package handler 提供 CMDB 服务的 HTTP 处理器。
//
// 本文件实现 CMDB 模块的所有 HTTP 接口，包括：
//   - 资产管理 (CRUD)
//   - 关系管理
//   - 拓扑查询
//   - 同步任务管理
//   - 审计日志查询
package handler

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	cmdbv1 "github.com/cy77cc/OpsPilot/api/cmdb/v1"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	cmdblogic "github.com/cy77cc/OpsPilot/internal/service/cmdb/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// Handler 是 CMDB 服务的 HTTP 处理器。
//
// 封装业务逻辑层和服务上下文，提供统一的请求处理入口。
type Handler struct {
	logic  *cmdblogic.Logic // 业务逻辑层
	svcCtx *svc.ServiceContext
}

// NewHandler 创建 CMDB 处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//
// 返回: 初始化后的 Handler 实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: cmdblogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

// ListAssets 获取资产列表。
//
// @Summary 获取资产列表
// @Description 分页查询 CMDB 中的资产，支持按类型、状态、关键字筛选
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param asset_type query string false "资产类型"
// @Param status query string false "资产状态"
// @Param keyword query string false "搜索关键字"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/assets [get]
func (h *Handler) ListAssets(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:read") {
		return
	}
	filter := cmdblogic.CIFilter{
		Type:      strings.TrimSpace(c.Query("asset_type")),
		Status:    strings.TrimSpace(c.Query("status")),
		Keyword:   strings.TrimSpace(c.Query("keyword")),
		ProjectID: h.projectIDFromHeader(c),
		TeamID:    h.teamIDFromHeader(c),
		Page:      atoiDefault(c.Query("page"), 1),
		PageSize:  atoiDefault(c.Query("page_size"), 20),
	}
	rows, total, err := h.logic.ListCIs(c.Request.Context(), filter)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": total})
}

// CreateAsset 创建资产。
//
// @Summary 创建资产
// @Description 在 CMDB 中创建新的资产记录
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body cmdbv1.CreateCIReq true "资产创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/assets [post]
func (h *Handler) CreateAsset(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:write") {
		return
	}
	var req cmdbv1.CreateCIReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	uid := uint(httpx.UIDFromCtx(c))
	created, err := h.logic.CreateCI(c.Request.Context(), uid, model.CMDBCI{
		CIType:     req.CIType,
		Name:       req.Name,
		Source:     req.Source,
		ExternalID: req.ExternalID,
		ProjectID:  req.ProjectID,
		TeamID:     req.TeamID,
		Owner:      req.Owner,
		Status:     req.Status,
		TagsJSON:   req.TagsJSON,
		AttrsJSON:  req.AttrsJSON,
	})
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.auditCI(c, "ci.create", uid, nil, created)
	httpx.OK(c, created)
}

// GetAsset 获取资产详情。
//
// @Summary 获取资产详情
// @Description 根据ID获取资产详细信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "资产ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /cmdb/assets/{id} [get]
func (h *Handler) GetAsset(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:read") {
		return
	}
	id := uint(atoiDefault(c.Param("id"), 0))
	row, err := h.logic.GetCI(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "asset not found")
		return
	}
	httpx.OK(c, row)
}

// UpdateAsset 更新资产。
//
// @Summary 更新资产
// @Description 更新指定资产的属性信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "资产ID"
// @Param request body cmdbv1.UpdateCIReq true "资产更新请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/assets/{id} [put]
func (h *Handler) UpdateAsset(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:write") {
		return
	}
	id := uint(atoiDefault(c.Param("id"), 0))
	before, err := h.logic.GetCI(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "asset not found")
		return
	}
	var req cmdbv1.UpdateCIReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Owner != nil {
		updates["owner"] = strings.TrimSpace(*req.Owner)
	}
	if req.Status != nil {
		updates["status"] = strings.TrimSpace(*req.Status)
	}
	if req.TagsJSON != nil {
		updates["tags_json"] = *req.TagsJSON
	}
	if req.AttrsJSON != nil {
		updates["attrs_json"] = *req.AttrsJSON
	}
	after, err := h.logic.UpdateCI(c.Request.Context(), uint(httpx.UIDFromCtx(c)), id, updates)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.auditCI(c, "ci.update", uint(httpx.UIDFromCtx(c)), before, after)
	httpx.OK(c, after)
}

// DeleteAsset 删除资产。
//
// @Summary 删除资产
// @Description 删除指定的资产记录（软删除）
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "资产ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/assets/{id} [delete]
func (h *Handler) DeleteAsset(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:write") {
		return
	}
	id := uint(atoiDefault(c.Param("id"), 0))
	before, err := h.logic.GetCI(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "asset not found")
		return
	}
	if err := h.logic.DeleteCI(c.Request.Context(), id); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.auditCI(c, "ci.delete", uint(httpx.UIDFromCtx(c)), before, nil)
	httpx.OK(c, nil)
}

// ListRelations 获取关系列表。
//
// @Summary 获取关系列表
// @Description 查询资产之间的关系，可按资产ID筛选
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param asset_id query int false "资产ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/relations [get]
func (h *Handler) ListRelations(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:read") {
		return
	}
	ciID := uint(atoiDefault(c.Query("asset_id"), 0))
	rels, err := h.logic.ListRelations(c.Request.Context(), ciID)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	out := make([]gin.H, 0, len(rels))
	for _, r := range rels {
		out = append(out, gin.H{"id": r.ID, "from_asset_id": r.FromCIID, "to_asset_id": r.ToCIID, "relation_type": r.RelationType})
	}
	httpx.OK(c, gin.H{"list": out, "total": len(out)})
}

// CreateRelation 创建关系。
//
// @Summary 创建关系
// @Description 在两个资产之间创建关系
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body cmdbv1.CreateRelationReq true "关系创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/relations [post]
func (h *Handler) CreateRelation(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:write") {
		return
	}
	var req cmdbv1.CreateRelationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	uid := uint(httpx.UIDFromCtx(c))
	rel, err := h.logic.CreateRelation(c.Request.Context(), uid, model.CMDBRelation{
		FromCIID:     req.FromCIID,
		ToCIID:       req.ToCIID,
		RelationType: req.RelationType,
	})
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.auditRelation(c, "relation.create", uid, nil, rel)
	httpx.OK(c, rel)
}

// DeleteRelation 删除关系。
//
// @Summary 删除关系
// @Description 删除指定的资产关系
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "关系ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/relations/{id} [delete]
func (h *Handler) DeleteRelation(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:write") {
		return
	}
	id := uint(atoiDefault(c.Param("id"), 0))
	rows, _ := h.logic.ListRelations(c.Request.Context(), 0)
	var before *model.CMDBRelation
	for i := range rows {
		if rows[i].ID == id {
			before = &rows[i]
			break
		}
	}
	if err := h.logic.DeleteRelation(c.Request.Context(), id); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.auditRelation(c, "relation.delete", uint(httpx.UIDFromCtx(c)), before, nil)
	httpx.OK(c, nil)
}

// Topology 获取资产拓扑图。
//
// @Summary 获取资产拓扑图
// @Description 获取资产及其关系的拓扑结构，用于可视化展示
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param X-Project-ID header int false "项目ID"
// @Param X-Team-ID header int false "团队ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/topology [get]
func (h *Handler) Topology(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:read") {
		return
	}
	graph, err := h.logic.Topology(c.Request.Context(), h.projectIDFromHeader(c), h.teamIDFromHeader(c))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, graph)
}

// TriggerSync 触发同步任务。
//
// @Summary 触发同步任务
// @Description 触发从外部数据源同步资产信息到 CMDB
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body cmdbv1.TriggerSyncReq false "同步请求"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/sync/jobs [post]
func (h *Handler) TriggerSync(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:sync") {
		return
	}
	var req cmdbv1.TriggerSyncReq
	_ = c.ShouldBindJSON(&req)
	uid := uint(httpx.UIDFromCtx(c))
	job, err := h.logic.RunSync(c.Request.Context(), uid, req.Source)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.logic.WriteAudit(c.Request.Context(), model.CMDBAudit{Action: "sync.trigger", ActorID: uid, Detail: job.ID})
	httpx.OK(c, job)
}

// GetSyncJob 获取同步任务详情。
//
// @Summary 获取同步任务详情
// @Description 根据任务ID获取同步任务的详细信息和执行状态
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path string true "同步任务ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /cmdb/sync/jobs/{id} [get]
func (h *Handler) GetSyncJob(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:sync") {
		return
	}
	job, err := h.logic.GetSyncJob(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "sync job not found")
		return
	}
	httpx.OK(c, job)
}

// RetrySyncJob 重试同步任务。
//
// @Summary 重试同步任务
// @Description 重新执行同步任务，从所有数据源同步资产信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path string true "同步任务ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/sync/jobs/{id}/retry [post]
func (h *Handler) RetrySyncJob(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:sync") {
		return
	}
	job, err := h.logic.RunSync(c.Request.Context(), uint(httpx.UIDFromCtx(c)), "all")
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, job)
}

// ListChanges 获取变更记录列表。
//
// @Summary 获取变更记录列表
// @Description 查询资产的变更历史记录，可按资产ID筛选
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param asset_id query int false "资产ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/changes [get]
func (h *Handler) ListChanges(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cmdb:audit", "cmdb:read") {
		return
	}
	ciID := uint(atoiDefault(c.Query("asset_id"), 0))
	items, err := h.logic.ListAudits(c.Request.Context(), ciID)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// ListAudits 获取审计日志列表。
//
// @Summary 获取审计日志列表
// @Description 查询 CMDB 的审计日志，等同于 ListChanges 接口
// @Tags CMDB
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param asset_id query int false "资产ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cmdb/audits [get]
func (h *Handler) ListAudits(c *gin.Context) { h.ListChanges(c) }

// projectIDFromHeader 从请求头获取项目ID。
//
// 参数:
//   - c: Gin 上下文
//
// 返回: 项目ID，未设置时返回 0
func (h *Handler) projectIDFromHeader(c *gin.Context) uint {
	v := strings.TrimSpace(c.GetHeader("X-Project-ID"))
	if v == "" {
		return 0
	}
	n, _ := strconv.ParseUint(v, 10, 64)
	return uint(n)
}

// teamIDFromHeader 从请求头获取团队ID。
//
// 参数:
//   - c: Gin 上下文
//
// 返回: 团队ID，未设置时返回 0
func (h *Handler) teamIDFromHeader(c *gin.Context) uint {
	v := strings.TrimSpace(c.GetHeader("X-Team-ID"))
	if v == "" {
		return 0
	}
	n, _ := strconv.ParseUint(v, 10, 64)
	return uint(n)
}

// auditCI 记录资产变更审计日志。
//
// 参数:
//   - c: Gin 上下文
//   - action: 操作类型 (如 ci.create, ci.update, ci.delete)
//   - actor: 操作者用户ID
//   - before: 变更前的资产数据，新增时为 nil
//   - after: 变更后的资产数据，删除时为 nil
func (h *Handler) auditCI(c *gin.Context, action string, actor uint, before *model.CMDBCI, after *model.CMDBCI) {
	var ciID uint
	if after != nil {
		ciID = after.ID
	} else if before != nil {
		ciID = before.ID
	}
	beforeJSON := ""
	afterJSON := ""
	if before != nil {
		buf, _ := json.Marshal(before)
		beforeJSON = string(buf)
	}
	if after != nil {
		buf, _ := json.Marshal(after)
		afterJSON = string(buf)
	}
	h.logic.WriteAudit(c.Request.Context(), model.CMDBAudit{CIID: ciID, Action: action, ActorID: actor, BeforeJSON: beforeJSON, AfterJSON: afterJSON, Detail: c.Request.URL.Path})
}

// auditRelation 记录关系变更审计日志。
//
// 参数:
//   - c: Gin 上下文
//   - action: 操作类型 (如 relation.create, relation.delete)
//   - actor: 操作者用户ID
//   - before: 变更前的关系数据，新增时为 nil
//   - after: 变更后的关系数据，删除时为 nil
func (h *Handler) auditRelation(c *gin.Context, action string, actor uint, before *model.CMDBRelation, after *model.CMDBRelation) {
	var relID uint
	if after != nil {
		relID = after.ID
	} else if before != nil {
		relID = before.ID
	}
	beforeJSON := ""
	afterJSON := ""
	if before != nil {
		buf, _ := json.Marshal(before)
		beforeJSON = string(buf)
	}
	if after != nil {
		buf, _ := json.Marshal(after)
		afterJSON = string(buf)
	}
	h.logic.WriteAudit(c.Request.Context(), model.CMDBAudit{RelationID: relID, Action: action, ActorID: actor, BeforeJSON: beforeJSON, AfterJSON: afterJSON, Detail: c.Request.URL.Path})
}

// atoiDefault 将字符串转换为整数，失败时返回默认值。
//
// 参数:
//   - v: 待转换的字符串
//   - d: 默认值
//
// 返回: 转换后的整数，转换失败时返回默认值
func atoiDefault(v string, d int) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return d
	}
	return n
}

var _ = time.Now
