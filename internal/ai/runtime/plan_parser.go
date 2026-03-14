package runtime

import (
	"encoding/json"
	"strings"
	"unicode"
)

// PlanParser 从 planner 文本输出中保守提取结构化步骤。
// 仅识别稳定的 numbered list / markdown list；提取失败时返回 false。
type PlanParser struct{}

func NewPlanParser() *PlanParser {
	return &PlanParser{}
}

func (p *PlanParser) Extract(planID, turnID, raw string) (PlanEvent, bool) {
	if steps, ok := extractJSONPlanSteps(raw); ok {
		return PlanEvent{
			PlanID:  strings.TrimSpace(planID),
			TurnID:  strings.TrimSpace(turnID),
			Source:  "planner_json",
			Summary: steps[0].Content,
			Steps:   steps,
			Raw:     strings.TrimSpace(raw),
		}, true
	}

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	steps := make([]PlanStep, 0, len(lines))
	for _, line := range lines {
		content, ok := normalizePlanLine(line)
		if !ok {
			continue
		}
		steps = append(steps, PlanStep{
			ID:      "step-" + itoa(len(steps)+1),
			Title:   content,
			Content: content,
			Status:  string(StepPending),
		})
	}
	if len(steps) < 2 {
		return PlanEvent{}, false
	}
	return PlanEvent{
		PlanID:  strings.TrimSpace(planID),
		TurnID:  strings.TrimSpace(turnID),
		Source:  "planner_text",
		Summary: steps[0].Content,
		Steps:   steps,
		Raw:     strings.TrimSpace(raw),
	}, true
}

func extractJSONPlanSteps(raw string) ([]PlanStep, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	}
	if strings.HasPrefix(trimmed, "```") {
		parts := strings.Split(trimmed, "\n")
		if len(parts) >= 3 {
			trimmed = strings.Join(parts[1:len(parts)-1], "\n")
		}
	}

	type jsonStep struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Content  string `json:"content"`
		ToolHint string `json:"toolHint"`
	}

	var parsed []jsonStep
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, false
	}
	if len(parsed) < 2 {
		return nil, false
	}

	steps := make([]PlanStep, 0, len(parsed))
	for index, step := range parsed {
		content := strings.TrimSpace(firstNonEmptyString(step.Content, step.Title))
		if content == "" {
			continue
		}
		steps = append(steps, PlanStep{
			ID:       firstNonEmptyString(step.ID, "step-"+itoa(index+1)),
			Title:    strings.TrimSpace(step.Title),
			Content:  content,
			Status:   string(StepPending),
			ToolHint: strings.TrimSpace(step.ToolHint),
		})
	}
	if len(steps) < 2 {
		return nil, false
	}
	return steps, true
}

func normalizePlanLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", false
	}

	switch {
	case strings.HasPrefix(trimmed, "- "), strings.HasPrefix(trimmed, "* "), strings.HasPrefix(trimmed, "+ "):
		content := strings.TrimSpace(trimmed[2:])
		return content, content != ""
	case hasOrderedListPrefix(trimmed):
		content := strings.TrimSpace(stripOrderedListPrefix(trimmed))
		return content, content != ""
	default:
		return "", false
	}
}

func hasOrderedListPrefix(line string) bool {
	seenDigit := false
	for i, r := range line {
		if unicode.IsDigit(r) {
			seenDigit = true
			continue
		}
		if !seenDigit {
			return false
		}
		if (r == '.' || r == ')') && i+1 < len(line) {
			return true
		}
		return false
	}
	return false
}

func stripOrderedListPrefix(line string) string {
	for i, r := range line {
		if r == '.' || r == ')' {
			if i+1 < len(line) {
				return line[i+1:]
			}
			return ""
		}
	}
	return line
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for v > 0 {
		pos--
		buf[pos] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[pos:])
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
