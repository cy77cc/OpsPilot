// Package handler 提供 Kubernetes 集群管理的 HTTP Handler 实现。
//
// 本文件实现 Argo Rollout 相关的 HTTP Handler，包括 Rollout 预览、
// 应用、列表查询和操作 (promote/abort/rollback) 等功能。
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

// rolloutGVR Argo Rollout 的 GroupVersionResource。
var rolloutGVR = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "rollouts"}

// RolloutPreview 预览 Rollout 配置。
//
// @Summary 预览 Rollout
// @Description 生成 Rollout YAML 清单预览
// @Tags 集群管理-Rollout
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body rolloutApplyReq true "Rollout 请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/rollout/preview [post]
func (h *Handler) RolloutPreview(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:deploy", "k8s:write", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	var req rolloutApplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if !h.namespaceWritable(c, cluster.ID, req.Namespace) {
		return
	}
	manifest := buildRolloutManifest(req)
	out, _ := yaml.Marshal(manifest.Object)
	httpx.OK(c, gin.H{"manifest": string(out), "strategy": req.Strategy})
}

// rolloutApplyReq Rollout 应用请求结构。
type rolloutApplyReq struct {
	Namespace     string            `json:"namespace" binding:"required"` // 命名空间 (必填)
	Name          string            `json:"name" binding:"required"`      // 名称 (必填)
	Image         string            `json:"image" binding:"required"`     // 镜像 (必填)
	Replicas      int32             `json:"replicas"`                     // 副本数
	Strategy      string            `json:"strategy"`                     // 策略: rolling/canary/blue-green
	Labels        map[string]string `json:"labels"`                       // 标签
	CanarySteps   []map[string]any  `json:"canary_steps"`                 // Canary 步骤
	ActiveService string            `json:"active_service"`               // 活跃服务
	PreviewSvc    string            `json:"preview_service"`              // 预览服务
	ApprovalToken string            `json:"approval_token"`               // 审批票据
}

// buildRolloutManifest 构建 Rollout 资源清单。
//
// 参数:
//   - req: Rollout 应用请求
//
// 返回: Unstructured 对象
func buildRolloutManifest(req rolloutApplyReq) *unstructured.Unstructured {
	if req.Replicas <= 0 {
		req.Replicas = 1
	}
	if req.Strategy == "" {
		req.Strategy = "rolling"
	}
	labels := req.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	if _, ok := labels["app"]; !ok {
		labels["app"] = req.Name
	}
	strategy := map[string]any{}
	switch strings.ToLower(req.Strategy) {
	case "blue-green", "bluegreen":
		strategy["blueGreen"] = map[string]any{
			"activeService":  defaultString(req.ActiveService, req.Name),
			"previewService": defaultString(req.PreviewSvc, req.Name+"-preview"),
		}
	case "canary":
		steps := req.CanarySteps
		if len(steps) == 0 {
			steps = []map[string]any{{"setWeight": 20}, {"pause": map[string]any{"duration": "30s"}}, {"setWeight": 50}, {"pause": map[string]any{"duration": "30s"}}}
		}
		strategy["canary"] = map[string]any{"steps": steps}
	default:
		strategy["canary"] = map[string]any{"steps": []map[string]any{{"setWeight": 100}}}
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Rollout",
		"metadata": map[string]any{
			"name":      req.Name,
			"namespace": req.Namespace,
			"labels":    labels,
		},
		"spec": map[string]any{
			"replicas": req.Replicas,
			"selector": map[string]any{"matchLabels": map[string]any{"app": req.Name}},
			"template": map[string]any{"metadata": map[string]any{"labels": map[string]any{"app": req.Name}}, "spec": map[string]any{"containers": []map[string]any{{"name": req.Name, "image": req.Image}}}},
			"strategy": strategy,
		},
	}}
}

