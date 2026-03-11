// Package summarizer 实现 AI 编排的总结阶段。
//
// Summarizer 负责汇总执行结果，生成用户可见的最终答案。
// 输出包含摘要、关键发现、建议和重规划提示。
package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/availability"
	"github.com/cy77cc/OpsPilot/internal/ai/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

// Input 是总结器的输入结构。
type Input struct {
	Message string               // 用户原始消息
	Plan    *planner.ExecutionPlan // 执行计划
	State   runtime.ExecutionState // 执行状态
	Steps   []executor.StepResult  // 步骤结果列表
}

// ReplanHint 表示重规划提示。
type ReplanHint struct {
	Reason          string   `json:"reason,omitempty"`          // 重规划原因
	Focus           string   `json:"focus,omitempty"`           // 关注点
	MissingEvidence []string `json:"missing_evidence,omitempty"` // 缺失证据
}

// SummaryOutput 表示总结输出。
type SummaryOutput struct {
	Summary               string      `json:"summary"`                     // 总结
	Headline              string      `json:"headline,omitempty"`          // 标题
	Conclusion            string      `json:"conclusion,omitempty"`        // 结论
	KeyFindings           []string    `json:"key_findings,omitempty"`      // 关键发现
	ResourceSummaries     []string    `json:"resource_summaries,omitempty"` // 资源摘要
	Recommendations       []string    `json:"recommendations,omitempty"`   // 建议
	RawOutputPolicy       string      `json:"raw_output_policy,omitempty"` // 原始输出策略
	NextActions           []string    `json:"next_actions,omitempty"`      // 后续动作
	NeedMoreInvestigation bool        `json:"need_more_investigation"`     // 是否需要更多调查
	Narrative             string      `json:"narrative"`                   // 自然语言描述
	ReplanHint            *ReplanHint `json:"replan_hint,omitempty"`       // 重规划提示
}

// Summarizer 是总结器核心。
type Summarizer struct {
	runner *adk.Runner                                             // ADK 运行器
	runFn  func(context.Context, Input, func(string)) (SummaryOutput, error) // 执行函数
}

// New 创建新的总结器实例。
func New(runner *adk.Runner) *Summarizer {
	return &Summarizer{runner: runner}
}

// NewWithFunc 使用自定义执行函数创建总结器。
func NewWithFunc(runFn func(context.Context, Input, func(string)) (SummaryOutput, error)) *Summarizer {
	return &Summarizer{runFn: runFn}
}

// UnavailableError 表示总结器不可用错误。
type UnavailableError struct {
	Code              string // 错误码
	UserVisibleReason string // 用户可见原因
	Cause             error  // 原始错误
}

func (e *UnavailableError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", strings.TrimSpace(e.Code), e.Cause)
	}
	return firstNonEmpty(e.Code, "summarizer_unavailable")
}

func (e *UnavailableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *UnavailableError) UserVisibleMessage() string {
	if e == nil {
		return availability.UnavailableMessage(availability.LayerSummarizer)
	}
	return firstNonEmpty(e.UserVisibleReason, availability.UnavailableMessage(availability.LayerSummarizer))
}

func (s *Summarizer) Summarize(ctx context.Context, in Input) (SummaryOutput, error) {
	return s.summarize(ctx, in, nil)
}

func (s *Summarizer) SummarizeStream(ctx context.Context, in Input, onDelta func(string)) (SummaryOutput, error) {
	return s.summarize(ctx, in, onDelta)
}

func (s *Summarizer) summarize(ctx context.Context, in Input, onDelta func(string)) (SummaryOutput, error) {
	if s != nil && s.runFn != nil {
		return s.runFn(ctx, in, onDelta)
	}
	if s == nil || s.runner == nil {
		return SummaryOutput{}, &UnavailableError{
			Code:              "summarizer_runner_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerSummarizer),
		}
	}
	raw, err := runADKSummarizer(ctx, s.runner, buildPromptInput(in), onDelta)
	if err != nil {
		return SummaryOutput{}, &UnavailableError{
			Code:              "summarizer_model_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerSummarizer),
			Cause:             err,
		}
	}
	parsed, err := parseSummaryOutput(raw)
	if err != nil {
		return normalizeSummary(buildPlainTextSummaryOutput(raw))
	}
	return normalizeSummary(parsed)
}

func buildPromptInput(in Input) string {
	data, _ := json.Marshal(in)
	return string(data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeSummary(parsed SummaryOutput) (SummaryOutput, error) {
	parsed.Summary = firstNonEmpty(parsed.Summary, parsed.Headline, parsed.Conclusion, parsed.Narrative)
	parsed.Headline = firstNonEmpty(parsed.Headline, parsed.Summary, parsed.Conclusion)
	parsed.Conclusion = firstNonEmpty(parsed.Conclusion, parsed.Summary, parsed.Narrative)
	if !hasRenderableSummary(parsed) {
		return SummaryOutput{}, &UnavailableError{
			Code:              "summarizer_invalid_output",
			UserVisibleReason: availability.InvalidOutputMessage(availability.LayerSummarizer),
			Cause:             fmt.Errorf("summary output missing renderable fields"),
		}
	}
	if strings.TrimSpace(parsed.RawOutputPolicy) == "" {
		parsed.RawOutputPolicy = "summary_only"
	}
	parsed.KeyFindings = dedupe(parsed.KeyFindings)
	parsed.ResourceSummaries = dedupe(parsed.ResourceSummaries)
	parsed.Recommendations = dedupe(parsed.Recommendations)
	parsed.NextActions = dedupe(parsed.NextActions)
	return parsed, nil
}

func parseSummaryOutput(raw string) (SummaryOutput, error) {
	candidates := []string{
		strings.TrimSpace(raw),
		extractFencedJSON(raw),
		extractJSONObject(raw),
	}
	var lastErr error
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		var parsed SummaryOutput
		if err := json.Unmarshal([]byte(candidate), &parsed); err == nil {
			return parsed, nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("summary output is empty")
	}
	return SummaryOutput{}, lastErr
}

func buildPlainTextSummaryOutput(raw string) SummaryOutput {
	text := strings.TrimSpace(raw)
	text = strings.TrimSpace(strings.TrimPrefix(text, "```"))
	text = strings.TrimSpace(strings.TrimPrefix(text, "json"))
	text = strings.TrimSpace(strings.TrimSuffix(text, "```"))
	return SummaryOutput{
		Summary:    text,
		Headline:   text,
		Conclusion: text,
		Narrative:  text,
	}
}

func hasRenderableSummary(parsed SummaryOutput) bool {
	if firstNonEmpty(parsed.Summary, parsed.Headline, parsed.Conclusion, parsed.Narrative) != "" {
		return true
	}
	if len(parsed.KeyFindings) > 0 || len(parsed.ResourceSummaries) > 0 || len(parsed.Recommendations) > 0 || len(parsed.NextActions) > 0 {
		return true
	}
	if parsed.ReplanHint != nil && (parsed.ReplanHint.Reason != "" || parsed.ReplanHint.Focus != "" || len(parsed.ReplanHint.MissingEvidence) > 0) {
		return true
	}
	return false
}

func extractFencedJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if !strings.Contains(trimmed, "```") {
		return ""
	}
	parts := strings.Split(trimmed, "```")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		part = strings.TrimPrefix(part, "json")
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "{") || strings.HasPrefix(part, "[") {
			return part
		}
	}
	return ""
}

func extractJSONObject(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	start := strings.Index(trimmed, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(trimmed); i++ {
		ch := trimmed[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return trimmed[start : i+1]
			}
		}
	}
	return ""
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
