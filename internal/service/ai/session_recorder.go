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
	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/google/uuid"
)

// chatRecorder 聊天记录器，负责记录和渲染对话内容。
type chatRecorder struct {
	store           *aistate.ChatStore        // 聊天存储
	userID          uint64                    // 用户 ID
	scene           string                    // 场景
	title           string                    // 会话标题
	prompt          string                    // 用户提示
	sessionID       string                    // 会话 ID
	assistantID     string                    // 助手消息 ID
	assistantTurnID string                    // 助手 turn ID
	assistant       aistate.ChatMessageRecord // 助手消息记录
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

	// === 新增: 原生链节点事件处理 ===
	case events.ChainStarted:
		r.handleChainStarted(payload)
	case events.ChainNodeOpen:
		r.handleChainNodeOpen(payload)
	case events.ChainNodePatch:
		r.handleChainNodePatch(payload)
	case events.ChainNodeClose:
		r.handleChainNodeClose(payload)
	case events.ChainCollapsed:
		r.handleChainCollapsed(payload)
	case events.FinalAnswerStart:
		r.handleFinalAnswerStart(payload)
	case events.FinalAnswerDelta:
		r.handleFinalAnswerDelta(payload)
	case events.FinalAnswerDone:
		r.handleFinalAnswerDone(payload)

	// === 兼容层: 旧事件处理 ===
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
		r.assistant.Content += firstRawString(payload["content_chunk"], payload["contentChunk"], payload["message"], payload["content"])
	case events.ThinkingDelta:
		r.assistant.Thinking += firstRawString(payload["content_chunk"], payload["contentChunk"], payload["message"], payload["content"])
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
	r.assistantTurnID = firstString(payload["turn_id"], payload["turnId"])
	r.assistant.TraceID = firstString(payload["trace_id"], payload["traceId"])
	if r.sessionID == "" {
		r.sessionID = uuid.NewString()
		logRecorder("generated_session_id", r, nil)
	}
	if err := r.store.EnsureSession(ctx, r.sessionID, r.userID, r.scene, r.title); err != nil {
		logRecorder("ensure_session_failed", r, err)
		return
	}
	if err := r.store.AppendUserMessage(ctx, r.sessionID, r.userID, r.scene, r.title, r.prompt); err != nil {
		logRecorder("append_user_message_failed", r, err)
		return
	}
	if r.assistantID == "" {
		id, err := r.store.CreateAssistantMessage(ctx, r.sessionID, r.userID, r.scene, r.title, r.assistantTurnID)
		if err == nil {
			r.assistantID = id
			return
		}
		logRecorder("create_assistant_message_failed", r, err)
	}
}

func (r *chatRecorder) persist(ctx context.Context) error {
	if r == nil || r.assistantID == "" || r.sessionID == "" {
		return nil
	}
	err := r.store.UpdateAssistantMessage(ctx, r.sessionID, r.assistantID, r.assistantTurnID, r.assistant)
	if err != nil {
		logRecorder("update_assistant_message_failed", r, err)
	}
	return err
}

