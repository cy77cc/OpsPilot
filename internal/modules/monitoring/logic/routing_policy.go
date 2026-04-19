package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"gorm.io/gorm"
)

// SeverityRouteInput 定义严重级别路由写入参数。
type SeverityRouteInput struct {
	Scope      string
	Severity   string
	ChannelIDs []uint
	Enabled    bool
}

// ResolveChannelsForAlert 根据规则绑定和严重级别路由解析目标渠道。
//
// 优先级:
// 1. 规则绑定渠道
// 2. 严重级别路由渠道
// 3. 默认日志渠道
func (l *Logic) ResolveChannelsForAlert(ctx context.Context, projectID uint, ruleID uint, severity string) ([]model.AlertNotificationChannel, error) {
	bound, err := l.listBoundChannels(ctx, projectID, ruleID)
	if err != nil {
		return nil, err
	}
	if len(bound) > 0 {
		return bound, nil
	}
	routed, err := l.listSeverityRoutedChannels(ctx, projectID, severity)
	if err != nil {
		return nil, err
	}
	if len(routed) > 0 {
		return routed, nil
	}
	return l.listDefaultLogChannel(ctx)
}

func (l *Logic) listBoundChannels(ctx context.Context, projectID, ruleID uint) ([]model.AlertNotificationChannel, error) {
	rows := make([]model.AlertNotificationChannel, 0, 8)
	err := l.svcCtx.DB.WithContext(ctx).
		Table("alert_rule_channel_bindings AS b").
		Select("c.*").
		Joins("JOIN alert_notification_channels AS c ON c.id = b.channel_id").
		Where("b.rule_id = ? AND b.enabled = 1", ruleID).
		Where("c.enabled = 1").
		Where("(b.project_id = ? OR b.project_id IS NULL)", projectID).
		Order("CASE WHEN b.project_id IS NULL THEN 1 ELSE 0 END ASC").
		Order("b.priority ASC").
		Order("b.id ASC").
		Find(&rows).Error
	return rows, err
}

func (l *Logic) listSeverityRoutedChannels(ctx context.Context, projectID uint, severity string) ([]model.AlertNotificationChannel, error) {
	normalizedSeverity := strings.ToLower(strings.TrimSpace(severity))
	if normalizedSeverity == "" {
		normalizedSeverity = "warning"
	}

	routes := make([]model.AlertSeverityRoute, 0, 4)
	q := l.svcCtx.DB.WithContext(ctx).
		Model(&model.AlertSeverityRoute{}).
		Where("enabled = 1").
		Where("LOWER(severity) = ?", normalizedSeverity)
	if projectID > 0 {
		q = q.Where("(project_id = ? OR project_id IS NULL)", projectID)
	} else {
		q = q.Where("project_id IS NULL")
	}
	if err := q.
		Order("CASE WHEN project_id IS NULL THEN 1 ELSE 0 END ASC").
		Order("id DESC").
		Find(&routes).Error; err != nil {
		return nil, err
	}

	for _, route := range routes {
		channelIDs := parseChannelIDs(route.ChannelIDsJSON)
		if len(channelIDs) == 0 {
			continue
		}
		channels, err := l.fetchEnabledChannelsByID(ctx, channelIDs)
		if err != nil {
			return nil, err
		}
		if len(channels) > 0 {
			return channels, nil
		}
	}
	return nil, nil
}

