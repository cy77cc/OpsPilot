// Package handler 提供 Kubernetes 集群管理的 HTTP Handler 实现。
//
// 本文件实现集群资源查询和简单部署操作的 HTTP Handler，
// 包括集群列表、节点、Pod、Service、Ingress、事件和日志等查询功能。
package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	projectlogic "github.com/cy77cc/OpsPilot/internal/service/project/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Handler 集群资源 HTTP 处理器。
type Handler struct{ svcCtx *svc.ServiceContext }

// NewHandler 创建集群资源处理器。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: Handler 实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler { return &Handler{svcCtx: svcCtx} }

// List 获取集群列表。
//
// @Summary 获取集群列表
// @Description 获取所有集群信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters [get]
func (h *Handler) List(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "kubernetes:read") {
		return
	}
	var list []model.Cluster
	if err := h.svcCtx.DB.Find(&list).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// Create 创建集群。
//
// @Summary 创建集群
// @Description 创建新的 Kubernetes 集群记录
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body map[string]interface{} true "集群创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters [post]
func (h *Handler) Create(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:write", "kubernetes:write") {
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Server      string `json:"server" binding:"required"`
		Kubeconfig  string `json:"kubeconfig"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	cluster := model.Cluster{Name: req.Name, Description: req.Description, Endpoint: req.Server, KubeConfig: req.Kubeconfig, Status: "created", Type: "kubernetes", AuthMethod: "kubeconfig", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := h.svcCtx.DB.Create(&cluster).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, cluster)
}

// Get 获取集群详情。
//
// @Summary 获取集群详情
// @Description 根据 ID 获取集群信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /clusters/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	httpx.OK(c, cluster)
}

// Nodes 获取集群节点列表。
//
// @Summary 获取集群节点
// @Description 获取指定集群的所有节点信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /clusters/{id}/nodes [get]
func (h *Handler) Nodes(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, dataSource, err := h.getClient(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "total": 0, "data_source": dataSource})
		return
	}
	nodes, err := cli.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	data := make([]gin.H, 0, len(nodes.Items))
	for _, n := range nodes.Items {
		role := n.Labels["kubernetes.io/role"]
		data = append(data, gin.H{"id": n.Name, "name": n.Name, "role": role, "status": nodeReadyStatus(&n), "cpu_cores": n.Status.Capacity.Cpu().Value(), "memory": n.Status.Capacity.Memory().Value() / 1024 / 1024, "labels": n.Labels, "ip": nodeInternalIP(&n), "pods": 0})
	}
	httpx.OK(c, gin.H{"list": data, "total": len(data), "data_source": dataSource})
}

// Deployments 获取 Deployment 列表。
//
// @Summary 获取 Deployment 列表
// @Description 获取指定集群的 Deployment 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string false "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /clusters/{id}/deployments [get]
func (h *Handler) Deployments(c *gin.Context) {
	h.listDeployLike(c, "deployments")
}

// Pods 获取 Pod 列表。
//
// @Summary 获取 Pod 列表
// @Description 获取指定集群的 Pod 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string false "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /clusters/{id}/pods [get]
func (h *Handler) Pods(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, dataSource, err := h.getClient(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "data_source": dataSource})
		return
	}
	ns := c.Query("namespace")
	if ns == "" {
		ns = corev1.NamespaceAll
	}
	if ns != corev1.NamespaceAll && !h.namespaceReadable(c, cluster.ID, ns) {
		return
	}
	items, err := cli.CoreV1().Pods(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	out := make([]gin.H, 0, len(items.Items))
	for _, p := range items.Items {
		out = append(out, gin.H{"id": p.UID, "name": p.Name, "namespace": p.Namespace, "status": string(p.Status.Phase), "phase": string(p.Status.Phase), "node": p.Spec.NodeName, "restarts": totalRestarts(p.Status.ContainerStatuses), "createdAt": p.CreationTimestamp.Time, "startTime": p.Status.StartTime})
	}
	httpx.OK(c, gin.H{"list": out, "data_source": dataSource})
}

// Services 获取 Service 列表。
//
// @Summary 获取 Service 列表
// @Description 获取指定集群的 Service 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string false "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /clusters/{id}/services [get]
func (h *Handler) Services(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, dataSource, err := h.getClient(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "data_source": dataSource})
		return
	}
	ns := c.Query("namespace")
	if ns == "" {
		ns = corev1.NamespaceAll
	}
	if ns != corev1.NamespaceAll && !h.namespaceReadable(c, cluster.ID, ns) {
		return
	}
	items, err := cli.CoreV1().Services(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	out := make([]gin.H, 0, len(items.Items))
	for _, s := range items.Items {
		ports := make([]gin.H, 0, len(s.Spec.Ports))
		for _, p := range s.Spec.Ports {
			ports = append(ports, gin.H{"port": p.Port, "targetPort": p.TargetPort.IntVal})
		}
		out = append(out, gin.H{"id": s.UID, "name": s.Name, "namespace": s.Namespace, "type": string(s.Spec.Type), "cluster_ip": s.Spec.ClusterIP, "ports": ports})
	}
	httpx.OK(c, gin.H{"list": out, "data_source": dataSource})
}

// Ingresses 获取 Ingress 列表。
//
// @Summary 获取 Ingress 列表
// @Description 获取指定集群的 Ingress 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string false "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /clusters/{id}/ingresses [get]
func (h *Handler) Ingresses(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, dataSource, err := h.getClient(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "data_source": dataSource})
		return
	}
	ns := c.Query("namespace")
	if ns == "" {
		ns = corev1.NamespaceAll
	}
	if ns != corev1.NamespaceAll && !h.namespaceReadable(c, cluster.ID, ns) {
		return
	}
	items, err := cli.NetworkingV1().Ingresses(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	out := make([]gin.H, 0)
	for _, ing := range items.Items {
		out = append(out, mapIngress(ing)...)
	}
	httpx.OK(c, gin.H{"list": out, "data_source": dataSource})
}

// Events 获取集群事件列表。
//
// @Summary 获取集群事件
// @Description 获取指定集群的事件列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string false "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /clusters/{id}/events [get]
func (h *Handler) Events(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, dataSource, err := h.getClient(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "data_source": dataSource})
		return
	}
	ns := c.Query("namespace")
	if ns == "" {
		ns = corev1.NamespaceAll
	}
	if ns != corev1.NamespaceAll && !h.namespaceReadable(c, cluster.ID, ns) {
		return
	}
	items, err := cli.CoreV1().Events(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	out := make([]gin.H, 0, len(items.Items))
	for _, e := range items.Items {
		out = append(out, gin.H{"type": e.Type, "reason": e.Reason, "message": e.Message, "time": e.LastTimestamp})
	}
	httpx.OK(c, gin.H{"list": out, "data_source": dataSource})
}

// Logs 获取 Pod 日志。
//
// @Summary 获取 Pod 日志
// @Description 获取指定 Pod 的容器日志
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string true "命名空间"
// @Param pod query string true "Pod 名称"
// @Param container query string false "容器名称"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /clusters/{id}/logs [get]
func (h *Handler) Logs(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, _, err := h.getClient(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"logs": "cluster client unavailable"})
		return
	}
	ns := c.Query("namespace")
	if ns == "" {
		ns = "default"
	}
	if !h.namespaceReadable(c, cluster.ID, ns) {
		return
	}
	pod := c.Query("pod")
	if pod == "" {
		httpx.Fail(c, xcode.ParamError, "pod required")
		return
	}
	req := cli.CoreV1().Pods(ns).GetLogs(pod, &corev1.PodLogOptions{Container: c.Query("container"), TailLines: int64Ptr(200)})
	buf, err := req.DoRaw(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"logs": string(buf)})
}

// ConnectTest 测试集群连通性。
//
// @Summary 测试集群连通性
// @Description 测试与集群 API Server 的连通性
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /clusters/{id}/connect-test [get]
func (h *Handler) ConnectTest(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read", "kubernetes:read") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	start := time.Now()
	cli, dataSource, err := h.getClient(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"connected": false, "message": err.Error(), "data_source": dataSource})
		return
	}
	_, err = cli.Discovery().ServerVersion()
	if err != nil {
		httpx.OK(c, gin.H{"connected": false, "message": err.Error(), "data_source": dataSource})
		return
	}
	httpx.OK(c, gin.H{"connected": true, "latency_ms": time.Since(start).Milliseconds(), "data_source": dataSource})
}

// DeployPreview 预览部署配置。
//
// @Summary 预览部署
// @Description 预览 Deployment YAML 清单
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body map[string]interface{} true "部署请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /clusters/{id}/deploy/preview [post]
func (h *Handler) DeployPreview(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:deploy", "kubernetes:write") {
		return
	}
	var req struct {
		Namespace string `json:"namespace" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Image     string `json:"image" binding:"required"`
		Replicas  int32  `json:"replicas"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if req.Replicas <= 0 {
		req.Replicas = 1
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	if !h.namespaceWritable(c, cluster.ID, req.Namespace) {
		return
	}
	httpx.OK(c, gin.H{"summary": "preview only", "manifest": gin.H{"namespace": req.Namespace, "name": req.Name, "image": req.Image, "replicas": req.Replicas}})
}

// DeployApply 应用部署配置。
//
// @Summary 应用部署
// @Description 创建或更新 Deployment
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body map[string]interface{} true "部署请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/deploy/apply [post]
func (h *Handler) DeployApply(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:deploy", "kubernetes:write") {
		return
	}
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	var req struct {
		Namespace     string `json:"namespace" binding:"required"`
		Name          string `json:"name" binding:"required"`
		Image         string `json:"image" binding:"required"`
		Replicas      int32  `json:"replicas"`
		ApprovalToken string `json:"approval_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if req.Replicas <= 0 {
		req.Replicas = 1
	}
	if !h.namespaceWritable(c, cluster.ID, req.Namespace) {
		return
	}
	if !h.requireProdApproval(c, cluster.ID, req.Namespace, "deploy", req.ApprovalToken) {
		return
	}
	yaml := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s
