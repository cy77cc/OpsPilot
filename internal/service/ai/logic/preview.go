// Package logic provides business logic for AI service
package logic

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"gorm.io/gorm"
)

// PreviewBuilder builds operation preview payloads before execution.
type PreviewBuilder struct {
	db    *gorm.DB
	metas map[string]tools.ToolMeta
}

// OperationPreview represents a preview of an operation
type OperationPreview struct {
	ToolName        string           `json:"tool_name"`
	ToolDescription string           `json:"tool_description"`
	Params          map[string]any   `json:"params"`
	RiskLevel       string           `json:"risk_level"`
	Mode            string           `json:"mode"`
	TargetResources []TargetResource `json:"target_resources"`
	ImpactScope     string           `json:"impact_scope"`
	PreviewDiff     string           `json:"preview_diff,omitempty"`
	Timeout         time.Duration    `json:"timeout"`
}

// TargetResource represents a target resource for an operation
type TargetResource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// NewPreviewBuilder creates a new PreviewBuilder
func NewPreviewBuilder(db *gorm.DB, metas []tools.ToolMeta) *PreviewBuilder {
	metaMap := make(map[string]tools.ToolMeta, len(metas))
	for _, item := range metas {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		metaMap[item.Name] = item
	}
	return &PreviewBuilder{db: db, metas: metaMap}
}

// BuildPreview builds an operation preview
func (b *PreviewBuilder) BuildPreview(toolName string, params map[string]any) OperationPreview {
	name := strings.TrimSpace(toolName)
	if params == nil {
		params = map[string]any{}
	}
	meta := b.metas[name]
	targets := b.ExtractTargetResources(name, params)
	preview := OperationPreview{
		ToolName:        name,
		ToolDescription: meta.Description,
		Params:          params,
		RiskLevel:       string(meta.Risk),
		Mode:            string(meta.Mode),
		TargetResources: targets,
		ImpactScope:     b.GenerateImpactScope(name, targets, params),
		Timeout:         defaultTimeoutForRisk(meta.Risk),
	}
	if diff := buildDeployDiffPreview(name, params); diff != "" {
		preview.PreviewDiff = diff
	}
	return preview
}

func (b *PreviewBuilder) ExtractTargetResources(toolName string, params map[string]any) []TargetResource {
	_ = b.db
	toolName = strings.TrimSpace(toolName)
	out := make([]TargetResource, 0, 4)
	appendOne := func(kind, id, name string) {
		if strings.TrimSpace(id) == "" {
			return
		}
		out = append(out, TargetResource{Type: kind, ID: id, Name: name})
	}

	if id, ok := intString(params["host_id"]); ok {
		appendOne("host", id, "")
	}
	for _, item := range intStringSlice(params["host_ids"]) {
		appendOne("host", item, "")
	}
	if id, ok := intString(params["cluster_id"]); ok {
		appendOne("cluster", id, "")
	}
	if id, ok := intString(params["service_id"]); ok {
		appendOne("service", id, "")
	}
	if id, ok := intString(params["target_id"]); ok {
		appendOne("target", id, "")
	}
	if strings.TrimSpace(toolName) != "" && len(out) == 0 {
		appendOne("tool", toolName, toolName)
	}
	return out
}

func (b *PreviewBuilder) GenerateImpactScope(toolName string, targets []TargetResource, params map[string]any) string {
	_ = b.db
	if len(targets) == 0 {
		return "影响范围未知，建议先预览并确认参数。"
	}
	types := make(map[string]int, 4)
	for _, t := range targets {
		types[t.Type]++
	}
	parts := make([]string, 0, len(types))
	for kind, n := range types {
		parts = append(parts, fmt.Sprintf("%s:%d", kind, n))
	}
	sort.Strings(parts)
	scope := fmt.Sprintf("将影响资源 %s", strings.Join(parts, ", "))
	if strings.Contains(strings.ToLower(toolName), "batch") {
		scope += "，为批量操作。"
		return scope
	}
	if _, ok := params["namespace"]; ok {
		scope += "，包含命名空间维度。"
	}
	return scope
}

func defaultTimeoutForRisk(risk tools.ToolRisk) time.Duration {
	switch risk {
	case tools.ToolRiskLow:
		return 5 * time.Minute
	case tools.ToolRiskHigh:
		return 30 * time.Minute
	default:
		return 15 * time.Minute
	}
}

func buildDeployDiffPreview(toolName string, params map[string]any) string {
	name := strings.ToLower(strings.TrimSpace(toolName))
	if !strings.Contains(name, "deploy") && !strings.Contains(name, "release") {
		return ""
	}
	current := strings.TrimSpace(toString(params["current_version"]))
	target := strings.TrimSpace(toString(params["target_version"]))
	if current == "" && target == "" {
		return "部署类操作，可在执行前补充版本差异信息。"
	}
	if current == "" {
		return fmt.Sprintf("目标版本: %s", target)
	}
	if target == "" {
		return fmt.Sprintf("当前版本: %s", current)
	}
	return fmt.Sprintf("版本变更: %s -> %s", current, target)
}

func intString(v any) (string, bool) {
	switch x := v.(type) {
	case int:
		return fmt.Sprintf("%d", x), true
	case int64:
		return fmt.Sprintf("%d", x), true
	case float64:
		return fmt.Sprintf("%d", int64(x)), true
	case string:
		if strings.TrimSpace(x) == "" {
			return "", false
		}
		return strings.TrimSpace(x), true
	default:
		return "", false
	}
}

func intStringSlice(v any) []string {
	switch x := v.(type) {
	case []int:
		out := make([]string, 0, len(x))
		for _, item := range x {
			out = append(out, fmt.Sprintf("%d", item))
		}
		return out
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if text, ok := intString(item); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
