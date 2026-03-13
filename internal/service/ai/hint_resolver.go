package ai

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	coreai "github.com/cy77cc/OpsPilot/internal/ai"
	aitools "github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/model"
)

type HintResolver struct {
	deps common.PlatformDeps
}

func NewHintResolver(deps common.PlatformDeps) *HintResolver {
	return &HintResolver{deps: deps}
}

func (r *HintResolver) Resolve(ctx context.Context, spec aitools.ToolSpec, runtimeCtx coreai.RuntimeContext, params map[string]any) (map[string]aitools.ParamHint, error) {
	base := cloneHints(spec.Schema)
	if len(base) == 0 {
		return base, nil
	}

	for name, hint := range base {
		source := r.resolveSource(spec, name, hint, params)
		if source == "" {
			continue
		}
		hint.EnumSource = source
		hint.DependsOn = hintDependsOn(source)
		options, resolvedDefault, err := r.resolveOptions(ctx, source, runtimeCtx, params)
		if err != nil {
			return nil, err
		}
		if len(options) > 0 {
			hint.Options = options
		}
		if hint.Default == nil && resolvedDefault != nil {
			hint.Default = resolvedDefault
		}
		base[name] = hint
	}
	return base, nil
}

func (r *HintResolver) resolveSource(spec aitools.ToolSpec, name string, hint aitools.ParamHint, params map[string]any) string {
	if hint.EnumSource != "" {
		return hint.EnumSource
	}

	switch spec.Name {
	case "k8s_logs", "k8s_get_pod_logs":
		if name == "pod" {
			return "pods"
		}
	case "k8s_query":
		if name == "name" {
			switch strings.ToLower(strings.TrimSpace(stringValue(params["resource"]))) {
			case "pods":
				return "pods"
			case "deployments":
				return "deployments"
			}
		}
	case "k8s_events", "k8s_get_events":
		if name == "name" {
			switch strings.ToLower(strings.TrimSpace(stringValue(params["kind"]))) {
			case "pod":
				return "pods"
			case "deployment":
				return "deployments"
			}
		}
	}

	return ""
}

func (r *HintResolver) resolveOptions(ctx context.Context, source string, runtimeCtx coreai.RuntimeContext, _ map[string]any) ([]aitools.ParamOption, any, error) {
	switch source {
	case "clusters":
		return r.clusterOptions(ctx, runtimeCtx), nil, nil
	case "hosts":
		return r.hostOptions(ctx, runtimeCtx), nil, nil
	case "namespaces":
		options := r.namespaceOptions(ctx, runtimeCtx)
		return options, selectedResourceDefault(runtimeCtx.SelectedResources, "namespace"), nil
	case "deployments":
		options := r.deploymentOptions(ctx, runtimeCtx)
		return options, selectedResourceName(runtimeCtx.SelectedResources, "deployment"), nil
	case "pods":
		options := r.podOptions(runtimeCtx)
		return options, selectedResourceName(runtimeCtx.SelectedResources, "pod"), nil
	case "services":
		return r.serviceOptions(ctx, runtimeCtx), nil, nil
	default:
		return nil, nil, nil
	}
}

func (r *HintResolver) clusterOptions(ctx context.Context, runtimeCtx coreai.RuntimeContext) []aitools.ParamOption {
	if r.deps.DB == nil {
		return nil
	}
	query := r.deps.DB.WithContext(ctx).Model(&model.Cluster{}).Order("name asc")
	if projectID := parseUint(runtimeCtx.ProjectID); projectID > 0 {
		subQuery := r.deps.DB.WithContext(ctx).Model(&model.DeploymentTarget{}).Select("distinct cluster_id").Where("project_id = ? AND cluster_id > 0", projectID)
		query = query.Where("id IN (?)", subQuery)
	}
	rows := make([]model.Cluster, 0)
	if err := query.Find(&rows).Error; err != nil {
		return nil
	}
	options := make([]aitools.ParamOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, aitools.ParamOption{
			Value: row.ID,
			Label: row.Name,
			Metadata: map[string]any{
				"status":   row.Status,
				"env_type": row.EnvType,
			},
		})
	}
	return options
}

func (r *HintResolver) hostOptions(ctx context.Context, runtimeCtx coreai.RuntimeContext) []aitools.ParamOption {
	if r.deps.DB == nil {
		return nil
	}
	query := r.deps.DB.WithContext(ctx).Model(&model.Node{}).Order("name asc")
	if projectID := parseUint(runtimeCtx.ProjectID); projectID > 0 {
		subQuery := r.deps.DB.WithContext(ctx).
			Table("deployment_target_nodes as dtn").
			Select("distinct dtn.host_id").
			Joins("join deployment_targets dt on dt.id = dtn.target_id").
			Where("dt.project_id = ?", projectID)
		query = query.Where("id IN (?)", subQuery)
	}
	rows := make([]model.Node, 0)
	if err := query.Find(&rows).Error; err != nil {
		return nil
	}
	options := make([]aitools.ParamOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, aitools.ParamOption{
			Value: row.ID,
			Label: firstNonEmpty(row.Name, row.Hostname, row.IP),
			Metadata: map[string]any{
				"ip":         row.IP,
				"status":     row.Status,
				"cluster_id": row.ClusterID,
			},
		})
	}
	return options
}

