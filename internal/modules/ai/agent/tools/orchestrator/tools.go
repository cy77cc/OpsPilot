package orchestrator

import (
	"context"
	"fmt"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools/toolutil"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	aitools "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
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
		return toolutil.UnavailableInvokableTool("load_session_history", err)
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
			assembler := newContextAssembler()
			layers := assembler.Assemble(contextAssembleInput{
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
		return toolutil.UnavailableInvokableTool("load_task_context", err)
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
			result := newArtifactService(input.MaxInlineChars).BuildReference(content, input.ArtifactID)
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
		return toolutil.UnavailableInvokableTool("load_artifact_context", err)
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

			query := strings.TrimSpace(input.Query)
			results := defaultToolCatalog().Search(query, limit, domain)
			return map[string]any{
				"query":   query,
				"domain":  domain,
				"count":   len(results),
				"results": results,
			}, nil
		},
	)
	if err != nil {
		return toolutil.UnavailableInvokableTool("tool_search", err)
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
	assembler := newContextAssembler()
	layers := assembler.Assemble(contextAssembleInput{
		Instruction:   instructionLayer,
		SessionMemory: episodicLayer,
		TaskMemory:    workingLayer,
	})
	assembled := strings.Join(layers, "\n\n")
	artifactRef := newArtifactService(maxChars).BuildReference(assembled, "")
	result := assembled
	if artifactRef.Mode == artifactModeArtifact {
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
