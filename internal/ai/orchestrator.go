package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	adkcore "github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/service/ai/knowledge"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
)

type runnerAPI interface {
	ToolMetas() []aitools.ToolMeta
	Query(ctx context.Context, sessionID, message string, opts ...adkcore.AgentRunOption) *adkcore.AsyncIterator[*adkcore.AgentEvent]
	Resume(ctx context.Context, checkpointID string, targets map[string]any, opts ...adkcore.AgentRunOption) (*adkcore.AsyncIterator[*adkcore.AgentEvent], error)
	Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error)
}

type sessionStore interface {
	CurrentSession(userID uint64, scene string) (*logic.AISession, bool)
	AppendMessage(userID uint64, scene, sessionID string, message map[string]any) (*logic.AISession, error)
}

type ChatStreamRequest struct {
	UserID    uint64
	SessionID string
	Message   string
	Context   map[string]any
}

// Orchestrator is the high-level AI core entrypoint for gateway chat and resume flows.
type Orchestrator struct {
	runner   runnerAPI
	sessions sessionStore
	runtime  *logic.RuntimeStore
	control  *ControlPlane
}

func NewOrchestrator(runner runnerAPI, sessions sessionStore, runtime *logic.RuntimeStore, control *ControlPlane) *Orchestrator {
	return &Orchestrator{
		runner:   runner,
		sessions: sessions,
		runtime:  runtime,
		control:  control,
	}
}

func (o *Orchestrator) ChatStream(ctx context.Context, req ChatStreamRequest, emit func(string, map[string]any) bool) error {
	if o == nil || o.runner == nil {
		return errors.New("ai adk agent not initialized")
	}
	if o.sessions == nil || o.runtime == nil {
		return errors.New("ai control plane not initialized")
	}

	msg := strings.TrimSpace(req.Message)
	scene := logic.NormalizeScene(logic.ToString(req.Context["scene"]))
	sid := strings.TrimSpace(req.SessionID)
	if sid == "" {
		if session, ok := o.sessions.CurrentSession(req.UserID, scene); ok {
			sid = session.ID
		} else {
			sid = formatID("sess", time.Now())
		}
	}

	userTime := time.Now()
	session, err := o.sessions.AppendMessage(req.UserID, scene, sid, map[string]any{
		"id":        formatID("u", userTime),
		"role":      "user",
		"content":   msg,
		"timestamp": userTime,
	})
	if err != nil {
		_ = emit("error", map[string]any{"message": err.Error()})
		return nil
	}
	if !emit("meta", map[string]any{"sessionId": session.ID, "createdAt": session.CreatedAt}) {
		return nil
	}

	emitWithSession := func(event string, payload map[string]any) bool {
		if payload == nil {
			payload = map[string]any{}
		}
		switch event {
		case "approval_required", "review_required", "interrupt_required":
			payload["sessionId"] = sid
			payload["checkpoint_id"] = sid
		}
		return emit(event, payload)
	}

	approvalToken := strings.TrimSpace(logic.ToString(req.Context["approval_token"]))
	tracker := newToolEventTracker()
	o.runtime.RememberContext(req.UserID, scene, extractResourceContext(req.Context, msg))
	streamCtx := o.buildToolContext(ctx, req.UserID, approvalToken, scene, msg, req.Context, emitWithSession, tracker)
	prompt := o.buildPrompt(msg, scene, req.Context)

	iter := o.runner.Query(streamCtx, sid, prompt)
	var assistantContent strings.Builder
	var reasoningContent strings.Builder
	var fatalErr *streamErrorPayload
	var finalOnce sync.Once
	emitFinal := func(event string, payload map[string]any) {
		finalOnce.Do(func() {
			_ = emit(event, payload)
		})
	}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if err := o.processADKEvent(emitWithSession, tracker, event, &assistantContent, &reasoningContent); err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			fatalErr = &streamErrorPayload{Code: "stream_interrupted", Message: err.Error(), Recoverable: true}
			break
		}
	}

	summary := tracker.summary()
	content := strings.TrimSpace(assistantContent.String())
	if content == "" {
		if fatalErr != nil {
			content = fmt.Sprintf("本轮执行未完整结束：%s", fatalErr.Message)
		} else {
			content = "无输出。"
		}
	}

	assistantTime := time.Now()
	session, err = o.sessions.AppendMessage(req.UserID, scene, sid, map[string]any{
		"id":        formatID("a", assistantTime),
		"role":      "assistant",
		"content":   content,
		"thinking":  strings.TrimSpace(reasoningContent.String()),
		"timestamp": assistantTime,
	})
	if err != nil {
		emitFinal("error", map[string]any{"message": err.Error()})
		return nil
	}

	streamState := resolveStreamState(fatalErr, summary)
	recs := o.refreshSuggestions(req.UserID, scene, content)
	if fatalErr != nil {
		emitFinal("error", map[string]any{
			"code":         fatalErr.Code,
			"message":      fatalErr.Message,
			"recoverable":  fatalErr.Recoverable,
			"tool_summary": summary,
		})
	}
	emitFinal("done", buildDonePayload(session, streamState, summary, recs))
	return nil
}