`, req.Name, req.Namespace, req.Replicas, req.Name, req.Name, req.Name, req.Image)
	if err := projectlogic.DeployToCluster(c.Request.Context(), cluster, yaml); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	h.createAudit(cluster.ID, req.Namespace, "deploy.apply", "deployment", req.Name, "success", "legacy deployment applied", uint(httpx.UIDFromCtx(c)))
	httpx.OK(c, gin.H{"applied": true})
}

// mustCluster 获取集群模型，失败时输出错误响应。
//
// 参数:
//   - c: Gin 上下文
//
// 返回: 集群模型和成功标志
func (h *Handler) mustCluster(c *gin.Context) (*model.Cluster, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return nil, false
	}
	var cluster model.Cluster
	if err := h.svcCtx.DB.First(&cluster, id).Error; err != nil {
		httpx.Fail(c, xcode.NotFound, "cluster not found")
		return nil, false
	}
	return &cluster, true
}

// getClient 获取集群的 Kubernetes 客户端。
//
// 参数:
//   - cluster: 集群模型
//
// 返回: 客户端、数据源标识和错误
func (h *Handler) getClient(cluster *model.Cluster) (*kubernetes.Clientset, string, error) {
	if cluster != nil && strings.TrimSpace(cluster.KubeConfig) != "" {
		cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
		if err != nil {
			return nil, "db", err
		}
		cli, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, "db", err
		}
		return cli, "live", nil
	}
	return nil, "none", fmt.Errorf("kubernetes client unavailable")
}

// mapIngress 转换 Ingress 为响应格式。
//
// 参数:
//   - ing: Ingress 对象
//
// 返回: 响应数据列表
func mapIngress(ing networkingv1.Ingress) []gin.H {
	out := make([]gin.H, 0)
	if len(ing.Spec.Rules) == 0 {
		return []gin.H{{"id": ing.UID, "name": ing.Name, "namespace": ing.Namespace, "host": "", "path": "/", "service": "", "tls": len(ing.Spec.TLS) > 0}}
	}
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil || len(rule.HTTP.Paths) == 0 {
			out = append(out, gin.H{"id": ing.UID, "name": ing.Name, "namespace": ing.Namespace, "host": rule.Host, "path": "/", "service": "", "tls": len(ing.Spec.TLS) > 0})
			continue
		}
		for _, p := range rule.HTTP.Paths {
			svc := ""
			if p.Backend.Service != nil {
				svc = p.Backend.Service.Name
			}
			out = append(out, gin.H{"id": ing.UID, "name": ing.Name, "namespace": ing.Namespace, "host": rule.Host, "path": p.Path, "service": svc, "tls": len(ing.Spec.TLS) > 0})
		}
	}
	return out
}

// nodeReadyStatus 获取节点就绪状态。
//
// 参数:
//   - n: Node 对象
//
// 返回: 状态字符串 (online/offline/unknown)
func nodeReadyStatus(n *corev1.Node) string {
	for _, cond := range n.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionTrue {
				return "online"
			}
			return "offline"
		}
	}
	return "unknown"
}

// nodeInternalIP 获取节点内网 IP。
//
// 参数:
//   - n: Node 对象
//
// 返回: 内网 IP 字符串
func nodeInternalIP(n *corev1.Node) string {
	for _, addr := range n.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

// totalRestarts 计算容器总重启次数。
//
// 参数:
//   - statuses: 容器状态列表
//
// 返回: 总重启次数
func totalRestarts(statuses []corev1.ContainerStatus) int32 {
	var total int32
	for _, st := range statuses {
		total += st.RestartCount
	}
	return total
}

// int64Ptr 创建 int64 指针。
func int64Ptr(v int64) *int64 { return &v }

// listDeployLike 查询 Deployment 类资源列表。
//
// 参数:
//   - c: Gin 上下文
//   - _: 资源类型 (保留参数)
func (h *Handler) listDeployLike(c *gin.Context, _ string) {
	cluster, ok := h.mustCluster(c)
	if !ok {
		return
	}
	cli, dataSource, err := h.getClient(cluster)
	if err != nil {
		httpx.OK(c, gin.H{"list": []any{}, "data_source": dataSource})
		return
	}
	ns := c.Query("namespace")
	if ns == "" {
		ns = corev1.NamespaceAll
	}
	items, err := cli.AppsV1().Deployments(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	out := make([]gin.H, 0, len(items.Items))
	for _, d := range items.Items {
		image := ""
		if len(d.Spec.Template.Spec.Containers) > 0 {
			image = d.Spec.Template.Spec.Containers[0].Image
		}
		status := "syncing"
		if d.Status.AvailableReplicas == d.Status.Replicas && d.Status.Replicas > 0 {
			status = "running"
		}
		if d.Status.Replicas == 0 {
			status = "stopped"
		}
		out = append(out, gin.H{"id": d.UID, "namespace": d.Namespace, "name": d.Name, "image": image, "replicas": d.Status.ReadyReplicas, "status": status})
	}
	httpx.OK(c, gin.H{"list": out, "data_source": dataSource})
}
