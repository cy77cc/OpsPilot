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

type Handler struct {
	logic  *cmdblogic.Logic
	svcCtx *svc.ServiceContext
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: cmdblogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

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

func (h *Handler) ListAudits(c *gin.Context) { h.ListChanges(c) }

func (h *Handler) projectIDFromHeader(c *gin.Context) uint {
	v := strings.TrimSpace(c.GetHeader("X-Project-ID"))
	if v == "" {
		return 0
	}
	n, _ := strconv.ParseUint(v, 10, 64)
	return uint(n)
}

func (h *Handler) teamIDFromHeader(c *gin.Context) uint {
	v := strings.TrimSpace(c.GetHeader("X-Team-ID"))
	if v == "" {
		return 0
	}
	n, _ := strconv.ParseUint(v, 10, 64)
	return uint(n)
}

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

func atoiDefault(v string, d int) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return d
	}
	return n
}

var _ = time.Now
