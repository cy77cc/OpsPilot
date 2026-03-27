package history

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
