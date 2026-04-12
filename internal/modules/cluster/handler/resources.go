// Package handler 提供 Kubernetes 集群管理的 HTTP Handler 实现。
//
// 本文件实现集群资源查询相关的 HTTP Handler，包括命名空间、工作负载、
// 服务、配置和存储等资源的查询功能。
package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1typed "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1typed "k8s.io/client-go/kubernetes/typed/core/v1"
)

// NamespaceInfo 命名空间信息结构。
type NamespaceInfo struct {
	Name      string            `json:"name"`       // 命名空间名称
	Status    string            `json:"status"`     // 状态: Active/Terminating
	Labels    map[string]string `json:"labels"`     // 标签
	CreatedAt string            `json:"created_at"` // 创建时间
}

// PodInfo Pod 信息结构。
type PodInfo struct {
	Name      string            `json:"name"`       // Pod 名称
	Namespace string            `json:"namespace"`  // 命名空间
	Status    string            `json:"status"`     // 状态: Running/Pending/Failed/Succeeded
	PodIP     string            `json:"pod_ip"`     // Pod IP
	NodeName  string            `json:"node_name"`  // 所在节点
	Ready     string            `json:"ready"`      // 就绪状态: n/m
	Restarts  int32             `json:"restarts"`   // 重启次数
	Age       string            `json:"age"`        // 创建时长
	Labels    map[string]string `json:"labels"`     // 标签
	CreatedAt string            `json:"created_at"` // 创建时间
}

// DeploymentInfo Deployment 信息结构。
type DeploymentInfo struct {
	Name      string `json:"name"`       // Deployment 名称
	Namespace string `json:"namespace"`  // 命名空间
	Replicas  int32  `json:"replicas"`   // 期望副本数
	Ready     int32  `json:"ready"`      // 就绪副本数
	Updated   int32  `json:"updated"`    // 已更新副本数
	Available int32  `json:"available"`  // 可用副本数
	Age       string `json:"age"`        // 创建时长
	CreatedAt string `json:"created_at"` // 创建时间
}

// StatefulSetInfo StatefulSet 信息结构。
type StatefulSetInfo struct {
	Name      string `json:"name"`       // StatefulSet 名称
	Namespace string `json:"namespace"`  // 命名空间
	Replicas  int32  `json:"replicas"`   // 期望副本数
	Ready     int32  `json:"ready"`      // 就绪副本数
	Age       string `json:"age"`        // 创建时长
	CreatedAt string `json:"created_at"` // 创建时间
}

const workloadRestartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"

type kubernetesWorkloadClient interface {
	AppsV1() appsv1typed.AppsV1Interface
	CoreV1() corev1typed.CoreV1Interface
}

type workloadOperationTarget struct {
	Namespace string
	Resource  string
	Name      string
	Action    string
}

type WorkloadHighRiskReq struct {
	ApprovalToken string `json:"approval_token"`
}

type WorkloadScaleReq struct {
	Replicas      *int32 `json:"replicas"`
	ApprovalToken string `json:"approval_token"`
}

// DaemonSetInfo DaemonSet 信息结构。
type DaemonSetInfo struct {
	Name      string `json:"name"`       // DaemonSet 名称
	Namespace string `json:"namespace"`  // 命名空间
	Desired   int32  `json:"desired"`    // 期望副本数
	Ready     int32  `json:"ready"`      // 就绪副本数
	Age       string `json:"age"`        // 创建时长
	CreatedAt string `json:"created_at"` // 创建时间
}

// JobInfo Job 信息结构。
type JobInfo struct {
	Name        string `json:"name"`        // Job 名称
	Namespace   string `json:"namespace"`   // 命名空间
	Completions int32  `json:"completions"` // 完成数
	Succeeded   int32  `json:"succeeded"`   // 成功数
	Failed      int32  `json:"failed"`      // 失败数
	Status      string `json:"status"`      // 状态: Running/Completed/Failed
	Age         string `json:"age"`         // 创建时长
	CreatedAt   string `json:"created_at"`  // 创建时间
}

// ServiceInfo Service 信息结构。
type ServiceInfo struct {
	Name      string            `json:"name"`       // Service 名称
	Namespace string            `json:"namespace"`  // 命名空间
	Type      string            `json:"type"`       // 类型: ClusterIP/NodePort/LoadBalancer
	ClusterIP string            `json:"cluster_ip"` // 集群 IP
	Ports     []ServicePort     `json:"ports"`      // 端口列表
	Selector  map[string]string `json:"selector"`   // 选择器
	Age       string            `json:"age"`        // 创建时长
	CreatedAt string            `json:"created_at"` // 创建时间
}

