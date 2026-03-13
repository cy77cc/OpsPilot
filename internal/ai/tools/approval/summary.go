package approval

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

var placeholderRegexp = regexp.MustCompile(`\{([a-zA-Z0-9_.-]+)\}`)

type SummaryRenderer struct{}

func NewSummaryRenderer() *SummaryRenderer {
	return &SummaryRenderer{}
}

func (r *SummaryRenderer) Render(decision airuntime.ApprovalDecision, params map[string]any) string {
	template := strings.TrimSpace(decision.SummaryTemplate)
	if template != "" {
		if rendered := strings.TrimSpace(r.renderTemplate(template, decision, params)); rendered != "" {
			return rendered
		}
	}
	return r.generateActionSummary(decision, params)
}

func (r *SummaryRenderer) generateActionSummary(decision airuntime.ApprovalDecision, params map[string]any) string {
	verb := defaultVerb(decision.Tool.Name, decision.Tool.Mode)
	subject := summaryFirstValue(
		stringFrom(params, "name"),
		stringFrom(params, "target"),
		stringFrom(params, "service_name"),
		stringFrom(params, "service_id"),
		stringFrom(params, "deployment"),
		stringFrom(params, "deployment_name"),
		stringFrom(params, "pod"),
		stringFrom(params, "resource"),
	)
	if subject == "" {
		subject = summaryFirstValue(decision.Tool.DisplayName, decision.Tool.Name, "target resource")
	}

	parts := []string{fmt.Sprintf("%s %s", verb, subject)}
	if namespace := stringFrom(params, "namespace"); namespace != "" {
		parts = append(parts, fmt.Sprintf("(namespace: %s)", namespace))
	}
	if replicas := stringFrom(params, "replicas"); replicas != "" {
		parts = append(parts, fmt.Sprintf("to %s replicas", replicas))
	}
	if command := stringFrom(params, "command"); command != "" {
		parts = append(parts, fmt.Sprintf("command: %s", command))
	}
	if extras := compactParamSummary(params); extras != "" {
		parts = append(parts, extras)
	}
	return strings.Join(parts, " ")
}

func (r *SummaryRenderer) renderTemplate(template string, decision airuntime.ApprovalDecision, params map[string]any) string {
	values := map[string]string{
		"tool_name":         decision.Tool.Name,
		"tool_display_name": summaryFirstValue(decision.Tool.DisplayName, decision.Tool.Name),
		"mode":              decision.Tool.Mode,
		"risk":              decision.Tool.Risk,
		"risk_level":        decision.Tool.Risk,
		"environment":       decision.Environment,
	}
	for key, value := range params {
		values[key] = stringify(value)
	}
	return placeholderRegexp.ReplaceAllStringFunc(template, func(token string) string {
		matches := placeholderRegexp.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		if value, ok := values[matches[1]]; ok && value != "" {
			return value
		}
		return ""
	})
}

func defaultVerb(toolName, mode string) string {
	toolName = strings.ToLower(strings.TrimSpace(toolName))
	switch {
	case strings.Contains(toolName, "delete"):
		return "Delete"
	case strings.Contains(toolName, "restart"):
		return "Restart"
	case strings.Contains(toolName, "scale"):
		return "Scale"
	case strings.Contains(toolName, "deploy"), strings.Contains(toolName, "apply"):
		return "Apply"
	case strings.Contains(toolName, "exec"), strings.Contains(toolName, "command"):
		return "Run"
	case strings.EqualFold(mode, "mutating"):
		return "Execute"
	default:
		return "Run"
	}
}

func compactParamSummary(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for key := range params {
		switch key {
		case "name", "target", "namespace", "replicas", "command":
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	summary := make([]string, 0, minInt(2, len(keys)))
	for _, key := range keys[:minInt(2, len(keys))] {
		summary = append(summary, fmt.Sprintf("%s=%s", key, stringify(params[key])))
	}
	return strings.Join(summary, ", ")
}

func stringify(v any) string {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value)
	default:
		raw, _ := json.Marshal(value)
		return strings.TrimSpace(string(raw))
	}
}

func stringFrom(params map[string]any, key string) string {
	return stringify(params[key])
}

func summaryFirstValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
