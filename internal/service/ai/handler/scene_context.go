package handler

import (
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"gopkg.in/yaml.v3"
)

type sceneMeta struct {
	Scene        string   `json:"scene"`
	Description  string   `json:"description"`
	Keywords     []string `json:"keywords"`
	Tools        []string `json:"tools"`
	ContextHints []string `json:"context_hints"`
}

type sceneMappingItem struct {
	Description  string   `yaml:"description"`
	Keywords     []string `yaml:"keywords"`
	Tools        []string `yaml:"tools"`
	ContextHints []string `yaml:"context_hints"`
}

type sceneMappingConfig struct {
	Mappings map[string]sceneMappingItem `yaml:"mappings"`
}

var (
	sceneRegistryOnce sync.Once
	sceneRegistry     map[string]sceneMeta
)

func loadSceneRegistry() {
	sceneRegistry = map[string]sceneMeta{}
	var content []byte
	var err error
	for _, path := range []string{
		"configs/scene_mappings.yaml",
		"../configs/scene_mappings.yaml",
		"../../configs/scene_mappings.yaml",
		"../../../configs/scene_mappings.yaml",
	} {
		content, err = os.ReadFile(path)
		if err == nil && len(content) > 0 {
			break
		}
	}
	if err != nil || len(content) == 0 {
		return
	}
	var cfg sceneMappingConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil || len(cfg.Mappings) == 0 {
		return
	}
	for scene, item := range cfg.Mappings {
		sceneRegistry[scene] = sceneMeta{
			Scene:        scene,
			Description:  item.Description,
			Keywords:     append([]string{}, item.Keywords...),
			Tools:        append([]string{}, item.Tools...),
			ContextHints: append([]string{}, item.ContextHints...),
		}
	}
}

func normalizeSceneKey(scene string) string {
	v := strings.TrimSpace(scene)
	v = strings.TrimPrefix(v, "scene:")
	return strings.ToLower(v)
}

func sceneMetaByKey(scene string) (sceneMeta, bool) {
	sceneRegistryOnce.Do(loadSceneRegistry)
	meta, ok := sceneRegistry[normalizeSceneKey(scene)]
	return meta, ok
}

func (h *AIHandler) sceneRecommendedTools(scene string) []tools.ToolMeta {
	if h == nil || h.ai == nil {
		return nil
	}
	meta, ok := sceneMetaByKey(scene)
	if !ok {
		return nil
	}
	all := h.ai.ToolMetas()
	metaByName := make(map[string]tools.ToolMeta, len(all))
	for _, item := range all {
		metaByName[item.Name] = item
	}
	out := make([]tools.ToolMeta, 0, len(meta.Tools))
	for _, name := range meta.Tools {
		if item, exists := metaByName[name]; exists {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