// ServicePort Service 端口结构。
type ServicePort struct {
	Name       string `json:"name"`        // 端口名称
	Port       int32  `json:"port"`        // 端口号
	TargetPort string `json:"target_port"` // 目标端口
	Protocol   string `json:"protocol"`    // 协议: TCP/UDP
}

// IngressInfo Ingress 信息结构。
type IngressInfo struct {
	Name      string        `json:"name"`       // Ingress 名称
	Namespace string        `json:"namespace"`  // 命名空间
	Hosts     []IngressHost `json:"hosts"`      // 主机列表
	Age       string        `json:"age"`        // 创建时长
	CreatedAt string        `json:"created_at"` // 创建时间
}

// IngressHost Ingress 主机结构。
type IngressHost struct {
	Host  string   `json:"host"`  // 主机名
	Paths []string `json:"paths"` // 路径列表
}

// ConfigMapInfo ConfigMap 信息结构。
type ConfigMapInfo struct {
	Name      string   `json:"name"`       // ConfigMap 名称
	Namespace string   `json:"namespace"`  // 命名空间
	DataKeys  []string `json:"data_keys"`  // 数据键列表
	Age       string   `json:"age"`        // 创建时长
	CreatedAt string   `json:"created_at"` // 创建时间
}

// SecretInfo Secret 信息结构 (仅元数据)。
type SecretInfo struct {
	Name      string   `json:"name"`       // Secret 名称
	Namespace string   `json:"namespace"`  // 命名空间
	Type      string   `json:"type"`       // 类型: Opaque/TLS/ServiceAccountToken
	DataKeys  []string `json:"data_keys"`  // 数据键列表
	Age       string   `json:"age"`        // 创建时长
	CreatedAt string   `json:"created_at"` // 创建时间
}

// PVCInfo PVC 信息结构。
type PVCInfo struct {
	Name         string `json:"name"`          // PVC 名称
	Namespace    string `json:"namespace"`     // 命名空间
	Status       string `json:"status"`        // 状态: Bound/Pending
	Capacity     string `json:"capacity"`      // 容量
	AccessModes  string `json:"access_modes"`  // 访问模式
	StorageClass string `json:"storage_class"` // 存储类
	VolumeName   string `json:"volume_name"`   // 卷名称
	Age          string `json:"age"`           // 创建时长
	CreatedAt    string `json:"created_at"`    // 创建时间
}

// PVInfo PV 信息结构。
type PVInfo struct {
	Name         string `json:"name"`          // PV 名称
	Status       string `json:"status"`        // 状态: Available/Bound/Released
	Capacity     string `json:"capacity"`      // 容量
	AccessModes  string `json:"access_modes"`  // 访问模式
	StorageClass string `json:"storage_class"` // 存储类
	ClaimRef     string `json:"claim_ref"`     // 绑定声明: namespace/pvc-name
	Age          string `json:"age"`           // 创建时长
	CreatedAt    string `json:"created_at"`    // 创建时间
}

// getClusterClient 获取集群的 Kubernetes 客户端。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//
// 返回: Kubernetes 客户端，失败返回错误
func (h *Handler) getClusterClient(ctx context.Context, clusterID uint) (*kubernetes.Clientset, error) {
	var cred model.ClusterCredential
	if err := h.svcCtx.DB.WithContext(ctx).
		Where("cluster_id = ?", clusterID).
		First(&cred).Error; err != nil {
		return nil, fmt.Errorf("credential not found: %w", err)
	}

	restConfig, err := h.buildRestConfigFromCredential(&cred)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restConfig)
}

