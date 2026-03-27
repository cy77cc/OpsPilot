// Package cluster 提供集群管理相关的业务逻辑。
//
// 本文件实现集群状态指标采集器，定时采集 K8s 集群的节点和 Pod 状态，
// 并推送到 Prometheus 用于监控和告警。
package cluster

import (
	"context"
	"sync"
	"time"

	prominfra "github.com/cy77cc/OpsPilot/internal/infra/prometheus"
	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Collector 集群状态指标采集器。
type Collector struct {
	svcCtx        *svc.ServiceContext
	pusher        *prominfra.MetricsPusher
	collectorOnce sync.Once
}

// NewCollector 创建集群指标采集器。
func NewCollector(svcCtx *svc.ServiceContext) *Collector {
	return &Collector{
		svcCtx: svcCtx,
		pusher: svcCtx.MetricsPusher,
	}
}

// Start 启动定时采集。
func (c *Collector) Start() {
	if c.pusher == nil {
		logger.L().Warn("MetricsPusher is nil, cluster metrics collection disabled")
		return
	}

	c.collectorOnce.Do(func() {
		rootCtx := runtimectx.WithServices(context.Background(), c.svcCtx)
		// 首次立即采集
		ctx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
		c.Collect(ctx)
		cancel()

		// 启动定时采集
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for {
				<-ticker.C
				ctx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
				c.Collect(ctx)
				cancel()
			}
		}()

		logger.L().Info("Cluster metrics collector started")
	})
}

// Collect 执行一轮采集，遍历所有集群并采集指标。
func (c *Collector) Collect(ctx context.Context) {
	var clusters []model.Cluster
	if err := c.svcCtx.DB.WithContext(ctx).Find(&clusters).Error; err != nil {
		logger.L().Warn("failed to list clusters for metrics collection", logger.Error(err))
		return
	}

	if len(clusters) == 0 {
		return
	}

	for i := range clusters {
		c.collectClusterMetrics(ctx, &clusters[i])
	}
}

// collectClusterMetrics 采集单个集群的指标。
func (c *Collector) collectClusterMetrics(ctx context.Context, cluster *model.Cluster) {
	// 获取 K8s 客户端
	cli, err := c.getK8sClient(cluster)
	if err != nil {
		logger.L().Warn("failed to get k8s client for cluster",
			logger.Error(err),
			logger.Int("cluster_id", int(cluster.ID)),
			logger.String("cluster_name", cluster.Name),
		)
		return
	}

	// 采集节点状态
	nodes, err := cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.L().Warn("failed to list nodes for cluster",
			logger.Error(err),
			logger.Int("cluster_id", int(cluster.ID)),
		)
		return
	}

	var readyCount, notReadyCount int
	for _, node := range nodes.Items {
		if isNodeReady(&node) {
			readyCount++
		} else {
			notReadyCount++
		}
	}

	// 采集 Pod 状态
	pods, err := cli.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.L().Warn("failed to list pods for cluster",
			logger.Error(err),
			logger.Int("cluster_id", int(cluster.ID)),
		)
		// 继续推送节点指标，不因为 Pod 查询失败而中断
	}

	var runningCount, pendingCount, failedCount int
	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			runningCount++
		case corev1.PodPending:
			pendingCount++
		case corev1.PodFailed:
			failedCount++
		}
	}

	// 推送指标
	snapshot := prominfra.ClusterMetricSnapshot{
		ClusterID:    uint64(cluster.ID),
		ClusterName:  cluster.Name,
		NodeTotal:    len(nodes.Items),
		NodeReady:    readyCount,
		NodeNotReady: notReadyCount,
		PodTotal:     len(pods.Items),
		PodRunning:   runningCount,
		PodPending:   pendingCount,
		PodFailed:    failedCount,
	}

	if err := c.pusher.PushClusterMetrics(ctx, snapshot); err != nil {
		logger.L().Warn("failed to push cluster metrics to prometheus",
			logger.Error(err),
			logger.Int("cluster_id", int(cluster.ID)),
			logger.String("cluster_name", cluster.Name),
		)
	}
}

// getK8sClient 获取集群的 K8s 客户端。
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

// isNodeReady 检查节点是否处于 Ready 状态。
func isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// ErrKubeConfigNotFound 表示未找到 KubeConfig。
var ErrKubeConfigNotFound = &KubeConfigNotFoundError{}

// KubeConfigNotFoundError KubeConfig 未找到错误。
type KubeConfigNotFoundError struct{}

func (e *KubeConfigNotFoundError) Error() string {
	return "kubeconfig not found"
}
