package handler

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
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

func (h *AIHandler) chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		httpx.Fail(c, xcode.ParamError, "message is required")
		return
	}
	if h.svcCtx.AI == nil {
		httpx.Fail(c, xcode.ServerError, "AI service not available: LLM not configured or initialization failed. Please check llm.enable, llm.provider, llm.base_url, llm.api_key in config.")
		return
	}
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	h.chatWithADK(c, req, uid, msg)
}

func firstUserMessageContent(messages []map[string]any) string {
	for _, msg := range messages {
		if strings.TrimSpace(logic.ToString(msg["role"])) != "user" {
			continue
		}
		content := strings.TrimSpace(logic.ToString(msg["content"]))
		if content != "" {
			return content
		}
	}
	return ""
}

func recommendationPayload(items []RecommendationRecord) []gin.H {
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
	isHostAction := strings.Contains(msg, "写入") ||
		strings.Contains(msg, "创建文件") ||
		strings.Contains(msg, "写文件") ||
		strings.Contains(msg, "执行命令") ||
		strings.Contains(msg, "运行命令") ||
		strings.Contains(msg, "修改") ||
		strings.Contains(msg, "删除") ||
		strings.Contains(msg, "重启") ||
		strings.Contains(msg, "apply") ||
		strings.Contains(msg, "cat >") ||
		strings.Contains(msg, "echo >") ||
		strings.Contains(msg, "tee ")
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
	if isHostAction && (strings.Contains(msg, "主机") || strings.Contains(msg, "服务器") || strings.Contains(msg, "/tmp/") || strings.Contains(msg, "火山云服务器") || strings.Contains(msg, "香港云服务器")) {
		return `执行要求（必须遵守）:
1) 这是主机变更/写操作请求，不能只做口头说明，必须实际调用工具完成流程。
2) 若用户提到主机名称而不是 host_id，必须先调用 host_list_inventory(keyword=<主机名关键词>) 获取准确主机 ID。
3) 对命令执行类写操作，必须继续调用 host_batch_exec_preview 做预检查。
4) 如果预检查未阻断，必须继续调用 host_batch_exec_apply 发起实际执行；该工具需要审批时，应触发审批，不要在审批前自行结束对话。
5) 不允许只输出“应该使用 host_batch_exec_apply/preview”这类说明后停止。`
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

func buildStrictToolUseDirective(toolNames []string) string {
	if len(toolNames) == 0 {
		return ""
	}
	names := append([]string(nil), toolNames...)
	sort.Strings(names)
	return "工具调用硬约束:\n1) 只能调用以下真实存在的工具，工具名必须逐字匹配，禁止改名、缩写、拼接或臆造新工具。\n2) 如果下列工具都不适用，就直接说明原因或继续提问，不要假装调用工具。\n3) 不要把工具原始返回直接原样贴给用户，先提炼结论再引用关键字段。\n\n可用工具:\n- " + strings.Join(names, "\n- ")
}

func toolNamesFromMetas(metas []tools.ToolMeta) []string {
	out := make([]string, 0, len(metas))
	for _, meta := range metas {
		name := strings.TrimSpace(meta.Name)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	return out
}

func (h *AIHandler) buildToolContext(ctx context.Context, uid uint64, approvalToken, scene, userMessage string, runtime map[string]any, emit func(event string, payload gin.H) bool, tracker *toolEventTracker) context.Context {
	ctx = tools.WithToolUser(ctx, uid, approvalToken)
	normalized := normalizeRuntimeContext(runtime)
	if _, ok := normalized["require_confirmation"]; !ok {
		normalized["require_confirmation"] = true
	}
	for k, v := range h.runtime.GetRememberedContext(uid, scene) {
		if strings.TrimSpace(logic.ToString(normalized[k])) == "" {
			normalized[k] = v
		}
	}
	for k, v := range resolveReferencePronouns(userMessage, h.runtime.GetRememberedContext(uid, scene)) {
		if strings.TrimSpace(logic.ToString(normalized[k])) == "" {
			normalized[k] = v
		}
	}
	ctx = tools.WithToolRuntimeContext(ctx, normalized)
	ctx = tools.WithToolMemoryAccessor(ctx, logic.NewToolMemoryAccessor(h.runtime, uid, scene))
	ctx = tools.WithToolPolicyChecker(ctx, h.toolPolicy)
	ctx = tools.WithToolEventEmitter(ctx, func(event string, payload any) {
		switch event {
		case "tool_call", "tool_result":
			pm := toPayloadMap(payload)
			toolName := strings.TrimSpace(logic.ToString(pm["tool"]))
			callID := strings.TrimSpace(logic.ToString(pm["call_id"]))
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
		default:
			_ = emit(event, toPayloadMap(payload))
		}
	})
	return ctx
}

func normalizeRuntimeContext(runtime map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range runtime {
		out[k] = v
	}
	pageData, _ := runtime["pageData"].(map[string]any)
	for _, key := range []string{"host_id", "cluster_id", "service_id", "target_id", "namespace", "env", "runtime_type"} {
		if _, exists := out[key]; exists && strings.TrimSpace(logic.ToString(out[key])) != "" {
			continue
		}
		if pageData != nil {
			if v, ok := pageData[key]; ok {
				out[key] = v
			}
		}
	}
	return out
}

func extractResourceContext(runtime map[string]any, message string) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"host_id", "cluster_id", "service_id", "target_id", "namespace", "env"} {
		if v := strings.TrimSpace(logic.ToString(runtime[key])); v != "" {
			out[key] = runtime[key]
		}
	}
	pageData, _ := runtime["pageData"].(map[string]any)
	for _, key := range []string{"host_id", "cluster_id", "service_id", "target_id", "namespace", "env"} {
		if strings.TrimSpace(logic.ToString(out[key])) != "" {
			continue
		}
		if pageData != nil && strings.TrimSpace(logic.ToString(pageData[key])) != "" {
			out[key] = pageData[key]
		}
	}
	for _, pair := range []struct {
		key     string
		pattern *regexp.Regexp
	}{
		{"cluster_id", regexp.MustCompile(`cluster_id\s*=\s*(\d+)`)},
		{"service_id", regexp.MustCompile(`service_id\s*=\s*(\d+)`)},
		{"host_id", regexp.MustCompile(`host_id\s*=\s*(\d+)`)},
		{"target_id", regexp.MustCompile(`target_id\s*=\s*(\d+)`)},
	} {
		matched := pair.pattern.FindStringSubmatch(strings.ToLower(message))
		if len(matched) > 1 && strings.TrimSpace(matched[1]) != "" {
			out[pair.key] = matched[1]
		}
	}
	return out
}

func resolveReferencePronouns(message string, remembered map[string]any) map[string]any {
	out := map[string]any{}
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return out
	}
	if strings.Contains(msg, "刚才那个集群") || strings.Contains(msg, "那个集群") {
		if v := remembered["cluster_id"]; strings.TrimSpace(logic.ToString(v)) != "" {
			out["cluster_id"] = v
		}
	}
	if strings.Contains(msg, "刚才那个服务") || strings.Contains(msg, "那个服务") {
		if v := remembered["service_id"]; strings.TrimSpace(logic.ToString(v)) != "" {
			out["service_id"] = v
		}
	}
	if strings.Contains(msg, "刚才那台主机") || strings.Contains(msg, "那台主机") {
		if v := remembered["host_id"]; strings.TrimSpace(logic.ToString(v)) != "" {
			out["host_id"] = v
		}
	}
	if strings.Contains(msg, "刚才那个目标") || strings.Contains(msg, "那个目标") {
		if v := remembered["target_id"]; strings.TrimSpace(logic.ToString(v)) != "" {
			out["target_id"] = v
		}
	}
	return out
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

func (h *AIHandler) buildConversationMessages(history []map[string]any, originalMsg, finalPrompt string) []*schema.Message {
	if len(history) == 0 {
		return []*schema.Message{schema.UserMessage(finalPrompt)}
	}
	start := 0
	if len(history) > 20 {
		start = len(history) - 20
	}
	out := make([]*schema.Message, 0, len(history[start:])+1)
	for i := start; i < len(history); i++ {
		role := strings.TrimSpace(logic.ToString(history[i]["role"]))
		content := logic.ToString(history[i]["content"])
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
