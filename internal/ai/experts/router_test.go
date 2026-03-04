package experts

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeRegistry struct {
	experts  map[string]*Expert
	keywords map[string][]*RankedExpert
	domains  map[string][]*RankedExpert
}

func (f *fakeRegistry) GetExpert(name string) (*Expert, bool) {
	e, ok := f.experts[name]
	return e, ok
}

func (f *fakeRegistry) ListExperts() []*Expert {
	out := make([]*Expert, 0, len(f.experts))
	for _, item := range f.experts {
		out = append(out, item)
	}
	return out
}

func (f *fakeRegistry) Reload() error { return nil }

func (f *fakeRegistry) MatchByKeywords(content string) []*RankedExpert {
	return f.keywords[content]
}

func (f *fakeRegistry) MatchByDomain(domain string) []*RankedExpert {
	return f.domains[domain]
}

func TestHybridRouterRouteSceneKeywordDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "scene_mappings.yaml")
	raw := `version: "1.0"
mappings:
  services:detail:
    primary_expert: service_expert
    helper_experts: [k8s_expert]
    strategy: sequential
`
	if err := os.WriteFile(cfgPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	reg := &fakeRegistry{
		experts: map[string]*Expert{
			"general_expert": {Name: "general_expert"},
			"service_expert": {Name: "service_expert"},
		},
		keywords: map[string][]*RankedExpert{
			"service issue": {
				{Expert: &Expert{Name: "service_expert"}, Score: 3},
			},
		},
	}
	router, err := NewHybridRouter(reg, cfgPath)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	sceneDecision := router.Route(context.Background(), &RouteRequest{Scene: "scene:services:detail"})
	if sceneDecision.Source != "scene" || sceneDecision.PrimaryExpert != "service_expert" {
		t.Fatalf("unexpected scene decision: %#v", sceneDecision)
	}

	keywordDecision := router.Route(context.Background(), &RouteRequest{Message: "service issue"})
	if keywordDecision.Source != "keyword" || keywordDecision.PrimaryExpert != "service_expert" {
		t.Fatalf("unexpected keyword decision: %#v", keywordDecision)
	}

	defaultDecision := router.Route(context.Background(), &RouteRequest{Message: "unknown"})
	if defaultDecision.Source != "default" || defaultDecision.PrimaryExpert != "general_expert" {
		t.Fatalf("unexpected default decision: %#v", defaultDecision)
	}
}