func logRecorder(action string, r *chatRecorder, err error) {
	l := logger.L()
	if l == nil || r == nil {
		return
	}
	fields := []logger.Field{
		logger.String("action", action),
		logger.String("session_id", r.sessionID),
		logger.String("turn_id", r.assistantTurnID),
		logger.String("scene", r.scene),
		{Key: "user_id", Value: r.userID},
	}
	if err != nil {
		fields = append(fields, logger.Error(err))
		l.Warn("ai session recorder event", fields...)
		return
	}
	l.Debug("ai session recorder event", fields...)
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

func firstRawString(values ...any) string {
	for _, value := range values {
		switch v := value.(type) {
		case string:
			if v != "" {
				return v
			}
		case []byte:
			if len(v) > 0 {
				return string(v)
			}
		default:
			if value == nil {
				continue
			}
			text := fmt.Sprint(value)
			if text != "" && text != "<nil>" {
				return text
			}
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

// === 原生链节点事件处理器 ===

func (r *chatRecorder) ensureRuntime() *aistate.RuntimeState {
	if r.assistant.Runtime == nil {
		r.assistant.Runtime = &aistate.RuntimeState{}
	}
	return r.assistant.Runtime
}

func (r *chatRecorder) handleChainStarted(payload map[string]any) {
	runtime := r.ensureRuntime()
	runtime.TurnID = firstString(payload["turn_id"])
	runtime.Nodes = nil
	runtime.IsCollapsed = false
	runtime.ActiveNodeID = ""
	if runtime.FinalAnswer == nil {
		runtime.FinalAnswer = &aistate.RuntimeFinalAnswer{}
	}
	runtime.FinalAnswer.Visible = false
	runtime.FinalAnswer.Content = ""
}

func (r *chatRecorder) handleChainNodeOpen(payload map[string]any) {
	runtime := r.ensureRuntime()
	nodeID := firstString(payload["node_id"])
	if nodeID == "" {
		return
	}
	node := aistate.RuntimeChainNode{
		NodeID:    nodeID,
		Kind:      firstString(payload["kind"]),
		Title:     firstString(payload["title"]),
		Status:    firstString(payload["status"]),
		Summary:   firstString(payload["summary"]),
		StartedAt: firstString(payload["started_at"]),
	}
	if details, ok := payload["details"].([]any); ok {
		node.Details = details
	}
	if approval, ok := payload["approval"].(map[string]any); ok {
		node.Approval = approval
	}
	// 关闭之前的活动节点
	if runtime.ActiveNodeID != "" && runtime.ActiveNodeID != nodeID {
		for i, n := range runtime.Nodes {
			if n.NodeID == runtime.ActiveNodeID {
				runtime.Nodes[i].Status = "done"
				break
			}
		}
	}
	// 更新或追加节点
	found := false
	for i, n := range runtime.Nodes {
		if n.NodeID == nodeID {
			runtime.Nodes[i] = node
			found = true
			break
		}
	}
	if !found {
		runtime.Nodes = append(runtime.Nodes, node)
	}
	if node.Status == "loading" || node.Status == "active" || node.Status == "waiting" {
		runtime.ActiveNodeID = nodeID
	}
}

func (r *chatRecorder) handleChainNodePatch(payload map[string]any) {
	runtime := r.ensureRuntime()
	nodeID := firstString(payload["node_id"])
	if nodeID == "" {
		return
	}
	for i, n := range runtime.Nodes {
		if n.NodeID == nodeID {
			if title := firstString(payload["title"]); title != "" {
				runtime.Nodes[i].Title = title
			}
			if summary := firstString(payload["summary"]); summary != "" {
				runtime.Nodes[i].Summary = summary
			}
			if status := firstString(payload["status"]); status != "" {
				runtime.Nodes[i].Status = status
			}
			if details, ok := payload["details"].([]any); ok {
				runtime.Nodes[i].Details = details
			}
			break
		}
	}
}

func (r *chatRecorder) handleChainNodeClose(payload map[string]any) {
	runtime := r.ensureRuntime()
	nodeID := firstString(payload["node_id"])
	if nodeID == "" {
		return
	}
	status := firstString(payload["status"])
	if status == "" {
		status = "done"
	}
	for i, n := range runtime.Nodes {
		if n.NodeID == nodeID {
			runtime.Nodes[i].Status = status
			break
		}
	}
	if runtime.ActiveNodeID == nodeID {
		runtime.ActiveNodeID = ""
	}
}

func (r *chatRecorder) handleChainCollapsed(payload map[string]any) {
	runtime := r.ensureRuntime()
	runtime.IsCollapsed = true
	runtime.ActiveNodeID = ""
	// 标记所有节点为完成
	for i := range runtime.Nodes {
		if runtime.Nodes[i].Status == "loading" || runtime.Nodes[i].Status == "active" || runtime.Nodes[i].Status == "waiting" {
			runtime.Nodes[i].Status = "done"
		}
	}
}

func (r *chatRecorder) handleFinalAnswerStart(payload map[string]any) {
	runtime := r.ensureRuntime()
	if runtime.FinalAnswer == nil {
		runtime.FinalAnswer = &aistate.RuntimeFinalAnswer{}
	}
	runtime.FinalAnswer.Visible = true
	runtime.FinalAnswer.Streaming = true
	runtime.FinalAnswer.RevealState = "revealing"
}

func (r *chatRecorder) handleFinalAnswerDelta(payload map[string]any) {
	runtime := r.ensureRuntime()
	if runtime.FinalAnswer == nil {
		runtime.FinalAnswer = &aistate.RuntimeFinalAnswer{}
	}
	chunk := firstString(payload["chunk"])
	if chunk != "" {
		runtime.FinalAnswer.Content += chunk
	}
	runtime.FinalAnswer.Visible = true
	runtime.FinalAnswer.Streaming = true
}

func (r *chatRecorder) handleFinalAnswerDone(payload map[string]any) {
	runtime := r.ensureRuntime()
	if runtime.FinalAnswer == nil {
		runtime.FinalAnswer = &aistate.RuntimeFinalAnswer{}
	}
	runtime.FinalAnswer.Streaming = false
	runtime.FinalAnswer.RevealState = "complete"
}
