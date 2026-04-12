package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	aiartifact "github.com/cy77cc/OpsPilot/internal/modules/ai/artifact"
	aicontext "github.com/cy77cc/OpsPilot/internal/modules/ai/context"
	aitools "github.com/cy77cc/OpsPilot/internal/modules/ai/agent"
	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	deploymentmodel "github.com/cy77cc/OpsPilot/internal/modules/deployment/model"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	projectmodel "github.com/cy77cc/OpsPilot/internal/modules/project/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

type LoadTaskContextInput struct {
	Instruction      string   `json:"instruction,omitempty" jsonschema_description:"optional stable instruction layer"`
	SessionMemory    string   `json:"session_memory,omitempty" jsonschema_description:"optional session memory summary"`
	TaskMemory       string   `json:"task_memory,omitempty" jsonschema_description:"optional task memory summary"`
	RunScratchpad    string   `json:"run_scratchpad,omitempty" jsonschema_description:"optional run scratchpad summary"`
	ArtifactExcerpts []string `json:"artifact_excerpts,omitempty" jsonschema_description:"optional artifact excerpts to inject"`
}

type LoadArtifactContextInput struct {
	Content        string `json:"content" jsonschema_description:"required content to classify as inline or artifact reference"`
	ArtifactID     string `json:"artifact_id,omitempty" jsonschema_description:"optional preferred artifact id"`
	MaxInlineChars int    `json:"max_inline_chars,omitempty" jsonschema_description:"optional max inline chars before artifact reference, default 512"`
}

type ToolSearchInput struct {
	Query  string `json:"query" jsonschema_description:"required search query for tool capability/domain"`
	Limit  int    `json:"limit,omitempty" jsonschema_description:"optional max result count, default 5"`
	Domain string `json:"domain,omitempty" jsonschema_description:"optional domain filter: host, kubernetes, monitoring"`
}

func defaultToolCatalog() aitools.Catalog {
	return aitools.NewCatalog(aitools.AllCatalogEntries())
}

