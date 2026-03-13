package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	aiobs "github.com/cy77cc/OpsPilot/internal/ai/observability"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/host"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/kubernetes"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/service"
	"github.com/cy77cc/OpsPilot/internal/model"
)

type Mode string
type Risk string

const (
	ModeReadonly Mode = "readonly"
	ModeMutating Mode = "mutating"

	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

type ParamHint struct {
	Type       string        `json:"type"`
	Required   bool          `json:"required"`
	Hint       string        `json:"hint,omitempty"`
	Default    any           `json:"default,omitempty"`
	EnumSource string        `json:"enum_source,omitempty"`
	DependsOn  []string      `json:"depends_on,omitempty"`
	Options    []ParamOption `json:"options,omitempty"`
}

type ParamOption struct {
	Value       any            `json:"value"`
	Label       string         `json:"label"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type Capability struct {
	Name        string               `json:"name"`
	DisplayName string               `json:"display_name,omitempty"`
	Description string               `json:"description"`
	Mode        Mode                 `json:"mode"`
	Risk        Risk                 `json:"risk"`
	Category    string               `json:"category,omitempty"`
	Tags        []string             `json:"tags,omitempty"`
	Scenes      []string             `json:"scenes,omitempty"`
	Schema      map[string]ParamHint `json:"schema,omitempty"`
}

type Execution struct {
	Result   any
	Summary  string
	Metadata map[string]any
	Usage    *aiobs.Usage
}

type ToolSpec struct {
	Capability
	Input   any
	Preview func(context.Context, common.PlatformDeps, map[string]any) (any, error)
	Execute func(context.Context, common.PlatformDeps, map[string]any) (*Execution, error)
}

type Registry struct {
	deps  common.PlatformDeps
	mu    sync.RWMutex
	tools map[string]ToolSpec
}

func NewRegistry(deps common.PlatformDeps) *Registry {
	r := &Registry{deps: deps, tools: map[string]ToolSpec{}}
	r.registerDefaults()
	return r
}

func (r *Registry) List() []Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Capability, 0, len(r.tools))
	for _, spec := range r.tools {
		cap := spec.Capability
		if cap.Schema == nil {
			cap.Schema = hintsFor(spec.Input)
		}
		out = append(out, cap)
	}
	return out
}

func (r *Registry) Get(name string) (ToolSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	spec, ok := r.tools[strings.TrimSpace(name)]
	if ok && spec.Schema == nil {
		spec.Schema = hintsFor(spec.Input)
	}
	return spec, ok
}

func (r *Registry) Hints(name string) (map[string]ParamHint, bool) {
	spec, ok := r.Get(name)
	if !ok {
		return nil, false
	}
	return spec.Schema, true
}

func (r *Registry) FilterByScene(scene string, cfg *SceneConfig) []Capability {
	scene = strings.TrimSpace(scene)
	caps := r.List()
	if cfg == nil {
		return caps
	}
	allowed := make(map[string]struct{}, len(cfg.AllowedTools))
	for _, name := range cfg.AllowedTools {
		allowed[strings.TrimSpace(name)] = struct{}{}
	}
	blocked := make(map[string]struct{}, len(cfg.BlockedTools))
	for _, name := range cfg.BlockedTools {
		blocked[strings.TrimSpace(name)] = struct{}{}
	}
	out := make([]Capability, 0, len(caps))
	for _, cap := range caps {
		if _, found := blocked[cap.Name]; found {
			continue
		}
		if len(allowed) > 0 {
			if _, found := allowed[cap.Name]; !found {
				continue
			}
		}
		if scene != "" && len(cap.Scenes) > 0 {
			matched := false
			for _, candidate := range cap.Scenes {
				if candidate == scene {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, cap)
	}
	return out
}

func (r *Registry) register(spec ToolSpec) {
	spec.Name = strings.TrimSpace(spec.Name)
	if spec.Name == "" {
		return
	}
	spec.Schema = hintsFor(spec.Input)
	r.tools[spec.Name] = spec
}

func (r *Registry) registerDefaults() {
	r.register(ToolSpec{
		Capability: Capability{
			Name:        "service_status",
			DisplayName: "Service Status",
			Description: "Get current service runtime status",
			Mode:        ModeReadonly,
			Risk:        RiskLow,
			Category:    "service",
			Tags:        []string{"service", "status"},
			Scenes:      []string{"global", "service"},
		},
		Input: service.ServiceStatusInput{},
		Execute: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (*Execution, error) {
			id := toInt(params["service_id"])
			if id <= 0 {
				return nil, fmt.Errorf("service_id is required")
			}
			var row model.Service
			if err := deps.DB.WithContext(ctx).First(&row, id).Error; err != nil {
				return nil, err
			}
			result := map[string]any{
				"service_id":   row.ID,
				"name":         row.Name,
				"status":       row.Status,
				"env":          row.Env,
				"runtime_type": row.RuntimeType,
				"image":        row.Image,
				"replicas":     row.Replicas,
			}
			return &Execution{Result: result, Summary: fmt.Sprintf("service %s status is %s", row.Name, row.Status)}, nil
		},
	})
	r.register(ToolSpec{
		Capability: Capability{
			Name:        "service_status_by_target",
			DisplayName: "Service Status By Target",
			Description: "Resolve service by name or id and inspect status",
			Mode:        ModeReadonly,
			Risk:        RiskLow,
			Category:    "service",
			Tags:        []string{"service", "lookup"},
			Scenes:      []string{"global", "service"},
		},
		Input: service.ServiceStatusByTargetInput{},
		Execute: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (*Execution, error) {
			target := strings.TrimSpace(toString(params["target"]))
			if target == "" {
				return nil, fmt.Errorf("target is required")
			}
			var row model.Service
			q := deps.DB.WithContext(ctx)
			if id, err := strconv.Atoi(target); err == nil && id > 0 {
				if err := q.First(&row, id).Error; err != nil {
					return nil, err
				}
			} else if err := q.Where("name = ?", target).First(&row).Error; err != nil {
				return nil, err
			}
			return &Execution{
				Result:  map[string]any{"service_id": row.ID, "name": row.Name, "status": row.Status, "env": row.Env},
				Summary: fmt.Sprintf("service %s status is %s", row.Name, row.Status),
			}, nil
		},
	})
	r.register(ToolSpec{
		Capability: Capability{
			Name:        "service_deploy_preview",
			DisplayName: "Service Deploy Preview",
			Description: "Preview service deployment changes",
			Mode:        ModeReadonly,
			Risk:        RiskLow,
			Category:    "service",
			Tags:        []string{"service", "preview", "deployment"},
			Scenes:      []string{"global", "service"},
		},
		Input: service.ServiceDeployPreviewInput{},
		Preview: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (any, error) {
			id := toInt(params["service_id"])
			clusterID := toInt(params["cluster_id"])
			if id <= 0 || clusterID <= 0 {
				return nil, fmt.Errorf("service_id and cluster_id are required")
			}
			var row model.Service
			if err := deps.DB.WithContext(ctx).First(&row, id).Error; err != nil {
				return nil, err
			}
			return map[string]any{
				"service_id": id,
				"cluster_id": clusterID,
				"name":       row.Name,
				"image":      row.Image,
				"replicas":   row.Replicas,
				"dry_run":    true,
			}, nil
		},
		Execute: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (*Execution, error) {
			preview, err := r.tools["service_deploy_preview"].Preview(ctx, deps, params)
			if err != nil {
				return nil, err
			}
			return &Execution{Result: preview, Summary: "service deployment preview generated"}, nil
		},
	})
	r.register(ToolSpec{
		Capability: Capability{
			Name:        "service_deploy_apply",
			DisplayName: "Service Deploy Apply",
			Description: "Apply a service deployment",
			Mode:        ModeMutating,
			Risk:        RiskMedium,
			Category:    "service",
			Tags:        []string{"service", "deploy"},
			Scenes:      []string{"global", "service"},
		},
		Input: service.ServiceDeployApplyInput{},
		Preview: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (any, error) {
			preview, err := r.tools["service_deploy_preview"].Preview(ctx, deps, params)
			if err != nil {
				return nil, err
			}
			return map[string]any{"preview": preview, "approval_required": true}, nil
		},
		Execute: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (*Execution, error) {
			preview, err := r.tools["service_deploy_preview"].Preview(ctx, deps, params)
			if err != nil {
				return nil, err
			}
			return &Execution{Result: map[string]any{"applied": true, "deployment": preview}, Summary: "service deployment applied"}, nil
		},
	})
	r.register(ToolSpec{
		Capability: Capability{
			Name:        "host_list_inventory",
			DisplayName: "Host Inventory",
			Description: "List hosts with status and metadata",
			Mode:        ModeReadonly,
			Risk:        RiskLow,
			Category:    "host",
			Tags:        []string{"host", "inventory"},
			Scenes:      []string{"global", "host"},
		},
		Input: host.HostInventoryInput{},
		Execute: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (*Execution, error) {
			limit := toInt(params["limit"])
			if limit <= 0 {
				limit = 50
			}
			query := deps.DB.WithContext(ctx).Model(&model.Node{})
			if status := strings.TrimSpace(toString(params["status"])); status != "" {
				query = query.Where("status = ?", status)
			}
			if keyword := strings.TrimSpace(toString(params["keyword"])); keyword != "" {
				pattern := "%" + keyword + "%"
				query = query.Where("name LIKE ? OR ip LIKE ? OR hostname LIKE ?", pattern, pattern, pattern)
			}
			rows := make([]model.Node, 0)
			if err := query.Order("id desc").Limit(limit).Find(&rows).Error; err != nil {
				return nil, err
			}
			result := make([]map[string]any, 0, len(rows))
			for _, row := range rows {
				result = append(result, map[string]any{"id": row.ID, "name": row.Name, "ip": row.IP, "hostname": row.Hostname, "status": row.Status})
			}
			return &Execution{Result: map[string]any{"total": len(result), "list": result}, Summary: fmt.Sprintf("listed %d hosts", len(result))}, nil
		},
	})
	r.register(ToolSpec{
		Capability: Capability{
			Name:        "host_batch_exec_preview",
			DisplayName: "Host Batch Preview",
			Description: "Preview a multi-host command execution",
			Mode:        ModeReadonly,
			Risk:        RiskLow,
			Category:    "host",
			Tags:        []string{"host", "preview", "batch"},
			Scenes:      []string{"global", "host"},
		},
		Input: host.HostBatchExecPreviewInput{},
		Preview: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (any, error) {
			hostIDs := toUintSlice(params["host_ids"])
			command := strings.TrimSpace(toString(params["command"]))
			if len(hostIDs) == 0 || command == "" {
				return nil, fmt.Errorf("host_ids and command are required")
			}
			rows := make([]model.Node, 0)
			if err := deps.DB.WithContext(ctx).Where("id IN ?", hostIDs).Find(&rows).Error; err != nil {
				return nil, err
			}
			targets := make([]map[string]any, 0, len(rows))
			for _, row := range rows {
				targets = append(targets, map[string]any{"id": row.ID, "name": row.Name, "ip": row.IP})
			}
			class, risk := classifyCommand(command)
			return map[string]any{
				"command":        command,
				"command_class":  class,
				"risk":           risk,
				"target_count":   len(hostIDs),
				"resolved_count": len(rows),
				"targets":        targets,
			}, nil
		},
		Execute: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (*Execution, error) {
			preview, err := r.tools["host_batch_exec_preview"].Preview(ctx, deps, params)
			if err != nil {
				return nil, err
			}
			return &Execution{Result: preview, Summary: "host batch preview generated"}, nil
		},
	})
	r.register(ToolSpec{
		Capability: Capability{
			Name:        "host_batch_exec_apply",
			DisplayName: "Host Batch Apply",
			Description: "Execute a multi-host command after approval",
			Mode:        ModeMutating,
			Risk:        RiskHigh,
			Category:    "host",
			Tags:        []string{"host", "apply", "batch"},
			Scenes:      []string{"global", "host"},
		},
		Input: host.HostBatchExecApplyInput{},
		Preview: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (any, error) {
			preview, err := r.tools["host_batch_exec_preview"].Preview(ctx, deps, params)
			if err != nil {
				return nil, err
			}
			return map[string]any{"preview": preview, "approval_required": true}, nil
		},
		Execute: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (*Execution, error) {
			hostIDs := toUintSlice(params["host_ids"])
			command := strings.TrimSpace(toString(params["command"]))
			if len(hostIDs) == 0 || command == "" {
				return nil, fmt.Errorf("host_ids and command are required")
			}
			rows := make([]model.Node, 0)
			if err := deps.DB.WithContext(ctx).Where("id IN ?", hostIDs).Find(&rows).Error; err != nil {
				return nil, err
			}
			results := make([]map[string]any, 0, len(rows))
			for _, row := range rows {
				results = append(results, map[string]any{
					"host_id":   row.ID,
					"host_name": row.Name,
					"status":    "simulated_success",
					"stdout":    fmt.Sprintf("simulated execution of %q on %s", command, row.Name),
					"stderr":    "",
					"exit_code": 0,
				})
			}
			return &Execution{Result: map[string]any{"command": command, "results": results}, Summary: "host batch command executed"}, nil
		},
	})
	r.register(ToolSpec{
		Capability: Capability{
			Name:        "k8s_query",
			DisplayName: "Kubernetes Query",
			Description: "Query kubernetes resources",
			Mode:        ModeReadonly,
			Risk:        RiskLow,
			Category:    "kubernetes",
			Tags:        []string{"k8s", "query"},
			Scenes:      []string{"global", "service"},
		},
		Input: kubernetes.K8sQueryInput{},
		Execute: func(ctx context.Context, deps common.PlatformDeps, params map[string]any) (*Execution, error) {
			return &Execution{Result: map[string]any{"resource": toString(params["resource"]), "filters": params, "items": []any{}}, Summary: "kubernetes query prepared"}, nil
		},
	})
}

func hintsFor(input any) map[string]ParamHint {
	if input == nil {
		return nil
	}
	t := reflect.TypeOf(input)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	out := make(map[string]ParamHint, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		desc := field.Tag.Get("jsonschema_description")
		hint := ParamHint{
			Type:       schemaType(field.Type),
			Required:   strings.Contains(desc, "required"),
			Hint:       cleanDescription(desc),
			EnumSource: enumSource(name),
		}
		out[name] = hint
	}
	return out
}

func schemaType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map:
		return "object"
	default:
		return "string"
	}
}

func cleanDescription(desc string) string {
	if desc == "" {
		return ""
	}
	desc = strings.ReplaceAll(desc, "required,", "")
	desc = strings.ReplaceAll(desc, "optional ", "")
	return strings.TrimSpace(desc)
}

func enumSource(name string) string {
	switch name {
	case "service_id":
		return "services"
	case "cluster_id":
		return "clusters"
	case "host_ids", "host_id":
		return "hosts"
	case "namespace":
		return "namespaces"
	case "pod":
		return "pods"
	default:
		return ""
	}
}

func classifyCommand(command string) (string, string) {
	cmd := strings.ToLower(strings.TrimSpace(command))
	switch {
	case strings.Contains(cmd, "restart"), strings.Contains(cmd, "stop"), strings.Contains(cmd, "kill"):
		return "mutating", string(RiskHigh)
	case strings.Contains(cmd, "status"), strings.Contains(cmd, "cat "), strings.Contains(cmd, "tail "), strings.Contains(cmd, "df "), strings.Contains(cmd, "free "), strings.Contains(cmd, "ps "):
		return "readonly", string(RiskLow)
	default:
		return "mutating", string(RiskMedium)
	}
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	default:
		return 0
	}
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func toUintSlice(v any) []uint64 {
	raw, ok := v.([]any)
	if !ok {
		if values, ok := v.([]uint64); ok {
			return values
		}
		return nil
	}
	out := make([]uint64, 0, len(raw))
	for _, item := range raw {
		if n := uint64(toInt(item)); n > 0 {
			out = append(out, n)
		}
	}
	return out
}

type SceneConfig struct {
	Scene          string         `json:"scene"`
	Name           string         `json:"name,omitempty"`
	Description    string         `json:"description,omitempty"`
	Constraints    []string       `json:"constraints,omitempty"`
	AllowedTools   []string       `json:"allowed_tools,omitempty"`
	BlockedTools   []string       `json:"blocked_tools,omitempty"`
	Examples       []string       `json:"examples,omitempty"`
	ApprovalConfig map[string]any `json:"approval_config,omitempty"`
}

func DecodeSceneConfig(row model.AISceneConfig) SceneConfig {
	out := SceneConfig{
		Scene:       row.Scene,
		Name:        row.Name,
		Description: row.Description,
	}
	_ = json.Unmarshal([]byte(row.ConstraintsJSON), &out.Constraints)
	_ = json.Unmarshal([]byte(row.AllowedToolsJSON), &out.AllowedTools)
	_ = json.Unmarshal([]byte(row.BlockedToolsJSON), &out.BlockedTools)
	_ = json.Unmarshal([]byte(row.ExamplesJSON), &out.Examples)
	_ = json.Unmarshal([]byte(row.ApprovalConfigJSON), &out.ApprovalConfig)
	return out
}

func EncodeSceneConfig(scene SceneConfig) model.AISceneConfig {
	return model.AISceneConfig{
		Scene:              scene.Scene,
		Name:               scene.Name,
		Description:        scene.Description,
		ConstraintsJSON:    mustJSON(scene.Constraints),
		AllowedToolsJSON:   mustJSON(scene.AllowedTools),
		BlockedToolsJSON:   mustJSON(scene.BlockedTools),
		ExamplesJSON:       mustJSON(scene.Examples),
		ApprovalConfigJSON: mustJSON(scene.ApprovalConfig),
	}
}

func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