// GetNamespaces 获取集群命名空间列表。
//
// @Summary 获取命名空间列表
// @Description 获取指定集群的所有命名空间
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces [get]
func (h *Handler) GetNamespaces(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]NamespaceInfo, 0, len(list.Items))
	for _, ns := range list.Items {
		items = append(items, NamespaceInfo{
			Name:      ns.Name,
			Status:    string(ns.Status.Phase),
			Labels:    ns.Labels,
			CreatedAt: ns.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetPods 获取命名空间的 Pod 列表。
//
// @Summary 获取 Pod 列表
// @Description 获取指定命名空间的 Pod 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/pods [get]
func (h *Handler) GetPods(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.CoreV1().Pods(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]PodInfo, 0, len(list.Items))
	for _, pod := range list.Items {
		ready, restarts := getPodReadyAndRestarts(&pod)
		items = append(items, PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			PodIP:     pod.Status.PodIP,
			NodeName:  pod.Spec.NodeName,
			Ready:     ready,
			Restarts:  restarts,
			Age:       getAge(pod.CreationTimestamp),
			Labels:    pod.Labels,
			CreatedAt: pod.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetDeployments 获取命名空间的 Deployment 列表。
//
// @Summary 获取 Deployment 列表
// @Description 获取指定命名空间的 Deployment 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/deployments [get]
func (h *Handler) GetDeployments(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.AppsV1().Deployments(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]DeploymentInfo, 0, len(list.Items))
	for _, dep := range list.Items {
		replicas := int32(0)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}
		items = append(items, DeploymentInfo{
			Name:      dep.Name,
			Namespace: dep.Namespace,
			Replicas:  replicas,
			Ready:     dep.Status.ReadyReplicas,
			Updated:   dep.Status.UpdatedReplicas,
			Available: dep.Status.AvailableReplicas,
			Age:       getAge(dep.CreationTimestamp),
			CreatedAt: dep.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetStatefulSets 获取命名空间的 StatefulSet 列表。
//
// @Summary 获取 StatefulSet 列表
// @Description 获取指定命名空间的 StatefulSet 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/statefulsets [get]
func (h *Handler) GetStatefulSets(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.AppsV1().StatefulSets(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]StatefulSetInfo, 0, len(list.Items))
	for _, sts := range list.Items {
		replicas := int32(0)
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}
		items = append(items, StatefulSetInfo{
			Name:      sts.Name,
			Namespace: sts.Namespace,
			Replicas:  replicas,
			Ready:     sts.Status.ReadyReplicas,
			Age:       getAge(sts.CreationTimestamp),
			CreatedAt: sts.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// RestartDeployment 重启 Deployment。
func (h *Handler) RestartDeployment(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	var req WorkloadHighRiskReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpx.BindErr(c, err)
		return
	}

	clusterID, target, ok := parseWorkloadOperationTarget(c, "deployment", "deployment.restart")
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	result, err := h.executeHighRiskWorkloadOperation(c, clusterID, target, req.ApprovalToken, func(ctx context.Context, client kubernetesWorkloadClient) (map[string]any, error) {
		return h.restartDeployment(ctx, client, target.Namespace, target.Name)
	})
	if err != nil {
		h.handleHighRiskWorkloadOperationError(c, target.Resource, err)
		return
	}
	httpx.OK(c, result)
}

// ScaleDeployment 调整 Deployment 副本数。
func (h *Handler) ScaleDeployment(c *gin.Context) {
	h.ScaleDeploymentImpl(c)
}

// DeleteDeployment 删除 Deployment。
func (h *Handler) DeleteDeployment(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	var req WorkloadHighRiskReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpx.BindErr(c, err)
		return
	}

	clusterID, target, ok := parseWorkloadOperationTarget(c, "deployment", "deployment.delete")
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	result, err := h.executeHighRiskWorkloadOperation(c, clusterID, target, req.ApprovalToken, func(ctx context.Context, client kubernetesWorkloadClient) (map[string]any, error) {
		return h.deleteDeployment(ctx, client, target.Namespace, target.Name)
	})
	if err != nil {
		h.handleHighRiskWorkloadOperationError(c, target.Resource, err)
		return
	}
	httpx.OK(c, result)
}

// RestartStatefulSet 重启 StatefulSet。
func (h *Handler) RestartStatefulSet(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	var req WorkloadHighRiskReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpx.BindErr(c, err)
		return
	}

	clusterID, target, ok := parseWorkloadOperationTarget(c, "statefulset", "statefulset.restart")
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	result, err := h.executeHighRiskWorkloadOperation(c, clusterID, target, req.ApprovalToken, func(ctx context.Context, client kubernetesWorkloadClient) (map[string]any, error) {
		return h.restartStatefulSet(ctx, client, target.Namespace, target.Name)
	})
	if err != nil {
		h.handleHighRiskWorkloadOperationError(c, target.Resource, err)
		return
	}
	httpx.OK(c, result)
}

// ScaleStatefulSet 调整 StatefulSet 副本数。
func (h *Handler) ScaleStatefulSet(c *gin.Context) {
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

	clusterID, target, ok := parseWorkloadOperationTarget(c, "statefulset", "statefulset.scale")
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	result, err := h.executeHighRiskWorkloadOperation(c, clusterID, target, req.ApprovalToken, func(ctx context.Context, client kubernetesWorkloadClient) (map[string]any, error) {
		return h.scaleStatefulSet(ctx, client, target.Namespace, target.Name, *req.Replicas)
	})
	if err != nil {
		h.handleHighRiskWorkloadOperationError(c, target.Resource, err)
		return
	}
	httpx.OK(c, result)
}

// DeleteStatefulSet 删除 StatefulSet。
func (h *Handler) DeleteStatefulSet(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	var req WorkloadHighRiskReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpx.BindErr(c, err)
		return
	}

	clusterID, target, ok := parseWorkloadOperationTarget(c, "statefulset", "statefulset.delete")
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	result, err := h.executeHighRiskWorkloadOperation(c, clusterID, target, req.ApprovalToken, func(ctx context.Context, client kubernetesWorkloadClient) (map[string]any, error) {
		return h.deleteStatefulSet(ctx, client, target.Namespace, target.Name)
	})
	if err != nil {
		h.handleHighRiskWorkloadOperationError(c, target.Resource, err)
		return
	}
	httpx.OK(c, result)
}

// DeletePod 删除 Pod。
func (h *Handler) DeletePod(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	var req WorkloadHighRiskReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpx.BindErr(c, err)
		return
	}

	clusterID, target, ok := parseWorkloadOperationTarget(c, "pod", "pod.delete")
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	result, err := h.executeHighRiskWorkloadOperation(c, clusterID, target, req.ApprovalToken, func(ctx context.Context, client kubernetesWorkloadClient) (map[string]any, error) {
		return h.deletePod(ctx, client, target.Namespace, target.Name)
	})
	if err != nil {
		h.handleHighRiskWorkloadOperationError(c, target.Resource, err)
		return
	}
	httpx.OK(c, result)
}

func parseWorkloadOperationTarget(c *gin.Context, resource, action string) (uint, workloadOperationTarget, bool) {
	clusterID := httpx.UintFromParam(c, "id")
	namespace := c.Param("namespace")
	name := c.Param("name")
	target := workloadOperationTarget{
		Namespace: namespace,
		Resource:  resource,
		Name:      name,
		Action:    action,
	}
	if clusterID == 0 || target.Namespace == "" || target.Name == "" {
		return 0, workloadOperationTarget{}, false
	}
	return clusterID, target, true
}

func (h *Handler) executeHighRiskWorkloadOperation(c *gin.Context, clusterID uint, target workloadOperationTarget, approvalToken string, fn func(context.Context, kubernetesWorkloadClient) (map[string]any, error)) (ClusterOperationResponse, error) {
	client, err := h.getClusterClient(c.Request.Context(), clusterID)
	if err != nil {
		return ClusterOperationResponse{}, err
	}
	return h.executeHighRiskWorkloadOperationWithClient(c, clusterID, target, approvalToken, client, fn)
}

func (h *Handler) executeHighRiskWorkloadOperationWithClient(c *gin.Context, clusterID uint, target workloadOperationTarget, approvalToken string, client kubernetesWorkloadClient, fn func(context.Context, kubernetesWorkloadClient) (map[string]any, error)) (ClusterOperationResponse, error) {
	ctx := c.Request.Context()
	var cluster model.Cluster
	if err := h.svcCtx.DB.WithContext(ctx).First(&cluster, clusterID).Error; err != nil {
		return ClusterOperationResponse{}, fmt.Errorf("cluster not found: %w", err)
	}
	if cluster.Source != "platform_managed" {
		return ClusterOperationResponse{}, fmt.Errorf("cannot modify workloads in externally managed cluster")
	}
	if err := h.ensureWorkloadExists(ctx, client, target); err != nil {
		return ClusterOperationResponse{}, err
	}

	gate := h.requireHighRiskApproval(ctx, clusterID, target.Namespace, target.Action, target.Resource, target.Name, approvalToken, httpx.UIDFromCtx(c))
	if !gate.Allowed {
		return operationResponseFromGate(clusterID, target.Resource, target.Name, gate), nil
	}

	details, opErr := fn(ctx, client)
	operatorID := uint(httpx.UIDFromCtx(c))
	if opErr != nil {
		audit, _ := h.RecordClusterOperationAudit(ctx, clusterID, target.Namespace, target.Action, target.Resource, target.Name, "failed", opErr.Error(), operatorID)
		return ClusterOperationResponse{
			State:       OperationStateFailed,
			Code:        OperationCodeFailed,
			Message:     sanitizeOperationText(opErr.Error()),
			ClusterID:   clusterID,
			Resource:    target.Resource,
			ResourceID:  target.Name,
			AuditID:     audit.ID,
			Diagnostics: sanitizeOperationText(opErr.Error()),
		}, nil
	}

	audit, err := h.RecordClusterOperationAudit(ctx, clusterID, target.Namespace, target.Action, target.Resource, target.Name, "success", target.Action+" executed", operatorID)
	if err != nil {
		return operationSuccessResponse(clusterID, target.Resource, target.Name, target.Action+" executed", 0, details), nil
	}
	return operationSuccessResponse(clusterID, target.Resource, target.Name, target.Action+" executed", audit.ID, details), nil
}

func (h *Handler) handleHighRiskWorkloadOperationError(c *gin.Context, resource string, err error) {
	switch {
	case err == nil:
		return
	case apierrors.IsNotFound(err):
		httpx.NotFound(c, resource+" not found")
	case errors.Is(err, context.Canceled):
		httpx.ServerErr(c, err)
	case err.Error() == "cannot modify workloads in externally managed cluster":
		httpx.BadRequest(c, err.Error())
	case strings.Contains(err.Error(), "cluster not found"):
		httpx.NotFound(c, "cluster not found")
	default:
		httpx.ServerErr(c, err)
	}
}

func (h *Handler) ensureWorkloadExists(ctx context.Context, client kubernetesWorkloadClient, target workloadOperationTarget) error {
	switch target.Resource {
	case "deployment":
		_, err := client.AppsV1().Deployments(target.Namespace).Get(ctx, target.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("deployment not found: %w", err)
		}
	case "statefulset":
		_, err := client.AppsV1().StatefulSets(target.Namespace).Get(ctx, target.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("statefulset not found: %w", err)
		}
	case "pod":
		_, err := client.CoreV1().Pods(target.Namespace).Get(ctx, target.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("pod not found: %w", err)
		}
	default:
		return fmt.Errorf("unsupported workload resource %q", target.Resource)
	}
	return nil
}

func (h *Handler) restartDeployment(ctx context.Context, client kubernetesWorkloadClient, namespace, name string) (map[string]any, error) {
	deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	restartedAt := time.Now().UTC().Format(time.RFC3339)
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations[workloadRestartedAtAnnotation] = restartedAt
	if _, err := client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "deployment", "name": name, "restarted_at": restartedAt}, nil
}

func (h *Handler) scaleDeployment(ctx context.Context, client kubernetesWorkloadClient, namespace, name string, replicas int32) (map[string]any, error) {
	deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	deployment.Spec.Replicas = &replicas
	if _, err := client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "deployment", "name": name, "replicas": replicas}, nil
}

func (h *Handler) deleteDeployment(ctx context.Context, client kubernetesWorkloadClient, namespace, name string) (map[string]any, error) {
	if err := client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "deployment", "name": name, "deleted": true}, nil
}

func (h *Handler) restartStatefulSet(ctx context.Context, client kubernetesWorkloadClient, namespace, name string) (map[string]any, error) {
	statefulset, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	restartedAt := time.Now().UTC().Format(time.RFC3339)
	if statefulset.Spec.Template.Annotations == nil {
		statefulset.Spec.Template.Annotations = map[string]string{}
	}
	statefulset.Spec.Template.Annotations[workloadRestartedAtAnnotation] = restartedAt
	if _, err := client.AppsV1().StatefulSets(namespace).Update(ctx, statefulset, metav1.UpdateOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "statefulset", "name": name, "restarted_at": restartedAt}, nil
}

func (h *Handler) scaleStatefulSet(ctx context.Context, client kubernetesWorkloadClient, namespace, name string, replicas int32) (map[string]any, error) {
	statefulset, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	statefulset.Spec.Replicas = &replicas
	if _, err := client.AppsV1().StatefulSets(namespace).Update(ctx, statefulset, metav1.UpdateOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "statefulset", "name": name, "replicas": replicas}, nil
}

func (h *Handler) deleteStatefulSet(ctx context.Context, client kubernetesWorkloadClient, namespace, name string) (map[string]any, error) {
	if err := client.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "statefulset", "name": name, "deleted": true}, nil
}