// defaultString 返回非空字符串或默认值。
//
// 参数:
//   - v: 待检查值
//   - d: 默认值
//
// 返回: 非空值或默认值
func defaultString(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

// RolloutApply 应用 Rollout 配置。
//
// @Summary 应用 Rollout
// @Description 创建或更新 Argo Rollout 资源
// @Tags 集群管理-Rollout
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body rolloutApplyReq true "Rollout 请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/rollout/apply [post]
func (h *Handler) RolloutApply(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:deploy", "k8s:write", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	var req rolloutApplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if !h.namespaceWritable(c, cluster.ID, req.Namespace) {
		return
	}
	if !h.requireProdApproval(c, cluster.ID, req.Namespace, "deploy", req.ApprovalToken) {
		return
	}
	_, dc, err := h.getClients(cluster)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	obj := buildRolloutManifest(req)
	resource := dc.Resource(rolloutGVR).Namespace(req.Namespace)
	existing, err := resource.Get(c.Request.Context(), req.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			if _, err := resource.Create(c.Request.Context(), obj, metav1.CreateOptions{}); err != nil {
				if isRolloutCRDMissing(err) {
					httpx.Fail(c, xcode.ServerError, "rollout_crd_missing")
					return
				}
				httpx.Fail(c, xcode.ServerError, err.Error())
				return
			}
		} else {
			if isRolloutCRDMissing(err) {
				httpx.Fail(c, xcode.ServerError, "rollout_crd_missing")
				return
			}
			httpx.Fail(c, xcode.ServerError, err.Error())
			return
		}
	} else {
		obj.SetResourceVersion(existing.GetResourceVersion())
		if _, err := resource.Update(c.Request.Context(), obj, metav1.UpdateOptions{}); err != nil {
			httpx.Fail(c, xcode.ServerError, err.Error())
			return
		}
	}
	raw, _ := json.Marshal(req)
	uid := httpx.UIDFromCtx(c)
	rec := model.ClusterReleaseRecord{ClusterID: cluster.ID, Namespace: req.Namespace, App: req.Name, Strategy: req.Strategy, RolloutName: req.Name, Revision: int(req.Replicas), Status: "applied", Operator: strconv.FormatUint(uid, 10), PayloadJSON: string(raw)}
	_ = h.svcCtx.DB.Create(&rec).Error
	h.createAudit(cluster.ID, req.Namespace, "rollout.apply", "rollout", req.Name, "success", "rollout applied", uint(uid))
	httpx.OK(c, gin.H{"applied": true, "name": req.Name})
}

// isRolloutCRDMissing 检查错误是否为 CRD 缺失。
//
// 参数:
//   - err: 错误对象
//
// 返回: CRD 缺失返回 true
func isRolloutCRDMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "the server could not find the requested resource") || strings.Contains(msg, "no matches for kind") || strings.Contains(msg, "argoproj.io")
}

