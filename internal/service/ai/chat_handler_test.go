package ai

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestToolEventTrackerSummary(t *testing.T) {
	tracker := newToolEventTracker()
	tracker.noteCall("c1", "os_get_cpu_mem")
	tracker.noteCall("c2", "os_get_cpu_mem")
	tracker.noteCall("c3", "k8s_get_events")
	tracker.noteResult("c1", "os_get_cpu_mem")

	summary := tracker.summary()
	if summary.Calls != 3 {
		t.Fatalf("expected 3 calls, got %d", summary.Calls)
	}
	if summary.Results != 1 {
		t.Fatalf("expected 1 result, got %d", summary.Results)
	}
	if len(summary.Missing) != 2 {
		t.Fatalf("expected 2 missing results, got %d", len(summary.Missing))
	}
	if len(summary.MissingCallIDs) != 2 {
		t.Fatalf("expected 2 missing call ids, got %d", len(summary.MissingCallIDs))
	}
}

func TestResolveStreamState(t *testing.T) {
	ok := resolveStreamState(nil, toolSummary{})
	if ok != "ok" {
		t.Fatalf("expected ok state, got %s", ok)
	}

	partial := resolveStreamState(nil, toolSummary{MissingCallIDs: []string{"c1"}})
	if partial != "partial" {
		t.Fatalf("expected partial state, got %s", partial)
	}

	failed := resolveStreamState(&streamErrorPayload{
		Code:        "stream_interrupted",
		Message:     "broken",
		Recoverable: true,
	}, toolSummary{})
	if failed != "failed" {
		t.Fatalf("expected failed state, got %s", failed)
	}
}

func TestRecommendationPayload(t *testing.T) {
	in := []recommendationRecord{
		{ID: "1", Type: "suggestion", Title: "A", Content: "a", Relevance: 0.8, FollowupPrompt: "next a"},
		{ID: "2", Type: "suggestion", Title: "B", Content: "b", Relevance: 0.7},
		{ID: "3", Type: "suggestion", Title: "C", Content: "c", Relevance: 0.6},
		{ID: "4", Type: "suggestion", Title: "D", Content: "d", Relevance: 0.5},
	}
	out := recommendationPayload(in)
	if len(out) != 3 {
		t.Fatalf("expected 3 items, got %d", len(out))
	}
	if out[0]["followup_prompt"] != "next a" {
		t.Fatalf("expected followup prompt to be kept")
	}
}

func TestDonePayloadIncludesTurnRecommendations(t *testing.T) {
	session := &aiSession{ID: "sess-1"}
	summary := toolSummary{Calls: 1, Results: 1}
	recs := []recommendationRecord{
		{ID: "1", Type: "suggestion", Title: "A", Content: "a", Relevance: 0.8, FollowupPrompt: "next a"},
	}

	out := buildDonePayload(session, "ok", summary, recs)
	items, ok := out["turn_recommendations"].([]gin.H)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one turn recommendation, got %#v", out["turn_recommendations"])
	}
	if items[0]["title"] != "A" {
		t.Fatalf("unexpected recommendation payload: %#v", items[0])
	}
}

func TestDetectUnresolvedToolIntent(t *testing.T) {
	tool := detectUnresolvedToolIntent("我将调用 host_list_inventory 查询主机", "")
	if tool != "host_list_inventory" {
		t.Fatalf("expected host_list_inventory, got %q", tool)
	}
	if got := detectUnresolvedToolIntent("普通思考文本", "没有工具名"); got != "" {
		t.Fatalf("expected empty tool, got %q", got)
	}
}

func TestBuildToolExecutionDirective(t *testing.T) {
	if got := buildToolExecutionDirective("查看香港云服务器硬盘使用情况", "scene:hosts"); got == "" {
		t.Fatalf("expected directive for host diagnostic query")
	}
	if got := buildToolExecutionDirective("向火山云服务器的 /tmp/test.txt 写入当前系统状态", "scene:hosts"); !strings.Contains(got, "host_batch_exec_apply") {
		t.Fatalf("expected mutating host execution directive, got %q", got)
	}
	if got := buildToolExecutionDirective("帮我写一段周报", "scene:hosts"); got != "" {
		t.Fatalf("expected empty directive for non-diagnostic query")
	}
	if got := buildToolExecutionDirective("查看服务资源", "scene:services"); got != "" {
		t.Fatalf("expected empty directive for non-host scene")
	}
}

func TestBuildHelpKnowledgeDirective(t *testing.T) {
	if got := buildHelpKnowledgeDirective("如何添加新主机并完成纳管"); got == "" {
		t.Fatalf("expected help directive for help intent")
	}
	if got := buildHelpKnowledgeDirective("帮我写一段周报"); got != "" {
		t.Fatalf("expected empty help directive for non-help intent")
	}
}

func TestComposePromptDirectives(t *testing.T) {
	out := composePromptDirectives("A", "", "B")
	if out != "A\n\nB" {
		t.Fatalf("unexpected directive compose result: %q", out)
	}
}

func TestBuildStrictToolUseDirective(t *testing.T) {
	out := buildStrictToolUseDirective([]string{
		"host_list_inventory",
		"host_ssh_exec_readonly",
		"service_deploy_apply",
	})
	if !strings.Contains(out, "只能调用以下真实存在的工具") {
		t.Fatalf("expected strict tool wording, got %q", out)
	}
	if !strings.Contains(out, "host_list_inventory") || !strings.Contains(out, "service_deploy_apply") {
		t.Fatalf("expected tool allowlist in directive, got %q", out)
	}
}

func TestMatchFAQKnowledgeFromEntries(t *testing.T) {
	entries := []faqKnowledgeEntry{
		{ID: "FAQ-001", Question: "登录失败提示账号或密码错误怎么办？", Answer: "先确认账号状态并重置密码。"},
		{ID: "FAQ-056", Question: "发布前必须检查什么？", Answer: "检查变更范围和回滚方案。"},
	}
	got, score := matchFAQKnowledgeFromEntries("登录失败怎么办", entries)
	if got == nil || got.ID != "FAQ-001" {
		t.Fatalf("expected FAQ-001, got %#v", got)
	}
	if score <= 0 {
		t.Fatalf("expected score > 0")
	}
}

func TestMatchFAQKnowledgeFromEntriesNoMatch(t *testing.T) {
	entries := []faqKnowledgeEntry{
		{ID: "FAQ-001", Question: "登录失败提示账号或密码错误怎么办？", Answer: "先确认账号状态并重置密码。"},
	}
	got, score := matchFAQKnowledgeFromEntries("帮我写周报", entries)
	if got != nil || score != 0 {
		t.Fatalf("expected no match, got=%#v score=%d", got, score)
	}
}
