package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/core"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Input types

type K8sListInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Resource  string `json:"resource" jsonschema_description:"required,resource type,enum=pods,enum=services,enum=deployments,enum=nodes"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max items,default=50"`
}

type K8sQueryInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Resource  string `json:"resource" jsonschema_description:"required,resource type,enum=pods,enum=services,enum=deployments,enum=nodes"`
	Name      string `json:"name,omitempty" jsonschema_description:"resource name for exact lookup"`
	Label     string `json:"label,omitempty" jsonschema_description:"label selector"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max items,default=50"`
}

type K8sEventsInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max events,default=50"`
}

type K8sEventsQueryInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Kind      string `json:"kind,omitempty" jsonschema_description:"involved object kind,enum=Pod,enum=Deployment,enum=Service,enum=Node"`
	Name      string `json:"name,omitempty" jsonschema_description:"involved object name"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max events,default=50"`
}

type K8sPodLogsInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Pod       string `json:"pod" jsonschema_description:"required,pod name"`
	Container string `json:"container,omitempty" jsonschema_description:"container name"`
	TailLines int    `json:"tail_lines,omitempty" jsonschema_description:"tail lines,default=200"`
}

type K8sLogsInput struct {
	ClusterID int    `json:"cluster_id,omitempty" jsonschema_description:"cluster id in database"`
	Namespace string `json:"namespace,omitempty" jsonschema_description:"kubernetes namespace,default=default"`
	Pod       string `json:"pod" jsonschema_description:"required,pod name"`
	Container string `json:"container,omitempty" jsonschema_description:"container name"`
	TailLines int    `json:"tail_lines,omitempty" jsonschema_description:"tail lines,default=200"`
}

// NewKubernetesTools returns all kubernetes tools.
func NewKubernetesTools(ctx context.Context, deps core.PlatformDeps) []tool.InvokableTool {
	return []tool.InvokableTool{
		K8sQuery(ctx, deps),
		K8sListResources(ctx, deps),
		K8sEvents(ctx, deps),
		K8sGetEvents(ctx, deps),
		K8sLogs(ctx, deps),
		K8sGetPodLogs(ctx, deps),
	}
}

type K8sQueryOutput struct {
	Items []map[string]any `json:"items"`
}

func K8sQuery(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_query",
		"Query Kubernetes resources with filtering options. resource is required and specifies the resource type (pods/services/deployments/nodes). Optional parameters: cluster_id targets a specific cluster, namespace limits scope (default: all namespaces), name filters by exact name, label uses label selector, limit caps results (default 50). Returns resource details with status and metadata. Example: {\"resource\":\"pods\",\"namespace\":\"default\",\"label\":\"app=nginx\"}.",
		func(ctx context.Context, input *K8sQueryInput, opts ...tool.Option) (*K8sQueryOutput, error) {
			if strings.TrimSpace(input.Resource) == "" {
				return nil, fmt.Errorf("resource is required")
			}
			cli, _, err := core.ResolveK8sClient(deps, core.StructToMap(input))
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

func K8sListResources(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_list_resources",
		"List Kubernetes resources of a specific type. resource is required and must be one of: pods, services, deployments, nodes. Optional parameters: cluster_id targets a specific cluster, namespace limits scope (default: all namespaces), limit caps results (default 50). Returns a simplified list of resources with basic information. Example: {\"resource\":\"pods\",\"namespace\":\"kube-system\",\"limit\":20}.",
		func(ctx context.Context, input *K8sListInput, opts ...tool.Option) (*K8sListResourcesOutput, error) {
			if strings.TrimSpace(input.Resource) == "" {
				return nil, fmt.Errorf("resource is required")
			}
			cli, _, err := core.ResolveK8sClient(deps, core.StructToMap(input))
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

func K8sEvents(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_events",
		"Query Kubernetes events with optional filtering. Optional parameters: cluster_id targets a specific cluster, namespace limits scope (default: all namespaces), kind filters by involved object kind (Pod/Deployment/Service/Node), name filters by object name, limit caps results (default 50). Returns events with type, reason, message, and involved object info. Example: {\"namespace\":\"default\",\"kind\":\"Pod\",\"limit\":20}.",
		func(ctx context.Context, input *K8sEventsQueryInput, opts ...tool.Option) (*K8sEventsOutput, error) {
			cli, _, err := core.ResolveK8sClient(deps, core.StructToMap(input))
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

func K8sGetEvents(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_get_events",
		"Get Kubernetes events from a namespace. Optional parameters: cluster_id targets a specific cluster, namespace limits scope (default: all namespaces), limit caps results (default 50). Returns events with type, reason, and message. Use this for a quick event overview. Example: {\"namespace\":\"default\",\"limit\":30}.",
		func(ctx context.Context, input *K8sEventsInput, opts ...tool.Option) (*K8sGetEventsOutput, error) {
			cli, _, err := core.ResolveK8sClient(deps, core.StructToMap(input))
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

func K8sLogs(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_logs",
		"Get logs from a Kubernetes pod. pod is required. Optional parameters: cluster_id targets a specific cluster, namespace (default: default), container specifies which container in a multi-container pod, tail_lines limits log lines (default 200). Returns pod logs as a string. Example: {\"namespace\":\"default\",\"pod\":\"nginx-abc123\",\"tail_lines\":100}.",
		func(ctx context.Context, input *K8sLogsInput, opts ...tool.Option) (*K8sLogsOutput, error) {
			cli, _, err := core.ResolveK8sClient(deps, core.StructToMap(input))
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

func K8sGetPodLogs(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_get_pod_logs",
		"Get logs from a specific Kubernetes pod. pod is required. Optional parameters: cluster_id targets a specific cluster, namespace (default: default), container for multi-container pods, tail_lines limits output (default 200). Returns pod logs for debugging and troubleshooting. Example: {\"namespace\":\"production\",\"pod\":\"api-server-xyz789\",\"tail_lines\":500}.",
		func(ctx context.Context, input *K8sPodLogsInput, opts ...tool.Option) (*K8sGetPodLogsOutput, error) {
			cli, _, err := core.ResolveK8sClient(deps, core.StructToMap(input))
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