// ListRollouts 获取 Rollout 列表。
//
// @Summary 获取 Rollout 列表
// @Description 获取指定集群的 Argo Rollout 列表
// @Tags 集群管理-Rollout
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string false "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/rollouts [get]
func (h *Handler) ListRollouts(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "k8s:deploy", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	_, dc, err := h.getClients(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "total": 0})
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	if ns == "" {
		ns = corev1.NamespaceAll
	}
	if ns != corev1.NamespaceAll && !h.namespaceReadable(c, cluster.ID, ns) {
		return
	}
	items, err := dc.Resource(rolloutGVR).Namespace(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		if isRolloutCRDMissing(err) {
			httpx.OK(c, gin.H{"list": []any{}, "total": 0, "diagnostics": []string{"rollout_crd_missing"}})
			return
		}
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	list := make([]gin.H, 0, len(items.Items))
	for _, item := range items.Items {
		strategy := "rolling"
		if _, ok, _ := unstructured.NestedMap(item.Object, "spec", "strategy", "canary"); ok {
			strategy = "canary"
		}
		if _, ok, _ := unstructured.NestedMap(item.Object, "spec", "strategy", "blueGreen"); ok {
			strategy = "blue-green"
		}
		ready, _, _ := unstructured.NestedInt64(item.Object, "status", "readyReplicas")
		replicas, _, _ := unstructured.NestedInt64(item.Object, "status", "replicas")
		phase, _, _ := unstructured.NestedString(item.Object, "status", "phase")
		list = append(list, gin.H{"name": item.GetName(), "namespace": item.GetNamespace(), "strategy": strategy, "phase": phase, "ready_replicas": ready, "replicas": replicas, "created_at": item.GetCreationTimestamp().Time})
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// RolloutPromote 推进 Rollout。
//
// @Summary 推进 Rollout
// @Description 推进 Argo Rollout 到下一阶段
// @Tags 集群管理-Rollout
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param name path string true "Rollout 名称"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/rollouts/{name}/promote [post]
func (h *Handler) RolloutPromote(c *gin.Context) {
	h.rolloutAction(c, "promote", "k8s:deploy")
}

// RolloutAbort 中止 Rollout。
//
// @Summary 中止 Rollout
// @Description 中止 Argo Rollout 当前部署
// @Tags 集群管理-Rollout
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param name path string true "Rollout 名称"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/rollouts/{name}/abort [post]
func (h *Handler) RolloutAbort(c *gin.Context) {
	h.rolloutAction(c, "abort", "k8s:deploy")
}

// RolloutRollback 回滚 Rollout。
//
// @Summary 回滚 Rollout
// @Description 回滚 Argo Rollout 到上一版本
// @Tags 集群管理-Rollout
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param name path string true "Rollout 名称"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/rollouts/{name}/rollback [post]
func (h *Handler) RolloutRollback(c *gin.Context) {
	h.rolloutAction(c, "undo", "k8s:rollback")
}

// rolloutAction 执行 Rollout 操作。
//
// 参数:
//   - c: Gin 上下文
//   - action: 操作类型 (promote/abort/undo)
//   - perm: 所需权限
func (h *Handler) rolloutAction(c *gin.Context, action, perm string) {
	if !httpx.Authorize(c, h.svcCtx.DB, perm, "k8s:write", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpx.Fail(c, xcode.ParamError, "rollout name required")
		return
	}
	var req struct {
		Namespace     string `json:"namespace"`
		ApprovalToken string `json:"approval_token"`
		Full          bool   `json:"full"`
	}
	_ = c.ShouldBindJSON(&req)
	if strings.TrimSpace(req.Namespace) == "" {
		req.Namespace = strings.TrimSpace(c.Query("namespace"))
	}
	if req.Namespace == "" {
		req.Namespace = "default"
	}
	if !h.namespaceWritable(c, cluster.ID, req.Namespace) {
		return
	}
	if !h.requireProdApproval(c, cluster.ID, req.Namespace, map[string]string{"promote": "deploy", "abort": "rollback", "undo": "rollback"}[action], req.ApprovalToken) {
		return
	}
	if err := h.execRolloutCLI(c.Request.Context(), cluster, req.Namespace, name, action, req.Full); err != nil {
		if isRolloutCRDMissing(err) {
			httpx.Fail(c, xcode.ServerError, "rollout_crd_missing")
			return
		}
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.createAudit(cluster.ID, req.Namespace, "rollout."+action, "rollout", name, "success", "rollout action executed", uint(httpx.UIDFromCtx(c)))
	httpx.OK(c, gin.H{"action": action, "name": name})
}

// execRolloutCLI 通过 kubectl argo-rollouts 插件执行操作。
//
// 参数:
//   - ctx: 上下文
//   - cluster: 集群模型
//   - namespace: 命名空间
//   - name: Rollout 名称
//   - action: 操作类型
//   - full: 是否完全推进
//
// 返回: 失败返回错误
func (h *Handler) execRolloutCLI(ctx context.Context, cluster *model.Cluster, namespace, name, action string, full bool) error {
	if strings.TrimSpace(cluster.KubeConfig) == "" {
		return errors.New("cluster kubeconfig required for rollout action")
	}
	kubeFile, err := os.CreateTemp("", "cluster-kubeconfig-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(kubeFile.Name())
	if _, err := kubeFile.WriteString(cluster.KubeConfig); err != nil {
		_ = kubeFile.Close()
		return err
	}
	_ = kubeFile.Close()

	args := []string{"argo", "rollouts", action, name, "-n", namespace, "--kubeconfig", kubeFile.Name()}
	if action == "promote" && full {
		args = append(args, "--full")
	}
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	msg := strings.ToLower(string(out) + err.Error())
	if strings.Contains(msg, "unknown command") || strings.Contains(msg, "argo") && strings.Contains(msg, "not found") {
		return fmt.Errorf("rollout_cli_missing: kubectl argo rollouts plugin is required")
	}
	if strings.Contains(msg, "no matches for kind") || strings.Contains(msg, "argoproj.io") {
		return fmt.Errorf("rollout_crd_missing")
	}
	return fmt.Errorf("rollout action failed: %s", strings.TrimSpace(string(out)))
}
