package rewrite

import (
	"context"
	"errors"
	"testing"
)

func TestEmitContentDeltaSupportsIncrementalChunks(t *testing.T) {
	var emitted []string
	aggregated := ""
	aggregated = emitContentDelta(aggregated, "先看", func(chunk string) {
		emitted = append(emitted, chunk)
	})
	aggregated = emitContentDelta(aggregated, "执行证据", func(chunk string) {
		emitted = append(emitted, chunk)
	})
	if aggregated != "执行证据" {
		t.Fatalf("aggregated = %q, want latest chunk snapshot", aggregated)
	}
	if len(emitted) != 2 || emitted[0] != "先看" || emitted[1] != "执行证据" {
		t.Fatalf("emitted = %#v", emitted)
	}
}

func TestEmitContentDeltaSupportsCumulativeSnapshotsWithWhitespace(t *testing.T) {
	var emitted []string
	aggregated := ""
	aggregated = emitContentDelta(aggregated, "hello", func(chunk string) {
		emitted = append(emitted, chunk)
	})
	aggregated = emitContentDelta(aggregated, "hello world", func(chunk string) {
		emitted = append(emitted, chunk)
	})
	if aggregated != "hello world" {
		t.Fatalf("aggregated = %q", aggregated)
	}
	if len(emitted) != 2 || emitted[0] != "hello" || emitted[1] != " world" {
		t.Fatalf("emitted = %#v", emitted)
	}
}

func TestMergeStreamContentSupportsIncrementalAndSnapshotChunks(t *testing.T) {
	aggregated := ""
	aggregated = mergeStreamContent(aggregated, "hello")
	aggregated = mergeStreamContent(aggregated, " world")
	if aggregated != "hello world" {
		t.Fatalf("incremental aggregated = %q", aggregated)
	}

	aggregated = ""
	aggregated = mergeStreamContent(aggregated, "hello")
	aggregated = mergeStreamContent(aggregated, "hello world")
	if aggregated != "hello world" {
		t.Fatalf("snapshot aggregated = %q", aggregated)
	}
}

func TestRewriteFailsExplicitlyWhenRunnerUnavailable(t *testing.T) {
	_, err := New(nil).Rewrite(context.Background(), Input{Message: "查看所有主机的状态"})
	if err == nil {
		t.Fatalf("Rewrite() error = nil, want unavailable error")
	}
	var unavailable *ModelUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("Rewrite() error = %v, want ModelUnavailableError", err)
	}
	if unavailable.Code != "rewrite_runner_unavailable" {
		t.Fatalf("Code = %q, want rewrite_runner_unavailable", unavailable.Code)
	}
	if unavailable.UserVisibleMessage() == "" {
		t.Fatalf("UserVisibleMessage() should not be empty")
	}
}

func TestParseModelOutputReturnsInvalidJSONError(t *testing.T) {
	base := buildBaseOutput(Input{Message: "查看所有主机的状态"})

	_, err := parseModelOutput(base, "{not-json")
	if err == nil {
		t.Fatalf("parseModelOutput() error = nil, want invalid json")
	}
	var unavailable *ModelUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("parseModelOutput() error = %v, want ModelUnavailableError", err)
	}
	if unavailable.Code != "rewrite_invalid_json" {
		t.Fatalf("Code = %q, want rewrite_invalid_json", unavailable.Code)
	}
}

