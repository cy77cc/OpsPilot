package cluster

import (
	"fmt"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handler) UpgradeClusterImpl(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	var req UpgradeClusterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	var cluster model.Cluster
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).First(&cluster, id).Error; err != nil {
		httpx.NotFound(c, "cluster not found")
		return
	}

	if cluster.Source != "platform_managed" {
		httpx.BadRequest(c, "only platform-managed clusters can be upgraded through this API")
		return
	}

	gate := h.requireHighRiskApproval(c.Request.Context(), id, "", "cluster.upgrade", "cluster", cluster.Name, req.ApprovalToken, httpx.UIDFromCtx(c))
	if !gate.Allowed {
		httpx.OK(c, operationResponseFromGate(id, "cluster", cluster.Name, gate))
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	version, err := client.Discovery().ServerVersion()
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	result := UpgradeClusterResult{
		ClusterID:   id,
		FromVersion: version.GitVersion,
		ToVersion:   req.TargetVersion,
		Status:      "preview",
		Message:     "Cluster upgrade would require SSH access to all nodes. This is a preview.",
		UpgradeSteps: []string{
			fmt.Sprintf("1. Drain and cordon control plane nodes"),
			fmt.Sprintf("2. Upgrade kubeadm to v%s on control plane", req.TargetVersion),
			fmt.Sprintf("3. Run 'kubeadm upgrade apply v%s' on control plane", req.TargetVersion),
			fmt.Sprintf("4. Upgrade kubelet and kubectl on control plane"),
			fmt.Sprintf("5. Uncordon control plane nodes"),
			fmt.Sprintf("6. Repeat steps 1-5 for worker nodes"),
			"7. Verify cluster health",
		},
	}

	audit, _ := h.RecordClusterOperationAudit(c.Request.Context(), id, "", "cluster.upgrade", "cluster", cluster.Name, "success", "upgrade preview generated", uint(httpx.UIDFromCtx(c)))
	httpx.OK(c, operationSuccessResponse(id, "cluster", cluster.Name, "upgrade preview generated", audit.ID, map[string]any{
		"cluster_id":    result.ClusterID,
		"from_version":  result.FromVersion,
		"to_version":    result.ToVersion,
		"status":        result.Status,
		"message":       result.Message,
		"upgrade_steps": result.UpgradeSteps,
		"current_stage": "preview",
	}))
}
