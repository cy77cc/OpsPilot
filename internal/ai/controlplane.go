package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
	"gorm.io/gorm"
)

type toolMetaProvider interface {
	ToolMetas() []aitools.ToolMeta
}

// ControlPlane centralizes AI control-plane semantics that were previously owned by handlers.
type ControlPlane struct {
	db       *gorm.DB
	runtime  *logic.RuntimeStore
	provider toolMetaProvider
}

func NewControlPlane(db *gorm.DB, runtime *logic.RuntimeStore, provider toolMetaProvider) *ControlPlane {
	return &ControlPlane{db: db, runtime: runtime, provider: provider}
}

func (c *ControlPlane) FindMeta(name string) (aitools.ToolMeta, bool) {
	if c == nil || c.provider == nil {
		return aitools.ToolMeta{}, false
	}
	normalized := aitools.NormalizeToolName(name)
	for _, item := range c.provider.ToolMetas() {
		if item.Name == normalized {
			return item, true
		}
	}
	return aitools.ToolMeta{}, false
}

func (c *ControlPlane) HasPermission(uid uint64, code string) bool {
	if uid == 0 {
		return false
	}
	if c.IsAdmin(uid) {
		return true
	}
	if code == "" {
		return true
	}
	perms, err := c.fetchPermissions(uid)
	if err != nil {
		return false
	}
	resource := code
	if parts := strings.Split(code, ":"); len(parts) > 0 {
		resource = parts[0]
	}
	for _, p := range perms {
		if p == code || p == resource+":*" || p == "*:*" {
			return true
		}
	}
	return false
}

func (c *ControlPlane) IsAdmin(uid uint64) bool {
	if c == nil || c.db == nil || uid == 0 {
		return false
	}
	var u model.User
	if err := c.db.Select("id", "username").Where("id = ?", uid).First(&u).Error; err == nil {
		if strings.EqualFold(strings.TrimSpace(u.Username), "admin") {
			return true
		}
	}
	type roleRow struct {
		Code string `gorm:"column:code"`
	}
	var rows []roleRow
	err := c.db.Table("roles").
		Select("roles.code").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", uid).
		Scan(&rows).Error
	if err != nil {
		return false
	}
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.Code), "admin") {
			return true
		}
	}
	return false
}

func (c *ControlPlane) ToolPolicy(ctx context.Context, meta aitools.ToolMeta, params map[string]any) error {
	uid, approvalToken := aitools.ToolUserFromContext(ctx)
	if uid == 0 {
		return errors.New("unauthorized")
	}
	if !c.HasPermission(uid, meta.Permission) {
		return errors.New("permission denied")
	}
	if shouldSkipToolApproval(meta, params) {
		return nil
	}
	if meta.Mode == aitools.ToolModeReadonly {
		return nil
	}

	runtime := aitools.ToolRuntimeContextFromContext(ctx)
	requireConfirmation := toBool(runtime["require_confirmation"])
	if requireConfirmation {
		confirmationToken := strings.TrimSpace(logic.ToString(runtime["confirmation_token"]))
		confirmationSvc := logic.NewConfirmationService(c.db)
		if confirmationToken == "" {
			return c.newConfirmationRequiredError(ctx, uid, runtime, meta, params, confirmationSvc)
		}
		cf, err := confirmationSvc.Get(ctx, confirmationToken)
		if err != nil {
			return c.newConfirmationRequiredError(ctx, uid, runtime, meta, params, confirmationSvc)
		}
		if cf.ToolName != meta.Name {
			return errors.New("confirmation tool mismatch")
		}
		if cf.RequestUserID != uid && !c.IsAdmin(uid) {
			return errors.New("confirmation owner mismatch")
		}
		if time.Now().After(cf.ExpiresAt) {
			_, _ = confirmationSvc.ExpirePending(ctx, time.Now())
			return errors.New("confirmation expired")
		}
		if cf.Status != "confirmed" {
			return errors.New("confirmation not confirmed")
		}
	}

	if strings.TrimSpace(approvalToken) == "" {
		if c.runtime == nil {
			return errors.New("approval runtime unavailable")
		}
		t := c.runtime.NewApproval(uid, logic.ApprovalTicket{
			Tool:   meta.Name,
			Params: params,
			Risk:   meta.Risk,
			Mode:   meta.Mode,
			Meta:   meta,
		})
		return &aitools.ApprovalRequiredError{
			Token:     t.ID,
			Tool:      t.Tool,
			ExpiresAt: t.ExpiresAt,
			Message:   "approval required",
		}
	}

	if c.runtime == nil {
		return errors.New("approval runtime unavailable")
	}
	t, ok := c.runtime.GetApproval(approvalToken)
	if !ok {
		return errors.New("approval not found")
	}
	if t.Tool != meta.Name {
		return errors.New("approval tool mismatch")
	}
	if t.RequestUID != uid && !c.IsAdmin(uid) {
		return errors.New("approval owner mismatch")
	}
	if time.Now().After(t.ExpiresAt) {
		return errors.New("approval expired")
	}
	if t.Status != "approved" {
		return errors.New("approval not approved")
	}
	return nil
}

