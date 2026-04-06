package cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
)

func (h *Handler) executeHighRiskNodeOperationImpl(c *gin.Context, clusterID uint, nodeName, action, approvalToken string, fn func(context.Context, *model.Cluster, *model.ClusterNode, *kubernetes.Clientset) (map[string]any, error)) (ClusterOperationResponse, error) {
	ctx := c.Request.Context()
	var cluster model.Cluster
	if err := h.svcCtx.DB.WithContext(ctx).First(&cluster, clusterID).Error; err != nil {
		return ClusterOperationResponse{}, fmt.Errorf("cluster not found: %w", err)
	}
	if strings.TrimSpace(cluster.Source) != "platform_managed" {
		return ClusterOperationResponse{}, fmt.Errorf("cannot modify nodes in externally managed cluster")
	}

	var node model.ClusterNode
	if err := h.svcCtx.DB.WithContext(ctx).Where("cluster_id = ? AND name = ?", clusterID, nodeName).First(&node).Error; err != nil {
		return ClusterOperationResponse{}, fmt.Errorf("node not found: %w", err)
	}

	gate := h.requireHighRiskApproval(ctx, clusterID, "", action, "node", nodeName, approvalToken, httpx.UIDFromCtx(c))
	if !gate.Allowed {
		return operationResponseFromGate(clusterID, "node", nodeName, gate), nil
	}

	client, err := h.getClusterClient(ctx, clusterID)
	if err != nil {
		return ClusterOperationResponse{}, err
	}

	details, opErr := fn(ctx, &cluster, &node, client)
	operatorID := uint(httpx.UIDFromCtx(c))
	if opErr != nil {
		audit, _ := h.RecordClusterOperationAudit(ctx, clusterID, "", action, "node", nodeName, "failed", opErr.Error(), operatorID)
		return ClusterOperationResponse{
			State:       OperationStateFailed,
			Code:        "failed",
			Message:     sanitizeOperationText(opErr.Error()),
			ClusterID:   clusterID,
			Resource:    "node",
			ResourceID:  nodeName,
			AuditID:     audit.ID,
			Diagnostics: sanitizeOperationText(opErr.Error()),
		}, nil
	}

	audit, err := h.RecordClusterOperationAudit(ctx, clusterID, "", action, "node", nodeName, "success", action+" executed", operatorID)
	if err != nil {
		return operationSuccessResponse(clusterID, "node", nodeName, action+" executed", 0, details), nil
	}
	if syncErr := h.SyncClusterNodes(ctx, clusterID); syncErr != nil {
		if details == nil {
			details = map[string]any{}
		}
		details["sync_warning"] = sanitizeOperationText(syncErr.Error())
	}
	return operationSuccessResponse(clusterID, "node", nodeName, action+" executed", audit.ID, details), nil
}
