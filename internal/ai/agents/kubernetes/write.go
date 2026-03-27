// Package kubernetes 提供 Kubernetes 集群操作相关的工具实现。
//
// 本文件实现 K8s 写操作工具集，包括：
//   - Deployment 扩缩容
//   - Deployment 滚动重启
//   - Pod 删除
//   - Deployment 回滚
//   - Deployment 删除
package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// =============================================================================
// 输入类型定义
// =============================================================================

// K8sScaleDeploymentInput Deployment 扩缩容输入。
type K8sScaleDeploymentInput struct {
	ClusterID int    `json:"cluster_id" jsonschema_description:"required,cluster id in database"`
	Namespace string `json:"namespace" jsonschema_description:"required,kubernetes namespace"`
	Name      string `json:"name" jsonschema_description:"required,deployment name"`
	Replicas  int32  `json:"replicas" jsonschema_description:"required,target replica count"`
}

// K8sRestartDeploymentInput Deployment 滚动重启输入。
type K8sRestartDeploymentInput struct {
	ClusterID int    `json:"cluster_id" jsonschema_description:"required,cluster id in database"`
	Namespace string `json:"namespace" jsonschema_description:"required,kubernetes namespace"`
	Name      string `json:"name" jsonschema_description:"required,deployment name"`
}

// K8sDeletePodInput Pod 删除输入。
type K8sDeletePodInput struct {
	ClusterID          int    `json:"cluster_id" jsonschema_description:"required,cluster id in database"`
	Namespace          string `json:"namespace" jsonschema_description:"required,kubernetes namespace"`
	Name               string `json:"name" jsonschema_description:"required,pod name"`
	GracePeriodSeconds *int64 `json:"grace_period_seconds,omitempty" jsonschema_description:"optional,grace period in seconds,default=30"`
}

// K8sRollbackDeploymentInput Deployment 回滚输入。
type K8sRollbackDeploymentInput struct {
	ClusterID int    `json:"cluster_id" jsonschema_description:"required,cluster id in database"`
	Namespace string `json:"namespace" jsonschema_description:"required,kubernetes namespace"`
	Name      string `json:"name" jsonschema_description:"required,deployment name"`
	Revision  int64  `json:"revision,omitempty" jsonschema_description:"optional,revision to rollback to,default=previous"`
}

// K8sDeleteDeploymentInput Deployment 删除输入。
type K8sDeleteDeploymentInput struct {
	ClusterID int    `json:"cluster_id" jsonschema_description:"required,cluster id in database"`
	Namespace string `json:"namespace" jsonschema_description:"required,kubernetes namespace"`
	Name      string `json:"name" jsonschema_description:"required,deployment name"`
}

// =============================================================================
// 写操作工具实现
// =============================================================================

