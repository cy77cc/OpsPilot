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
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	model "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// RuleSyncService 是告警规则同步服务。
//
// 负责将数据库中的启用状态告警规则同步到 Prometheus 告警规则文件，
// 并通过 HTTP 请求触发 Prometheus 重载配置。
type RuleSyncService struct {
	db        *gorm.DB     // 数据库连接
	rulesFile string       // 规则文件路径
	rulesURL  string       // Prometheus rules API URL
	reloadURL string       // Prometheus 重载 URL
	client    *http.Client // HTTP 客户端
	mu        sync.Mutex   // 并发锁
}

const importedPrometheusSource = "prometheus"

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

type promRulesResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
	Data      struct {
		Groups []promRulesGroup `json:"groups"`
	} `json:"data"`
}

type promRulesGroup struct {
	Rules []promRulesAPIRule `json:"rules"`
}

type promRulesAPIRule struct {
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Query       string            `json:"query"`
	Duration    json.RawMessage   `json:"duration"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
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
		rulesURL:  strings.TrimRight(address, "/") + "/api/v1/rules",
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

	if err := s.syncPrometheusRulesToDB(ctx); err != nil {
		return 0, err
	}

	rules := make([]model.AlertRule, 0, 64)
	if err := s.db.WithContext(ctx).
		Where("enabled = ?", true).
		Where("COALESCE(source, '') <> ?", importedPrometheusSource).
		Order("id ASC").
		Find(&rules).Error; err != nil {
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
	if metric == "" && strings.TrimSpace(rule.PromQLExpr) == "" {
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

func (s *RuleSyncService) syncPrometheusRulesToDB(ctx context.Context) error {
	promRules, err := s.fetchPrometheusRules(ctx)
	if err != nil {
		return err
	}
	if len(promRules) == 0 {
		return nil
	}

	existing := make([]model.AlertRule, 0, len(promRules))
	if err := s.db.WithContext(ctx).
		Where("project_id IS NULL").
		Order("id ASC").
		Find(&existing).Error; err != nil {
		return err
	}

	existingByName := make(map[string]model.AlertRule, len(existing))
	for _, row := range existing {
		key := normalizeRuleName(row.Name)
		if key == "" {
			continue
		}
		if _, ok := existingByName[key]; !ok {
			existingByName[key] = row
		}
	}

	for _, pr := range promRules {
		converted := convertPrometheusRuleToModel(pr)
		key := normalizeRuleName(converted.Name)
		if key == "" {
			continue
		}
		if row, ok := existingByName[key]; ok {
			if err := s.updateRuleFromPrometheus(ctx, row, converted); err != nil {
				return err
			}
			continue
		}
		if err := s.db.WithContext(ctx).Create(&converted).Error; err != nil {
			return err
		}
		existingByName[key] = converted
	}
	return nil
}

func (s *RuleSyncService) fetchPrometheusRules(ctx context.Context) ([]promRule, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.rulesURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("prometheus rules query failed: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed promRulesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.Status != "success" {
		msg := strings.TrimSpace(parsed.Error)
		if msg == "" {
			msg = strings.TrimSpace(parsed.ErrorType)
		}
		if msg == "" {
			msg = "unknown error"
		}
		return nil, fmt.Errorf("prometheus rules query failed: %s", msg)
	}

	rules := make([]promRule, 0, 64)
	for _, group := range parsed.Data.Groups {
		for _, item := range group.Rules {
			if strings.TrimSpace(strings.ToLower(item.Type)) != "alerting" {
				continue
			}
			name := strings.TrimSpace(item.Name)
			expr := strings.TrimSpace(item.Query)
			if name == "" || expr == "" {
				continue
			}
			rules = append(rules, promRule{
				Alert:       name,
				Expr:        expr,
				For:         parsePrometheusDuration(item.Duration),
				Labels:      cloneStringMap(item.Labels),
				Annotations: cloneStringMap(item.Annotations),
			})
		}
	}
	return rules, nil
}

func (s *RuleSyncService) updateRuleFromPrometheus(ctx context.Context, existing, imported model.AlertRule) error {
	metric, operator, threshold, simpleExpr := parseSimplePromExpression(imported.PromQLExpr)

	updates := map[string]any{
		"name":             imported.Name,
		"promql_expr":      imported.PromQLExpr,
		"duration_sec":     imported.DurationSec,
		"labels_json":      imported.LabelsJSON,
		"annotations_json": imported.AnnotationsJSON,
		"severity":         imported.Severity,
		"source":           imported.Source,
		"state":            "enabled",
		"enabled":          true,
	}

	if imported.DurationSec <= 0 {
		delete(updates, "duration_sec")
	}
	if strings.TrimSpace(imported.Operator) == "" {
		delete(updates, "operator")
	}
	if imported.PromQLExpr == "" {
		delete(updates, "promql_expr")
	}
	if imported.LabelsJSON == "" {
		delete(updates, "labels_json")
	}
	if imported.AnnotationsJSON == "" {
		delete(updates, "annotations_json")
	}
	if simpleExpr {
		updates["metric"] = metric
		updates["operator"] = operator
		updates["threshold"] = threshold
	} else {
		delete(updates, "metric")
		delete(updates, "operator")
		delete(updates, "threshold")
	}
	if existing.Metric == "" && !simpleExpr {
		updates["metric"] = imported.Metric
		updates["operator"] = "gt"
		updates["threshold"] = 0
	}
	return s.db.WithContext(ctx).
		Model(&model.AlertRule{}).
		Where("id = ?", existing.ID).
		Updates(updates).Error
}

func convertPrometheusRuleToModel(rule promRule) model.AlertRule {
	expr := strings.TrimSpace(rule.Expr)
	metric, operator, threshold, ok := parseSimplePromExpression(expr)
	if !ok {
		metric = inferMetric(expr)
		operator = "gt"
		threshold = 0
	}
	if metric == "" {
		metric = "prometheus_expression"
	}
	labelsJSON := encodeJSONMap(rule.Labels)
	annotationsJSON := encodeJSONMap(rule.Annotations)
	severity := normalizeSeverity(rule.Labels["severity"])

	return model.AlertRule{
		Name:            strings.TrimSpace(rule.Alert),
		Metric:          metric,
		PromQLExpr:      expr,
		Operator:        operator,
		Threshold:       threshold,
		DurationSec:     parseDurationSeconds(rule.For),
		WindowSec:       3600,
		GranularitySec:  60,
		LabelsJSON:      labelsJSON,
		AnnotationsJSON: annotationsJSON,
		Severity:        severity,
		Source:          importedPrometheusSource,
		Scope:           "global",
		Enabled:         true,
		State:           "enabled",
	}
}

func parsePrometheusDuration(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		trimmed := strings.TrimSpace(str)
		if trimmed == "" {
			return ""
		}
		if sec, err := strconv.ParseFloat(trimmed, 64); err == nil && sec > 0 {
			return (time.Duration(sec * float64(time.Second))).String()
		}
		return trimmed
	}
	var num float64
	if err := json.Unmarshal(raw, &num); err == nil && num > 0 {
		return (time.Duration(num * float64(time.Second))).String()
	}
	return ""
}

func parseSimplePromExpression(expr string) (metric, operator string, threshold float64, ok bool) {
	compact := strings.Join(strings.Fields(strings.TrimSpace(expr)), " ")
	if compact == "" {
		return "", "", 0, false
	}

	ops := []string{">=", "<=", "==", "=", ">", "<"}
	selected := ""
	selectedIdx := -1
	for _, op := range ops {
		if idx := strings.Index(compact, op); idx > 0 {
			selected = op
			selectedIdx = idx
			break
		}
	}
	if selected == "" {
		return "", "", 0, false
	}

	left := strings.TrimSpace(compact[:selectedIdx])
	right := strings.TrimSpace(compact[selectedIdx+len(selected):])
	if left == "" || right == "" {
		return "", "", 0, false
	}

	n, err := strconv.ParseFloat(right, 64)
	if err != nil {
		return "", "", 0, false
	}
	if !isPromMetricName(left) {
		return "", "", 0, false
	}

	return left, toModelOperator(selected), n, true
}

func isPromMetricName(v string) bool {
	if v == "" {
		return false
	}
	for i, r := range v {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '_' || r == ':':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func inferMetric(expr string) string {
	for _, token := range strings.FieldsFunc(expr, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= 'A' && r <= 'Z':
			return false
		case r >= '0' && r <= '9':
			return false
		case r == '_' || r == ':':
			return false
		default:
			return true
		}
	}) {
		if isPromMetricName(token) {
			return token
		}
	}
	return ""
}

func toModelOperator(op string) string {
	switch strings.TrimSpace(op) {
	case ">":
		return "gt"
	case ">=":
		return "gte"
	case "<":
		return "lt"
	case "<=":
		return "lte"
	case "=", "==":
		return "eq"
	default:
		return "gt"
	}
}

func parseDurationSeconds(v string) int {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return 0
	}
	if sec, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return int(math.Round(sec))
	}
	d, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0
	}
	return int(d.Seconds())
}

func encodeJSONMap(data map[string]string) string {
	if len(data) == 0 {
		return ""
	}
	b, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(b)
}

func normalizeRuleName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
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
