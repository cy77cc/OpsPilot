package runtime

import "testing"

func TestPlanParserExtractJSONCodeFence(t *testing.T) {
	t.Parallel()

	parser := NewPlanParser()
	plan, ok := parser.Extract("plan-1", "turn-1", "```json\n[{\"id\":\"step-1\",\"content\":\"检查集群状态\",\"toolHint\":\"get_cluster_info\"},{\"id\":\"step-2\",\"content\":\"获取 deployment 列表\",\"toolHint\":\"list_deployments\"}]\n```")
	if !ok {
		t.Fatalf("Extract() = false, want true")
	}
	if plan.Source != "planner_json" {
		t.Fatalf("Source = %q, want planner_json", plan.Source)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps len = %d, want 2", len(plan.Steps))
	}
	if plan.Steps[0].ToolHint != "get_cluster_info" {
		t.Fatalf("ToolHint = %q, want get_cluster_info", plan.Steps[0].ToolHint)
	}
}

func TestPlanParserExtractMarkdownList(t *testing.T) {
	t.Parallel()

	parser := NewPlanParser()
	plan, ok := parser.Extract("plan-2", "turn-2", "- 检查集群状态\n- 获取 deployment 列表\n- 输出结论")
	if !ok {
		t.Fatalf("Extract() = false, want true")
	}
	if plan.Source != "planner_text" {
		t.Fatalf("Source = %q, want planner_text", plan.Source)
	}
	if len(plan.Steps) != 3 {
		t.Fatalf("steps len = %d, want 3", len(plan.Steps))
	}
	if plan.Steps[1].Content != "获取 deployment 列表" {
		t.Fatalf("Content = %q, want 获取 deployment 列表", plan.Steps[1].Content)
	}
}

func TestPlanParserExtractRequiresAtLeastTwoSteps(t *testing.T) {
	t.Parallel()

	parser := NewPlanParser()
	if _, ok := parser.Extract("plan-3", "turn-3", "- 只有一步"); ok {
		t.Fatalf("Extract() = true, want false")
	}
}
