package experts

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

type ResultAggregator struct {
	mode       AggregationMode
	aggregator AggregatorLLM
}

func NewResultAggregator(mode AggregationMode, chatModel AggregatorLLM) *ResultAggregator {
	if mode == "" {
		mode = AggregationTemplate
	}
	return &ResultAggregator{
		mode:       mode,
		aggregator: chatModel,
	}
}

func (a *ResultAggregator) Aggregate(ctx context.Context, results []ExpertResult, originalQuery string) (string, error) {
	if len(results) == 0 {
		return "未获得专家输出，请补充上下文后重试。", nil
	}
	if a == nil || a.mode == AggregationTemplate {
		return a.aggregateByTemplate(results), nil
	}
	if a.mode == AggregationLLM {
		return a.aggregateByLLM(ctx, results, originalQuery)
	}
	return a.aggregateByTemplate(results), nil
}

func (a *ResultAggregator) aggregateByTemplate(results []ExpertResult) string {
	var b strings.Builder
	b.WriteString("诊断报告\n\n")
	for _, item := range results {
		if strings.TrimSpace(item.ExpertName) == "" {
			continue
		}
		b.WriteString("【")
		b.WriteString(item.ExpertName)
		b.WriteString("】\n")
		if item.Error != nil {
			b.WriteString("执行失败: ")
			b.WriteString(item.Error.Error())
			b.WriteString("\n\n")
			continue
		}
		b.WriteString(strings.TrimSpace(item.Output))
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

func (a *ResultAggregator) aggregateByLLM(ctx context.Context, results []ExpertResult, originalQuery string) (string, error) {
	if a == nil || a.aggregator == nil {
		return a.aggregateByTemplate(results), nil
	}
	prompt := "请将多个专家结果合并为最终答复，保持结构化、可执行。\n"
	prompt += "用户请求: " + strings.TrimSpace(originalQuery) + "\n\n"
	for _, item := range results {
		prompt += fmt.Sprintf("[%s]\n%s\n\n", item.ExpertName, strings.TrimSpace(item.Output))
	}
	msg, err := a.aggregator.Generate(ctx, []*schema.Message{schema.UserMessage(prompt)})
	if err != nil {
		return "", err
	}
	if msg == nil {
		return "", fmt.Errorf("llm aggregation returned nil message")
	}
	return strings.TrimSpace(msg.Content), nil
}
