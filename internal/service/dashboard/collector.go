// Package dashboard 提供主控台相关的业务逻辑。
//
// 本文件实现主控台数据采集器，定时采集 K8s 集群资源、工作负载状态和异常 Pod。
package dashboard

import (
	"context"
	"sync"
	"time"

	prominfra "github.com/cy77cc/OpsPilot/internal/infra/prometheus"
	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Collector 主控台数据采集器。
//
// 负责定时采集 K8s 集群资源、工作负载状态和异常 Pod，并写入缓存表。
type Collector struct {
	svcCtx        *svc.ServiceContext
	db            *gorm.DB
	prometheus    prominfra.Client
	collectorOnce sync.Once
}

// NewCollector 创建主控台数据采集器。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接和 Prometheus 客户端
//
// 返回: 采集器实例
func NewCollector(svcCtx *svc.ServiceContext) *Collector {
	return &Collector{
		svcCtx:     svcCtx,
		db:         svcCtx.DB,
		prometheus: svcCtx.Prometheus,
	}
}

// Start 启动定时采集。
//
// 启动三个定时任务：
//   - 每 5 分钟采集集群资源
//   - 每 1 分钟采集工作负载状态
//   - 每 30 秒采集异常 Pod
func (c *Collector) Start() {
	c.collectorOnce.Do(func() {
		rootCtx := runtimectx.WithServices(context.Background(), c.svcCtx)
		// 首次立即采集
		ctx, cancel := context.WithTimeout(rootCtx, 60*time.Second)
		c.Collect(ctx)
		cancel()

		// 启动定时采集
		go func() {
			// 每 5 分钟采集集群资源
			resourceTicker := time.NewTicker(5 * time.Minute)
			defer resourceTicker.Stop()

			// 每 1 分钟采集工作负载状态
			workloadTicker := time.NewTicker(1 * time.Minute)
			defer workloadTicker.Stop()

			// 每 30 秒采集异常 Pod
			issuePodTicker := time.NewTicker(30 * time.Second)
			defer issuePodTicker.Stop()

			for {
				select {
				case <-resourceTicker.C:
					ctx, cancel := context.WithTimeout(rootCtx, 60*time.Second)
					c.collectClusterResources(ctx)
					cancel()
				case <-workloadTicker.C:
					ctx, cancel := context.WithTimeout(rootCtx, 60*time.Second)
					c.collectWorkloadStats(ctx)
					cancel()
				case <-issuePodTicker.C:
					ctx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
					c.collectIssuePods(ctx)
					cancel()
				}
			}
		}()

		logger.L().Info("Dashboard collector started")
	})
}

// Collect 执行一轮完整采集。
//
// 依次采集集群资源、工作负载状态和异常 Pod。
func (c *Collector) Collect(ctx context.Context) {
	c.collectClusterResources(ctx)
	c.collectWorkloadStats(ctx)
	c.collectIssuePods(ctx)
}

// collectClusterResources 采集所有集群的资源使用情况。
func (c *Collector) collectClusterResources(ctx context.Context) {
	var clusters []model.Cluster
	if err := c.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		logger.L().Warn("failed to list clusters for resource collection", logger.Error(err))
		return
	}

	for i := range clusters {
		c.collectClusterResource(ctx, &clusters[i])
	}
}

