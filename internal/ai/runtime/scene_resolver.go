package runtime

import (
	"context"
	"strings"
	"sync"
)

type SceneConfigLoader interface {
	LoadSceneConfigs(ctx context.Context) (map[string]SceneConfig, error)
}

type SceneConfigResolver struct {
	loader SceneConfigLoader

	mu            sync.RWMutex
	domainConfigs map[string]SceneConfig
	subSceneRules map[string]SubSceneRule
}

func NewSceneConfigResolver(loader SceneConfigLoader) *SceneConfigResolver {
	return &SceneConfigResolver{
		loader:        loader,
		domainConfigs: defaultSceneConfigs(),
		subSceneRules: defaultSubSceneRules(),
	}
}

func defaultSceneConfigResolver() *SceneConfigResolver {
	return NewSceneConfigResolver(nil)
}

func (r *SceneConfigResolver) Resolve(sceneKey string) ResolvedScene {
	sceneKey = strings.TrimSpace(sceneKey)
	if sceneKey == "" {
		sceneKey = "global"
	}
	parts := strings.SplitN(sceneKey, ":", 2)
	domain := parts[0]
	subScene := ""
	if len(parts) == 2 {
		subScene = parts[1]
	}

	r.mu.RLock()
	config, ok := r.domainConfigs[domain]
	if !ok {
		config = r.domainConfigs["global"]
	}
	rule, hasRule := r.subSceneRules[subScene]
	r.mu.RUnlock()

	resolved := ResolvedScene{
		SceneKey:     sceneKey,
		Domain:       domain,
		SubScene:     subScene,
		SceneConfig:  config,
		AllowedTools: cloneStrings(config.AllowedTools),
		BlockedTools: cloneStrings(config.BlockedTools),
		Constraints:  cloneStrings(config.Constraints),
		ExampleIDs:   cloneStrings(config.Examples),
	}
	if hasRule && subScene != "" {
		resolved = r.applySubSceneRule(resolved, rule)
	}
	return resolved
}

func (r *SceneConfigResolver) applySubSceneRule(resolved ResolvedScene, rule SubSceneRule) ResolvedScene {
	if len(rule.IncludeTools) > 0 {
		resolved.AllowedTools = cloneStrings(rule.IncludeTools)
	}
	if len(rule.ExcludeTools) > 0 {
		resolved.AllowedTools = removeItems(resolved.AllowedTools, rule.ExcludeTools)
		resolved.BlockedTools = append(resolved.BlockedTools, cloneStrings(rule.ExcludeTools)...)
	}
	if len(rule.ExtraConstraints) > 0 {
		resolved.Constraints = append(resolved.Constraints, cloneStrings(rule.ExtraConstraints)...)
	}
	return resolved
}

func (r *SceneConfigResolver) ReloadCache(ctx context.Context) error {
	if r.loader == nil {
		return nil
	}
	configs, err := r.loader.LoadSceneConfigs(ctx)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.domainConfigs = mergeSceneConfigs(defaultSceneConfigs(), configs)
	r.mu.Unlock()
	return nil
}

func (r *SceneConfigResolver) WatchChanges(ctx context.Context) <-chan error {
	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		<-ctx.Done()
	}()
	return ch
}

func defaultSceneConfigs() map[string]SceneConfig {
	return map[string]SceneConfig{
		"global": {
			Name:        "全局助手",
			Description: "适用于通用平台问题和排障。",
			Constraints: []string{"优先基于工具结果回答，不要臆测环境状态。"},
			Examples:    []string{"global-troubleshooting"},
		},
		"deployment": {
			Name:         "部署管理",
			Description:  "面向部署、发布、扩缩容和回滚操作。",
			Constraints:  []string{"生产环境变更需要显式确认。"},
			AllowedTools: []string{"clusterListInventory", "k8sListResources"},
			Examples:     []string{"deployment-scale", "deployment-rollout"},
		},
		"monitor": {
			Name:         "监控中心",
			Description:  "关注告警、健康状态和指标分析。",
			Constraints:  []string{"默认使用只读检查路径。"},
			AllowedTools: []string{"monitorAlertRuleList", "serviceCatalogList"},
			Examples:     []string{"monitor-alert-triage"},
		},
		"host": {
			Name:         "主机管理",
			Description:  "主机诊断、清单和执行能力。",
			Constraints:  []string{"主机命令执行前需要再次确认影响范围。"},
			AllowedTools: []string{"hostListInventory", "credentialList"},
			Examples:     []string{"host-restart-service"},
		},
		"cicd": {
			Name:         "CI/CD",
			Description:  "流水线、发布和交付流程。",
			Constraints:  []string{"涉及触发类动作时先检查当前流水线状态。"},
			AllowedTools: []string{"cicdPipelineList", "clusterListInventory"},
			Examples:     []string{"pipeline-rerun"},
		},
	}
}

func defaultSubSceneRules() map[string]SubSceneRule {
	return map[string]SubSceneRule{
		"clusters": {
			ExtraConstraints: []string{"集群级操作需要明确目标集群。"},
		},
		"hosts": {
			ExtraConstraints: []string{"主机命令执行需要审批。"},
		},
	}
}

func mergeSceneConfigs(base, overrides map[string]SceneConfig) map[string]SceneConfig {
	out := make(map[string]SceneConfig, len(base)+len(overrides))
	for key, cfg := range base {
		out[key] = cfg
	}
	for key, cfg := range overrides {
		if current, ok := out[key]; ok {
			if strings.TrimSpace(cfg.Name) != "" {
				current.Name = cfg.Name
			}
			if strings.TrimSpace(cfg.Description) != "" {
				current.Description = cfg.Description
			}
			if len(cfg.Constraints) > 0 {
				current.Constraints = cloneStrings(cfg.Constraints)
			}
			if len(cfg.AllowedTools) > 0 {
				current.AllowedTools = cloneStrings(cfg.AllowedTools)
			}
			if len(cfg.BlockedTools) > 0 {
				current.BlockedTools = cloneStrings(cfg.BlockedTools)
			}
			if len(cfg.Examples) > 0 {
				current.Examples = cloneStrings(cfg.Examples)
			}
			if cfg.ApprovalConfig != nil {
				current.ApprovalConfig = cfg.ApprovalConfig
			}
			out[key] = current
			continue
		}
		out[key] = cfg
	}
	return out
}

func removeItems(items, blocked []string) []string {
	if len(items) == 0 {
		return nil
	}
	denied := make(map[string]struct{}, len(blocked))
	for _, item := range blocked {
		denied[strings.TrimSpace(item)] = struct{}{}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := denied[strings.TrimSpace(item)]; ok {
			continue
		}
		out = append(out, strings.TrimSpace(item))
	}
	return out
}