func TestNormalizeOutputKeepsModelSemanticsAndRetrievalFields(t *testing.T) {
	base := buildBaseOutput(Input{
		Message: "查看状态",
		SelectedResources: []SelectedResource{
			{Type: "service", Name: "payment-api"},
		},
	})
	parsed := Output{
		OperationMode:     "investigate",
		DomainHints:       []string{"service", "observability", "service"},
		RetrievalIntent:   "runbook_lookup",
		RetrievalQueries:  []string{"payment-api health check", "payment-api health check"},
		RetrievalKeywords: []string{"runbook", "incident", "runbook"},
		KnowledgeScope:    []string{"service_runbooks", "incident_history", "service_runbooks"},
		RequiresRAG:       true,
		NormalizedRequest: NormalizedRequest{
			Intent: "service_health_check",
		},
		Assumptions: []string{"llm_inferred_scope"},
	}

	out := normalizeOutput(base, parsed)
	if out.RawUserInput != "查看状态" {
		t.Fatalf("RawUserInput = %q", out.RawUserInput)
	}
	if out.ResourceHints.ServiceName != "payment-api" {
		t.Fatalf("ResourceHints.ServiceName = %q, want payment-api", out.ResourceHints.ServiceName)
	}
	if out.NormalizedRequest.Intent != "service_health_check" {
		t.Fatalf("Intent = %q, want service_health_check", out.NormalizedRequest.Intent)
	}
	if out.RetrievalIntent != "runbook_lookup" {
		t.Fatalf("RetrievalIntent = %q, want runbook_lookup", out.RetrievalIntent)
	}
	if len(out.RetrievalQueries) != 1 || out.RetrievalQueries[0] != "payment-api health check" {
		t.Fatalf("RetrievalQueries = %#v", out.RetrievalQueries)
	}
	if len(out.RetrievalKeywords) != 2 {
		t.Fatalf("RetrievalKeywords = %#v, want 2 unique items", out.RetrievalKeywords)
	}
	if len(out.KnowledgeScope) != 2 {
		t.Fatalf("KnowledgeScope = %#v, want 2 unique items", out.KnowledgeScope)
	}
	if !out.RequiresRAG {
		t.Fatalf("RequiresRAG = false, want true")
	}
	if out.NormalizedGoal != "" {
		t.Fatalf("NormalizedGoal = %q, want empty because model did not provide it", out.NormalizedGoal)
	}
	if out.OperationMode != "investigate" {
		t.Fatalf("OperationMode = %q, want investigate", out.OperationMode)
	}
	if out.Narrative != "" {
		t.Fatalf("Narrative = %q, want model-owned empty narrative preserved", out.Narrative)
	}
}

func TestRewriteDetectsNumericResourceIDsFromSelection(t *testing.T) {
	base := buildBaseOutput(Input{
		Message: "查看 default 命名空间 mysql-0 的日志",
		SelectedResources: []SelectedResource{
			{Type: "cluster", ID: "12", Name: "prod-cluster"},
			{Type: "service", ID: "34", Name: "payment-api"},
			{Type: "host", ID: "56", Name: "node-a"},
		},
	})
	if base.ResourceHints.ClusterID != 12 {
		t.Fatalf("ClusterID = %d, want 12", base.ResourceHints.ClusterID)
	}
	if base.ResourceHints.ServiceID != 34 {
		t.Fatalf("ServiceID = %d, want 34", base.ResourceHints.ServiceID)
	}
	if base.ResourceHints.HostID != 56 {
		t.Fatalf("HostID = %d, want 56", base.ResourceHints.HostID)
	}
}

func TestNormalizeOutputDoesNotRebuildNarrativeOrGoalFromBase(t *testing.T) {
	base := buildBaseOutput(Input{
		Message: "查看 local 集群 pod 日志",
		SelectedResources: []SelectedResource{
			{Type: "cluster", ID: "12", Name: "local"},
		},
	})
	out := normalizeOutput(base, Output{
		RawUserInput: "查看 local 集群 pod 日志",
		Narrative:    "",
	})
	if out.NormalizedGoal != "" {
		t.Fatalf("NormalizedGoal = %q, want empty", out.NormalizedGoal)
	}
	if out.Narrative != "" {
		t.Fatalf("Narrative = %q, want empty", out.Narrative)
	}
	if out.ResourceHints.ClusterID != 12 {
		t.Fatalf("ClusterID = %d, want 12", out.ResourceHints.ClusterID)
	}
}