func (h *Handler) deletePod(ctx context.Context, client kubernetesWorkloadClient, namespace, name string) (map[string]any, error) {
	if err := client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "pod", "name": name, "deleted": true}, nil
}

// GetDaemonSets 获取命名空间的 DaemonSet 列表。
//
// @Summary 获取 DaemonSet 列表
// @Description 获取指定命名空间的 DaemonSet 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/daemonsets [get]
func (h *Handler) GetDaemonSets(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.AppsV1().DaemonSets(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]DaemonSetInfo, 0, len(list.Items))
	for _, ds := range list.Items {
		items = append(items, DaemonSetInfo{
			Name:      ds.Name,
			Namespace: ds.Namespace,
			Desired:   ds.Status.DesiredNumberScheduled,
			Ready:     ds.Status.NumberReady,
			Age:       getAge(ds.CreationTimestamp),
			CreatedAt: ds.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetJobs 获取命名空间的 Job 列表。
//
// @Summary 获取 Job 列表
// @Description 获取指定命名空间的 Job 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/jobs [get]
func (h *Handler) GetJobs(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.BatchV1().Jobs(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]JobInfo, 0, len(list.Items))
	for _, job := range list.Items {
		completions := int32(0)
		if job.Spec.Completions != nil {
			completions = *job.Spec.Completions
		}
		status := "Running"
		if job.Status.Succeeded > 0 {
			status = "Completed"
		} else if job.Status.Failed > 0 {
			status = "Failed"
		}
		items = append(items, JobInfo{
			Name:        job.Name,
			Namespace:   job.Namespace,
			Completions: completions,
			Succeeded:   job.Status.Succeeded,
			Failed:      job.Status.Failed,
			Status:      status,
			Age:         getAge(job.CreationTimestamp),
			CreatedAt:   job.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetServices 获取命名空间的 Service 列表。
//
// @Summary 获取 Service 列表
// @Description 获取指定命名空间的 Service 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/services [get]
func (h *Handler) GetServices(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.CoreV1().Services(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]ServiceInfo, 0, len(list.Items))
	for _, svc := range list.Items {
		ports := make([]ServicePort, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			targetPort := ""
			if p.TargetPort.IntVal != 0 {
				targetPort = fmt.Sprintf("%d", p.TargetPort.IntVal)
			} else if p.TargetPort.StrVal != "" {
				targetPort = p.TargetPort.StrVal
			}
			ports = append(ports, ServicePort{
				Name:       p.Name,
				Port:       p.Port,
				TargetPort: targetPort,
				Protocol:   string(p.Protocol),
			})
		}
		items = append(items, ServiceInfo{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Type:      string(svc.Spec.Type),
			ClusterIP: svc.Spec.ClusterIP,
			Ports:     ports,
			Selector:  svc.Spec.Selector,
			Age:       getAge(svc.CreationTimestamp),
			CreatedAt: svc.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetIngresses 获取命名空间的 Ingress 列表。
//
// @Summary 获取 Ingress 列表
// @Description 获取指定命名空间的 Ingress 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/ingresses [get]
func (h *Handler) GetIngresses(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.NetworkingV1().Ingresses(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]IngressInfo, 0, len(list.Items))
	for _, ing := range list.Items {
		hosts := make([]IngressHost, 0)
		for _, rule := range ing.Spec.Rules {
			paths := make([]string, 0)
			if rule.HTTP != nil {
				for _, p := range rule.HTTP.Paths {
					paths = append(paths, p.Path)
				}
			}
			hosts = append(hosts, IngressHost{
				Host:  rule.Host,
				Paths: paths,
			})
		}
		items = append(items, IngressInfo{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Hosts:     hosts,
			Age:       getAge(ing.CreationTimestamp),
			CreatedAt: ing.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetConfigMaps 获取命名空间的 ConfigMap 列表。
//
// @Summary 获取 ConfigMap 列表
// @Description 获取指定命名空间的 ConfigMap 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/configmaps [get]
func (h *Handler) GetConfigMaps(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.CoreV1().ConfigMaps(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]ConfigMapInfo, 0, len(list.Items))
	for _, cm := range list.Items {
		keys := make([]string, 0, len(cm.Data))
		for k := range cm.Data {
			keys = append(keys, k)
		}
		items = append(items, ConfigMapInfo{
			Name:      cm.Name,
			Namespace: cm.Namespace,
			DataKeys:  keys,
			Age:       getAge(cm.CreationTimestamp),
			CreatedAt: cm.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetSecrets 获取命名空间的 Secret 元数据列表。
//
// @Summary 获取 Secret 列表
// @Description 获取指定命名空间的 Secret 元数据列表 (不含数据内容)
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/secrets [get]
func (h *Handler) GetSecrets(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.CoreV1().Secrets(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]SecretInfo, 0, len(list.Items))
	for _, sec := range list.Items {
		keys := make([]string, 0, len(sec.Data))
		for k := range sec.Data {
			keys = append(keys, k)
		}
		items = append(items, SecretInfo{
			Name:      sec.Name,
			Namespace: sec.Namespace,
			Type:      string(sec.Type),
			DataKeys:  keys,
			Age:       getAge(sec.CreationTimestamp),
			CreatedAt: sec.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetPVCs 获取命名空间的 PVC 列表。
//
// @Summary 获取 PVC 列表
// @Description 获取指定命名空间的 PersistentVolumeClaim 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/pvcs [get]
func (h *Handler) GetPVCs(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.CoreV1().PersistentVolumeClaims(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]PVCInfo, 0, len(list.Items))
	for _, pvc := range list.Items {
		capacity := ""
		if pvc.Status.Capacity != nil {
			if q, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
				capacity = q.String()
			}
		}
		accessModes := ""
		for i, am := range pvc.Spec.AccessModes {
			if i > 0 {
				accessModes += ","
			}
			accessModes += string(am)
		}
		items = append(items, PVCInfo{
			Name:         pvc.Name,
			Namespace:    pvc.Namespace,
			Status:       string(pvc.Status.Phase),
			Capacity:     capacity,
			AccessModes:  accessModes,
			StorageClass: *pvc.Spec.StorageClassName,
			VolumeName:   pvc.Spec.VolumeName,
			Age:          getAge(pvc.CreationTimestamp),
			CreatedAt:    pvc.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetPVs 获取集群的 PV 列表。
//
// @Summary 获取 PV 列表
// @Description 获取指定集群的 PersistentVolume 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/pvs [get]
func (h *Handler) GetPVs(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.CoreV1().PersistentVolumes().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]PVInfo, 0, len(list.Items))
	for _, pv := range list.Items {
		capacity := ""
		if pv.Spec.Capacity != nil {
			if q, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
				capacity = q.String()
			}
		}
		accessModes := ""
		for i, am := range pv.Spec.AccessModes {
			if i > 0 {
				accessModes += ","
			}
			accessModes += string(am)
		}
		claimRef := ""
		if pv.Spec.ClaimRef != nil {
			claimRef = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
		}
		items = append(items, PVInfo{
			Name:         pv.Name,
			Status:       string(pv.Status.Phase),
			Capacity:     capacity,
			AccessModes:  accessModes,
			StorageClass: pv.Spec.StorageClassName,
			ClaimRef:     claimRef,
			Age:          getAge(pv.CreationTimestamp),
			CreatedAt:    pv.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// Helper functions - 辅助函数

// getPodReadyAndRestarts 获取 Pod 的就绪状态和重启次数。
//
// 参数:
//   - pod: Pod 对象
//
// 返回: 就绪状态字符串 (n/m) 和重启次数
func getPodReadyAndRestarts(pod *corev1.Pod) (string, int32) {
	var ready, total int
	var restarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		total++
		if cs.Ready {
			ready++
		}
		restarts += cs.RestartCount
	}
	return fmt.Sprintf("%d/%d", ready, total), restarts
}

// getAge 计算资源年龄字符串。
//
// 参数:
//   - t: 创建时间
//
// 返回: 年龄字符串 (如: 1d, 2h, 30m)
func getAge(t metav1.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := metav1.Now().Sub(t.Time)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
