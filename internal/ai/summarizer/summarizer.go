// Package summarizer 实现 AI 编排的总结阶段。
//
// Summarizer 负责汇总执行结果，生成用户可见的最终答案。
package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/ai/availability"
	"github.com/cy77cc/OpsPilot/internal/ai/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

// Input 是总结器的输入结构。
type Input struct {
	Message string                 // 用户原始消息
	Plan    *planner.ExecutionPlan // 执行计划
	State   runtime.ExecutionState // 执行状态
	Steps   []executor.StepResult  // 步骤结果列表
}

// Summarizer 是总结器核心。
type Summarizer struct {
	runner *adk.Runner
	runFn  func(context.Context, Input, func(string), func(string)) (string, error)
}

// New 创建新的总结器实例。
func New(runner *adk.Runner) *Summarizer {
	return &Summarizer{runner: runner}
}

// NewWithFunc 使用自定义执行函数创建总结器。
func NewWithFunc(runFn func(context.Context, Input, func(string), func(string)) (string, error)) *Summarizer {
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

func (s *Summarizer) Summarize(ctx context.Context, in Input) (string, error) {
	return s.summarize(ctx, in, nil, nil)
}

func (s *Summarizer) SummarizeStream(ctx context.Context, in Input, onThinkingDelta func(string), onAnswerDelta func(string)) (string, error) {
	return s.summarize(ctx, in, onThinkingDelta, onAnswerDelta)
}

func (s *Summarizer) summarize(ctx context.Context, in Input, onThinkingDelta func(string), onAnswerDelta func(string)) (string, error) {
	if s != nil && s.runFn != nil {
		return s.runFn(ctx, in, onThinkingDelta, onAnswerDelta)
	}
	if s == nil || s.runner == nil {
		return "", &UnavailableError{
			Code:              "summarizer_runner_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerSummarizer),
		}
	}
	message := buildPromptInput(ctx, in)
	if message != nil {
		return "", fmt.Errorf("构建 prompt 失败")
	}
	raw, err := runADKSummarizer(ctx, s.runner, message, onThinkingDelta, onAnswerDelta)
	if err != nil {
		return "", &UnavailableError{
			Code:              "summarizer_model_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerSummarizer),
			Cause:             err,
		}
	}
	return normalizeSummary(raw)
}

func buildPromptInput(ctx context.Context, in Input) []*schema.Message {
	template := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage(SystemPrompt()),
		schema.UserMessage(userPrompt()),
	)

	message, err := template.Format(ctx, map[string]any{
		"Message": in.Message,
		"Steps":   in.Steps,
		"Plan":    in.Plan,
		"State":   in.State,
	})
	if err != nil {
		return nil
	}
	return message
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

func normalizeSummary(raw string) (string, error) {
	text := strings.TrimSpace(raw)
	text = strings.TrimSpace(strings.TrimPrefix(text, "```"))
	text = strings.TrimSpace(strings.TrimPrefix(text, "json"))
	text = strings.TrimSpace(strings.TrimSuffix(text, "```"))
	if text == "" {
		return "", &UnavailableError{
			Code:              "summarizer_invalid_output",
			UserVisibleReason: availability.InvalidOutputMessage(availability.LayerSummarizer),
			Cause:             fmt.Errorf("summary output is empty"),
		}
	}
	return text, nil
}
