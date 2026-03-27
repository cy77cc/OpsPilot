// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件实现集群 HTTP Handler，处理集群相关的 CRUD 请求。
package cluster

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 集群服务 HTTP 处理器。
//
// 职责:
//   - 处理集群 CRUD 请求
//   - 管理缓存策略
//   - 协调 Repository 和业务逻辑
type Handler struct {
	svcCtx *svc.ServiceContext // 服务上下文
	repo   *Repository         // 数据访问层
}

// NewHandler 创建集群服务处理器。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库、缓存等依赖
//
// 返回: 集群处理器实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{
		svcCtx: svcCtx,
		repo:   NewRepository(svcCtx.DB),
	}
}

// GetClusters 获取集群列表。
//
// @Summary 获取集群列表
// @Description 获取当前用户有权限访问的所有集群信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param status query string false "按状态筛选"
// @Param source query string false "按来源筛选"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters [get]
func (h *Handler) GetClusters(c *gin.Context) {
	status := c.Query("status")
	source := c.Query("source")
	cacheKey := CacheKeyClusterList(status, source)
	items := make([]ClusterListItem, 0)
	raw, _, err := h.svcCtx.CacheFacade.GetOrLoad(c.Request.Context(), cacheKey, ClusterPhase1CachePolicies["clusters.list"].TTL, func(ctx context.Context) (string, error) {
		rows, qerr := h.repo.ListClusters(ctx, status, source)
		if qerr != nil {
			return "", qerr
		}
		raw, merr := json.Marshal(rows)
		if merr != nil {
			return "", merr
		}
		return string(raw), nil
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	_ = json.Unmarshal([]byte(raw), &items)

	httpx.OK(c, gin.H{
		"list":  items,
		"total": len(items),
	})
}

// GetClusterDetail 获取集群详情。
//
// @Summary 获取集群详情
// @Description 根据 ID 获取指定集群的详细信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=ClusterDetail}
// @Failure 400 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id} [get]
func (h *Handler) GetClusterDetail(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	cacheKey := CacheKeyClusterDetail(id)
	var detail ClusterDetail
	raw, _, err := h.svcCtx.CacheFacade.GetOrLoad(c.Request.Context(), cacheKey, ClusterPhase1CachePolicies["clusters.detail"].TTL, func(ctx context.Context) (string, error) {
		d, derr := h.repo.GetClusterDetail(ctx, id)
		if derr != nil {
			return "", derr
		}
		buf, merr := json.Marshal(d)
		if merr != nil {
			return "", merr
		}
		return string(buf), nil
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "cluster not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	if err := json.Unmarshal([]byte(raw), &detail); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, detail)
}

// GetClusterNodes 获取集群节点列表。
//
// @Summary 获取集群节点列表
// @Description 获取指定集群的所有节点信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/nodes [get]
func (h *Handler) GetClusterNodes(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	cacheKey := CacheKeyClusterNodes(id)
	items := make([]ClusterNode, 0)
	raw, _, err := h.svcCtx.CacheFacade.GetOrLoad(c.Request.Context(), cacheKey, ClusterPhase1CachePolicies["clusters.nodes"].TTL, func(ctx context.Context) (string, error) {
		rows, rerr := h.repo.ListClusterNodes(ctx, id)
		if rerr != nil {
			return "", rerr
		}
		buf, merr := json.Marshal(rows)
		if merr != nil {
			return "", merr
		}
		return string(buf), nil
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{
		"list":  items,
		"total": len(items),
	})
}

// CreateCluster 创建集群（导入外部集群）。
//
// @Summary 创建集群
// @Description 导入外部 Kubernetes 集群到平台管理
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body ClusterCreateReq true "集群创建请求"
// @Success 200 {object} httpx.Response{data=ClusterDetail}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters [post]
func (h *Handler) CreateCluster(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	var req ClusterCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	uid := httpx.UIDFromCtx(c)
	cluster, err := h.ImportCluster(c.Request.Context(), uid, req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, cluster)
}

// UpdateCluster 更新集群信息。
//
// @Summary 更新集群信息
// @Description 更新指定集群的基本信息（名称、描述等）
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body ClusterUpdateReq true "集群更新请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id} [put]
func (h *Handler) UpdateCluster(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	var req ClusterUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	cluster, err := h.repo.GetClusterModel(c.Request.Context(), id)
	if err != nil {
		httpx.NotFound(c, "cluster not found")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	updates["updated_at"] = time.Now()

	if err := h.repo.UpdateCluster(c.Request.Context(), id, updates); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	h.invalidateClusterCache(c.Request.Context(), id)

	httpx.OK(c, gin.H{"id": cluster.ID, "message": "updated"})
}

// DeleteCluster 删除集群。
//
// @Summary 删除集群
// @Description 删除指定集群及其关联数据（节点、凭证等）
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id} [delete]
func (h *Handler) DeleteCluster(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	cluster, err := h.repo.GetClusterModel(c.Request.Context(), id)
	if err != nil {
		httpx.NotFound(c, "cluster not found")
		return
	}

	if err := h.repo.DeleteClusterWithRelations(c.Request.Context(), cluster.ID); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	h.invalidateClusterCache(c.Request.Context(), cluster.ID)

	httpx.OK(c, gin.H{"id": cluster.ID, "message": "deleted"})
}

// invalidateClusterCache 使集群相关缓存失效。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
func (h *Handler) invalidateClusterCache(ctx context.Context, clusterID uint) {
	if h.svcCtx == nil || h.svcCtx.CacheFacade == nil {
		return
	}
	h.svcCtx.CacheFacade.Delete(ctx,
		CacheKeyClusterList("", ""),
		CacheKeyClusterList("active", ""),
		CacheKeyClusterDetail(clusterID),
		CacheKeyClusterNodes(clusterID),
	)
}

// TestCluster 测试集群连通性。
//
// @Summary 测试集群连通性
// @Description 测试指定集群的 API Server 连通性和延迟
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=ClusterTestResp}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/test [post]
func (h *Handler) TestCluster(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	result, err := h.TestConnectivity(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, result)
}
