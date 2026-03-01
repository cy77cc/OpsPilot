package handler

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func (h *Handler) teamIDFromHeader(c *gin.Context) uint {
	raw := strings.TrimSpace(c.GetHeader("X-Team-ID"))
	if raw == "" {
		return 0
	}
	n, _ := strconv.ParseUint(raw, 10, 64)
	return uint(n)
}

func (h *Handler) hasAnyPermission(userID uint, codes ...string) bool {
	return httpx.HasAnyPermission(h.svcCtx.DB, uint64(userID), codes...)
}

func (h *Handler) namespaceWritable(c *gin.Context, clusterID uint, namespace string) bool {
	if namespace == "" {
		return true
	}
	uid := httpx.UIDFromCtx(c)
	if httpx.IsAdmin(h.svcCtx.DB, uid) {
		return true
	}
	teamID := h.teamIDFromHeader(c)
	if teamID == 0 {
		httpx.Fail(c, xcode.Forbidden, "missing team header")
		return false
	}
	var binding model.ClusterNamespaceBinding
	err := h.svcCtx.DB.Where("cluster_id = ? AND team_id = ? AND namespace = ?", clusterID, teamID, namespace).First(&binding).Error
	if err != nil {
		httpx.Fail(c, xcode.Forbidden, "namespace not bound to team")
		return false
	}
	if binding.Readonly {
		httpx.Fail(c, xcode.Forbidden, "namespace is readonly for team")
		return false
	}
	return true
}

func (h *Handler) namespaceReadable(c *gin.Context, clusterID uint, namespace string) bool {
	if namespace == "" {
		return true
	}
	uid := httpx.UIDFromCtx(c)
	if httpx.IsAdmin(h.svcCtx.DB, uid) {
		return true
	}
	teamID := h.teamIDFromHeader(c)
	if teamID == 0 {
		httpx.Fail(c, xcode.Forbidden, "missing team header")
		return false
	}
	var count int64
	h.svcCtx.DB.Model(&model.ClusterNamespaceBinding{}).
		Where("cluster_id = ? AND team_id = ? AND namespace = ?", clusterID, teamID, namespace).
		Count(&count)
	if count == 0 {
		httpx.Fail(c, xcode.Forbidden, "namespace not bound to team")
		return false
	}
	return true
}

func (h *Handler) listBoundNamespaces(clusterID, teamID uint) ([]string, error) {
	if teamID == 0 {
		return nil, nil
	}
	var rows []model.ClusterNamespaceBinding
	err := h.svcCtx.DB.Where("cluster_id = ? AND team_id = ?", clusterID, teamID).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Namespace)
	}
	return out, nil
}

func (h *Handler) isProdNamespace(clusterID uint, namespace string) bool {
	lower := strings.ToLower(strings.TrimSpace(namespace))
	if strings.Contains(lower, "prod") || strings.Contains(lower, "production") {
		return true
	}
	var count int64
	h.svcCtx.DB.Model(&model.ClusterNamespaceBinding{}).
		Where("cluster_id = ? AND namespace = ? AND env = ?", clusterID, namespace, "production").
		Count(&count)
	return count > 0
}

func (h *Handler) requireProdApproval(c *gin.Context, clusterID uint, namespace, action, approvalToken string) bool {
	if !h.isProdNamespace(clusterID, namespace) {
		return true
	}
	uid := httpx.UIDFromCtx(c)
	if httpx.HasAnyPermission(h.svcCtx.DB, uid, "k8s:approve", "kubernetes:approve") {
		return true
	}
	if strings.TrimSpace(approvalToken) == "" {
		ticket := fmt.Sprintf("k8s-appr-%d", time.Now().UnixNano())
		rec := model.ClusterDeployApproval{
			Ticket:    ticket,
			ClusterID: clusterID,
			Namespace: namespace,
			Action:    action,
			Status:    "pending",
			RequestBy: uint(uid),
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		_ = h.svcCtx.DB.Create(&rec).Error
		httpx.Fail(c, xcode.Forbidden, "production action requires k8s:approve or approval_token")
		return false
	}
	var rec model.ClusterDeployApproval
	if err := h.svcCtx.DB.Where("ticket = ?", approvalToken).First(&rec).Error; err != nil {
		httpx.Fail(c, xcode.Forbidden, "invalid approval token")
		return false
	}
	if rec.ClusterID != clusterID || rec.Namespace != namespace || rec.Action != action {
		httpx.Fail(c, xcode.Forbidden, "approval token scope mismatch")
		return false
	}
	if rec.Status != "approved" {
		httpx.Fail(c, xcode.Forbidden, "approval token not approved")
		return false
	}
	if !rec.ExpiresAt.IsZero() && time.Now().After(rec.ExpiresAt) {
		httpx.Fail(c, xcode.Forbidden, "approval token expired")
		return false
	}
	return true
}

func (h *Handler) createAudit(clusterID uint, namespace, action, resource, resourceID, status, message string, operatorID uint) {
	rec := model.ClusterOperationAudit{
		ClusterID:  clusterID,
		Namespace:  namespace,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Status:     status,
		Message:    message,
		OperatorID: operatorID,
	}
	_ = h.svcCtx.DB.Create(&rec).Error
}

func (h *Handler) buildRESTConfig(cluster *model.Cluster) (*rest.Config, error) {
	if cluster != nil && strings.TrimSpace(cluster.KubeConfig) != "" {
		return clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
	}
	if home := homedir.HomeDir(); home != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
		if err == nil {
			return cfg, nil
		}
	}
	return rest.InClusterConfig()
}

func (h *Handler) getClients(cluster *model.Cluster) (*kubernetes.Clientset, dynamic.Interface, error) {
	cfg, err := h.buildRESTConfig(cluster)
	if err != nil {
		return nil, nil, err
	}
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return cli, dc, nil
}

func (h *Handler) tx(fn func(tx *gorm.DB) error) error {
	return h.svcCtx.DB.Transaction(fn)
}