func (o *Orchestrator) ResumePayload(ctx context.Context, checkpointID string, targets map[string]any) (map[string]any, error) {
	if o == nil || o.runner == nil {
		return nil, fmt.Errorf("ai adk agent not initialized")
	}
	iter, err := o.runner.Resume(ctx, checkpointID, targets)
	if err != nil {
		return nil, err
	}

	var output strings.Builder
	for {
		ev, ok := iter.Next()
		if !ok {
			break
		}
		if ev == nil {
			continue
		}
		if ev.Err != nil {
			var sig *adkcore.InterruptSignal
			if errors.As(ev.Err, &sig) {
				payload := interruptPayloadFromSignal(sig)
				payload["resumed"] = false
				payload["interrupted"] = true
				payload["sessionId"] = checkpointID
				return payload, nil
			}
			return nil, ev.Err
		}
		if ev.Action != nil && ev.Action.Interrupted != nil {
			payload := map[string]any{
				"resumed":            false,
				"interrupted":        true,
				"sessionId":          checkpointID,
				"interrupt_targets":  interruptRootTargets(ev.Action.Interrupted.InterruptContexts),
				"interrupt_contexts": ev.Action.Interrupted.InterruptContexts,
			}
			switch data := ev.Action.Interrupted.Data.(type) {
			case *aitools.ApprovalInfo:
				payload["tool"] = data.ToolName
				payload["arguments"] = data.ArgumentsInJSON
				payload["risk"] = data.Risk
				payload["preview"] = data.Preview
				payload["approval_required"] = true
			case *aitools.ReviewEditInfo:
				payload["tool"] = data.ToolName
				payload["arguments"] = data.ArgumentsInJSON
				payload["review_required"] = true
			default:
				if data != nil {
					payload["message"] = fmt.Sprintf("interrupt: %v", data)
				}
			}
			return payload, nil
		}
		if ev.Output != nil && ev.Output.MessageOutput != nil && ev.Output.MessageOutput.Message != nil {
			output.WriteString(ev.Output.MessageOutput.Message.Content)
		}
	}

	return map[string]any{
		"resumed":   true,
		"content":   strings.TrimSpace(output.String()),
		"sessionId": checkpointID,
	}, nil
}

func (o *Orchestrator) buildPrompt(message, scene string, runtime map[string]any) string {
	prompt := message
	directive := composePromptDirectives(
		buildStrictToolUseDirective(toolNamesFromMetas(o.runner.ToolMetas())),
		buildToolExecutionDirective(message, scene),
		knowledge.BuildHelpKnowledgeDirective(message),
	)
	if directive != "" {
		prompt = directive + "\n\n用户问题:\n" + message
	}
	if len(runtime) > 0 {
		prompt = message + "\n\n上下文:\n" + mustJSON(runtime)
		if directive != "" {
			prompt = directive + "\n\n用户问题:\n" + message + "\n\n上下文:\n" + mustJSON(runtime)
		}
	}
	return prompt
}

