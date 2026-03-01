package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func (h *Handler) CreateApproval(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:deploy", "k8s:rollback", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	var req struct {
		Namespace string `json:"namespace" binding:"required"`
		Action    string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	ticket := fmt.Sprintf("k8s-appr-%d", time.Now().UnixNano())
	rec := model.ClusterDeployApproval{Ticket: ticket, ClusterID: cluster.ID, Namespace: req.Namespace, Action: req.Action, Status: "pending", RequestBy: uint(httpx.UIDFromCtx(c)), ExpiresAt: time.Now().Add(30 * time.Minute)}
	if err := h.svcCtx.DB.Create(&rec).Error; err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, rec)
}

func (h *Handler) ConfirmApproval(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:approve", "kubernetes:approve") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	ticket := strings.TrimSpace(c.Param("ticket"))
	if ticket == "" {
		httpx.Fail(c, xcode.ParamError, "ticket required")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	_ = c.ShouldBindJSON(&req)
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "approved"
	}
	if status != "approved" && status != "rejected" {
		httpx.Fail(c, xcode.ParamError, "status must be approved or rejected")
		return
	}
	uid := httpx.UIDFromCtx(c)
	result := h.svcCtx.DB.Model(&model.ClusterDeployApproval{}).
		Where("ticket = ? AND cluster_id = ?", ticket, cluster.ID).
		Updates(map[string]any{"status": status, "review_by": uid, "updated_at": time.Now()})
	if result.Error != nil {
		httpx.Fail(c, xcode.ServerError, result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		httpx.Fail(c, xcode.NotFound, "approval ticket not found")
		return
	}
	httpx.OK(c, gin.H{"ticket": ticket, "status": status})
}

func (h *Handler) legacyDeployWithApproval(c *gin.Context, cli *kubernetes.Clientset, cluster *model.Cluster, req struct {
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Image     string `json:"image" binding:"required"`
	Replicas  int32  `json:"replicas"`
}) bool {
	if !h.namespaceWritable(c, cluster.ID, req.Namespace) {
		return false
	}
	var body struct {
		ApprovalToken string `json:"approval_token"`
	}
	_ = c.ShouldBindJSON(&body)
	if !h.requireProdApproval(c, cluster.ID, req.Namespace, "deploy", body.ApprovalToken) {
		return false
	}
	return true
}

func deploymentFromRollout(req rolloutApplyReq) *appsv1.Deployment {
	replicas := req.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	labels := map[string]string{"app": req.Name}
	return &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: req.Name, Image: req.Image}}}},
		},
	}
}

func (h *Handler) patchRolloutPaused(c *gin.Context, cluster *model.Cluster, namespace, name string, paused bool) error {
	_, dc, err := h.getClients(cluster)
	if err != nil {
		return err
	}
	patch := map[string]any{"spec": map[string]any{"paused": paused}}
	raw, _ := json.Marshal(patch)
	_, err = dc.Resource(rolloutGVR).Namespace(namespace).Patch(c.Request.Context(), name, types.MergePatchType, raw, metav1.PatchOptions{})
	return err
}
