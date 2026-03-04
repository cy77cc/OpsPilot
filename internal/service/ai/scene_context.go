package ai

import (
	"sort"
	"strings"
	"sync"

	"github.com/cy77cc/k8s-manage/internal/ai/experts"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
)

type sceneMeta struct {
	Scene        string   `json:"scene"`
	Description  string   `json:"description"`
	Keywords     []string `json:"keywords"`
	Tools        []string `json:"tools"`
	ContextHints []string `json:"context_hints"`
}

var (
	sceneRegistryOnce sync.Once
	sceneRegistry     map[string]sceneMeta
)

func loadSceneRegistry() {
	sceneRegistry = map[string]sceneMeta{}
	cfg, err := experts.LoadSceneMappings("configs/scene_mappings.yaml")
	if err != nil || cfg == nil || len(cfg.Mappings) == 0 {
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

func (h *handler) sceneRecommendedTools(scene string) []tools.ToolMeta {
	if h == nil || h.svcCtx == nil || h.svcCtx.AI == nil {
		return nil
	}
	meta, ok := sceneMetaByKey(scene)
	if !ok {
		return nil
	}
	all := h.svcCtx.AI.ToolMetas()
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
