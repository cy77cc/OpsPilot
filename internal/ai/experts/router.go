package experts

import (
	"context"
	"log/slog"
	"strings"
)

const defaultSceneMappingsPath = "configs/scene_mappings.yaml"

type HybridRouter struct {
	registry      ExpertRegistry
	sceneMappings map[string]SceneMapping
}

func NewHybridRouter(registry ExpertRegistry, sceneMappingsPath string) (*HybridRouter, error) {
	if strings.TrimSpace(sceneMappingsPath) == "" {
		sceneMappingsPath = defaultSceneMappingsPath
	}
	cfg, err := LoadSceneMappings(sceneMappingsPath)
	if err != nil {
		return nil, err
	}
	return &HybridRouter{
		registry:      registry,
		sceneMappings: cfg.Mappings,
	}, nil
}

func (r *HybridRouter) Route(ctx context.Context, req *RouteRequest) *RouteDecision {
	if req == nil {
		return r.routeDefault()
	}
	if decision := r.routeByScene(req.Scene); decision != nil {
		slog.DebugContext(ctx, "expert route selected", "source", "scene", "scene", req.Scene, "primary_expert", decision.PrimaryExpert)
		return decision
	}
	if decision := r.routeByKeywords(req.Message); decision != nil {
		slog.DebugContext(ctx, "expert route selected", "source", "keyword", "primary_expert", decision.PrimaryExpert)
		return decision
	}
	if decision := r.routeByDomain(req.Message); decision != nil {
		slog.DebugContext(ctx, "expert route selected", "source", "domain", "primary_expert", decision.PrimaryExpert)
		return decision
	}
	decision := r.routeDefault()
	slog.DebugContext(ctx, "expert route selected", "source", "default", "primary_expert", decision.PrimaryExpert)
	return decision
}

func (r *HybridRouter) routeByScene(scene string) *RouteDecision {
	key := normalizeSceneKey(scene)
	if key == "" {
		return nil
	}
	item, ok := r.sceneMappings[key]
	if !ok || strings.TrimSpace(item.PrimaryExpert) == "" {
		return nil
	}
	strategy := item.Strategy
	if strategy == "" {
		strategy = StrategySequential
	}
	return &RouteDecision{
		PrimaryExpert: item.PrimaryExpert,
		HelperExperts: append([]string{}, item.HelperExperts...),
		Strategy:      strategy,
		Confidence:    1.0,
		Source:        "scene",
	}
}

func (r *HybridRouter) routeByKeywords(content string) *RouteDecision {
	if r.registry == nil {
		return nil
	}
	matches := r.registry.MatchByKeywords(content)
	if len(matches) == 0 || matches[0] == nil || matches[0].Expert == nil {
		return nil
	}
	helpers := make([]string, 0, 2)
	for i := 1; i < len(matches) && len(helpers) < 2; i++ {
		if matches[i] == nil || matches[i].Expert == nil {
			continue
		}
		helpers = append(helpers, matches[i].Expert.Name)
	}
	return &RouteDecision{
		PrimaryExpert: matches[0].Expert.Name,
		HelperExperts: helpers,
		Strategy:      StrategySingle,
		Confidence:    clampScore(matches[0].Score / 5.0),
		Source:        "keyword",
	}
}

func (r *HybridRouter) routeByDomain(content string) *RouteDecision {
	if r.registry == nil {
		return nil
	}
	text := strings.ToLower(strings.TrimSpace(content))
	if text == "" {
		return nil
	}
	known := []string{
		"host_management",
		"os_diagnostics",
		"kubernetes",
		"workload_management",
		"service_management",
		"deployment_operations",
		"service_topology",
		"cicd",
		"job_scheduling",
		"observability",
		"audit_compliance",
		"security_governance",
	}
	for _, domain := range known {
		if !strings.Contains(text, strings.Split(domain, "_")[0]) {
			continue
		}
		matches := r.registry.MatchByDomain(domain)
		if len(matches) == 0 || matches[0] == nil || matches[0].Expert == nil {
			continue
		}
		return &RouteDecision{
			PrimaryExpert: matches[0].Expert.Name,
			Strategy:      StrategySingle,
			Confidence:    clampScore(matches[0].Score),
			Source:        "domain",
		}
	}
	return nil
}

func (r *HybridRouter) routeDefault() *RouteDecision {
	primary := "general_expert"
	if r.registry != nil {
		list := r.registry.ListExperts()
		if len(list) > 0 && list[0] != nil {
			primary = list[0].Name
		}
		if _, ok := r.registry.GetExpert("general_expert"); ok {
			primary = "general_expert"
		}
	}
	return &RouteDecision{
		PrimaryExpert: primary,
		Strategy:      StrategySingle,
		Confidence:    0.2,
		Source:        "default",
	}
}

func normalizeSceneKey(scene string) string {
	v := strings.TrimSpace(scene)
	v = strings.TrimPrefix(v, "scene:")
	return strings.ToLower(v)
}

func clampScore(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