func (l *Logic) listDefaultLogChannel(ctx context.Context) ([]model.AlertNotificationChannel, error) {
	rows := make([]model.AlertNotificationChannel, 0, 4)
	err := l.svcCtx.DB.WithContext(ctx).
		Where("enabled = 1").
		Where("(type = ? OR provider = ?)", "log", "log").
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

// ListRuleChannelBindings 查询规则-渠道绑定配置。
func (l *Logic) ListRuleChannelBindings(ctx context.Context, projectID, ruleID uint) ([]model.AlertRuleChannelBinding, error) {
	rows := make([]model.AlertRuleChannelBinding, 0, 8)
	q := l.svcCtx.DB.WithContext(ctx).Where("rule_id = ?", ruleID)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	} else {
		q = q.Where("project_id IS NULL")
	}
	if err := q.Order("priority ASC").Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ReplaceRuleChannelBindings 全量替换规则-渠道绑定配置。
func (l *Logic) ReplaceRuleChannelBindings(ctx context.Context, projectID, ruleID uint, channelIDs []uint) error {
	return l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Model(&model.AlertRuleChannelBinding{}).Where("rule_id = ?", ruleID)
		if projectID > 0 {
			q = q.Where("project_id = ?", projectID)
		} else {
			q = q.Where("project_id IS NULL")
		}
		if err := q.Delete(&model.AlertRuleChannelBinding{}).Error; err != nil {
			return err
		}

		seen := make(map[uint]struct{}, len(channelIDs))
		for idx, channelID := range channelIDs {
			if channelID == 0 {
				continue
			}
			if _, ok := seen[channelID]; ok {
				continue
			}
			seen[channelID] = struct{}{}
			row := model.AlertRuleChannelBinding{
				RuleID:    ruleID,
				ChannelID: channelID,
				Priority:  idx + 1,
				Enabled:   true,
			}
			if projectID > 0 {
				pid := projectID
				row.ProjectID = &pid
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListSeverityRoutes 查询严重级别路由配置。
func (l *Logic) ListSeverityRoutes(ctx context.Context, projectID uint) ([]model.AlertSeverityRoute, error) {
	rows := make([]model.AlertSeverityRoute, 0, 16)
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertSeverityRoute{})
	if projectID > 0 {
		q = q.Where("(project_id = ? OR project_id IS NULL)", projectID)
	} else {
		q = q.Where("project_id IS NULL")
	}
	if err := q.
		Order("CASE WHEN project_id IS NULL THEN 1 ELSE 0 END ASC").
		Order("severity ASC").
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ReplaceSeverityRoutes 全量替换严重级别路由配置。
func (l *Logic) ReplaceSeverityRoutes(ctx context.Context, projectID uint, routes []SeverityRouteInput) error {
	return l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Model(&model.AlertSeverityRoute{})
		if projectID > 0 {
			q = q.Where("project_id = ?", projectID)
		} else {
			q = q.Where("project_id IS NULL")
		}
		if err := q.Delete(&model.AlertSeverityRoute{}).Error; err != nil {
			return err
		}

		for _, item := range routes {
			scope, severity, channelIDs, err := normalizeSeverityRouteWrite(projectID, item)
			if err != nil {
				return err
			}
			b, err := json.Marshal(channelIDs)
			if err != nil {
				return err
			}
			row := model.AlertSeverityRoute{
				Scope:          scope,
				Severity:       severity,
				ChannelIDsJSON: string(b),
				Enabled:        item.Enabled,
			}
			if projectID > 0 {
				pid := projectID
				row.ProjectID = &pid
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (l *Logic) CreateSeverityRoute(ctx context.Context, projectID uint, input SeverityRouteInput) (*model.AlertSeverityRoute, error) {
	scope, severity, channelIDs, err := normalizeSeverityRouteWrite(projectID, input)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(channelIDs)
	if err != nil {
		return nil, err
	}
	row := model.AlertSeverityRoute{
		Scope:          scope,
		Severity:       severity,
		ChannelIDsJSON: string(b),
		Enabled:        input.Enabled,
	}
	if projectID > 0 {
		pid := projectID
		row.ProjectID = &pid
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (l *Logic) CreateRuleChannelBinding(ctx context.Context, projectID, ruleID, channelID uint, priority int, enabled bool) (*model.AlertRuleChannelBinding, error) {
	if ruleID == 0 || channelID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	row := model.AlertRuleChannelBinding{}
	err := l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").Where("id = ?", ruleID).Take(&model.AlertRule{}).Error; err != nil {
			return err
		}
		if err := tx.Select("id").Where("id = ?", channelID).Take(&model.AlertNotificationChannel{}).Error; err != nil {
			return err
		}

		existing, err := findScopedBinding(tx, projectID, ruleID, channelID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existing != nil {
			row = *existing
			return nil
		}

		row = model.AlertRuleChannelBinding{
			RuleID:    ruleID,
			ChannelID: channelID,
			Priority:  normalizeBindingPriority(priority),
			Enabled:   enabled,
		}
		if projectID > 0 {
			pid := projectID
			row.ProjectID = &pid
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (l *Logic) UpdateSeverityRoute(ctx context.Context, id uint, projectID uint, input SeverityRouteInput) (*model.AlertSeverityRoute, error) {
	scope, severity, channelIDs, err := normalizeSeverityRouteWrite(projectID, input)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(channelIDs)
	if err != nil {
		return nil, err
	}
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertSeverityRoute{}).Where("id = ?", id)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	} else {
		q = q.Where("project_id IS NULL")
	}
	updates := map[string]any{
		"scope":            scope,
		"severity":         severity,
		"channel_ids_json": string(b),
		"enabled":          input.Enabled,
	}
	result := q.Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var row model.AlertSeverityRoute
	fetch := l.svcCtx.DB.WithContext(ctx).Where("id = ?", id)
	if projectID > 0 {
		fetch = fetch.Where("project_id = ?", projectID)
	} else {
		fetch = fetch.Where("project_id IS NULL")
	}
	if err := fetch.Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (l *Logic) UpdateRuleChannelBinding(ctx context.Context, projectID, ruleID, channelID uint, priority int, enabled bool) (*model.AlertRuleChannelBinding, error) {
	if ruleID == 0 || channelID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var row model.AlertRuleChannelBinding
	err := l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := findScopedBinding(tx, projectID, ruleID, channelID)
		if err != nil {
			return err
		}

		updates := map[string]any{
			"priority": normalizeBindingPriority(priority),
			"enabled":  enabled,
		}
		result := tx.Model(&model.AlertRuleChannelBinding{}).Where("id = ?", existing.ID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("id = ?", existing.ID).Take(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (l *Logic) DeleteSeverityRoute(ctx context.Context, id uint, projectID uint) error {
	q := l.svcCtx.DB.WithContext(ctx).Where("id = ?", id)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	} else {
		q = q.Where("project_id IS NULL")
	}
	result := q.Delete(&model.AlertSeverityRoute{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (l *Logic) DeleteRuleChannelBinding(ctx context.Context, projectID, ruleID, channelID uint) error {
	return l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Model(&model.AlertRuleChannelBinding{}).
			Where("rule_id = ? AND channel_id = ?", ruleID, channelID)
		if projectID > 0 {
			q = q.Where("project_id = ?", projectID)
		} else {
			q = q.Where("project_id IS NULL")
		}

		ids := make([]uint, 0, 2)
		if err := q.Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return gorm.ErrRecordNotFound
		}
		if len(ids) > 1 {
			return fmt.Errorf("multiple bindings matched scoped delete: rule_id=%d channel_id=%d project_id=%d", ruleID, channelID, projectID)
		}

		result := tx.Delete(&model.AlertRuleChannelBinding{}, ids[0])
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func findScopedBinding(tx *gorm.DB, projectID, ruleID, channelID uint) (*model.AlertRuleChannelBinding, error) {
	q := tx.Model(&model.AlertRuleChannelBinding{}).
		Where("rule_id = ? AND channel_id = ?", ruleID, channelID)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	} else {
		q = q.Where("project_id IS NULL")
	}

	ids := make([]uint, 0, 2)
	if err := q.Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if len(ids) > 1 {
		return nil, fmt.Errorf("multiple bindings matched scoped write: rule_id=%d channel_id=%d project_id=%d", ruleID, channelID, projectID)
	}

	var row model.AlertRuleChannelBinding
	if err := tx.Where("id = ?", ids[0]).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func parseChannelIDs(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err == nil {
		return ids
	}
	var ints []int
	if err := json.Unmarshal([]byte(raw), &ints); err != nil {
		return nil
	}
	ids = make([]uint, 0, len(ints))
	for _, id := range ints {
		if id > 0 {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

func (l *Logic) fetchEnabledChannelsByID(ctx context.Context, channelIDs []uint) ([]model.AlertNotificationChannel, error) {
	if len(channelIDs) == 0 {
		return nil, nil
	}
	rows := make([]model.AlertNotificationChannel, 0, len(channelIDs))
	if err := l.svcCtx.DB.WithContext(ctx).
		Where("enabled = 1 AND id IN ?", channelIDs).
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func mustJSONChannelIDs(channelIDs []uint) string {
	b, err := json.Marshal(channelIDs)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func validateRouteScope(scope string) error {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", "global", "project":
		return nil
	default:
		return fmt.Errorf("invalid route scope: %s", scope)
	}
}

func normalizeSeverityRouteWrite(projectID uint, input SeverityRouteInput) (string, string, []uint, error) {
	scope := strings.ToLower(strings.TrimSpace(input.Scope))
	severity := strings.ToLower(strings.TrimSpace(input.Severity))
	if severity == "" {
		return "", "", nil, fmt.Errorf("severity is required")
	}
	switch severity {
	case "critical", "warning", "info":
	default:
		return "", "", nil, fmt.Errorf("invalid route severity: %s", severity)
	}
	if scope == "" {
		if projectID > 0 {
			scope = "project"
		} else {
			scope = "global"
		}
	}
	if err := validateRouteScope(scope); err != nil {
		return "", "", nil, err
	}
	if scope == "project" && projectID == 0 {
		return "", "", nil, fmt.Errorf("project scope requires project id")
	}

	channelIDs := make([]uint, 0, len(input.ChannelIDs))
	seen := make(map[uint]struct{}, len(input.ChannelIDs))
	for _, channelID := range input.ChannelIDs {
		if channelID == 0 {
			continue
		}
		if _, exists := seen[channelID]; exists {
			continue
		}
		seen[channelID] = struct{}{}
		channelIDs = append(channelIDs, channelID)
	}
	return scope, severity, channelIDs, nil
}

func normalizeBindingPriority(priority int) int {
	if priority > 0 {
		return priority
	}
	return 100
}
