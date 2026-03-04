package experts

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

const defaultExpertConfigPath = "configs/experts.yaml"

const defaultPersonaSuffix = `
请优先选择只读工具进行诊断，并在执行高风险操作前明确说明影响与前置条件。`

type Registry struct {
	ctx        context.Context
	configPath string
	allTools   map[string]tool.InvokableTool
	chatModel  model.ToolCallingChatModel

	mu      sync.RWMutex
	experts map[string]*Expert
}

func NewExpertRegistry(ctx context.Context, configPath string, allTools map[string]tool.InvokableTool, chatModel model.ToolCallingChatModel) (*Registry, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(configPath) == "" {
		configPath = defaultExpertConfigPath
	}
	r := &Registry{
		ctx:        ctx,
		configPath: configPath,
		allTools:   allTools,
		chatModel:  chatModel,
		experts:    map[string]*Expert{},
	}
	if err := r.Load(); err != nil {
		if err := r.loadDefaults(); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) Load() error {
	cfg, err := LoadExpertsConfig(r.configPath)
	if err != nil {
		return err
	}
	loaded := make(map[string]*Expert, len(cfg.Experts))
	for _, item := range cfg.Experts {
		expert, err := r.instantiateExpert(item)
		if err != nil {
			return err
		}
		loaded[item.Name] = expert
	}
	r.mu.Lock()
	r.experts = loaded
	r.mu.Unlock()
	return nil
}

func (r *Registry) loadDefaults() error {
	defaults := []ExpertConfig{
		{
			Name:         "general_expert",
			DisplayName:  "通用运维专家",
			Persona:      "你是通用运维专家，负责综合诊断与信息整合。",
			ToolPatterns: []string{"*"},
			Keywords:     []string{"运维", "diagnose", "诊断"},
			Capabilities: []string{"通用诊断"},
			RiskLevel:    "low",
		},
	}
	loaded := make(map[string]*Expert, len(defaults))
	for _, item := range defaults {
		expert, err := r.instantiateExpert(item)
		if err != nil {
			return err
		}
		loaded[item.Name] = expert
	}
	r.mu.Lock()
	r.experts = loaded
	r.mu.Unlock()
	return nil
}

func (r *Registry) instantiateExpert(cfg ExpertConfig) (*Expert, error) {
	selected := r.filterTools(cfg.ToolPatterns)
	agent, err := r.buildAgent(cfg.Persona, selected)
	if err != nil {
		return nil, err
	}
	domains := make([]DomainWeight, 0, len(cfg.Domains))
	domains = append(domains, cfg.Domains...)
	return &Expert{
		Name:         cfg.Name,
		DisplayName:  cfg.DisplayName,
		Persona:      cfg.Persona,
		ToolPatterns: append([]string{}, cfg.ToolPatterns...),
		Domains:      domains,
		Keywords:     append([]string{}, cfg.Keywords...),
		Capabilities: append([]string{}, cfg.Capabilities...),
		RiskLevel:    cfg.RiskLevel,
		Agent:        agent,
		Tools:        selected,
	}, nil
}

func (r *Registry) buildAgent(persona string, selected map[string]tool.InvokableTool) (*react.Agent, error) {
	if r.chatModel == nil {
		return nil, nil
	}
	baseTools := make([]tool.BaseTool, 0, len(selected))
	for _, item := range selected {
		bt, ok := item.(tool.BaseTool)
		if !ok {
			continue
		}
		baseTools = append(baseTools, bt)
	}
	agent, err := react.NewAgent(r.ctx, &react.AgentConfig{
		ToolCallingModel: r.chatModel,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: baseTools},
		MaxStep:          20,
		MessageModifier:  react.NewPersonaModifier(strings.TrimSpace(persona) + "\n" + strings.TrimSpace(defaultPersonaSuffix)),
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (r *Registry) filterTools(patterns []string) map[string]tool.InvokableTool {
	if len(r.allTools) == 0 {
		return map[string]tool.InvokableTool{}
	}
	if len(patterns) == 0 {
		patterns = []string{"*"}
	}
	out := make(map[string]tool.InvokableTool)
	for name, t := range r.allTools {
		for _, pattern := range patterns {
			p := strings.TrimSpace(pattern)
			if p == "" {
				continue
			}
			ok, err := filepath.Match(p, name)
			if err != nil {
				continue
			}
			if ok {
				out[name] = t
				break
			}
		}
	}
	if len(out) == 0 {
		for name, t := range r.allTools {
			out[name] = t
		}
	}
	return out
}

func (r *Registry) GetExpert(name string) (*Expert, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	expert, ok := r.experts[name]
	return expert, ok
}

func (r *Registry) ListExperts() []*Expert {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.experts))
	for name := range r.experts {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*Expert, 0, len(names))
	for _, name := range names {
		out = append(out, r.experts[name])
	}
	return out
}

func (r *Registry) Reload() error {
	if err := r.Load(); err != nil {
		return fmt.Errorf("reload experts config: %w", err)
	}
	return nil
}

func (r *Registry) MatchByKeywords(content string) []*RankedExpert {
	text := strings.ToLower(strings.TrimSpace(content))
	if text == "" {
		return nil
	}
	candidates := r.ListExperts()
	out := make([]*RankedExpert, 0, len(candidates))
	for _, expert := range candidates {
		if expert == nil {
			continue
		}
		score := 0.0
		for _, kw := range expert.Keywords {
			key := strings.ToLower(strings.TrimSpace(kw))
			if key == "" {
				continue
			}
			if strings.Contains(text, key) {
				score += 1
			}
		}
		if score > 0 {
			out = append(out, &RankedExpert{Expert: expert, Score: score})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func (r *Registry) MatchByDomain(domain string) []*RankedExpert {
	key := strings.ToLower(strings.TrimSpace(domain))
	if key == "" {
		return nil
	}
	candidates := r.ListExperts()
	out := make([]*RankedExpert, 0, len(candidates))
	for _, expert := range candidates {
		if expert == nil {
			continue
		}
		for _, item := range expert.Domains {
			if strings.EqualFold(strings.TrimSpace(item.Name), key) {
				out = append(out, &RankedExpert{Expert: expert, Score: item.Weight})
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}
