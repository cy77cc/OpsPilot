// Package kubernetes 提供 Kubernetes 集群操作相关的工具实现。
//
// 本文件实现 K8s 操作工具集，包括：
//   - 资源查询（Pod、Service、Deployment、Node）
//   - 事件检查
//   - 日志获取
package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// =============================================================================
// 输入类型定义
// =============================================================================

// K8sListInput 资源列表查询输入。
type K8sListInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Resource  string `json:"resource" jsonschema_description:"required,resource type,enum=pods,enum=services,enum=deployments,enum=nodes"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max items,default=50"`
}

// K8sQueryInput 资源查询输入，支持按名称和标签过滤。
type K8sQueryInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Resource  string `json:"resource" jsonschema_description:"required,resource type,enum=pods,enum=services,enum=deployments,enum=nodes"`
	Name      string `json:"name,omitempty" jsonschema_description:"resource name for exact lookup"`
	Label     string `json:"label,omitempty" jsonschema_description:"label selector"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max items,default=50"`
}

// K8sEventsInput 事件查询输入。
type K8sEventsInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max events,default=50"`
}

// K8sEventsQueryInput 事件查询输入，支持按对象过滤。
type K8sEventsQueryInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Kind      string `json:"kind,omitempty" jsonschema_description:"involved object kind,enum=Pod,enum=Deployment,enum=Service,enum=Node"`
	Name      string `json:"name,omitempty" jsonschema_description:"involved object name"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max events,default=50"`
}

// K8sPodLogsInput Pod 日志查询输入。
type K8sPodLogsInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Pod       string `json:"pod" jsonschema_description:"required,pod name"`
	Container string `json:"container,omitempty" jsonschema_description:"container name"`
	TailLines int    `json:"tail_lines,omitempty" jsonschema_description:"tail lines,default=200"`
}

// K8sLogsInput 日志查询输入。
type K8sLogsInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Pod       string `json:"pod" jsonschema_description:"required,pod name"`
	Container string `json:"container,omitempty" jsonschema_description:"container name"`
	TailLines int    `json:"tail_lines,omitempty" jsonschema_description:"tail lines,default=200"`
}

// NewKubernetesTools 创建所有 Kubernetes 工具。
func NewKubernetesTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{
		K8sQuery(ctx),
		K8sListResources(ctx),
		K8sEvents(ctx),
		K8sGetEvents(ctx),
		K8sLogs(ctx),
		K8sGetPodLogs(ctx),
	}
}

type K8sQueryOutput struct {
	Items []map[string]any `json:"items"`
}

