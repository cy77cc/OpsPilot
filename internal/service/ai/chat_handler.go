package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

type toolSummary struct {
	Calls          int      `json:"calls"`
	Results        int      `json:"results"`
	Missing        []string `json:"missing"`
	MissingCallIDs []string `json:"missing_call_ids,omitempty"`
}

type streamErrorPayload struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Recoverable bool   `json:"recoverable"`
}

var toolIntentPattern = regexp.MustCompile(`\b([a-z]+_[a-z0-9_]+)\b`)

type toolEventTracker struct {
	mu      sync.Mutex
	calls   map[string]int
	results map[string]int
	callIDs map[string]string
	doneIDs map[string]struct{}
}

func newToolEventTracker() *toolEventTracker {
	return &toolEventTracker{
		calls:   map[string]int{},
		results: map[string]int{},
		callIDs: map[string]string{},
		doneIDs: map[string]struct{}{},
	}
}

func (t *toolEventTracker) noteCall(callID, tool string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	name := strings.TrimSpace(tool)
	if name == "" {
		name = "unknown"
	}
	cid := strings.TrimSpace(callID)
	if cid == "" {
		cid = fmt.Sprintf("legacy-%s-%d", name, t.calls[name]+1)
	}
	t.callIDs[cid] = name
	t.calls[name]++
}

func (t *toolEventTracker) noteResult(callID, tool string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	cid := strings.TrimSpace(callID)
	if cid != "" {
		t.doneIDs[cid] = struct{}{}
	}
	name := strings.TrimSpace(tool)
	if name == "" {
		name = "unknown"
	}
	t.results[name]++
}

func (t *toolEventTracker) summary() toolSummary {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := toolSummary{}
	for _, count := range t.calls {
		out.Calls += count
	}
	for _, count := range t.results {
		out.Results += count
	}
	hasCallID := len(t.callIDs) > 0
	for callID, tool := range t.callIDs {
		if _, ok := t.doneIDs[callID]; ok {
			continue
		}
		out.MissingCallIDs = append(out.MissingCallIDs, callID)
		if tool != "" {
			out.Missing = append(out.Missing, tool)
		}
	}
	if !hasCallID {
		for tool, callCount := range t.calls {
			missing := callCount - t.results[tool]
			for i := 0; i < missing; i++ {
				out.Missing = append(out.Missing, tool)
			}
		}
	}
	return out
}

func resolveStreamState(fatalErr *streamErrorPayload, summary toolSummary) string {
	if fatalErr != nil {
		return "failed"
	}
	if len(summary.MissingCallIDs) > 0 || len(summary.Missing) > 0 {
		return "partial"
	}
	return "ok"
}

