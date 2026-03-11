// Package ai 提供 AI 编排服务的会话记录功能。
//
// 本文件实现聊天记录的持久化和思维链渲染，用于：
//   - 记录用户消息和助手响应
//   - 渲染思维链（思考过程）的可视化内容
//   - 持久化会话状态到数据库
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/ai/events"
	aistate "github.com/cy77cc/OpsPilot/internal/ai/state"
)

// chatRecorder 聊天记录器，负责记录和渲染对话内容。
type chatRecorder struct {
	store       *aistate.ChatStore       // 聊天存储
	userID      uint64                   // 用户 ID
	scene       string                   // 场景
	title       string                   // 会话标题
	prompt      string                   // 用户提示
	sessionID   string                   // 会话 ID
	assistantID string                   // 助手消息 ID
	assistant   aistate.ChatMessageRecord // 助手消息记录
}

// newChatRecorder 创建新的聊天记录器。
func newChatRecorder(store *aistate.ChatStore, userID uint64, scene, message string) *chatRecorder {
	if store == nil {
		return nil
	}
	return &chatRecorder{
		store:  store,
		userID: userID,
		scene:  normalizedScene(scene),
		title:  deriveChatTitle(message),
		prompt: strings.TrimSpace(message),
		assistant: aistate.ChatMessageRecord{
			Status:       "streaming",
			ThoughtChain: []map[string]any{},
		},
	}
}

func (r *chatRecorder) HandleEvent(ctx context.Context, eventType events.Name, payload map[string]any) {
	if r == nil {
		return
	}
	switch eventType {
	case events.Meta:
		r.handleMeta(ctx, payload)
	case events.RewriteResult:
		r.upsertStage(map[string]any{
			"key":         "rewrite",
			"title":       "理解你的问题",
			"status":      "success",
			"description": "已将口语化输入整理为可规划任务",
		})
	case events.PlannerState:
		r.upsertStage(map[string]any{
			"key":         "plan",
			"title":       "整理排查计划",
			"status":      normalizeThoughtStatus(payload["status"]),
			"description": firstString(payload["user_visible_summary"], "正在根据 Rewrite 结果整理计划"),
		})
	case events.PlanCreated:
		r.upsertStage(map[string]any{
			"key":         "plan",
			"title":       "整理排查计划",
			"status":      "success",
			"description": "已生成结构化计划",
		})
	case events.StageDelta:
		stageKey := firstString(payload["stage"])
		if stageKey != "" {
			if stageKey == "summary" {
				break
			}
			stage := r.findStage(stageKey)
			content := firstString(payload["content_chunk"], payload["contentChunk"], payload["message"], payload["content"])
			if replace, _ := payload["replace"].(bool); replace {
				stage["content"] = strings.TrimSpace(content)
			} else {
				stage["content"] = appendStageContent(toString(stage["content"]), content)
			}
			stage["status"] = normalizeThoughtStatus(payload["status"])
			if toString(stage["title"]) == "" {
				stage["title"] = resolveThoughtStageTitle(stageKey)
			}
			r.upsertStage(stage)
		}
	case events.StepUpdate:
		r.upsertStage(map[string]any{
			"key":         "execute",
			"title":       "调用专家执行",
			"status":      normalizeThoughtStatus(payload["status"]),
			"description": firstString(payload["title"], "正在推进计划步骤"),
		})
	case events.ToolCall:
		r.upsertStage(map[string]any{
			"key":         "execute",
			"title":       "调用专家执行",
			"status":      "loading",
			"description": firstString(payload["expert"], payload["tool_name"], "专家正在执行"),
		})
		r.upsertDetail("execute", map[string]any{
			"id":      firstString(payload["call_id"], payload["tool_name"], fmt.Sprintf("%d", time.Now().UnixNano())),
			"label":   firstString(payload["tool_name"], payload["expert"], "tool"),
			"status":  "loading",
			"content": firstString(payload["summary"]),
		})
	case events.ToolResult:
		status := "success"
		if firstString(payload["status"]) == "error" {
			status = "error"
		}
		if result, ok := payload["result"].(map[string]any); ok {
			if okValue, exists := result["ok"].(bool); exists && !okValue {
				status = "error"
			}
		}
		r.upsertDetail("execute", map[string]any{
			"id":      firstString(payload["call_id"], payload["tool_name"], fmt.Sprintf("%d", time.Now().UnixNano())),
			"label":   firstString(payload["tool_name"], payload["expert"], "tool"),
			"status":  status,
			"content": firstString(payload["error"], payload["summary"]),
		})
	case events.Delta:
		r.assistant.Content += firstString(payload["content_chunk"], payload["contentChunk"], payload["message"], payload["content"])
	case events.ThinkingDelta:
		r.assistant.Thinking += firstString(payload["content_chunk"], payload["contentChunk"], payload["message"], payload["content"])
	case events.ApprovalRequired:
		r.upsertStage(map[string]any{
			"key":         "user_action",
			"title":       "等待你确认",
			"status":      "loading",
			"description": firstString(payload["title"], "当前步骤需要审批后继续执行"),
			"content":     firstString(payload["user_visible_summary"]),
		})
	case events.ClarifyRequired:
		r.upsertStage(map[string]any{
			"key":         "user_action",
			"title":       "等待你补充信息",
			"status":      "loading",
			"description": firstString(payload["message"], payload["title"], "当前目标仍有歧义"),
		})
		if r.assistant.Content == "" {
			r.assistant.Content = firstString(payload["message"])
		}
	case events.ReplanStarted:
		r.upsertStage(map[string]any{
			"key":         "plan",
			"title":       "整理排查计划",
			"status":      "loading",
			"description": "正在开始新一轮规划",
		})
	case events.Error:
		r.assistant.Status = "error"
		r.markLastStage("error")
	case events.Done:
		r.assistant.Status = "completed"
		r.finalizeStages()
		if recommendations, ok := payload["turn_recommendations"].([]any); ok {
			r.assistant.Recommendations = normalizeAnySlice(recommendations)
		}
	}
	_ = r.persist(ctx)
}