func K8sQuery(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_query",
		"Query Kubernetes resources with filtering options. resource is required and specifies the resource type (pods/services/deployments/nodes). Optional parameters: cluster_id targets a specific cluster, namespace limits scope (default: all namespaces), name filters by exact name, label uses label selector, limit caps results (default 50). Returns resource details with status and metadata. Example: {\"resource\":\"pods\",\"namespace\":\"default\",\"label\":\"app=nginx\"}.",
		func(ctx context.Context, input *K8sQueryInput, opts ...tool.Option) (*K8sQueryOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context is unavailable")
			}
			if strings.TrimSpace(input.Resource) == "" {
				return nil, fmt.Errorf("resource is required")
			}
			cli, _, err := resolveK8sClient(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}
			ns := strings.TrimSpace(input.Namespace)
			if ns == "" {
				ns = corev1.NamespaceAll
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			name := strings.TrimSpace(input.Name)
			label := strings.TrimSpace(input.Label)
			resource := strings.ToLower(strings.TrimSpace(input.Resource))
			listOpts := metav1.ListOptions{}
			if label != "" {
				listOpts.LabelSelector = label
			}

			switch resource {
			case "pods":
				list, err := cli.CoreV1().Pods(ns).List(ctx, listOpts)
				if err != nil {
					return nil, err
				}
				out := make([]map[string]any, 0, len(list.Items))
				for _, p := range list.Items {
					if name != "" && p.Name != name {
						continue
					}
					out = append(out, map[string]any{
						"name":      p.Name,
						"namespace": p.Namespace,
						"phase":     p.Status.Phase,
						"node":      p.Spec.NodeName,
						"labels":    p.Labels,
					})
					if len(out) >= limit {
						break
					}
				}
				return &K8sQueryOutput{Items: out}, nil
			case "services":
				list, err := cli.CoreV1().Services(ns).List(ctx, listOpts)
				if err != nil {
					return nil, err
				}
				out := make([]map[string]any, 0, len(list.Items))
				for _, s := range list.Items {
					if name != "" && s.Name != name {
						continue
					}
					out = append(out, map[string]any{
						"name":       s.Name,
						"namespace":  s.Namespace,
						"type":       s.Spec.Type,
						"cluster_ip": s.Spec.ClusterIP,
						"labels":     s.Labels,
					})
					if len(out) >= limit {
						break
					}
				}
				return &K8sQueryOutput{Items: out}, nil
			case "deployments":
				list, err := cli.AppsV1().Deployments(ns).List(ctx, listOpts)
				if err != nil {
					return nil, err
				}
				out := make([]map[string]any, 0, len(list.Items))
				for _, d := range list.Items {
					if name != "" && d.Name != name {
						continue
					}
					out = append(out, map[string]any{
						"name":      d.Name,
						"namespace": d.Namespace,
						"ready":     d.Status.ReadyReplicas,
						"replicas":  d.Status.Replicas,
						"updated":   d.Status.UpdatedReplicas,
						"labels":    d.Labels,
					})
					if len(out) >= limit {
						break
					}
				}
				return &K8sQueryOutput{Items: out}, nil
			case "nodes":
				list, err := cli.CoreV1().Nodes().List(ctx, listOpts)
				if err != nil {
					return nil, err
				}
				out := make([]map[string]any, 0, len(list.Items))
				for _, n := range list.Items {
					if name != "" && n.Name != name {
						continue
					}
					out = append(out, map[string]any{
						"name":   n.Name,
						"labels": n.Labels,
					})
					if len(out) >= limit {
						break
					}
				}
				return &K8sQueryOutput{Items: out}, nil
			default:
				return nil, fmt.Errorf("unsupported resource type: %s", resource)
			}
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type K8sListResourcesOutput struct {
	Items []map[string]any `json:"items"`
}

func K8sListResources(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_list_resources",
		"List Kubernetes resources of a specific type. resource is required and must be one of: pods, services, deployments, nodes. Optional parameters: cluster_id targets a specific cluster, namespace limits scope (default: all namespaces), limit caps results (default 50). Returns a simplified list of resources with basic information. Example: {\"resource\":\"pods\",\"namespace\":\"kube-system\",\"limit\":20}.",
		func(ctx context.Context, input *K8sListInput, opts ...tool.Option) (*K8sListResourcesOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context is unavailable")
			}
			if strings.TrimSpace(input.Resource) == "" {
				return nil, fmt.Errorf("resource is required")
			}
			cli, _, err := resolveK8sClient(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}
			ns := strings.TrimSpace(input.Namespace)
			if ns == "" {
				ns = corev1.NamespaceAll
			}
			resource := strings.ToLower(strings.TrimSpace(input.Resource))
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			switch resource {
			case "pods":
				list, err := cli.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					return nil, err
				}
				out := make([]map[string]any, 0, len(list.Items))
				for i, p := range list.Items {
					if i >= limit {
						break
					}
					out = append(out, map[string]any{"name": p.Name, "namespace": p.Namespace, "phase": p.Status.Phase})
				}
				return &K8sListResourcesOutput{Items: out}, nil
			case "services":
				list, err := cli.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					return nil, err
				}
				out := make([]map[string]any, 0, len(list.Items))
				for i, s := range list.Items {
					if i >= limit {
						break
					}
					out = append(out, map[string]any{"name": s.Name, "namespace": s.Namespace, "type": s.Spec.Type})
				}
				return &K8sListResourcesOutput{Items: out}, nil
			case "deployments":
				list, err := cli.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					return nil, err
				}
				out := make([]map[string]any, 0, len(list.Items))
				for i, d := range list.Items {
					if i >= limit {
						break
					}
					out = append(out, map[string]any{"name": d.Name, "namespace": d.Namespace, "ready": d.Status.ReadyReplicas, "replicas": d.Status.Replicas})
				}
				return &K8sListResourcesOutput{Items: out}, nil
			case "nodes":
				list, err := cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				if err != nil {
					return nil, err
				}
				out := make([]map[string]any, 0, len(list.Items))
				for i, n := range list.Items {
					if i >= limit {
						break
					}
					out = append(out, map[string]any{"name": n.Name, "labels": n.Labels})
				}
				return &K8sListResourcesOutput{Items: out}, nil
			default:
				return nil, fmt.Errorf("unsupported resource type: %s", resource)
			}
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type K8sEventsOutput struct {
	Items []map[string]any `json:"items"`
}