func (h *handler) chat(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		httpx.Fail(c, xcode.ParamError, "message is required")
		return
	}
	if h.svcCtx.AI == nil || h.svcCtx.AI.Runnable == nil {
		httpx.Fail(c, xcode.ServerError, "ai agent not initialized")
		return
	}
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		httpx.Fail(c, xcode.ServerError, "streaming not supported")
		return
	}
	turnID := fmt.Sprintf("turn-%d", time.Now().UnixNano())
	writer := newSSEWriter(c, flusher, turnID)
	emit := writer.Emit
	var finalOnce sync.Once
	emitFinal := func(event string, payload gin.H) bool {
		sent := false
		finalOnce.Do(func() {
			sent = emit(event, payload)
		})
		return sent
	}
	defer writer.Close()
	stopHeartbeat := make(chan struct{})
	go heartbeatLoop(stopHeartbeat, emit)
	defer close(stopHeartbeat)

	sid := strings.TrimSpace(req.SessionID)
	scene := normalizeScene(toString(req.Context["scene"]))
	if sid == "" {
		if session, ok := h.store.currentSession(uid, scene); ok {
			sid = session.ID
		} else {
			sid = fmt.Sprintf("sess-%d", time.Now().UnixNano())
		}
	}

	userTime := time.Now()
	session, err := h.store.appendMessage(uid, scene, sid, map[string]any{
		"id":        fmt.Sprintf("u-%d", userTime.UnixNano()),
		"role":      "user",
		"content":   msg,
		"timestamp": userTime,
	})
	if err != nil {
		emitFinal("error", gin.H{"message": err.Error()})
		return
	}
	if !emit("meta", gin.H{"sessionId": session.ID, "createdAt": session.CreatedAt}) {
		return
	}

	approvalToken := strings.TrimSpace(toString(req.Context["approval_token"]))
	tracker := newToolEventTracker()
	streamCtx := h.buildToolContext(c.Request.Context(), uid, approvalToken, scene, req.Context, emit, tracker)
	prompt := msg
	directive := composePromptDirectives(
		buildToolExecutionDirective(msg, scene),
		buildHelpKnowledgeDirective(msg),
	)
	if directive != "" {
		prompt = directive + "\n\n用户问题:\n" + msg
	}
	if len(req.Context) > 0 {
		prompt = msg + "\n\n上下文:\n" + mustJSON(req.Context)
		if directive != "" {
			prompt = directive + "\n\n用户问题:\n" + msg + "\n\n上下文:\n" + mustJSON(req.Context)
		}
	}
	inputMessages := h.buildConversationMessages(session.Messages, msg, prompt)
	stream, err := h.svcCtx.AI.Stream(streamCtx, inputMessages)
	if err != nil {
		emitFinal("error", gin.H{"message": err.Error()})
		return
	}
	defer stream.Close()

	var fatalErr *streamErrorPayload
	var assistantContent strings.Builder
	var reasoningContent strings.Builder
	var streamErr error
	for {
		item, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			if apErr, ok := tools.IsApprovalRequired(recvErr); ok {
				_ = emit("approval_required", gin.H{
					"tool":           apErr.Tool,
					"approval_token": apErr.Token,
					"expiresAt":      apErr.ExpiresAt,
					"message":        apErr.Error(),
				})
				streamErr = recvErr
				break
			}
			streamErr = recvErr
			break
		}
		if item == nil {
			continue
		}
		if item.ReasoningContent != "" {
			reasoningContent.WriteString(item.ReasoningContent)
			_ = emit("thinking_delta", gin.H{"contentChunk": item.ReasoningContent})
		}
		if item.Content != "" {
			assistantContent.WriteString(item.Content)
			if !emit("delta", gin.H{"contentChunk": item.Content}) {
				return
			}
		}
	}
	if streamErr != nil && !errors.Is(streamErr, io.EOF) {
		if _, ok := tools.IsApprovalRequired(streamErr); !ok {
			fatalErr = &streamErrorPayload{
				Code:        "stream_interrupted",
				Message:     streamErr.Error(),
				Recoverable: true,
			}
		}
	}

	summary := tracker.summary()
	if summary.Calls == 0 {
		if hint := detectUnresolvedToolIntent(reasoningContent.String(), assistantContent.String()); hint != "" {
			_ = emit("tool_intent_unresolved", gin.H{
				"tool":    hint,
				"message": "检测到工具意图但未触发实际工具调用，可重试本轮或补充更明确参数。",
			})
		}
	}
	if len(summary.MissingCallIDs) > 0 {
		toolErr := &streamErrorPayload{
			Code:        "tool_result_missing",
			Message:     fmt.Sprintf("tool result missing for %d call(s)", len(summary.MissingCallIDs)),
			Recoverable: true,
		}
		_ = emitFinal("error", gin.H{
			"code":         toolErr.Code,
			"message":      toolErr.Message,
			"recoverable":  toolErr.Recoverable,
			"tool_summary": summary,
		})
	}

	content := strings.TrimSpace(assistantContent.String())
	if content == "" {
		switch {
		case fatalErr != nil:
			content = fmt.Sprintf("本轮执行未完整结束：%s", fatalErr.Message)
		case len(summary.MissingCallIDs) > 0:
			content = "本轮工具调用结果不完整。"
		default:
			content = "无输出。"
		}
	}
	assistantTime := time.Now()
	session, err = h.store.appendMessage(uid, scene, sid, map[string]any{
		"id":        fmt.Sprintf("a-%d", assistantTime.UnixNano()),
		"role":      "assistant",
		"content":   content,
		"thinking":  strings.TrimSpace(reasoningContent.String()),
		"timestamp": assistantTime,
	})
	if err != nil {
		emitFinal("error", gin.H{"message": err.Error()})
		return
	}
	if strings.TrimSpace(session.Title) == "" || strings.EqualFold(strings.TrimSpace(session.Title), defaultAISessionTitle) {
		firstUserContent := firstUserMessageContent(session.Messages)
		if firstUserContent != "" {
			if renamed, renameErr := h.store.updateSessionTitle(uid, sid, inferSessionTitle(firstUserContent)); renameErr == nil && renamed != nil {
				session = renamed
			}
		}
	}
	turnSuggestions := h.refreshSuggestions(uid, scene, content)
	streamState := resolveStreamState(fatalErr, summary)
	if fatalErr != nil {
		_ = emitFinal("error", gin.H{
			"code":         fatalErr.Code,
			"message":      fatalErr.Message,
			"recoverable":  fatalErr.Recoverable,
			"tool_summary": summary,
		})
	}
	emitFinal("done", gin.H{
		"session":              session,
		"stream_state":         streamState,
		"tool_summary":         summary,
		"turn_recommendations": recommendationPayload(turnSuggestions),
	})
}

func firstUserMessageContent(messages []map[string]any) string {
	for _, msg := range messages {
		if strings.TrimSpace(toString(msg["role"])) != "user" {
			continue
		}
		content := strings.TrimSpace(toString(msg["content"]))
		if content != "" {
			return content
		}
	}
	return ""
}

func recommendationPayload(items []recommendationRecord) []gin.H {
	if len(items) == 0 {
		return nil
	}
	limit := len(items)
	if limit > 3 {
		limit = 3
	}
	out := make([]gin.H, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, gin.H{
			"id":              items[i].ID,
			"type":            items[i].Type,
			"title":           items[i].Title,
			"content":         items[i].Content,
			"reasoning":       items[i].Reasoning,
			"relevance":       items[i].Relevance,
			"followup_prompt": items[i].FollowupPrompt,
		})
	}
	return out
}

