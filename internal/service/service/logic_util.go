package service

import (
	"encoding/json"
	"strings"
)

func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func normalizeStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(v)
	}
	return out
}

func buildLegacyEnvs(cfg *StandardServiceConfig) string {
	if cfg == nil {
		return ""
	}
	b, _ := json.Marshal(cfg.Envs)
	return string(b)
}

func buildLegacyResources(cfg *StandardServiceConfig) string {
	if cfg == nil {
		return ""
	}
	b, _ := json.Marshal(map[string]any{"limits": cfg.Resources})
	return string(b)
}

func truncateStr(v string, max int) string {
	s := strings.TrimSpace(v)
	if len(s) <= max || max <= 0 {
		return s
	}
	return s[:max]
}

func ensureStandardConfig(cfg *StandardServiceConfig) *StandardServiceConfig {
	if cfg == nil {
		cfg = &StandardServiceConfig{
			Image:    "nginx:latest",
			Replicas: 1,
			Resources: map[string]string{
				"cpu":    "500m",
				"memory": "512Mi",
			},
		}
	}
	if strings.TrimSpace(cfg.Image) == "" {
		cfg.Image = "nginx:latest"
	}
	if cfg.Replicas <= 0 {
		cfg.Replicas = 1
	}
	if len(cfg.Ports) == 0 {
		cfg.Ports = []PortConfig{{
			Name:          "http",
			Protocol:      "TCP",
			ContainerPort: 8080,
			ServicePort:   80,
		}}
	}
	return cfg
}