func LoadSessionHistory(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"load_session_history",
		"Load final user and assistant messages from the current authorized chat session. "+
			"Use when: you need to recall context from earlier in the conversation or understand the current task's history. "+
			"Don't use when: the user's last message is sufficient or when starting a completely new task. "+
			"Example: {\"mode\":\"compact\",\"max_turns\":6}. The tool reads the active session from runtime context and enforces ownership automatically.",
		func(ctx context.Context, input *LoadSessionHistoryInput, _ ...tool.Option) (map[string]any, error) {
			svcCtx, _ := runtimectx.ServicesAs[*svc.ServiceContext](ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context unavailable. Suggestion: retry in a few moments or report system error")
			}

			meta := runtimectx.AIMetadataFrom(ctx)
			if strings.TrimSpace(meta.SessionID) == "" || meta.UserID == 0 {
				return nil, fmt.Errorf("ai session context unavailable. Suggestion: ensure you are in an active chat session")
			}

			session, err := loadSession(ctx, svcCtx.DB, meta.SessionID, meta.UserID)
			if err != nil {
				return nil, fmt.Errorf("failed to load session history: %v. Suggestion: check session connectivity", err)
			}
			if session == nil {
				return nil, fmt.Errorf("session not found or access denied. Suggestion: verify you have permission for this session")
			}

			messages, err := listMessagesBySession(ctx, svcCtx.DB, meta.SessionID)
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

func LoadTaskContext(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"load_task_context",
		"Assemble layered task context from instruction, session memory, task memory, run scratchpad, and artifact excerpts. "+
			"Use this to produce deterministic context layers for task execution.",
		func(_ context.Context, input *LoadTaskContextInput, _ ...tool.Option) (map[string]any, error) {
			if input == nil {
				input = &LoadTaskContextInput{}
			}
			assembler := aicontext.NewAssembler()
			layers := assembler.Assemble(aicontext.Input{
				Instruction:      input.Instruction,
				SessionMemory:    input.SessionMemory,
				TaskMemory:       input.TaskMemory,
				RunScratchpad:    input.RunScratchpad,
				ArtifactExcerpts: input.ArtifactExcerpts,
			})
			return map[string]any{
				"layer_count":       len(layers),
				"context_layers":    layers,
				"assembled_context": strings.Join(layers, "\n\n"),
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func LoadArtifactContext(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"load_artifact_context",
		"Convert large content into either inline context or scaffolding artifact-reference metadata. "+
			"Use this to avoid prompt bloat while retaining a stable summary contract. Artifact handles may be absent until persistent artifact storage exists.",
		func(_ context.Context, input *LoadArtifactContextInput, _ ...tool.Option) (map[string]any, error) {
			if input == nil {
				return nil, fmt.Errorf("content is required")
			}
			content := strings.TrimSpace(input.Content)
			if content == "" {
				return nil, fmt.Errorf("content is required")
			}
			result := aiartifact.NewService(input.MaxInlineChars).BuildReference(content, input.ArtifactID)
			payload := map[string]any{
				"mode":    result.Mode,
				"summary": result.Summary,
			}
			if result.ArtifactID != "" {
				payload["artifact_id"] = result.ArtifactID
			}
			if result.Content != "" {
				payload["content"] = result.Content
			}
			return payload, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func ToolSearch(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"tool_search",
		"Search available tools from the metadata catalog by capability/domain keywords and return top candidates. "+
			"Use this before calling domain tools directly when tool count is large.",
		func(_ context.Context, input *ToolSearchInput, _ ...tool.Option) (map[string]any, error) {
			if input == nil || strings.TrimSpace(input.Query) == "" {
				return nil, fmt.Errorf("query is required")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 5
			}
			if limit > 20 {
				limit = 20
			}
			domain := strings.ToLower(strings.TrimSpace(input.Domain))

			// FIXME: Tools catalog removed
			var results []interface{}
			return map[string]any{
				"query":   strings.TrimSpace(input.Query),
				"domain":  domain,
				"count":   len(results),
				"results": results,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type chatSessionRecord struct {
	ID     string `gorm:"column:id"`
	UserID uint64 `gorm:"column:user_id"`
}

func (chatSessionRecord) TableName() string { return "ai_chat_sessions" }

type chatMessageRecord struct {
	Role    string `gorm:"column:role"`
	Content string `gorm:"column:content"`
	Status  string `gorm:"column:status"`
}

func (chatMessageRecord) TableName() string { return "ai_chat_messages" }

func loadSession(ctx context.Context, db *gorm.DB, sessionID string, userID uint64) (*chatSessionRecord, error) {
	var session chatSessionRecord
	err := db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func listMessagesBySession(ctx context.Context, db *gorm.DB, sessionID string) ([]chatMessageRecord, error) {
	var messages []chatMessageRecord
	err := db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("session_id_num ASC, created_at ASC, id ASC").
		Find(&messages).Error
	return messages, err
}

func filterFinalConversationMessages(messages []chatMessageRecord) []chatMessageRecord {
	filtered := make([]chatMessageRecord, 0, len(messages))
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

func buildHistoryPayload(sessionID, mode string, messages []chatMessageRecord, maxTurns, maxChars int) map[string]any {
	recentCount := maxTurns * 2
	if recentCount <= 0 {
		recentCount = defaultMaxTurns * 2
	}

	recentStart := 0
	if len(messages) > recentCount {
		recentStart = len(messages) - recentCount
	}

	recent := messages[recentStart:]
	instructionLayer := "### SEMANTIC MEMORY (Core Facts)\n- Role: OpsPilot AI assistant\n- Session: " + sessionID
	episodicLayer := ""
	if mode == "compact" && recentStart > 0 {
		older := messages[:recentStart]
		summary := summarizeMessages(older)
		if summary != "" {
			episodicLayer = "### EPISODIC MEMORY (Earlier History)\n" + summary
		}
	}
	workingLayer := "### WORKING MEMORY (Recent Turns)\n" + formatMessages(recent, maxRecentMessageChars)
	assembler := aicontext.NewAssembler()
	layers := assembler.Assemble(aicontext.Input{
		Instruction:   instructionLayer,
		SessionMemory: episodicLayer,
		TaskMemory:    workingLayer,
	})
	assembled := strings.Join(layers, "\n\n")
	artifactRef := aiartifact.NewService(maxChars).BuildReference(assembled, "")
	result := assembled
	if artifactRef.Mode == aiartifact.ModeArtifact {
		result = artifactRef.Summary
	}

	payload := map[string]any{
		"session_id":        sessionID,
		"mode":              mode,
		"message_count":     len(messages),
		"recent_messages":   len(recent),
		"context_layers":    layers,
		"formatted_history": result,
	}
	if artifactRef.ArtifactID != "" {
		payload["history_artifact_id"] = artifactRef.ArtifactID
	}
	return payload
}

func summarizeMessages(messages []chatMessageRecord) string {
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

func formatMessages(messages []chatMessageRecord, maxMessageChars int) string {
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
	valueRunes := []rune(value)
	if maxChars <= 0 || len(valueRunes) <= maxChars {
		return value
	}
	if maxChars <= len("...") {
		return string(valueRunes[:maxChars])
	}
	return string(valueRunes[:maxChars-3]) + "..."
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
		"Discover platform resources like clusters, hosts, services, and namespaces. "+
			"Use when: you need to find IDs, list available environments, or need an overview of available resources. "+
			"Don't use when: you already know the ID and want to perform specific operations (e.g., query pods, scale deployments). "+
			"Example: {\"resource_type\":\"clusters\"}. Omit resource_type to get a general overview of all categories.",
		func(ctx context.Context, input *PlatformDiscoverInput, opts ...tool.Option) (map[string]any, error) {
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context unavailable. Suggestion: retry or check system status")
			}

			resourceType := strings.ToLower(strings.TrimSpace(input.ResourceType))
			if resourceType == "namespaces" && input.ClusterID == 0 {
				return nil, fmt.Errorf("cluster_id is required for resource_type='namespaces'. Suggestion: call platform_discover_resources(resource_type='clusters') first to find valid cluster IDs")
			}
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
	var clusters []clustermodel.Cluster
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
	var nodes []hostmodel.Node
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
	var services []projectmodel.Service
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
		svcCtx.DB.Model(&clustermodel.Cluster{}).Count(&clusterCount)
		svcCtx.DB.Model(&hostmodel.Node{}).Count(&hostCount)
		svcCtx.DB.Model(&projectmodel.Service{}).Count(&serviceCount)
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
func resolveK8sClient(svcCtx *svc.ServiceContext, clusterID int) (*kubernetesclient.Clientset, string, error) {
	if clusterID <= 0 {
		return nil, "", fmt.Errorf("cluster_id is required")
	}
	if svcCtx.DB == nil {
		return nil, "", fmt.Errorf("database unavailable")
	}
	var cluster clustermodel.Cluster
	if err := svcCtx.DB.First(&cluster, clusterID).Error; err != nil {
		return nil, "", fmt.Errorf("cluster not found: %v", err)
	}
	cfg, err := buildRestConfigFromClusterOrCredential(svcCtx, &cluster)
	if err != nil {
		return nil, "", err
	}
	cli, err := kubernetesclient.NewForConfig(cfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create k8s client: %v", err)
	}
	return cli, cluster.Name, nil
}

type clusterCredentialMeta struct {
	SkipTLSVerify bool `json:"skip_tls_verify,omitempty"`
}

func buildRestConfigFromClusterOrCredential(svcCtx *svc.ServiceContext, cluster *clustermodel.Cluster) (*rest.Config, error) {
	if cluster == nil {
		return nil, fmt.Errorf("cluster is nil")
	}
	if strings.TrimSpace(cluster.KubeConfig) != "" {
		cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
		if err == nil {
			return cfg, nil
		}
	}
	if svcCtx == nil || svcCtx.DB == nil {
		return nil, fmt.Errorf("cluster %d has no kubeconfig and database unavailable", cluster.ID)
	}

	query := svcCtx.DB.Model(&deploymentmodel.ClusterCredential{}).Where("cluster_id = ? AND status = ?", cluster.ID, "active")
	if cluster.CredentialID != nil && *cluster.CredentialID > 0 {
		query = query.Where("id = ?", *cluster.CredentialID)
	}
	var cred deploymentmodel.ClusterCredential
	if err := query.Order("id DESC").First(&cred).Error; err != nil {
		return nil, fmt.Errorf("cluster %d has no kubeconfig or active credential", cluster.ID)
	}
	return buildRestConfigFromCredential(&cred)
}

func buildRestConfigFromCredential(cred *deploymentmodel.ClusterCredential) (*rest.Config, error) {
	enc := strings.TrimSpace(config.CFG.Security.EncryptionKey)
	if enc == "" {
		return nil, fmt.Errorf("security.encryption_key is required")
	}
	if cred == nil {
		return nil, fmt.Errorf("credential is nil")
	}
	if strings.TrimSpace(cred.KubeconfigEnc) != "" {
		kubeconfig, err := utils.DecryptText(cred.KubeconfigEnc, enc)
		if err != nil {
			return nil, fmt.Errorf("decrypt kubeconfig failed: %w", err)
		}
		return clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	}

	meta := clusterCredentialMeta{}
	if strings.TrimSpace(cred.MetadataJSON) != "" {
		_ = json.Unmarshal([]byte(cred.MetadataJSON), &meta)
	}
	cfg := &rest.Config{
		Host: strings.TrimSpace(cred.Endpoint),
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: meta.SkipTLSVerify,
		},
	}
	if strings.TrimSpace(cred.CACertEnc) != "" {
		ca, err := utils.DecryptText(cred.CACertEnc, enc)
		if err != nil {
			return nil, fmt.Errorf("decrypt ca cert failed: %w", err)
		}
		cfg.TLSClientConfig.CAData = []byte(ca)
	}
	if strings.TrimSpace(cred.CertEnc) != "" {
		cert, err := utils.DecryptText(cred.CertEnc, enc)
		if err != nil {
			return nil, fmt.Errorf("decrypt cert failed: %w", err)
		}
		key, err := utils.DecryptText(cred.KeyEnc, enc)
		if err != nil {
			return nil, fmt.Errorf("decrypt key failed: %w", err)
		}
		cfg.TLSClientConfig.CertData = []byte(cert)
		cfg.TLSClientConfig.KeyData = []byte(key)
	}
	if strings.TrimSpace(cred.TokenEnc) != "" {
		token, err := utils.DecryptText(cred.TokenEnc, enc)
		if err != nil {
			return nil, fmt.Errorf("decrypt token failed: %w", err)
		}
		cfg.BearerToken = token
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return nil, fmt.Errorf("credential endpoint is empty")
	}
	return cfg, nil
}
