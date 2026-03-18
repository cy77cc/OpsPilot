package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

type RuleSyncService struct {
	db        *gorm.DB
	rulesFile string
	reloadURL string
	client    *http.Client
	mu        sync.Mutex
}

type promRulesFile struct {
	Groups []promRuleGroup `yaml:"groups"`
}

type promRuleGroup struct {
	Name  string     `yaml:"name"`
	Rules []promRule `yaml:"rules"`
}

type promRule struct {
	Alert       string            `yaml:"alert"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

func NewRuleSyncService(db *gorm.DB) *RuleSyncService {
	cfg := config.CFG.Prometheus
	address := strings.TrimSpace(cfg.Address)

	// 如果 address 已设置但没有 scheme，添加 http:// 前缀
	if address != "" && !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}

	// 如果 address 为空，使用 host:port 构建
	if address == "" && strings.TrimSpace(cfg.Host) != "" {
		port := strings.TrimSpace(cfg.Port)
		if port == "" {
			port = "9090"
		}
		address = fmt.Sprintf("http://%s:%s", cfg.Host, port)
	}
	if address == "" {
		address = "http://prometheus:9090"
	}

	rulesFile := strings.TrimSpace(os.Getenv("PROMETHEUS_ALERTING_RULES_FILE"))
	if rulesFile == "" {
		rulesFile = "deploy/compose/prometheus/alerting_rules.yml"
	}

	return &RuleSyncService{
		db:        db,
		rulesFile: rulesFile,
		reloadURL: strings.TrimRight(address, "/") + "/-/reload",
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *RuleSyncService) SyncRules(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rules := make([]model.AlertRule, 0, 64)
	if err := s.db.WithContext(ctx).Where("enabled = 1").Order("id ASC").Find(&rules).Error; err != nil {
		return 0, err
	}

	file := promRulesFile{Groups: []promRuleGroup{{Name: "OpsPilot-alerts", Rules: make([]promRule, 0, len(rules))}}}
	for _, r := range rules {
		pr, err := convertRuleToPrometheus(r)
		if err != nil {
			return 0, err
		}
		file.Groups[0].Rules = append(file.Groups[0].Rules, pr)
	}

	if err := s.writeRulesFile(file); err != nil {
		return 0, err
	}
	if err := s.reloadPrometheus(ctx); err != nil {
		return 0, err
	}
	return len(rules), nil
}

func convertRuleToPrometheus(rule model.AlertRule) (promRule, error) {
	metric := strings.TrimSpace(rule.Metric)
	if metric == "" {
		return promRule{}, fmt.Errorf("rule %d metric is empty", rule.ID)
	}
	op := strings.TrimSpace(rule.Operator)
	switch op {
	case "", "gt", ">":
		op = ">"
	case "gte", ">=":
		op = ">="
	case "lt", "<":
		op = "<"
	case "lte", "<=":
		op = "<="
	case "eq", "=":
		op = "=="
	default:
		op = ">"
	}

	expr := strings.TrimSpace(rule.PromQLExpr)
	if expr == "" {
		expr = fmt.Sprintf("%s %s %v", metric, op, rule.Threshold)
	}
	labels := map[string]string{
		"severity": normalizeSeverity(rule.Severity),
		"rule_id":  fmt.Sprintf("%d", rule.ID),
	}
	if strings.TrimSpace(rule.Source) != "" {
		labels["source"] = strings.TrimSpace(rule.Source)
	}
	if strings.TrimSpace(rule.DimensionsJSON) != "" {
		var dim map[string]any
		if err := json.Unmarshal([]byte(rule.DimensionsJSON), &dim); err == nil {
			for k, v := range dim {
				key := strings.TrimSpace(k)
				if key == "" || strings.ContainsAny(key, " {}[]\t\n\r\"") {
					continue
				}
				labels[key] = fmt.Sprintf("%v", v)
			}
		}
	}
	if strings.TrimSpace(rule.LabelsJSON) != "" {
		var custom map[string]any
		if err := json.Unmarshal([]byte(rule.LabelsJSON), &custom); err == nil {
			for k, v := range custom {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				labels[key] = fmt.Sprintf("%v", v)
			}
		}
	}

	result := promRule{
		Alert:  strings.TrimSpace(rule.Name),
		Expr:   expr,
		Labels: labels,
		Annotations: map[string]string{
			"summary": strings.TrimSpace(rule.Name),
		},
	}
	if strings.TrimSpace(rule.AnnotationsJSON) != "" {
		var custom map[string]any
		if err := json.Unmarshal([]byte(rule.AnnotationsJSON), &custom); err == nil {
			for k, v := range custom {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				result.Annotations[key] = fmt.Sprintf("%v", v)
			}
		}
	}
	if result.Alert == "" {
		result.Alert = fmt.Sprintf("rule_%d", rule.ID)
	}
	if rule.DurationSec > 0 {
		result.For = (time.Duration(rule.DurationSec) * time.Second).String()
	}
	return result, nil
}

func (s *RuleSyncService) writeRulesFile(file promRulesFile) error {
	if err := os.MkdirAll(filepath.Dir(s.rulesFile), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(file)
	if err != nil {
		return err
	}
	return os.WriteFile(s.rulesFile, b, 0o644)
}

func (s *RuleSyncService) reloadPrometheus(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.reloadURL, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("prometheus reload failed: %d", resp.StatusCode)
	}
	return nil
}

func (s *RuleSyncService) StartPeriodic(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.SyncRules(runtimectx.Detach(ctx))
			}
		}
	}()
}
