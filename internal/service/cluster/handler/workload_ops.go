package handler

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) ScaleDeploymentImpl(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	var req WorkloadScaleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if req.Replicas == nil || *req.Replicas < 0 {
		httpx.BadRequest(c, "replicas must be >= 0")
		return
	}

	clusterID, target, ok := parseWorkloadOperationTarget(c, "deployment", "deployment.scale")
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	result, err := h.executeHighRiskWorkloadOperation(c, clusterID, target, req.ApprovalToken, func(ctx context.Context, client kubernetesWorkloadClient) (map[string]any, error) {
		return h.scaleDeployment(ctx, client, target.Namespace, target.Name, *req.Replicas)
	})
	if err != nil {
		h.handleHighRiskWorkloadOperationError(c, target.Resource, err)
		return
	}
	httpx.OK(c, result)
}