func extractRawEvidenceFromThoughtChain(stages []map[string]any) []string {
	for _, stage := range stages {
		if toString(stage["key"]) != "execute" {
			continue
		}
		details, _ := stage["details"].([]map[string]any)
		if len(details) == 0 {
			if raw, ok := stage["details"].([]any); ok {
				details = make([]map[string]any, 0, len(raw))
				for _, item := range raw {
					if detail, okDetail := item.(map[string]any); okDetail {
						details = append(details, detail)
					}
				}
			}
		}
		out := make([]string, 0, len(details))
		for _, detail := range details {
			text := firstString(detail["content"])
			if text == "" {
				text = firstString(detail["label"])
			}
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	}
	return nil
}

func copyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func (r *chatRecorder) SessionPayload(ctx context.Context) map[string]any {
	if r == nil || r.sessionID == "" {
		return nil
	}
	row, err := r.store.GetSession(ctx, r.userID, r.scene, r.sessionID, true)
	if err != nil || row == nil {
		return nil
	}
	session := toAPISession(*row, true)
	return map[string]any{
		"id":        session.ID,
		"scene":     session.Scene,
		"title":     session.Title,
		"messages":  session.Messages,
		"createdAt": session.CreatedAt.Format(time.RFC3339),
		"updatedAt": session.UpdatedAt.Format(time.RFC3339),
	}
}

func (r *chatRecorder) handleMeta(ctx context.Context, payload map[string]any) {
	r.sessionID = firstString(payload["session_id"], payload["sessionId"])
	r.assistant.TraceID = firstString(payload["trace_id"], payload["traceId"])
	if r.sessionID == "" {
		return
	}
	_ = r.store.EnsureSession(ctx, r.sessionID, r.userID, r.scene, r.title)
	_ = r.store.AppendUserMessage(ctx, r.sessionID, r.userID, r.scene, r.title, r.prompt)
	if r.assistantID == "" {
		id, err := r.store.CreateAssistantMessage(ctx, r.sessionID, r.userID, r.scene, r.title)
		if err == nil {
			r.assistantID = id
		}
	}
}

func (r *chatRecorder) persist(ctx context.Context) error {
	if r == nil || r.assistantID == "" || r.sessionID == "" {
		return nil
	}
	return r.store.UpdateAssistantMessage(ctx, r.sessionID, r.assistantID, r.assistant)
}

func (r *chatRecorder) upsertStage(stage map[string]any) {
	key := firstString(stage["key"])
	if key == "" {
		return
	}
	stages := r.assistant.ThoughtChain
	index := -1
	for i, item := range stages {
		if toString(item["key"]) == key {
			index = i
			break
		}
	}
	if index == -1 {
		stage["collapsible"] = true
		stage["blink"] = stage["status"] == "loading"
		r.assistant.ThoughtChain = append(stages, stage)
		return
	}
	merged := stages[index]
	for k, v := range stage {
		if v != nil && !(toString(v) == "" && (k == "description" || k == "content" || k == "footer")) {
			merged[k] = v
		}
	}
	merged["collapsible"] = true
	merged["blink"] = merged["status"] == "loading"
	merged["content"] = renderThoughtContent(merged)
	stages[index] = merged
	r.assistant.ThoughtChain = stages
}

func (r *chatRecorder) findStage(key string) map[string]any {
	for _, item := range r.assistant.ThoughtChain {
		if toString(item["key"]) == key {
			copy := map[string]any{}
			for k, v := range item {
				copy[k] = v
			}
			return copy
		}
	}
	return map[string]any{
		"key":         key,
		"title":       resolveThoughtStageTitle(key),
		"status":      "loading",
		"collapsible": true,
	}
}

func (r *chatRecorder) upsertDetail(stageKey string, detail map[string]any) {
	stage := r.findStage(stageKey)
	details := normalizeAnySlice(detailSlice(stage["details"]))
	index := -1
	targetID := firstString(detail["id"])
	for i, item := range details {
		if toString(item["id"]) == targetID {
			index = i
			break
		}
	}
	if index == -1 {
		details = append(details, detail)
	} else {
		for k, v := range detail {
			details[index][k] = v
		}
	}
	stage["details"] = details
	stage["content"] = renderThoughtContent(stage)
	r.upsertStage(stage)
}

func (r *chatRecorder) markLastStage(status string) {
	if len(r.assistant.ThoughtChain) == 0 {
		return
	}
	last := r.assistant.ThoughtChain[len(r.assistant.ThoughtChain)-1]
	last["status"] = status
	last["blink"] = false
	r.assistant.ThoughtChain[len(r.assistant.ThoughtChain)-1] = last
}

func (r *chatRecorder) finalizeStages() {
	for i, item := range r.assistant.ThoughtChain {
		if toString(item["status"]) == "loading" {
			item["status"] = "success"
		}
		item["blink"] = false
		r.assistant.ThoughtChain[i] = item
	}
}

func renderThoughtContent(stage map[string]any) string {
	summary := strings.TrimSpace(toString(stage["content"]))
	details := normalizeAnySlice(detailSlice(stage["details"]))
	lines := make([]string, 0, len(details)+1)
	if summary != "" {
		lines = append(lines, summary)
	}
	for _, detail := range details {
		prefix := "[执行中]"
		switch toString(detail["status"]) {
		case "error":
			prefix = "[失败]"
		case "success":
			prefix = "[完成]"
		}
		body := strings.TrimSpace(toString(detail["content"]))
		label := firstString(detail["label"], "tool")
		if body != "" {
			lines = append(lines, fmt.Sprintf("%s %s: %s", prefix, label, body))
		} else {
			lines = append(lines, fmt.Sprintf("%s %s", prefix, label))
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func normalizeThoughtStatus(raw any) string {
	switch strings.TrimSpace(toString(raw)) {
	case "completed", "success":
		return "success"
	case "failed", "error", "blocked":
		return "error"
	case "cancelled", "rejected":
		return "abort"
	case "running", "waiting_approval", "planning", "replanning":
		return "loading"
	default:
		return "loading"
	}
}

func resolveThoughtStageTitle(stage string) string {
	switch stage {
	case "rewrite":
		return "理解你的问题"
	case "plan":
		return "整理排查计划"
	case "execute":
		return "调用专家执行"
	case "user_action":
		return "等待你操作"
	default:
		return "处理中"
	}
}

func appendStageContent(current, chunk string) string {
	current = strings.TrimSpace(current)
	chunk = strings.TrimSpace(chunk)
	if current == "" {
		return chunk
	}
	if chunk == "" {
		return current
	}
	return current + "\n" + chunk
}

func detailSlice(raw any) []any {
	switch v := raw.(type) {
	case []map[string]any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	case []any:
		return v
	default:
		return nil
	}
}

func normalizeAnySlice(items []any) []map[string]any {
	if len(items) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if row, ok := item.(map[string]any); ok {
			out = append(out, row)
		}
	}
	return out
}

func firstString(values ...any) string {
	for _, value := range values {
		if text := strings.TrimSpace(toString(value)); text != "" {
			return text
		}
	}
	return ""
}

func firstStringFromMap(row map[string]any, values ...string) string {
	for _, key := range values {
		if text := strings.TrimSpace(toString(row[key])); text != "" {
			return text
		}
	}
	return ""
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func deriveChatTitle(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "新对话"
	}
	runes := []rune(message)
	if len(runes) > 24 {
		return strings.TrimSpace(string(runes[:24])) + "..."
	}
	return message
}
