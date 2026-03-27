// Package handler 提供 Kubernetes 集群管理的 HTTP Handler 实现。
//
// 本文件实现 HPA (Horizontal Pod Autoscaler) 相关的 HTTP Handler，
// 包括 HPA 列表查询、创建、更新和删除等功能。
package handler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// ListHPA 获取 HPA 列表。
//
// @Summary 获取 HPA 列表
// @Description 获取指定集群的 Horizontal Pod Autoscaler 列表
// @Tags 集群管理-HPA
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string false "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/hpa [get]
func (h *Handler) ListHPA(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "k8s:hpa", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "total": 0})
		return
	}
	namespace := strings.TrimSpace(c.Query("namespace"))
	if namespace == "" {
		namespace = corev1.NamespaceAll
	}
	if namespace != corev1.NamespaceAll && !h.namespaceReadable(c, cluster.ID, namespace) {
		return
	}
	items, err := cli.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	out := make([]gin.H, 0, len(items.Items))
	for _, item := range items.Items {
		cpu, mem := int32(0), int32(0)
		for _, m := range item.Spec.Metrics {
			if m.Type != autoscalingv2.ResourceMetricSourceType || m.Resource == nil || m.Resource.Target.AverageUtilization == nil {
				continue
			}
			if m.Resource.Name == corev1.ResourceCPU {
				cpu = *m.Resource.Target.AverageUtilization
			}
			if m.Resource.Name == corev1.ResourceMemory {
				mem = *m.Resource.Target.AverageUtilization
			}
		}
		out = append(out, gin.H{
			"name":               item.Name,
			"namespace":          item.Namespace,
			"target_ref_kind":    item.Spec.ScaleTargetRef.Kind,
			"target_ref_name":    item.Spec.ScaleTargetRef.Name,
			"min_replicas":       ptr.Deref(item.Spec.MinReplicas, 1),
			"max_replicas":       item.Spec.MaxReplicas,
			"cpu_utilization":    cpu,
			"memory_utilization": mem,
		})
	}
	httpx.OK(c, gin.H{"list": out, "total": len(out)})
}

// CreateHPA 创建 HPA。
//
// @Summary 创建 HPA
// @Description 为指定工作负载创建 Horizontal Pod Autoscaler
// @Tags 集群管理-HPA
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body map[string]interface{} true "HPA 创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/hpa [post]
func (h *Handler) CreateHPA(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:hpa", "k8s:write", "kubernetes:write") {
		return
	}
	h.applyHPA(c, false)
}

// UpdateHPA 更新 HPA。
//
// @Summary 更新 HPA
// @Description 更新指定的 Horizontal Pod Autoscaler 配置
// @Tags 集群管理-HPA
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body map[string]interface{} true "HPA 更新请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/hpa [put]
func (h *Handler) UpdateHPA(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:hpa", "k8s:write", "kubernetes:write") {
		return
	}
	h.applyHPA(c, true)
}