func (o *Orchestrator) buildToolContext(ctx context.Context, uid uint64, approvalToken, scene, userMessage string, runtime map[string]any, emit func(string, map[string]any) bool, tracker *toolEventTracker) context.Context {
	ctx = aitools.WithToolUser(ctx, uid, approvalToken)
	normalized := normalizeRuntimeContext(runtime)
	if _, ok := normalized["require_confirmation"]; !ok {
		normalized["require_confirmation"] = true
	}
	for k, v := range o.runtime.GetRememberedContext(uid, scene) {
		if strings.TrimSpace(logic.ToString(normalized[k])) == "" {
			normalized[k] = v
		}
	}
	for k, v := range resolveReferencePronouns(userMessage, o.runtime.GetRememberedContext(uid, scene)) {
		if strings.TrimSpace(logic.ToString(normalized[k])) == "" {
			normalized[k] = v
		}
	}
	ctx = aitools.WithToolRuntimeContext(ctx, normalized)
	ctx = aitools.WithToolMemoryAccessor(ctx, logic.NewToolMemoryAccessor(o.runtime, uid, scene))
	if o.control != nil {
		ctx = aitools.WithToolPolicyChecker(ctx, o.control.ToolPolicy)
	}
	ctx = aitools.WithToolEventEmitter(ctx, func(event string, payload any) {
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
			_ = emit(event, map[string]any{
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

func (o *Orchestrator) refreshSuggestions(uid uint64, scene, answer string) []logic.RecommendationRecord {
	scene = logic.NormalizeScene(scene)
	prompt := "你是 suggestion 智能体。基于下面回答提炼 3 条可执行建议，每条一行，格式为：标题|内容|相关度(0-1)|思考摘要（不超过60字）。回答内容如下：\n" + answer
	out := []logic.RecommendationRecord{}
	if o.runner != nil {
		msg, err := o.runner.Generate(context.Background(), []*schema.Message{schema.UserMessage(prompt)})
		if err == nil && msg != nil {
			lines := strings.Split(msg.Content, "\n")
			for _, line := range lines {
				trim := strings.TrimSpace(line)
				if trim == "" {
					continue
				}
				parts := strings.SplitN(trim, "|", 4)
				if len(parts) < 2 {
					continue
				}
				rel := 0.7
				if len(parts) >= 3 {
					if v, err := strconvParseFloat(strings.TrimSpace(parts[2])); err == nil {
						rel = v
					}
				}
				reasoning := ""
				if len(parts) == 4 {
					reasoning = strings.TrimSpace(parts[3])
				}
				out = append(out, logic.RecommendationRecord{
					ID:             formatID("rec", time.Now()),
					UserID:         uid,
					Scene:          scene,
					Type:           "suggestion",
					Title:          strings.TrimSpace(parts[0]),
					Content:        strings.TrimSpace(parts[1]),
					FollowupPrompt: strings.TrimSpace(parts[1]),
					Reasoning:      reasoning,
					Relevance:      rel,
					CreatedAt:      time.Now(),
				})
			}
		}
	}
	if len(out) == 0 {
		out = append(out, logic.RecommendationRecord{
			ID:             formatID("rec", time.Now()),
			UserID:         uid,
			Scene:          scene,
			Type:           "suggestion",
			Title:          "先做健康检查",
			Content:        "优先检查资源/日志，再进行部署或配置变更。",
			FollowupPrompt: "先帮我做一次资源健康检查，然后再给变更建议。",
			Reasoning:      "先确认现状可降低误操作风险，再执行变更更稳妥。",
			Relevance:      0.7,
			CreatedAt:      time.Now(),
		})
	}
	o.runtime.SetRecommendations(uid, scene, out)
	return out
}

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

func buildDonePayload(session *logic.AISession, streamState string, summary toolSummary, recs []logic.RecommendationRecord) map[string]any {
	return map[string]any{
		"session":              session,
		"stream_state":         streamState,
		"tool_summary":         summary,
		"turn_recommendations": recommendationPayload(recs),
	}
}

func recommendationPayload(items []logic.RecommendationRecord) []map[string]any {
	if len(items) == 0 {
		return nil
	}
	limit := len(items)
	if limit > 3 {
		limit = 3
	}
	out := make([]map[string]any, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, map[string]any{
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

func toolNamesFromMetas(metas []aitools.ToolMeta) []string {
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

func strconvParseFloat(v string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(v), 64)
}