// collectClusterResource 采集单个集群的资源使用情况。
//
// 参数:
//   - ctx: 上下文
//   - cluster: 集群模型
//
// 采集内容:
//   - CPU 可分配/已请求/实际使用
//   - 内存可分配/已请求/实际使用
//   - Pod 总数/运行中/等待中/失败
func (c *Collector) collectClusterResource(ctx context.Context, cluster *model.Cluster) {
	cli, err := c.getK8sClient(cluster)
	if err != nil {
		logger.L().Debug("failed to get k8s client for cluster",
			logger.Error(err),
			logger.Int("cluster_id", int(cluster.ID)),
		)
		return
	}

	// 查询节点资源
	nodes, err := cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.L().Warn("failed to list nodes", logger.Error(err), logger.Int("cluster_id", int(cluster.ID)))
		return
	}

	var cpuAllocatable, memAllocatable float64
	for _, node := range nodes.Items {
		cpuAllocatable += float64(node.Status.Allocatable.Cpu().MilliValue()) / 1000
		memAllocatable += float64(node.Status.Allocatable.Memory().Value()) / 1024 / 1024
	}

	// 查询 Pod 资源请求
	pods, err := cli.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.L().Warn("failed to list pods", logger.Error(err), logger.Int("cluster_id", int(cluster.ID)))
		return
	}

	var cpuRequested, memRequested float64
	var runningCount, pendingCount, failedCount int
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if req := container.Resources.Requests; req != nil {
				cpuRequested += float64(req.Cpu().MilliValue()) / 1000
				memRequested += float64(req.Memory().Value()) / 1024 / 1024
			}
		}
		switch pod.Status.Phase {
		case corev1.PodRunning:
			runningCount++
		case corev1.PodPending:
			pendingCount++
		case corev1.PodFailed:
			failedCount++
		}
	}

	// 从 Prometheus 查询实际使用
	var cpuUsage, memUsage float64
	if c.prometheus != nil {
		cpuUsage = c.queryClusterCPUUsage(ctx, cluster.ID)
		memUsage = c.queryClusterMemoryUsage(ctx, cluster.ID)
	}

	// 写入快照
	snapshot := model.ClusterResourceSnapshot{
		ClusterID:           cluster.ID,
		CPUAllocatableCores: cpuAllocatable,
		CPURequestedCores:   cpuRequested,
		CPUUsageCores:       cpuUsage,
		MemoryAllocatableMB: int64(memAllocatable),
		MemoryRequestedMB:   int64(memRequested),
		MemoryUsageMB:       int64(memUsage),
		PodTotal:            len(pods.Items),
		PodRunning:          runningCount,
		PodPending:          pendingCount,
		PodFailed:           failedCount,
		CollectedAt:         time.Now().UTC(),
	}

	if err := c.db.Create(&snapshot).Error; err != nil {
		logger.L().Warn("failed to save cluster resource snapshot", logger.Error(err))
	}
}

// collectWorkloadStats 采集所有集群的工作负载状态。
func (c *Collector) collectWorkloadStats(ctx context.Context) {
	var clusters []model.Cluster
	if err := c.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		return
	}

	for i := range clusters {
		c.collectClusterWorkload(ctx, &clusters[i])
	}
}

// collectClusterWorkload 采集单个集群的工作负载状态。
//
// 采集 Deployment/StatefulSet/DaemonSet/Service/Ingress 数量和健康状态。
func (c *Collector) collectClusterWorkload(ctx context.Context, cluster *model.Cluster) {
	cli, err := c.getK8sClient(cluster)
	if err != nil {
		return
	}

	// 采集 Deployment
	deployments, _ := cli.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	var deployTotal, deployHealthy int
	for _, d := range deployments.Items {
		deployTotal++
		if d.Status.ReadyReplicas == d.Status.Replicas && d.Status.Replicas > 0 {
			deployHealthy++
		}
	}

	// 采集 StatefulSet
	statefulsets, _ := cli.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	var stsTotal, stsHealthy int
	for _, sts := range statefulsets.Items {
		stsTotal++
		if sts.Status.ReadyReplicas == sts.Status.Replicas && sts.Status.Replicas > 0 {
			stsHealthy++
		}
	}

	// 采集 DaemonSet
	daemonsets, _ := cli.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	var dsTotal, dsHealthy int
	for _, ds := range daemonsets.Items {
		dsTotal++
		if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
			dsHealthy++
		}
	}

	// 采集 Service
	services, _ := cli.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	serviceCount := len(services.Items)

	// 采集 Ingress
	ingresses, _ := cli.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	ingressCount := len(ingresses.Items)

	// 写入数据库（集群级别的聚合）
	now := time.Now().UTC()
	stats := model.K8sWorkloadStats{
		ClusterID:          cluster.ID,
		Namespace:          "",
		DeploymentTotal:    deployTotal,
		DeploymentHealthy:  deployHealthy,
		StatefulSetTotal:   stsTotal,
		StatefulSetHealthy: stsHealthy,
		DaemonSetTotal:     dsTotal,
		DaemonSetHealthy:   dsHealthy,
		ServiceCount:       serviceCount,
		IngressCount:       ingressCount,
		CollectedAt:        now,
	}
	c.db.Create(&stats)
}

// collectIssuePods 采集所有集群的异常 Pod。
func (c *Collector) collectIssuePods(ctx context.Context) {
	var clusters []model.Cluster
	if err := c.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		return
	}

	for i := range clusters {
		c.collectClusterIssuePods(ctx, &clusters[i])
	}
}

