// Package handler 提供监控告警服务的 HTTP 处理器。
//
// 本文件实现告警规则同步服务，负责将数据库中的告警规则
// 同步到 Prometheus 配置文件，并触发 Prometheus 配置重载。
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

// RuleSyncService 是告警规则同步服务。
//
// 负责将数据库中的启用状态告警规则同步到 Prometheus 告警规则文件，
// 并通过 HTTP 请求触发 Prometheus 重载配置。
type RuleSyncService struct {
	db        *gorm.DB       // 数据库连接
	rulesFile string         // 规则文件路径
	reloadURL string         // Prometheus 重载 URL
	client    *http.Client   // HTTP 客户端
	mu        sync.Mutex     // 并发锁
}

// promRulesFile 是 Prometheus 规则文件结构。
//
// 对应 Prometheus 告警规则 YAML 文件的顶层结构。
type promRulesFile struct {
	Groups []promRuleGroup `yaml:"groups"` // 规则组列表
}

// promRuleGroup 是 Prometheus 规则组结构。
//
// 一组相关的告警规则，共享相同的评估间隔。
type promRuleGroup struct {
	Name  string     `yaml:"name"`  // 规则组名称
	Rules []promRule `yaml:"rules"` // 规则列表
}

// promRule 是 Prometheus 告警规则结构。
//
// 定义单个告警规则的完整配置。
type promRule struct {
	Alert       string            `yaml:"alert"`                 // 告警名称
	Expr        string            `yaml:"expr"`                  // PromQL 表达式
	For         string            `yaml:"for,omitempty"`         // 持续时间
	Labels      map[string]string `yaml:"labels,omitempty"`      // 标签
	Annotations map[string]string `yaml:"annotations,omitempty"` // 注解
}

// NewRuleSyncService 创建规则同步服务实例。
//
// 从配置中读取 Prometheus 地址和规则文件路径，
// 初始化 HTTP 客户端用于触发配置重载。
//
// 参数:
//   - db: 数据库连接
//
// 返回: 初始化完成的 RuleSyncService 实例
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

// SyncRules 同步告警规则到 Prometheus。
//
// 从数据库读取所有启用的告警规则，转换为 Prometheus 格式，
// 写入规则文件并触发 Prometheus 配置重载。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 同步的规则数量和可能的错误
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

// convertRuleToPrometheus 将数据库规则转换为 Prometheus 规则格式。
//
// 解析操作符、阈值、标签和注解，生成完整的 PromQL 表达式。
//
// 参数:
//   - rule: 数据库中的告警规则
//
// 返回: Prometheus 格式的规则和可能的错误
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

// writeRulesFile 将规则写入文件。
//
// 确保 目录存在，将规则序列化为 YAML 格式并写入文件。
//
// 参数:
//   - file: Prometheus 规则文件结构
//
// 返回: 可能的错误
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

// reloadPrometheus 触发 Prometheus 配置重载。
//
// 向 Prometheus 的 /-/reload 端点发送 POST 请求。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 可能的错误
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

// StartPeriodic 启动定期同步任务。
//
// 在后台启动定时器，定期同步告警规则到 Prometheus。
//
// 参数:
//   - ctx: 上下文，用于控制任务生命周期
//   - interval: 同步间隔，小于等于 0 时默认 5 分钟
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
