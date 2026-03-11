package summarizer

import (
	"context"
	"errors"
	"testing"
)

func TestSummarizerReturnsUnavailableWhenRunnerMissing(t *testing.T) {
	_, err := New(nil).Summarize(context.Background(), Input{})
	if err == nil {
		t.Fatalf("Summarize() error = nil, want UnavailableError")
	}
	var unavailable *UnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("Summarize() error = %v, want UnavailableError", err)
	}
	if unavailable.Code != "summarizer_runner_unavailable" {
		t.Fatalf("Code = %q, want summarizer_runner_unavailable", unavailable.Code)
	}
}

func TestNormalizeSummaryRequiresRenderableFields(t *testing.T) {
	_, err := normalizeSummary(SummaryOutput{})
	if err == nil {
		t.Fatalf("normalizeSummary() error = nil, want invalid output error")
	}
	var unavailable *UnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("normalizeSummary() error = %v, want UnavailableError", err)
	}
	if unavailable.Code != "summarizer_invalid_output" {
		t.Fatalf("Code = %q, want summarizer_invalid_output", unavailable.Code)
	}
}

func TestNormalizeSummaryPreservesModelSemantics(t *testing.T) {
	out, err := normalizeSummary(SummaryOutput{
		Summary:               "日志显示正常",
		Headline:              "服务已经恢复",
		Conclusion:            "服务可能已经恢复，建议继续观察",
		Narrative:             "根据当前结果，服务表现正常，但仍建议继续观察一段时间。",
		KeyFindings:           []string{"日志显示正常", "日志显示正常"},
		Recommendations:       []string{"继续观察", "继续观察"},
		NeedMoreInvestigation: true,
	})
	if err != nil {
		t.Fatalf("normalizeSummary() error = %v", err)
	}
	if !out.NeedMoreInvestigation {
		t.Fatalf("NeedMoreInvestigation = false, want true")
	}
	if out.Conclusion != "服务可能已经恢复，建议继续观察" {
		t.Fatalf("Conclusion should be preserved, got %q", out.Conclusion)
	}
	if out.Narrative != "根据当前结果，服务表现正常，但仍建议继续观察一段时间。" {
		t.Fatalf("Narrative should be preserved, got %q", out.Narrative)
	}
	if len(out.KeyFindings) != 1 || len(out.Recommendations) != 1 {
		t.Fatalf("summary fields should be deduped: %#v", out)
	}
	if out.RawOutputPolicy != "summary_only" {
		t.Fatalf("RawOutputPolicy = %q, want summary_only", out.RawOutputPolicy)
	}
}

func TestNormalizeSummaryAcceptsNarrativeOnlyOutput(t *testing.T) {
	out, err := normalizeSummary(SummaryOutput{
		Narrative: "根据执行结果，payment-api 当前状态正常。",
	})
	if err != nil {
		t.Fatalf("normalizeSummary() error = %v", err)
	}
	if out.Summary != "根据执行结果，payment-api 当前状态正常。" {
		t.Fatalf("Summary = %q", out.Summary)
	}
	if out.Conclusion != "根据执行结果，payment-api 当前状态正常。" {
		t.Fatalf("Conclusion = %q", out.Conclusion)
	}
}

func TestParseSummaryOutputAcceptsFencedJSON(t *testing.T) {
	out, err := parseSummaryOutput("```json\n{\"summary\":\"ok\",\"narrative\":\"done\"}\n```")
	if err != nil {
		t.Fatalf("parseSummaryOutput() error = %v", err)
	}
	if out.Summary != "ok" {
		t.Fatalf("Summary = %q", out.Summary)
	}
}

func TestParseSummaryOutputExtractsJSONFromMixedText(t *testing.T) {
	raw := "tool result:\n{\"summary\":\"ok\",\"headline\":\"done\",\"narrative\":\"complete\"}\nextra trailing note"
	out, err := parseSummaryOutput(raw)
	if err != nil {
		t.Fatalf("parseSummaryOutput() error = %v", err)
	}
	if out.Headline != "done" {
		t.Fatalf("Headline = %q", out.Headline)
	}
}

func TestSummarizeFallsBackToPlainTextModelOutput(t *testing.T) {
	s := NewWithFunc(func(_ context.Context, _ Input, _ func(string)) (SummaryOutput, error) {
		return normalizeSummary(buildPlainTextSummaryOutput("服务运行正常，根分区使用率 27%，当前无需处理。"))
	})
	out, err := s.Summarize(context.Background(), Input{})
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if out.Summary != "服务运行正常，根分区使用率 27%，当前无需处理。" {
		t.Fatalf("Summary = %q", out.Summary)
	}
	if out.Conclusion != out.Summary || out.Narrative != out.Summary {
		t.Fatalf("plain text summary should be preserved: %#v", out)
	}
}