// collectClusterIssuePods 采集单个集群的异常 Pod。
//
// 检测并缓存 CrashLoopBackOff、ImagePullBackOff、OOMKilled 等异常状态的 Pod。
func (c *Collector) collectClusterIssuePods(ctx context.Context, cluster *model.Cluster) {
	cli, err := c.getK8sClient(cluster)
	if err != nil {
		return
	}

	pods, err := cli.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}

	now := time.Now().UTC()
	var issuePods []model.K8sIssuePod

	for _, pod := range pods.Items {
		issueType, issueReason, message := detectPodIssue(&pod)
		if issueType == "" {
			continue
		}

		issuePods = append(issuePods, model.K8sIssuePod{
			ClusterID:   cluster.ID,
			Namespace:   pod.Namespace,
			PodName:     pod.Name,
			IssueType:   issueType,
			IssueReason: issueReason,
			Message:     message,
			FirstSeenAt: now,
			LastSeenAt:  now,
		})
	}

	// 更新或创建异常 Pod 记录
	for _, ip := range issuePods {
		var existing model.K8sIssuePod
		err := c.db.Where("cluster_id = ? AND namespace = ? AND pod_name = ?", ip.ClusterID, ip.Namespace, ip.PodName).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			c.db.Create(&ip)
		} else if err == nil {
			c.db.Model(&existing).Updates(map[string]any{
				"issue_type":   ip.IssueType,
				"issue_reason": ip.IssueReason,
				"message":      ip.Message,
				"last_seen_at": now,
			})
		}
	}

	// 清理已恢复的异常 Pod（超过 5 分钟未更新）
	c.db.Where("cluster_id = ? AND last_seen_at < ?", cluster.ID, now.Add(-5*time.Minute)).Delete(&model.K8sIssuePod{})
}

// detectPodIssue 检测 Pod 是否有问题。
//
// 返回问题类型、原因和详细信息。如果没有问题返回空字符串。
func detectPodIssue(pod *corev1.Pod) (issueType, issueReason, message string) {
	// 检查 Pod 状态
	if pod.Status.Phase == corev1.PodFailed {
		return "Failed", string(pod.Status.Phase), "Pod is in failed state"
	}

	// 检查容器状态
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			switch cs.State.Waiting.Reason {
			case "CrashLoopBackOff":
				return model.IssueTypeCrashLoopBackOff, cs.State.Waiting.Reason, cs.State.Waiting.Message
			case "ImagePullBackOff":
				return model.IssueTypeImagePullBackOff, cs.State.Waiting.Reason, cs.State.Waiting.Message
			case "ErrImagePull":
				return model.IssueTypeErrImagePull, cs.State.Waiting.Reason, cs.State.Waiting.Message
			case "CreateContainerConfigError":
				return model.IssueTypeCreateContainerErr, cs.State.Waiting.Reason, cs.State.Waiting.Message
			}
		}
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			if cs.State.Terminated.Reason == "OOMKilled" {
				return model.IssueTypeOOMKilled, cs.State.Terminated.Reason, cs.State.Terminated.Message
			}
		}
	}

	// 检查 Pod 条件
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse {
			return model.IssueTypeUnknown, "NotReady", condition.Message
		}
	}

	return "", "", ""
}

// getK8sClient 获取集群的 K8s 客户端。
//
// 参数:
//   - cluster: 集群模型，包含 KubeConfig
//
// 返回: Kubernetes 客户端或错误
func (c *Collector) getK8sClient(cluster *model.Cluster) (*kubernetes.Clientset, error) {
	if cluster.KubeConfig == "" {
		return nil, ErrKubeConfigNotFound
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// queryClusterCPUUsage 从 Prometheus 查询集群 CPU 使用率。
//
// 返回 CPU 使用核数。
func (c *Collector) queryClusterCPUUsage(ctx context.Context, clusterID uint) float64 {
	if c.prometheus == nil {
		return 0
	}
	// 简化实现：查询节点 CPU 使用率总和
	result, err := c.prometheus.Query(ctx, `sum(rate(container_cpu_usage_seconds_total{container!=""}[5m]))`, time.Now())
	if err != nil {
		return 0
	}
	if len(result.Vector) > 0 && len(result.Vector[0].Value) >= 2 {
		if v, ok := result.Vector[0].Value[1].(float64); ok {
			return v
		}
	}
	return 0
}

// queryClusterMemoryUsage 从 Prometheus 查询集群内存使用量。
//
// 返回内存使用量（MB）。
func (c *Collector) queryClusterMemoryUsage(ctx context.Context, clusterID uint) float64 {
	if c.prometheus == nil {
		return 0
	}
	result, err := c.prometheus.Query(ctx, `sum(container_memory_working_set_bytes{container!=""})/1024/1024`, time.Now())
	if err != nil {
		return 0
	}
	if len(result.Vector) > 0 && len(result.Vector[0].Value) >= 2 {
		if v, ok := result.Vector[0].Value[1].(float64); ok {
			return v
		}
	}
	return 0
}

// ErrKubeConfigNotFound 表示未找到 KubeConfig。
var ErrKubeConfigNotFound = &KubeConfigNotFoundError{}

// KubeConfigNotFoundError KubeConfig 未找到错误。
type KubeConfigNotFoundError struct{}

func (e *KubeConfigNotFoundError) Error() string {
	return "kubeconfig not found"
}
