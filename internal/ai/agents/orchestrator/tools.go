package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultMaxTurns       = 6
	defaultMaxChars       = 4000
	maxSummaryMessages    = 8
	maxRecentMessageChars = 320
	maxSummaryLineChars   = 120
)

type LoadSessionHistoryInput struct {
	Mode     string `json:"mode,omitempty" jsonschema_description:"optional history mode: recent or compact. compact is recommended for longer sessions"`
	MaxTurns int    `json:"max_turns,omitempty" jsonschema_description:"optional number of recent turns to include, default 6"`
	MaxChars int    `json:"max_chars,omitempty" jsonschema_description:"optional maximum output size in characters, default 4000"`
}

func LoadSessionHistory(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"load_session_history",
		"Load final user and assistant messages from the current authorized chat session. Do not pass session_id; the tool reads the active session from runtime context and enforces ownership automatically. It never returns steps, tool traces, or runtime state. mode=recent returns the latest turns verbatim. mode=compact returns a compact summary of earlier history plus recent turns. Example: {\"mode\":\"compact\",\"max_turns\":6}.",
		func(ctx context.Context, input *LoadSessionHistoryInput, _ ...tool.Option) (map[string]any, error) {
			svcCtx, _ := runtimectx.ServicesAs[*svc.ServiceContext](ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context unavailable")
			}

			meta := runtimectx.AIMetadataFrom(ctx)
			if strings.TrimSpace(meta.SessionID) == "" || meta.UserID == 0 {
				return nil, fmt.Errorf("ai session context unavailable")
			}

			chatDAO := aidao.NewAIChatDAO(svcCtx.DB)
			session, err := chatDAO.GetSession(ctx, meta.SessionID, meta.UserID, "")
			if err != nil {
				return nil, err
			}
			if session == nil {
				return nil, fmt.Errorf("session not found or access denied")
			}

			messages, err := chatDAO.ListMessagesBySession(ctx, meta.SessionID)
			if err != nil {
				return nil, err
			}

			filtered := filterFinalConversationMessages(messages)
			mode := normalizeMode(input.Mode)
			maxTurns := normalizeMaxTurns(input.MaxTurns)
			maxChars := normalizeMaxChars(input.MaxChars)

			payload := buildHistoryPayload(meta.SessionID, mode, filtered, maxTurns, maxChars)
			return payload, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func filterFinalConversationMessages(messages []model.AIChatMessage) []model.AIChatMessage {
	filtered := make([]model.AIChatMessage, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(message.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		if strings.TrimSpace(message.Content) == "" {
			continue
		}
		if role == "assistant" && strings.EqualFold(strings.TrimSpace(message.Status), "streaming") {
			continue
		}
		filtered = append(filtered, message)
	}
	return filtered
}

func buildHistoryPayload(sessionID, mode string, messages []model.AIChatMessage, maxTurns, maxChars int) map[string]any {
	recentCount := maxTurns * 2
	if recentCount <= 0 {
		recentCount = defaultMaxTurns * 2
	}

	recentStart := 0
	if len(messages) > recentCount {
		recentStart = len(messages) - recentCount
	}

	recent := messages[recentStart:]
	var formatted string
	if mode == "compact" && recentStart > 0 {
		older := messages[:recentStart]
		summary := summarizeMessages(older)
		if summary != "" {
			formatted = "[Earlier conversation summary]\n" + summary + "\n\n"
		}
	}
	formatted += "[Recent conversation]\n" + formatMessages(recent, maxRecentMessageChars)
	formatted = enforceCharLimit(formatted, maxChars)

	return map[string]any{
		"session_id":        sessionID,
		"mode":              mode,
		"message_count":     len(messages),
		"recent_messages":   len(recent),
		"formatted_history": formatted,
	}
}

func summarizeMessages(messages []model.AIChatMessage) string {
	if len(messages) == 0 {
		return ""
	}
	if len(messages) > maxSummaryMessages {
		messages = messages[len(messages)-maxSummaryMessages:]
	}

	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		lines = append(lines, fmt.Sprintf("- %s: %s", roleLabel(message.Role), truncateText(message.Content, maxSummaryLineChars)))
	}
	return strings.Join(lines, "\n")
}

func formatMessages(messages []model.AIChatMessage, maxMessageChars int) string {
	if len(messages) == 0 {
		return "(no prior messages)"
	}
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		lines = append(lines, fmt.Sprintf("%s: %s", roleLabel(message.Role), truncateText(message.Content, maxMessageChars)))
	}
	return strings.Join(lines, "\n")
}

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "compact":
		return "compact"
	default:
		return "recent"
	}
}