func (r *HintResolver) namespaceOptions(ctx context.Context, runtimeCtx coreai.RuntimeContext) []aitools.ParamOption {
	namespaces := make(map[string]struct{})
	for _, resource := range runtimeCtx.SelectedResources {
		if ns := strings.TrimSpace(resource.Namespace); ns != "" {
			namespaces[ns] = struct{}{}
		}
	}
	if r.deps.DB != nil {
		query := r.deps.DB.WithContext(ctx).Model(&model.DeploymentRelease{}).
			Select("distinct deployment_releases.namespace_or_project").
			Joins("join services on services.id = deployment_releases.service_id").
			Where("deployment_releases.runtime_type = ?", "k8s").
			Where("deployment_releases.namespace_or_project <> ''")
		if projectID := parseUint(runtimeCtx.ProjectID); projectID > 0 {
			query = query.Where("services.project_id = ?", projectID)
		}
		rows := make([]string, 0)
		if err := query.Pluck("deployment_releases.namespace_or_project", &rows).Error; err == nil {
			for _, row := range rows {
				if ns := strings.TrimSpace(row); ns != "" {
					namespaces[ns] = struct{}{}
				}
			}
		}
	}
	return stringOptions(namespaces)
}

func (r *HintResolver) deploymentOptions(ctx context.Context, runtimeCtx coreai.RuntimeContext) []aitools.ParamOption {
	names := make(map[string]struct{})
	for _, resource := range runtimeCtx.SelectedResources {
		if resourceTypeMatches(resource.Type, "deployment") {
			if name := firstNonEmpty(resource.Name, resource.ID); name != "" {
				names[name] = struct{}{}
			}
		}
	}
	if r.deps.DB != nil {
		query := r.deps.DB.WithContext(ctx).Model(&model.Service{}).Select("distinct name").Where("runtime_type = ?", "k8s")
		if projectID := parseUint(runtimeCtx.ProjectID); projectID > 0 {
			query = query.Where("project_id = ?", projectID)
		}
		rows := make([]string, 0)
		if err := query.Order("name asc").Pluck("name", &rows).Error; err == nil {
			for _, row := range rows {
				if name := strings.TrimSpace(row); name != "" {
					names[name] = struct{}{}
				}
			}
		}
	}
	return stringOptions(names)
}

func (r *HintResolver) podOptions(runtimeCtx coreai.RuntimeContext) []aitools.ParamOption {
	names := make(map[string]struct{})
	for _, resource := range runtimeCtx.SelectedResources {
		if resourceTypeMatches(resource.Type, "pod") {
			if name := firstNonEmpty(resource.Name, resource.ID); name != "" {
				names[name] = struct{}{}
			}
		}
	}
	return stringOptions(names)
}

func (r *HintResolver) serviceOptions(ctx context.Context, runtimeCtx coreai.RuntimeContext) []aitools.ParamOption {
	if r.deps.DB == nil {
		return nil
	}
	query := r.deps.DB.WithContext(ctx).Model(&model.Service{}).Order("name asc")
	if projectID := parseUint(runtimeCtx.ProjectID); projectID > 0 {
		query = query.Where("project_id = ?", projectID)
	}
	rows := make([]model.Service, 0)
	if err := query.Find(&rows).Error; err != nil {
		return nil
	}
	options := make([]aitools.ParamOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, aitools.ParamOption{
			Value: row.ID,
			Label: row.Name,
			Metadata: map[string]any{
				"env":        row.Env,
				"project_id": row.ProjectID,
			},
		})
	}
	return options
}

func cloneHints(in map[string]aitools.ParamHint) map[string]aitools.ParamHint {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]aitools.ParamHint, len(in))
	for key, hint := range in {
		clone := hint
		if len(hint.DependsOn) > 0 {
			clone.DependsOn = append([]string(nil), hint.DependsOn...)
		}
		if len(hint.Options) > 0 {
			clone.Options = append([]aitools.ParamOption(nil), hint.Options...)
		}
		out[key] = clone
	}
	return out
}

func hintDependsOn(source string) []string {
	switch source {
	case "namespaces":
		return []string{"cluster_id"}
	case "deployments", "pods":
		return []string{"cluster_id", "namespace"}
	default:
		return nil
	}
}

func stringOptions(values map[string]struct{}) []aitools.ParamOption {
	if len(values) == 0 {
		return nil
	}
	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Strings(items)
	options := make([]aitools.ParamOption, 0, len(items))
	for _, value := range items {
		options = append(options, aitools.ParamOption{Value: value, Label: value})
	}
	return options
}

func selectedResourceDefault(resources []coreai.SelectedResource, field string) any {
	switch field {
	case "namespace":
		for _, resource := range resources {
			if ns := strings.TrimSpace(resource.Namespace); ns != "" {
				return ns
			}
		}
	}
	return nil
}

func selectedResourceName(resources []coreai.SelectedResource, kind string) any {
	for _, resource := range resources {
		if resourceTypeMatches(resource.Type, kind) {
			if name := firstNonEmpty(resource.Name, resource.ID); name != "" {
				return name
			}
		}
	}
	return nil
}

func resourceTypeMatches(resourceType, expected string) bool {
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	expected = strings.ToLower(strings.TrimSpace(expected))
	if resourceType == expected {
		return true
	}
	return resourceType == expected+"s"
}

func parseUint(raw string) uint {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0
	}
	return uint(value)
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
