package cluster

import (
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
)

// Handler handles cluster-related HTTP requests
type Handler struct {
	svcCtx *svc.ServiceContext
}

// NewHandler creates a new cluster handler
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{svcCtx: svcCtx}
}

// GetClusters returns list of clusters
func (h *Handler) GetClusters(c *gin.Context) {
	// Mock data for now - replace with actual database query
	clusters := []ClusterListItem{
		{
			ID:        1,
			Name:      "prod-cluster-01",
			Version:   "1.28.0",
			Status:    "active",
			NodeCount: 5,
		},
		{
			ID:        2,
			Name:      "staging-cluster-01",
			Version:   "1.27.0",
			Status:    "active",
			NodeCount: 3,
		},
	}

	httpx.OK(c, gin.H{
		"list":  clusters,
		"total": len(clusters),
	})
}

// GetClusterDetail returns detailed cluster information
func (h *Handler) GetClusterDetail(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	// Mock data - replace with actual database query
	cluster := ClusterDetail{
		ID:          id,
		Name:        "prod-cluster-01",
		Version:     "1.28.0",
		Status:      "active",
		NodeCount:   5,
		Endpoint:    "https://10.0.1.100:6443",
		Description: "Production Kubernetes cluster",
	}

	httpx.OK(c, cluster)
}

// GetClusterNodes returns nodes in a cluster
func (h *Handler) GetClusterNodes(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	// Mock data - replace with actual database query
	nodes := []ClusterNode{
		{
			ID:        1,
			ClusterID: id,
			HostID:    1,
			Name:      "master-01",
			IP:        "10.0.1.101",
			Role:      "control-plane",
			Status:    "ready",
		},
		{
			ID:        2,
			ClusterID: id,
			HostID:    2,
			Name:      "worker-01",
			IP:        "10.0.1.102",
			Role:      "worker",
			Status:    "ready",
		},
	}

	httpx.OK(c, gin.H{
		"list":  nodes,
		"total": len(nodes),
	})
}