func normalizeMaxTurns(maxTurns int) int {
	if maxTurns <= 0 {
		return defaultMaxTurns
	}
	if maxTurns > 20 {
		return 20
	}
	return maxTurns
}

func normalizeMaxChars(maxChars int) int {
	if maxChars <= 0 {
		return defaultMaxChars
	}
	if maxChars > 12000 {
		return 12000
	}
	return maxChars
}

func roleLabel(role string) string {
	if strings.EqualFold(strings.TrimSpace(role), "assistant") {
		return "Assistant"
	}
	return "User"
}

func truncateText(value string, maxChars int) string {
	value = strings.TrimSpace(strings.Join(strings.Fields(value), " "))
	if maxChars <= 0 || len(value) <= maxChars {
		return value
	}
	if maxChars <= len("...") {
		return value[:maxChars]
	}
	return value[:maxChars-3] + "..."
}

func enforceCharLimit(value string, maxChars int) string {
	if maxChars <= 0 || len(value) <= maxChars {
		return value
	}
	if maxChars <= len("...(truncated)") {
		return value[:maxChars]
	}
	return value[:maxChars-len("...(truncated)")] + "...(truncated)"
}

func serviceContextFromRuntime(ctx context.Context) *svc.ServiceContext {
	svcCtx, _ := runtimectx.ServicesAs[*svc.ServiceContext](ctx)
	return svcCtx
}

// =============================================================================
// 输入类型定义
// =============================================================================

// PlatformDiscoverInput 资源发现输入。
type PlatformDiscoverInput struct {
	ResourceType string `json:"resource_type,omitempty" jsonschema_description:"optional,resource type to discover: clusters/hosts/services/namespaces/metrics,omit for overview"`
	ClusterID    int    `json:"cluster_id,omitempty" jsonschema_description:"required when resource_type=namespaces,cluster id to query namespaces from"`
}

// =============================================================================
// 工具入口
// =============================================================================

// PlatformDiscoverResources 创建资源发现工具。
//
// 该工具允许 AI 查询平台内可用资源，无需预先知道资源 ID。
// 支持的资源类型：
//   - clusters: K8s 集群列表
//   - hosts: 主机列表
//   - services: 服务列表
//   - namespaces: 指定集群的命名空间（需 cluster_id）
//   - metrics: Prometheus 可用指标
//
// 不传 resource_type 时返回所有资源类型的概览。
func PlatformDiscoverResources(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"platform_discover_resources",
		"Discover platform resources available for operations. Optional resource_type filters results: clusters (K8s clusters), hosts (server list), services (service catalog), namespaces (K8s namespaces, requires cluster_id), metrics (Prometheus metrics). Omit resource_type to get an overview of all resource types with counts. Example: {\"resource_type\":\"clusters\"} or {\"resource_type\":\"namespaces\",\"cluster_id\":1}.",
		func(ctx context.Context, input *PlatformDiscoverInput, opts ...tool.Option) (map[string]any, error) {
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context unavailable")
			}

			resourceType := strings.ToLower(strings.TrimSpace(input.ResourceType))
			switch resourceType {
			case "clusters":
				return discoverClusters(ctx, svcCtx)
			case "hosts":
				return discoverHosts(ctx, svcCtx)
			case "services":
				return discoverServices(ctx, svcCtx)
			case "namespaces":
				if input.ClusterID <= 0 {
					return nil, fmt.Errorf("cluster_id is required when resource_type=namespaces")
				}
				return discoverNamespaces(ctx, svcCtx, input.ClusterID)
			case "metrics":
				return discoverMetrics(ctx, svcCtx)
			default:
				return discoverOverview(ctx, svcCtx)
			}
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

// =============================================================================
// 资源发现实现
// =============================================================================

// discoverClusters 发现所有 K8s 集群。
func discoverClusters(ctx context.Context, svcCtx *svc.ServiceContext) (map[string]any, error) {
	if svcCtx.DB == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	var clusters []model.Cluster
	if err := svcCtx.DB.Select("id", "name", "endpoint", "status", "type", "version", "env_type").
		Order("id asc").Find(&clusters).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(clusters))
	for _, c := range clusters {
		items = append(items, map[string]any{
			"id":       c.ID,
			"name":     c.Name,
			"endpoint": c.Endpoint,
			"status":   c.Status,
			"type":     c.Type,
			"version":  c.Version,
			"env_type": c.EnvType,
		})
	}
	return map[string]any{
		"resource_type": "clusters",
		"total":         len(items),
		"items":         items,
	}, nil
}

