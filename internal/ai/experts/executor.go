package experts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

type ExpertExecutor struct {
	registry ExpertRegistry
}

func NewExpertExecutor(registry ExpertRegistry) *ExpertExecutor {
	return &ExpertExecutor{registry: registry}
}

func (e *ExpertExecutor) ExecuteStep(ctx context.Context, step *ExecutionStep, priorResults []ExpertResult, baseMessage string) (*ExpertResult, error) {
	if step == nil {
		return nil, fmt.Errorf("execution step is nil")
	}
	if e == nil || e.registry == nil {
		return nil, fmt.Errorf("expert registry is nil")
	}
	exp, ok := e.registry.GetExpert(step.ExpertName)
	if !ok || exp == nil {
		return nil, fmt.Errorf("expert not found: %s", step.ExpertName)
	}
	msg := e.buildExpertMessage(step, priorResults, baseMessage)
	start := time.Now()
	if exp.Agent == nil {
		return &ExpertResult{
			ExpertName: exp.Name,
			Output:     "专家模型未初始化，返回静态诊断建议：" + msg,
			Duration:   time.Since(start),
		}, nil
	}
	resp, err := exp.Agent.Generate(ctx, []*schema.Message{
		schema.UserMessage(msg),
	})
	if err != nil {
		return &ExpertResult{
			ExpertName: exp.Name,
			Error:      err,
			Duration:   time.Since(start),
		}, err
	}
	output := ""
	if resp != nil {
		output = strings.TrimSpace(resp.Content)
	}
	return &ExpertResult{
		ExpertName: exp.Name,
		Output:     output,
		Duration:   time.Since(start),
	}, nil
}

func (e *ExpertExecutor) StreamStep(ctx context.Context, step *ExecutionStep, priorResults []ExpertResult, baseMessage string) (*schema.StreamReader[*schema.Message], error) {
	if step == nil {
		return nil, fmt.Errorf("execution step is nil")
	}
	if e == nil || e.registry == nil {
		return nil, fmt.Errorf("expert registry is nil")
	}
	exp, ok := e.registry.GetExpert(step.ExpertName)
	if !ok || exp == nil {
		return nil, fmt.Errorf("expert not found: %s", step.ExpertName)
	}
	msg := e.buildExpertMessage(step, priorResults, baseMessage)
	if exp.Agent == nil {
		return schema.StreamReaderFromArray([]*schema.Message{
			schema.AssistantMessage("专家模型未初始化，返回静态诊断建议："+msg, nil),
		}), nil
	}
	return exp.Agent.Stream(ctx, []*schema.Message{
		schema.UserMessage(msg),
	})
}

func (e *ExpertExecutor) buildExpertMessage(step *ExecutionStep, priorResults []ExpertResult, baseMessage string) string {
	var b strings.Builder
	base := strings.TrimSpace(baseMessage)
	if base != "" {
		b.WriteString("用户请求:\n")
		b.WriteString(base)
		b.WriteString("\n")
	}
	task := strings.TrimSpace(step.Task)
	if task != "" {
		b.WriteString("\n当前任务:\n")
		b.WriteString(task)
		b.WriteString("\n")
	}
	if len(step.ContextFrom) > 0 && len(priorResults) > 0 {
		b.WriteString("\n上游专家结果:\n")
		for _, idx := range step.ContextFrom {
			if idx < 0 || idx >= len(priorResults) {
				continue
			}
			item := priorResults[idx]
			b.WriteString("- ")
			b.WriteString(item.ExpertName)
			b.WriteString(": ")
			b.WriteString(strings.TrimSpace(item.Output))
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
}