func (c *ControlPlane) newConfirmationRequiredError(ctx context.Context, uid uint64, runtime map[string]any, meta aitools.ToolMeta, params map[string]any, confirmationSvc *logic.ConfirmationService) error {
	metas := []aitools.ToolMeta{}
	if c != nil && c.provider != nil {
		metas = c.provider.ToolMetas()
	}
	previewBuilder := logic.NewPreviewBuilder(c.db, metas)
	preview := previewBuilder.BuildPreview(meta.Name, params)
	req, err := confirmationSvc.RequestConfirmation(ctx, logic.ConfirmationRequestInput{
		RequestUserID: uid,
		TraceID:       strings.TrimSpace(logic.ToString(runtime["trace_id"])),
		ToolName:      meta.Name,
		ToolMode:      string(meta.Mode),
		RiskLevel:     string(meta.Risk),
		ParamsJSON:    mustJSON(params),
		PreviewJSON:   mustJSON(preview),
		Timeout:       preview.Timeout,
	})
	if err != nil {
		return err
	}
	return &aitools.ConfirmationRequiredError{
		Token:     req.ID,
		Tool:      meta.Name,
		ExpiresAt: req.ExpiresAt,
		Preview: map[string]any{
			"tool":             preview.ToolName,
			"tool_description": preview.ToolDescription,
			"risk_level":       preview.RiskLevel,
			"mode":             preview.Mode,
			"target_resources": preview.TargetResources,
			"impact_scope":     preview.ImpactScope,
			"preview_diff":     preview.PreviewDiff,
		},
		Message: "confirmation required",
	}
}

func (c *ControlPlane) fetchPermissions(uid uint64) ([]string, error) {
	if c == nil || c.db == nil {
		return nil, errors.New("db unavailable")
	}
	type row struct {
		Code string `gorm:"column:code"`
	}
	var rows []row
	err := c.db.Table("permissions").
		Select("permissions.code").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", uid).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Code)
	}
	return out, nil
}

func shouldSkipToolApproval(meta aitools.ToolMeta, params map[string]any) bool {
	if meta.Name != "service_deploy" {
		return false
	}
	applyRequested := toBool(params["apply"])
	previewRequested := toBool(params["preview"])
	return previewRequested && !applyRequested
}

func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func toBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true") || strings.TrimSpace(x) == "1"
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	default:
		return false
	}
}

func normalizeSessionTitle(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, trimmed)
	rs := []rune(strings.TrimSpace(trimmed))
	if len(rs) > 64 {
		rs = rs[:64]
	}
	return strings.TrimSpace(string(rs))
}

func formatID(prefix string, ts time.Time) string {
	return fmt.Sprintf("%s-%d", prefix, ts.UnixNano())
}