// applyHPA 应用 HPA 配置 (创建或更新)。
//
// 参数:
//   - c: Gin 上下文
//   - update: 是否为更新操作
func (h *Handler) applyHPA(c *gin.Context, update bool) {
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	var req struct {
		Namespace         string `json:"namespace" binding:"required"`
		Name              string `json:"name" binding:"required"`
		TargetRefKind     string `json:"target_ref_kind" binding:"required"`
		TargetRefName     string `json:"target_ref_name" binding:"required"`
		MinReplicas       int32  `json:"min_replicas"`
		MaxReplicas       int32  `json:"max_replicas" binding:"required"`
		CPUUtilization    *int32 `json:"cpu_utilization"`
		MemoryUtilization *int32 `json:"memory_utilization"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if !h.namespaceWritable(c, cluster.ID, req.Namespace) {
		return
	}
	if req.MinReplicas <= 0 {
		req.MinReplicas = 1
	}
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	metrics := make([]autoscalingv2.MetricSpec, 0, 2)
	if req.CPUUtilization != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{Type: autoscalingv2.ResourceMetricSourceType, Resource: &autoscalingv2.ResourceMetricSource{Name: corev1.ResourceCPU, Target: autoscalingv2.MetricTarget{Type: autoscalingv2.UtilizationMetricType, AverageUtilization: req.CPUUtilization}}})
	}
	if req.MemoryUtilization != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{Type: autoscalingv2.ResourceMetricSourceType, Resource: &autoscalingv2.ResourceMetricSource{Name: corev1.ResourceMemory, Target: autoscalingv2.MetricTarget{Type: autoscalingv2.UtilizationMetricType, AverageUtilization: req.MemoryUtilization}}})
	}
	if len(metrics) == 0 {
		httpx.Fail(c, xcode.ParamError, "at least one metric is required")
		return
	}
	obj := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: req.TargetRefKind, Name: req.TargetRefName},
			MinReplicas:    ptr.To(req.MinReplicas),
			MaxReplicas:    req.MaxReplicas,
			Metrics:        metrics,
		},
	}
	if update {
		if _, err := cli.AutoscalingV2().HorizontalPodAutoscalers(req.Namespace).Update(c.Request.Context(), obj, metav1.UpdateOptions{}); err != nil {
			httpx.Fail(c, xcode.ServerError, err.Error())
			return
		}
	} else {
		if _, err := cli.AutoscalingV2().HorizontalPodAutoscalers(req.Namespace).Create(c.Request.Context(), obj, metav1.CreateOptions{}); err != nil {
			httpx.Fail(c, xcode.ServerError, err.Error())
			return
		}
	}
	raw, _ := json.Marshal(req)
	_ = h.svcCtx.DB.Save(&model.ClusterHPAPolicy{ClusterID: cluster.ID, Namespace: req.Namespace, Name: req.Name, TargetRefKind: req.TargetRefKind, TargetRefName: req.TargetRefName, MinReplicas: req.MinReplicas, MaxReplicas: req.MaxReplicas, CPUUtilization: req.CPUUtilization, MemoryUtilization: req.MemoryUtilization, RawPolicyJSON: string(raw)}).Error
	h.createAudit(cluster.ID, req.Namespace, "hpa.apply", "hpa", req.Name, "success", "hpa policy applied", uint(httpx.UIDFromCtx(c)))
	httpx.OK(c, obj)
}

// DeleteHPA 删除 HPA。
//
// @Summary 删除 HPA
// @Description 删除指定的 Horizontal Pod Autoscaler
// @Tags 集群管理-HPA
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param name path string true "HPA 名称"
// @Param namespace query string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/hpa/{name} [delete]
func (h *Handler) DeleteHPA(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:hpa", "k8s:write", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	namespace := strings.TrimSpace(c.Query("namespace"))
	if namespace == "" {
		httpx.Fail(c, xcode.ParamError, "namespace required")
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpx.Fail(c, xcode.ParamError, "name required")
		return
	}
	if !h.namespaceWritable(c, cluster.ID, namespace) {
		return
	}
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	if err := cli.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	_ = h.svcCtx.DB.Where("cluster_id = ? AND namespace = ? AND name = ?", cluster.ID, namespace, name).Delete(&model.ClusterHPAPolicy{}).Error
	h.createAudit(cluster.ID, namespace, "hpa.delete", "hpa", name, "success", "hpa policy deleted", uint(httpx.UIDFromCtx(c)))
	httpx.OK(c, nil)
}

// ListQuotas 获取 ResourceQuota 列表。
//
// @Summary 获取配额列表
// @Description 获取指定集群的 ResourceQuota 列表
// @Tags 集群管理-配额
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string false "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/quotas [get]
func (h *Handler) ListQuotas(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "k8s:quota", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "total": 0})
		return
	}
	namespace := strings.TrimSpace(c.Query("namespace"))
	if namespace == "" {
		namespace = corev1.NamespaceAll
	}
	if namespace != corev1.NamespaceAll && !h.namespaceReadable(c, cluster.ID, namespace) {
		return
	}
	items, err := cli.CoreV1().ResourceQuotas(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	list := make([]gin.H, 0, len(items.Items))
	for _, item := range items.Items {
		hard := map[string]string{}
		for k, v := range item.Spec.Hard {
			hard[string(k)] = v.String()
		}
		list = append(list, gin.H{"name": item.Name, "namespace": item.Namespace, "hard": hard})
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// CreateOrUpdateQuota 创建或更新 ResourceQuota。
//
// @Summary 创建或更新配额
// @Description 创建或更新指定命名空间的 ResourceQuota
// @Tags 集群管理-配额
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body map[string]interface{} true "配额请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/quotas [post]
func (h *Handler) CreateOrUpdateQuota(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:quota", "k8s:write", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	var req struct {
		Namespace string            `json:"namespace" binding:"required"`
		Name      string            `json:"name" binding:"required"`
		Hard      map[string]string `json:"hard" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if !h.namespaceWritable(c, cluster.ID, req.Namespace) {
		return
	}
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	hard := corev1.ResourceList{}
	for k, v := range req.Hard {
		q, err := resource.ParseQuantity(v)
		if err != nil {
			httpx.Fail(c, xcode.ParamError, fmt.Sprintf("invalid quota %s: %v", k, err))
			return
		}
		hard[corev1.ResourceName(k)] = q
	}
	obj := &corev1.ResourceQuota{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: corev1.ResourceQuotaSpec{Hard: hard}}
	if _, err := cli.CoreV1().ResourceQuotas(req.Namespace).Get(c.Request.Context(), req.Name, metav1.GetOptions{}); err == nil {
		if _, err := cli.CoreV1().ResourceQuotas(req.Namespace).Update(c.Request.Context(), obj, metav1.UpdateOptions{}); err != nil {
			httpx.Fail(c, xcode.ServerError, err.Error())
			return
		}
	} else {
		if _, err := cli.CoreV1().ResourceQuotas(req.Namespace).Create(c.Request.Context(), obj, metav1.CreateOptions{}); err != nil {
			httpx.Fail(c, xcode.ServerError, err.Error())
			return
		}
	}
	raw, _ := json.Marshal(req)
	_ = h.svcCtx.DB.Save(&model.ClusterQuotaPolicy{ClusterID: cluster.ID, Namespace: req.Namespace, Name: req.Name, Type: "resourcequota", SpecJSON: string(raw)}).Error
	h.createAudit(cluster.ID, req.Namespace, "quota.apply", "resourcequota", req.Name, "success", "quota applied", uint(httpx.UIDFromCtx(c)))
	httpx.OK(c, obj)
}

// DeleteQuota 删除 ResourceQuota。
//
// @Summary 删除配额
// @Description 删除指定的 ResourceQuota
// @Tags 集群管理-配额
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param name path string true "配额名称"
// @Param namespace query string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/quotas/{name} [delete]
func (h *Handler) DeleteQuota(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:quota", "k8s:write", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	namespace := strings.TrimSpace(c.Query("namespace"))
	if namespace == "" {
		httpx.Fail(c, xcode.ParamError, "namespace required")
		return
	}
	if !h.namespaceWritable(c, cluster.ID, namespace) {
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	if err := cli.CoreV1().ResourceQuotas(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	_ = h.svcCtx.DB.Where("cluster_id = ? AND namespace = ? AND name = ? AND type = ?", cluster.ID, namespace, name, "resourcequota").Delete(&model.ClusterQuotaPolicy{}).Error
	h.createAudit(cluster.ID, namespace, "quota.delete", "resourcequota", name, "success", "quota deleted", uint(httpx.UIDFromCtx(c)))
	httpx.OK(c, nil)
}

// ListLimitRanges 获取 LimitRange 列表。
//
// @Summary 获取 LimitRange 列表
// @Description 获取指定集群的 LimitRange 列表
// @Tags 集群管理-配额
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string false "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/limitranges [get]
func (h *Handler) ListLimitRanges(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "k8s:quota", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "total": 0})
		return
	}
	namespace := strings.TrimSpace(c.Query("namespace"))
	if namespace == "" {
		namespace = corev1.NamespaceAll
	}
	if namespace != corev1.NamespaceAll && !h.namespaceReadable(c, cluster.ID, namespace) {
		return
	}
	items, err := cli.CoreV1().LimitRanges(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	list := make([]gin.H, 0, len(items.Items))
	for _, item := range items.Items {
		list = append(list, gin.H{"name": item.Name, "namespace": item.Namespace, "limits": item.Spec.Limits})
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// CreateLimitRange 创建 LimitRange。
//
// @Summary 创建 LimitRange
// @Description 为指定命名空间创建 LimitRange
// @Tags 集群管理-配额
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body map[string]interface{} true "LimitRange 请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/limitranges [post]
func (h *Handler) CreateLimitRange(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:quota", "k8s:write", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	var req struct {
		Namespace      string            `json:"namespace" binding:"required"`
		Name           string            `json:"name" binding:"required"`
		Default        map[string]string `json:"default"`
		DefaultRequest map[string]string `json:"default_request"`
		Min            map[string]string `json:"min"`
		Max            map[string]string `json:"max"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if !h.namespaceWritable(c, cluster.ID, req.Namespace) {
		return
	}
	buildRes := func(m map[string]string) (corev1.ResourceList, error) {
		out := corev1.ResourceList{}
		for k, v := range m {
			q, err := resource.ParseQuantity(v)
			if err != nil {
				return nil, err
			}
			out[corev1.ResourceName(k)] = q
		}
		return out, nil
	}
	def, err := buildRes(req.Default)
	if err != nil {
		httpx.BindErr(c, err)
		return
	}
	defReq, err := buildRes(req.DefaultRequest)
	if err != nil {
		httpx.BindErr(c, err)
		return
	}
	minRes, err := buildRes(req.Min)
	if err != nil {
		httpx.BindErr(c, err)
		return
	}
	maxRes, err := buildRes(req.Max)
	if err != nil {
		httpx.BindErr(c, err)
		return
	}
	obj := &corev1.LimitRange{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: corev1.LimitRangeSpec{Limits: []corev1.LimitRangeItem{{Type: corev1.LimitTypeContainer, Default: def, DefaultRequest: defReq, Min: minRes, Max: maxRes}}}}
	cli, _, err := h.getClients(cluster)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	if _, err := cli.CoreV1().LimitRanges(req.Namespace).Get(c.Request.Context(), req.Name, metav1.GetOptions{}); err == nil {
		if _, err := cli.CoreV1().LimitRanges(req.Namespace).Update(c.Request.Context(), obj, metav1.UpdateOptions{}); err != nil {
			httpx.Fail(c, xcode.ServerError, err.Error())
			return
		}
	} else {
		if _, err := cli.CoreV1().LimitRanges(req.Namespace).Create(c.Request.Context(), obj, metav1.CreateOptions{}); err != nil {
			httpx.Fail(c, xcode.ServerError, err.Error())
			return
		}
	}
	raw, _ := json.Marshal(req)
	_ = h.svcCtx.DB.Save(&model.ClusterQuotaPolicy{ClusterID: cluster.ID, Namespace: req.Namespace, Name: req.Name, Type: "limitrange", SpecJSON: string(raw)}).Error
	h.createAudit(cluster.ID, req.Namespace, "limitrange.apply", "limitrange", req.Name, "success", "limitrange applied", uint(httpx.UIDFromCtx(c)))
	httpx.OK(c, obj)
}