// discoverHosts 发现所有主机。
func discoverHosts(ctx context.Context, svcCtx *svc.ServiceContext) (map[string]any, error) {
	if svcCtx.DB == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	var nodes []model.Node
	if err := svcCtx.DB.Select("id", "name", "ip", "hostname", "status", "os", "cluster_id").
		Order("id asc").Find(&nodes).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		items = append(items, map[string]any{
			"id":         n.ID,
			"name":       n.Name,
			"ip":         n.IP,
			"hostname":   n.Hostname,
			"status":     n.Status,
			"os":         n.OS,
			"cluster_id": n.ClusterID,
		})
	}
	return map[string]any{
		"resource_type": "hosts",
		"total":         len(items),
		"items":         items,
	}, nil
}

// discoverServices 发现所有服务。
func discoverServices(ctx context.Context, svcCtx *svc.ServiceContext) (map[string]any, error) {
	if svcCtx.DB == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	var services []model.Service
	if err := svcCtx.DB.Select("id", "name", "env", "status", "runtime_type", "owner").
		Order("id asc").Find(&services).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(services))
	for _, s := range services {
		items = append(items, map[string]any{
			"id":           s.ID,
			"name":         s.Name,
			"env":          s.Env,
			"status":       s.Status,
			"runtime_type": s.RuntimeType,
			"owner":        s.Owner,
		})
	}
	return map[string]any{
		"resource_type": "services",
		"total":         len(items),
		"items":         items,
	}, nil
}

// discoverNamespaces 发现指定集群的命名空间。
func discoverNamespaces(ctx context.Context, svcCtx *svc.ServiceContext, clusterID int) (map[string]any, error) {
	cli, clusterName, err := resolveK8sClient(svcCtx, clusterID)
	if err != nil {
		return nil, err
	}
	nsList, err := cli.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		items = append(items, map[string]any{
			"name":   ns.Name,
			"status": string(ns.Status.Phase),
		})
	}
	return map[string]any{
		"resource_type": "namespaces",
		"cluster_id":    clusterID,
		"cluster_name":  clusterName,
		"total":         len(items),
		"items":         items,
	}, nil
}

// discoverMetrics 发现 Prometheus 可用指标。
func discoverMetrics(ctx context.Context, svcCtx *svc.ServiceContext) (map[string]any, error) {
	if svcCtx.Prometheus == nil {
		return nil, fmt.Errorf("prometheus client unavailable")
	}
	metadata, err := svcCtx.Prometheus.Metadata(ctx, "")
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(metadata))
	for _, m := range metadata {
		items = append(items, map[string]any{
			"name": m.Metric,
			"type": m.Type,
			"help": m.Help,
			"unit": m.Unit,
		})
	}
	return map[string]any{
		"resource_type": "metrics",
		"total":         len(items),
		"items":         items,
	}, nil
}

// discoverOverview 返回所有资源类型的概览。
func discoverOverview(ctx context.Context, svcCtx *svc.ServiceContext) (map[string]any, error) {
	result := map[string]any{
		"resource_type": "overview",
		"clusters":      map[string]any{"total": 0},
		"hosts":         map[string]any{"total": 0},
		"services":      map[string]any{"total": 0},
		"metrics":       map[string]any{"available": false},
	}

	if svcCtx.DB != nil {
		var clusterCount, hostCount, serviceCount int64
		svcCtx.DB.Model(&model.Cluster{}).Count(&clusterCount)
		svcCtx.DB.Model(&model.Node{}).Count(&hostCount)
		svcCtx.DB.Model(&model.Service{}).Count(&serviceCount)
		result["clusters"] = map[string]any{"total": clusterCount}
		result["hosts"] = map[string]any{"total": hostCount}
		result["services"] = map[string]any{"total": serviceCount}
	}

	if svcCtx.Prometheus != nil {
		result["metrics"] = map[string]any{"available": true}
	}

	return result, nil
}

// =============================================================================
// 辅助函数
// =============================================================================

// resolveK8sClient 解析 Kubernetes 客户端。
//
// 参数:
//   - svcCtx: 服务上下文
//   - clusterID: 集群 ID
//
// 返回:
//   - *kubernetes.Clientset: Kubernetes 客户端
//   - string: 集群名称
//   - error: 错误信息
func resolveK8sClient(svcCtx *svc.ServiceContext, clusterID int) (*kubernetes.Clientset, string, error) {
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