func detectUnresolvedToolIntent(reasoning, content string) string {
	combined := strings.ToLower(reasoning + "\n" + content)
	matches := toolIntentPattern.FindAllStringSubmatch(combined, -1)
	for _, item := range matches {
		if len(item) < 2 {
			continue
		}
		name := strings.TrimSpace(item[1])
		if strings.HasPrefix(name, "os_") ||
			strings.HasPrefix(name, "host_") ||
			strings.HasPrefix(name, "k8s_") ||
			strings.HasPrefix(name, "service_") ||
			strings.HasPrefix(name, "cluster_") {
			return name
		}
	}
	return ""
}

func buildToolExecutionDirective(message, scene string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return ""
	}
	isInventoryOrDiag := strings.Contains(msg, "查看") ||
		strings.Contains(msg, "查询") ||
		strings.Contains(msg, "清单") ||
		strings.Contains(msg, "资源") ||
		strings.Contains(msg, "服务器") ||
		strings.Contains(msg, "主机") ||
		strings.Contains(msg, "cpu") ||
		strings.Contains(msg, "内存") ||
		strings.Contains(msg, "磁盘") ||
		strings.Contains(msg, "硬盘") ||
		strings.Contains(msg, "disk") ||
		strings.Contains(msg, "memory")
	if !isInventoryOrDiag {
		return ""
	}
	sceneLower := strings.ToLower(scene)
	if sceneLower != "" && sceneLower != "global" && !strings.Contains(sceneLower, "host") && !strings.Contains(sceneLower, "scene:hosts") {
		return ""
	}
	return `执行要求（必须遵守）:
1) 这是资源查询/诊断请求，必须先调用至少一个只读工具，再给出结论。
2) 不允许仅输出“我将调用某工具”的计划性文字。
3) 当用户提到具体主机名称（如“香港云服务器”）时，必须按顺序执行：
   - 先调用 host_list_inventory(keyword=<主机名关键词>) 获取准确主机信息与 ID。
   - 再调用 host_ssh_exec_readonly(host_id=<命中的ID>, command="df -h") 查询磁盘使用。
4) 若第 2 步出现 SSH 认证失败，必须在结论中明确给出：
   - 已命中的主机 ID/名称/IP
   - 失败原因为认证失败
   - 建议下一步（更新凭据或检查 ssh_key_id/password）。`
}

func composePromptDirectives(directives ...string) string {
	out := make([]string, 0, len(directives))
	for _, directive := range directives {
		v := strings.TrimSpace(directive)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return strings.Join(out, "\n\n")
}

func (h *handler) buildToolContext(ctx context.Context, uid uint64, approvalToken, scene string, runtime map[string]any, emit func(event string, payload gin.H) bool, tracker *toolEventTracker) context.Context {
	ctx = tools.WithToolUser(ctx, uid, approvalToken)
	ctx = tools.WithToolRuntimeContext(ctx, runtime)
	ctx = tools.WithToolMemoryAccessor(ctx, &toolMemoryAccessor{
		store: h.store,
		uid:   uid,
		scene: scene,
	})
	ctx = tools.WithToolPolicyChecker(ctx, h.toolPolicy)
	ctx = tools.WithToolEventEmitter(ctx, func(event string, payload any) {
		switch event {
		case "tool_call", "tool_result":
			pm := toPayloadMap(payload)
			toolName := strings.TrimSpace(toString(pm["tool"]))
			callID := strings.TrimSpace(toString(pm["call_id"]))
			switch event {
			case "tool_call":
				tracker.noteCall(callID, toolName)
			case "tool_result":
				tracker.noteResult(callID, toolName)
			}
			_ = emit(event, gin.H{
				"tool":             toolName,
				"call_id":          callID,
				"payload":          pm,
				"ts":               time.Now().UTC().Format(time.RFC3339Nano),
				"retry":            pm["retry"],
				"param_resolution": pm["param_resolution"],
			})
		}
	})
	return ctx
}

func toPayloadMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{"raw": v}
}

func mustJSON(v any) string {
	raw, _ := jsonMarshal(v)
	return raw
}

func (h *handler) buildConversationMessages(history []map[string]any, originalMsg, finalPrompt string) []*schema.Message {
	if len(history) == 0 {
		return []*schema.Message{schema.UserMessage(finalPrompt)}
	}
	start := 0
	if len(history) > 20 {
		start = len(history) - 20
	}
	out := make([]*schema.Message, 0, len(history[start:])+1)
	for i := start; i < len(history); i++ {
		role := strings.TrimSpace(toString(history[i]["role"]))
		content := toString(history[i]["content"])
		if role == "assistant" && strings.TrimSpace(content) != "" {
			out = append(out, schema.AssistantMessage(content, nil))
			continue
		}
		if role == "user" {
			if i == len(history)-1 && content == originalMsg {
				out = append(out, schema.UserMessage(finalPrompt))
			} else {
				out = append(out, schema.UserMessage(content))
			}
		}
	}
	if len(out) == 0 {
		out = append(out, schema.UserMessage(finalPrompt))
	}
	return out
}
