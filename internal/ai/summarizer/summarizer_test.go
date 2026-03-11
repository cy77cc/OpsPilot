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

func TestNormalizeSummaryRequiresRenderableText(t *testing.T) {
	_, err := normalizeSummary("")
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

func TestNormalizeSummaryPreservesPlainText(t *testing.T) {
	out, err := normalizeSummary("日志显示正常，服务可能已经恢复，建议继续观察。")
	if err != nil {
		t.Fatalf("normalizeSummary() error = %v", err)
	}
	if out != "日志显示正常，服务可能已经恢复，建议继续观察。" {
		t.Fatalf("Summary = %q", out)
	}
}

func TestNormalizeSummaryStripsCodeFenceWrapper(t *testing.T) {
	out, err := normalizeSummary("```json\n根据执行结果，payment-api 当前状态正常。\n```")
	if err != nil {
		t.Fatalf("normalizeSummary() error = %v", err)
	}
	if out != "根据执行结果，payment-api 当前状态正常。" {
		t.Fatalf("Summary = %q", out)
	}
}

func TestSummarizeFallsBackToPlainTextModelOutput(t *testing.T) {
	s := NewWithFunc(func(_ context.Context, _ Input, _ func(string), _ func(string)) (string, error) {
		return normalizeSummary("服务运行正常，根分区使用率 27%，当前无需处理。")
	})
	out, err := s.Summarize(context.Background(), Input{})
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if out != "服务运行正常，根分区使用率 27%，当前无需处理。" {
		t.Fatalf("Summary = %q", out)
	}
}

func TestSummarizeStreamSeparatesThinkingAndAnswer(t *testing.T) {
	var thinking string
	var answer string
	s := NewWithFunc(func(_ context.Context, _ Input, onThinkingDelta func(string), onAnswerDelta func(string)) (string, error) {
		if onThinkingDelta != nil {
			onThinkingDelta("先看执行证据。")
		}
		if onAnswerDelta != nil {
			onAnswerDelta("服务运行正常。")
		}
		return normalizeSummary("服务运行正常。")
	})
	out, err := s.SummarizeStream(context.Background(), Input{}, func(chunk string) {
		thinking += chunk
	}, func(chunk string) {
		answer += chunk
	})
	if err != nil {
		t.Fatalf("SummarizeStream() error = %v", err)
	}
	if thinking != "先看执行证据。" {
		t.Fatalf("thinking = %q", thinking)
	}
	if answer != "服务运行正常。" {
		t.Fatalf("answer = %q", answer)
	}
	if out != "服务运行正常。" {
		t.Fatalf("Summary = %q", out)
	}
}

func TestEmitSummaryDeltaSupportsIncrementalChunks(t *testing.T) {
	var emitted []string
	aggregated := ""
	aggregated = emitSummaryDelta(aggregated, "先看", func(chunk string) {
		emitted = append(emitted, chunk)
	})
	aggregated = emitSummaryDelta(aggregated, "执行证据", func(chunk string) {
		emitted = append(emitted, chunk)
	})
	if aggregated != "先看执行证据" {
		t.Fatalf("aggregated = %q", aggregated)
	}
	if len(emitted) != 2 || emitted[0] != "先看" || emitted[1] != "执行证据" {
		t.Fatalf("emitted = %#v", emitted)
	}
}

func TestEmitSummaryDeltaSupportsCumulativeSnapshots(t *testing.T) {
	var emitted []string
	aggregated := ""
	aggregated = emitSummaryDelta(aggregated, "先看", func(chunk string) {
		emitted = append(emitted, chunk)
	})
	aggregated = emitSummaryDelta(aggregated, "先看执行证据", func(chunk string) {
		emitted = append(emitted, chunk)
	})
	if aggregated != "先看执行证据" {
		t.Fatalf("aggregated = %q", aggregated)
	}
	if len(emitted) != 2 || emitted[0] != "先看" || emitted[1] != "执行证据" {
		t.Fatalf("emitted = %#v", emitted)
	}
}