func K8sEvents(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_events",
		"Query Kubernetes events with optional filtering. Optional parameters: cluster_id targets a specific cluster, namespace limits scope (default: all namespaces), kind filters by involved object kind (Pod/Deployment/Service/Node), name filters by object name, limit caps results (default 50). Returns events with type, reason, message, and involved object info. Example: {\"namespace\":\"default\",\"kind\":\"Pod\",\"limit\":20}.",
		func(ctx context.Context, input *K8sEventsQueryInput, opts ...tool.Option) (*K8sEventsOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context is unavailable")
			}
			cli, _, err := resolveK8sClient(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}
			ns := strings.TrimSpace(input.Namespace)
			if ns == "" {
				ns = corev1.NamespaceAll
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			kind := strings.TrimSpace(input.Kind)
			name := strings.TrimSpace(input.Name)
			list, err := cli.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, err
			}
			out := make([]map[string]any, 0, len(list.Items))
			for _, e := range list.Items {
				if kind != "" && !strings.EqualFold(e.InvolvedObject.Kind, kind) {
					continue
				}
				if name != "" && e.InvolvedObject.Name != name {
					continue
				}
				out = append(out, map[string]any{
					"type":      e.Type,
					"reason":    e.Reason,
					"message":   e.Message,
					"namespace": e.Namespace,
					"kind":      e.InvolvedObject.Kind,
					"name":      e.InvolvedObject.Name,
				})
				if len(out) >= limit {
					break
				}
			}
			return &K8sEventsOutput{Items: out}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type K8sGetEventsOutput struct {
	Items []map[string]any `json:"items"`
}

func K8sGetEvents(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_get_events",
		"Get Kubernetes events from a namespace. Optional parameters: cluster_id targets a specific cluster, namespace limits scope (default: all namespaces), limit caps results (default 50). Returns events with type, reason, and message. Use this for a quick event overview. Example: {\"namespace\":\"default\",\"limit\":30}.",
		func(ctx context.Context, input *K8sEventsInput, opts ...tool.Option) (*K8sGetEventsOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context unavailable")
			}
			cli, _, err := resolveK8sClient(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}
			ns := strings.TrimSpace(input.Namespace)
			if ns == "" {
				ns = corev1.NamespaceAll
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			list, err := cli.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, err
			}
			out := make([]map[string]any, 0, len(list.Items))
			for i, e := range list.Items {
				if i >= limit {
					break
				}
				out = append(out, map[string]any{"type": e.Type, "reason": e.Reason, "message": e.Message})
			}
			return &K8sGetEventsOutput{Items: out}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type K8sLogsOutput struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
	Logs      string `json:"logs"`
}

func K8sLogs(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_logs",
		"Get logs from a Kubernetes pod. pod is required. Optional parameters: cluster_id targets a specific cluster, namespace (default: default), container specifies which container in a multi-container pod, tail_lines limits log lines (default 200). Returns pod logs as a string. Example: {\"namespace\":\"default\",\"pod\":\"nginx-abc123\",\"tail_lines\":100}.",
		func(ctx context.Context, input *K8sLogsInput, opts ...tool.Option) (*K8sLogsOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context is unavailable")
			}
			cli, _, err := resolveK8sClient(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}
			ns := strings.TrimSpace(input.Namespace)
			if ns == "" {
				ns = "default"
			}
			pod := strings.TrimSpace(input.Pod)
			if pod == "" {
				return nil, fmt.Errorf("pod is required")
			}
			tailLines := int64(input.TailLines)
			if tailLines <= 0 {
				tailLines = 200
			}
			opt := &corev1.PodLogOptions{Container: strings.TrimSpace(input.Container), TailLines: &tailLines}
			raw, err := cli.CoreV1().Pods(ns).GetLogs(pod, opt).DoRaw(ctx)
			if err != nil {
				return nil, err
			}
			return &K8sLogsOutput{
				Namespace: ns,
				Pod:       pod,
				Container: strings.TrimSpace(input.Container),
				Logs:      string(raw),
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type K8sGetPodLogsOutput struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Logs      string `json:"logs"`
}

func K8sGetPodLogs(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_get_pod_logs",
		"Get logs from a specific Kubernetes pod. pod is required. Optional parameters: cluster_id targets a specific cluster, namespace (default: default), container for multi-container pods, tail_lines limits output (default 200). Returns pod logs for debugging and troubleshooting. Example: {\"namespace\":\"production\",\"pod\":\"api-server-xyz789\",\"tail_lines\":500}.",
		func(ctx context.Context, input *K8sPodLogsInput, opts ...tool.Option) (*K8sGetPodLogsOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context is unavailable")
			}
			cli, _, err := resolveK8sClient(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}
			ns := strings.TrimSpace(input.Namespace)
			if ns == "" {
				ns = "default"
			}
			pod := strings.TrimSpace(input.Pod)
			if pod == "" {
				return nil, fmt.Errorf("pod is required")
			}
			tailLines := int64(input.TailLines)
			if tailLines <= 0 {
				tailLines = 200
			}
			opt := &corev1.PodLogOptions{Container: strings.TrimSpace(input.Container), TailLines: &tailLines}
			raw, err := cli.CoreV1().Pods(ns).GetLogs(pod, opt).DoRaw(ctx)
			if err != nil {
				return nil, err
			}
			return &K8sGetPodLogsOutput{
				Namespace: ns,
				Pod:       pod,
				Logs:      string(raw),
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

// resolveK8sClient 解析 Kubernetes 客户端，根据参数和依赖项选择合适的客户端。
//
// 参数:
//   - deps: 平台依赖项，包含数据库连接和默认客户端等
//   - params: 参数字典，可能包含 cluster_id 等信息
//
// 返回值:
//   - *kubernetes.Clientset: Kubernetes 客户端实例
//   - string: 客户端来源标识
//   - error: 解析过程中的错误
func resolveK8sClient(svcCtx *svc.ServiceContext, clusterID int) (*kubernetes.Clientset, string, error) {

	if clusterID <= 0 {
		return nil, "missing_cluster_id", fmt.Errorf("k8s client unavailable: cluster_id is required")
	}

	// 首先尝试从数据库中获取指定集群的客户端
	if svcCtx.DB != nil {
		var cluster model.Cluster
		// 从数据库中查询集群信息
		if err := svcCtx.DB.First(&cluster, clusterID).Error; err == nil && strings.TrimSpace(cluster.KubeConfig) != "" {
			// 使用集群的 KubeConfig 创建 REST 配置
			cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
			if err != nil {
				return nil, "cluster_kubeconfig", err
			}

			// 使用 REST 配置创建 Kubernetes 客户端
			cli, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return nil, "cluster_kubeconfig", err
			}

			// 返回从集群配置创建的客户端
			return cli, "cluster_kubeconfig", nil
		}
	}

	// 如果所有尝试都失败，返回错误
	return nil, "fallback", fmt.Errorf("k8s client unavailable: cluster %d has no usable kubeconfig or db access", clusterID)
}
