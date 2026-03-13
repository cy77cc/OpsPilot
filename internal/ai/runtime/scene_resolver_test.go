package runtime

import (
	"context"
	"testing"
)

type fakeSceneLoader struct {
	configs map[string]SceneConfig
}

func (f fakeSceneLoader) LoadSceneConfigs(context.Context) (map[string]SceneConfig, error) {
	return f.configs, nil
}

func TestSceneConfigResolverResolveAppliesSubSceneRules(t *testing.T) {
	resolver := NewSceneConfigResolver(nil)
	resolved := resolver.Resolve("host:hosts")

	if resolved.Domain != "host" || resolved.SubScene != "hosts" {
		t.Fatalf("resolved scene = %#v", resolved)
	}
	if len(resolved.Constraints) == 0 {
		t.Fatalf("expected sub-scene constraints: %#v", resolved)
	}
}

func TestSceneConfigResolverReloadCacheMergesOverrides(t *testing.T) {
	resolver := NewSceneConfigResolver(fakeSceneLoader{
		configs: map[string]SceneConfig{
			"deployment": {
				Name:         "部署管理增强版",
				AllowedTools: []string{"clusterListInventory"},
			},
		},
	})

	if err := resolver.ReloadCache(context.Background()); err != nil {
		t.Fatalf("ReloadCache error = %v", err)
	}
	resolved := resolver.Resolve("deployment")
	if resolved.SceneConfig.Name != "部署管理增强版" {
		t.Fatalf("resolved name = %q", resolved.SceneConfig.Name)
	}
	if len(resolved.EffectiveAllowedTools()) != 1 {
		t.Fatalf("allowed tools = %#v", resolved.EffectiveAllowedTools())
	}
}

func TestSceneConfigResolverResolveFallsBackToGlobalAndKeepsDomainSubScene(t *testing.T) {
	resolver := NewSceneConfigResolver(nil)

	global := resolver.Resolve("")
	if global.Domain != "global" {
		t.Fatalf("global domain = %q, want global", global.Domain)
	}

	resolved := resolver.Resolve("deployment:clusters")
	if resolved.Domain != "deployment" || resolved.SubScene != "clusters" {
		t.Fatalf("resolved scene = %#v", resolved)
	}
	if len(resolved.Constraints) == 0 {
		t.Fatalf("expected deployment sub-scene constraints: %#v", resolved)
	}
}
