package cluster

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *Handler) CreateServiceImpl(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	clusterID, namespace, ok := parseNamespacedMutation(c)
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	var req ServiceMutationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if err := validateServiceMutationReq(req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), clusterID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if _, err := client.CoreV1().Services(namespace).Get(c.Request.Context(), req.Name, metav1.GetOptions{}); err == nil {
		httpx.BadRequest(c, "service already exists")
		return
	} else if err != nil && !apierrors.IsNotFound(err) {
		h.handleServiceIngressMutationError(c, "service", err)
		return
	}

	target := serviceIngressMutationTarget{
		Namespace:       namespace,
		Resource:        "service",
		Name:            req.Name,
		Action:          "service.create",
		RequireApproval: serviceMutationRequiresApproval(req),
	}
	result, err := h.executeServiceIngressMutationWithClient(c, clusterID, target, req.ApprovalToken, client, func(ctx context.Context, kubeClient kubernetesServiceIngressClient) (map[string]any, error) {
		return h.createService(ctx, kubeClient, namespace, req)
	})
	if err != nil {
		h.handleServiceIngressMutationError(c, "service", err)
		return
	}
	httpx.OK(c, result)
}
