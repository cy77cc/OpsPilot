package handler

import (
	"strconv"
	"strings"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *Handler) Namespaces(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "total": 0, "data_source": "none"})
		return
	}
	nsList, err := cli.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	uid := httpx.UIDFromCtx(c)
	teamID := h.teamIDFromHeader(c)
	allowed := map[string]struct{}{}
	if !httpx.IsAdmin(h.svcCtx.DB, uid) && teamID > 0 {
		bound, _ := h.listBoundNamespaces(cluster.ID, teamID)
		for _, b := range bound {
			allowed[b] = struct{}{}
		}
	}
	list := make([]gin.H, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		if len(allowed) > 0 {
			if _, ok := allowed[ns.Name]; !ok {
				continue
			}
		}
		list = append(list, gin.H{"name": ns.Name, "status": string(ns.Status.Phase), "labels": ns.Labels, "created_at": ns.CreationTimestamp.Time})
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *Handler) CreateNamespace(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:write", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	var req struct {
		Name   string            `json:"name" binding:"required"`
		Env    string            `json:"env"`
		Labels map[string]string `json:"labels"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if !h.namespaceWritable(c, cluster.ID, req.Name) {
		if !httpx.IsAdmin(h.svcCtx.DB, httpx.UIDFromCtx(c)) {
			return
		}
	}
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	labels := req.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	if strings.TrimSpace(req.Env) != "" {
		labels["environment"] = req.Env
	}
	nsObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Labels: labels}}
	ns, err := cli.CoreV1().Namespaces().Create(c.Request.Context(), nsObj, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	if ns == nil {
		ns, _ = cli.CoreV1().Namespaces().Get(c.Request.Context(), req.Name, metav1.GetOptions{})
	}
	uid := httpx.UIDFromCtx(c)
	teamID := h.teamIDFromHeader(c)
	if teamID > 0 {
		var existing model.ClusterNamespaceBinding
		if h.svcCtx.DB.Where("cluster_id = ? AND team_id = ? AND namespace = ?", cluster.ID, teamID, req.Name).First(&existing).Error != nil {
			_ = h.svcCtx.DB.Create(&model.ClusterNamespaceBinding{ClusterID: cluster.ID, TeamID: teamID, Namespace: req.Name, Env: req.Env, Readonly: false, CreatedBy: uint(uid)}).Error
		}
	}
	h.createAudit(cluster.ID, req.Name, "namespace.create", "namespace", req.Name, "success", "namespace created", uint(uid))
	httpx.OK(c, ns)
}

func (h *Handler) DeleteNamespace(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:write", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpx.Fail(c, xcode.ParamError, "namespace required")
		return
	}
	if name == "kube-system" || name == "kube-public" {
		httpx.Fail(c, xcode.ParamError, "system namespace cannot be deleted")
		return
	}
	if !h.namespaceWritable(c, cluster.ID, name) {
		return
	}
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	if err := cli.CoreV1().Namespaces().Delete(c.Request.Context(), name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.createAudit(cluster.ID, name, "namespace.delete", "namespace", name, "success", "namespace deleted", uint(httpx.UIDFromCtx(c)))
	httpx.OK(c, nil)
}

func (h *Handler) ListNamespaceBindings(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:namespace:bind", "k8s:read", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	query := h.svcCtx.DB.Model(&model.ClusterNamespaceBinding{}).Where("cluster_id = ?", cluster.ID)
	if teamRaw := strings.TrimSpace(c.Query("team_id")); teamRaw != "" {
		query = query.Where("team_id = ?", teamRaw)
	}
	var rows []model.ClusterNamespaceBinding
	if err := query.Order("team_id ASC, namespace ASC").Find(&rows).Error; err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}

func (h *Handler) PutNamespaceBindings(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:namespace:bind", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	teamID64, err := strconv.ParseUint(c.Param("teamId"), 10, 64)
	if err != nil || teamID64 == 0 {
		httpx.Fail(c, xcode.ParamError, "invalid team id")
		return
	}
	var req struct {
		Bindings []struct {
			Namespace string `json:"namespace" binding:"required"`
			Env       string `json:"env"`
			Readonly  bool   `json:"readonly"`
		} `json:"bindings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	uid := uint(httpx.UIDFromCtx(c))
	teamID := uint(teamID64)
	if err := h.tx(func(tx *gorm.DB) error {
		if err := tx.Where("cluster_id = ? AND team_id = ?", cluster.ID, teamID).Delete(&model.ClusterNamespaceBinding{}).Error; err != nil {
			return err
		}
		for _, item := range req.Bindings {
			if strings.TrimSpace(item.Namespace) == "" {
				continue
			}
			row := model.ClusterNamespaceBinding{ClusterID: cluster.ID, TeamID: teamID, Namespace: strings.TrimSpace(item.Namespace), Env: strings.TrimSpace(item.Env), Readonly: item.Readonly, CreatedBy: uid}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.createAudit(cluster.ID, "", "namespace.bindings.update", "binding", strconv.FormatUint(teamID64, 10), "success", "namespace bindings updated", uid)
	h.ListNamespaceBindings(c)
}