// K8sScaleDeployment 创建 Deployment 扩缩容工具。
//
// 该工具需要审批，风险等级: medium。
func K8sScaleDeployment(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_scale_deployment",
		"Scale a Kubernetes Deployment to a specified number of replicas. cluster_id, namespace, name, and replicas are required. This triggers approval flow. Returns previous and new replica count. Example: {\"cluster_id\":1,\"namespace\":\"default\",\"name\":\"nginx\",\"replicas\":3}.",
		func(ctx context.Context, input *K8sScaleDeploymentInput, opts ...tool.Option) (map[string]any, error) {
			// 参数验证
			if err := validateScaleInput(input); err != nil {
				return nil, err
			}
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context unavailable")
			}
			cli, clusterName, err := resolveK8sClientForWrite(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}

			// 获取当前 Deployment
			deploy, err := cli.AppsV1().Deployments(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to get deployment: %v", err)
			}
			previousReplicas := *deploy.Spec.Replicas

			// 执行扩缩容
			deploy.Spec.Replicas = &input.Replicas
			_, err = cli.AppsV1().Deployments(input.Namespace).Update(ctx, deploy, metav1.UpdateOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to scale deployment: %v", err)
			}

			return map[string]any{
				"cluster_id":        input.ClusterID,
				"cluster_name":      clusterName,
				"namespace":         input.Namespace,
				"name":              input.Name,
				"previous_replicas": previousReplicas,
				"new_replicas":      input.Replicas,
				"action":            "scale",
				"status":            "success",
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

// K8sRestartDeployment 创建 Deployment 滚动重启工具。
//
// 该工具需要审批，风险等级: medium。
func K8sRestartDeployment(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_restart_deployment",
		"Trigger a rolling restart of a Kubernetes Deployment by updating the restartedAt annotation. cluster_id, namespace, and name are required. This triggers approval flow. Returns restart timestamp. Example: {\"cluster_id\":1,\"namespace\":\"default\",\"name\":\"nginx\"}.",
		func(ctx context.Context, input *K8sRestartDeploymentInput, opts ...tool.Option) (map[string]any, error) {
			// 参数验证
			if err := validateNamespaceNameInput(input.ClusterID, input.Namespace, input.Name); err != nil {
				return nil, err
			}
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context unavailable")
			}
			cli, clusterName, err := resolveK8sClientForWrite(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}

			// 获取当前 Deployment
			deploy, err := cli.AppsV1().Deployments(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to get deployment: %v", err)
			}

			// 更新 annotation 触发滚动重启
			if deploy.Spec.Template.Annotations == nil {
				deploy.Spec.Template.Annotations = make(map[string]string)
			}
			restartTimestamp := time.Now().Format(time.RFC3339)
			deploy.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartTimestamp

			_, err = cli.AppsV1().Deployments(input.Namespace).Update(ctx, deploy, metav1.UpdateOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to restart deployment: %v", err)
			}

			return map[string]any{
				"cluster_id":   input.ClusterID,
				"cluster_name": clusterName,
				"namespace":    input.Namespace,
				"name":         input.Name,
				"restarted_at": restartTimestamp,
				"action":       "restart",
				"status":       "success",
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

// K8sDeletePod 创建 Pod 删除工具。
//
// 该工具需要审批，风险等级: high。
func K8sDeletePod(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_delete_pod",
		"Delete a Kubernetes Pod. cluster_id, namespace, and name are required. Optional grace_period_seconds controls graceful termination (default 30). This triggers approval flow. Returns deleted pod name. Example: {\"cluster_id\":1,\"namespace\":\"default\",\"name\":\"nginx-abc123\",\"grace_period_seconds\":30}.",
		func(ctx context.Context, input *K8sDeletePodInput, opts ...tool.Option) (map[string]any, error) {
			// 参数验证
			if err := validateNamespaceNameInput(input.ClusterID, input.Namespace, input.Name); err != nil {
				return nil, err
			}
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context unavailable")
			}
			cli, clusterName, err := resolveK8sClientForWrite(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}

			// 设置优雅终止期
			gracePeriod := int64(30)
			if input.GracePeriodSeconds != nil && *input.GracePeriodSeconds >= 0 {
				gracePeriod = *input.GracePeriodSeconds
			}
			deleteOpts := metav1.DeleteOptions{
				GracePeriodSeconds: &gracePeriod,
			}

			err = cli.CoreV1().Pods(input.Namespace).Delete(ctx, input.Name, deleteOpts)
			if err != nil {
				return nil, fmt.Errorf("failed to delete pod: %v", err)
			}

			return map[string]any{
				"cluster_id":           input.ClusterID,
				"cluster_name":         clusterName,
				"namespace":            input.Namespace,
				"name":                 input.Name,
				"grace_period_seconds": gracePeriod,
				"action":               "delete_pod",
				"status":               "success",
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

// K8sRollbackDeployment 创建 Deployment 回滚工具。
//
// 该工具需要审批，风险等级: medium。
func K8sRollbackDeployment(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_rollback_deployment",
		"Rollback a Kubernetes Deployment to a previous revision. cluster_id, namespace, and name are required. Optional revision specifies target revision (default: previous). This triggers approval flow. Returns revision rolled back to. Example: {\"cluster_id\":1,\"namespace\":\"default\",\"name\":\"nginx\"}.",
		func(ctx context.Context, input *K8sRollbackDeploymentInput, opts ...tool.Option) (map[string]any, error) {
			// 参数验证
			if err := validateNamespaceNameInput(input.ClusterID, input.Namespace, input.Name); err != nil {
				return nil, err
			}
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context unavailable")
			}
			cli, clusterName, err := resolveK8sClientForWrite(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}

			// 获取 ReplicaSet 列表，找到上一个版本
			rsList, err := cli.AppsV1().ReplicaSets(input.Namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", input.Name),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to list replicasets: %v", err)
			}

			// 找到当前和上一个版本
			var currentRevision, previousRevision int64
			var previousRS string
			for _, rs := range rsList.Items {
				rev := rs.Annotations["deployment.kubernetes.io/revision"]
				if rev == "" {
					continue
				}
				var revNum int64
				fmt.Sscanf(rev, "%d", &revNum)
				if revNum > currentRevision {
					previousRevision = currentRevision
					previousRS = ""
					currentRevision = revNum
				} else if revNum > previousRevision {
					previousRevision = revNum
					previousRS = rs.Name
				}
			}

			// 如果指定了版本，使用指定版本
			targetRevision := previousRevision
			if input.Revision > 0 {
				targetRevision = input.Revision
			}

			// 使用 kubectl rollout undo 的方式：通过 REST API
			rollbackBody := fmt.Sprintf(`{"name":"%s","rollbackTo":{"revision":%d}}`, input.Name, targetRevision)
			err = cli.AppsV1().RESTClient().Post().
				Namespace(input.Namespace).
				Resource("deployments").
				Name(input.Name).
				SubResource("rollback").
				Body([]byte(rollbackBody)).
				Do(ctx).
				Error()
			if err != nil {
				return nil, fmt.Errorf("failed to rollback deployment: %v", err)
			}

			return map[string]any{
				"cluster_id":      input.ClusterID,
				"cluster_name":    clusterName,
				"namespace":       input.Namespace,
				"name":            input.Name,
				"target_revision": targetRevision,
				"previous_rs":     previousRS,
				"action":          "rollback",
				"status":          "success",
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

// K8sDeleteDeployment 创建 Deployment 删除工具。
//
// 该工具需要审批，风险等级: critical。
func K8sDeleteDeployment(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"k8s_delete_deployment",
		"Delete a Kubernetes Deployment permanently. cluster_id, namespace, and name are required. WARNING: This action is irreversible and will stop the service. This triggers approval flow with critical risk level. Example: {\"cluster_id\":1,\"namespace\":\"default\",\"name\":\"nginx\"}.",
		func(ctx context.Context, input *K8sDeleteDeploymentInput, opts ...tool.Option) (map[string]any, error) {
			// 参数验证
			if err := validateNamespaceNameInput(input.ClusterID, input.Namespace, input.Name); err != nil {
				return nil, err
			}
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context unavailable")
			}
			cli, clusterName, err := resolveK8sClientForWrite(svcCtx, input.ClusterID)
			if err != nil {
				return nil, err
			}

			err = cli.AppsV1().Deployments(input.Namespace).Delete(ctx, input.Name, metav1.DeleteOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to delete deployment: %v", err)
			}

			return map[string]any{
				"cluster_id":   input.ClusterID,
				"cluster_name": clusterName,
				"namespace":    input.Namespace,
				"name":         input.Name,
				"action":       "delete_deployment",
				"status":       "success",
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

// =============================================================================
// 辅助函数
// =============================================================================

// resolveK8sClientForWrite 解析 Kubernetes 客户端用于写操作。
//
// 参数:
//   - svcCtx: 服务上下文
//   - clusterID: 集群 ID
//
// 返回:
//   - *kubernetes.Clientset: Kubernetes 客户端
//   - string: 集群名称
//   - error: 错误信息
func resolveK8sClientForWrite(svcCtx *svc.ServiceContext, clusterID int) (*kubernetes.Clientset, string, error) {
	if clusterID <= 0 {
		return nil, "", fmt.Errorf("cluster_id is required")
	}
	if svcCtx.DB == nil {
		return nil, "", fmt.Errorf("database unavailable")
	}
	var cluster model.Cluster
	if err := svcCtx.DB.First(&cluster, clusterID).Error; err != nil {
		return nil, "", fmt.Errorf("cluster not found: %v", err)
	}
	if strings.TrimSpace(cluster.KubeConfig) == "" {
		return nil, "", fmt.Errorf("cluster %d has no kubeconfig", clusterID)
	}
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
	if err != nil {
		return nil, "", fmt.Errorf("invalid kubeconfig: %v", err)
	}
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create k8s client: %v", err)
	}
	return cli, cluster.Name, nil
}

// validateScaleInput 验证扩缩容输入。
func validateScaleInput(input *K8sScaleDeploymentInput) error {
	if input.ClusterID <= 0 {
		return fmt.Errorf("cluster_id is required")
	}
	if strings.TrimSpace(input.Namespace) == "" {
		return fmt.Errorf("namespace is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if input.Replicas < 0 {
		return fmt.Errorf("replicas must be >= 0")
	}
	return nil
}

// validateNamespaceNameInput 验证基础 namespace/name 输入。
func validateNamespaceNameInput(clusterID int, namespace, name string) error {
	if clusterID <= 0 {
		return fmt.Errorf("cluster_id is required")
	}
	if strings.TrimSpace(namespace) == "" {
		return fmt.Errorf("namespace is required")
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
