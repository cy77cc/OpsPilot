package core

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/cy77cc/k8s-manage/internal/model"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// EmitPolicyRequiredEvent 发射策略要求事件（审批/确认）。
// 当工具执行被策略拦截时，通过 SSE 通知前端。
func EmitPolicyRequiredEvent(ctx context.Context, meta ToolMeta, err error) {
	if apErr, ok := IsApprovalRequired(err); ok {
		EmitToolEvent(ctx, "approval_required", map[string]any{
			"tool":           meta.Name,
			"approval_token": apErr.Token,
			"expiresAt":      apErr.ExpiresAt,
			"message":        apErr.Error(),
		})
		return
	}
	if cfErr, ok := IsConfirmationRequired(err); ok {
		EmitToolEvent(ctx, "confirmation_required", map[string]any{
			"tool":               meta.Name,
			"confirmation_token": cfErr.Token,
			"expiresAt":          cfErr.ExpiresAt,
			"preview":            cfErr.Preview,
			"message":            cfErr.Error(),
		})
	}
}

// ResolveK8sClient 解析 Kubernetes 客户端，根据参数和依赖项选择合适的客户端。
//
// 参数:
//   - deps: 平台依赖项，包含数据库连接和默认客户端等
//   - params: 参数字典，可能包含 cluster_id 等信息
//
// 返回值:
//   - *kubernetes.Clientset: Kubernetes 客户端实例
//   - string: 客户端来源标识
//   - error: 解析过程中的错误
func ResolveK8sClient(deps PlatformDeps, params map[string]any) (*kubernetes.Clientset, string, error) {
	// 从参数中获取 cluster_id 并转换为整数
	clusterID := toInt(params["cluster_id"])

	// 首先尝试从数据库中获取指定集群的客户端
	if clusterID > 0 && deps.DB != nil {
		var cluster model.Cluster
		// 从数据库中查询集群信息
		if err := deps.DB.First(&cluster, clusterID).Error; err == nil && strings.TrimSpace(cluster.KubeConfig) != "" {
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

	// 如果没有指定集群或获取失败，尝试使用默认客户端
	if deps.Clientset != nil {
		return deps.Clientset, "default_clientset", nil
	}

	// 如果所有尝试都失败，返回错误
	return nil, "fallback", errors.New("k8s client unavailable")
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case uint64:
		return int(x)
	case json.Number:
		n, _ := strconv.Atoi(x.String())
		return n
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	default:
		return 0
	}
}

// StructToMap 将结构体转换为 map[string]any。
// 用于将工具输入转换为参数字典。
func StructToMap(v any) map[string]any {
	raw, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}
