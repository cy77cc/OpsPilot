package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
	"github.com/gin-gonic/gin"
)

func (h *AIHandler) toolPolicy(ctx context.Context, meta tools.ToolMeta, params map[string]any) error {
	uid, approvalToken := tools.ToolUserFromContext(ctx)
	if uid == 0 {
		return errors.New("unauthorized")
	}
	if !h.hasPermission(uid, meta.Permission) {
		return errors.New("permission denied")
	}
	if shouldSkipToolApproval(meta, params) {
		return nil
	}
	if meta.Mode == tools.ToolModeReadonly {
		return nil
	}
	runtime := tools.ToolRuntimeContextFromContext(ctx)
	requireConfirmation := toBool(runtime["require_confirmation"])
	if requireConfirmation {
		confirmationToken := strings.TrimSpace(logic.ToString(runtime["confirmation_token"]))
		confirmationSvc := logic.NewConfirmationService(h.svcCtx.DB)
		if confirmationToken == "" {
			return h.newConfirmationRequiredError(ctx, uid, runtime, meta, params, confirmationSvc)
		}
		cf, err := confirmationSvc.Get(ctx, confirmationToken)
		if err != nil {
			return h.newConfirmationRequiredError(ctx, uid, runtime, meta, params, confirmationSvc)
		}
		if cf.ToolName != meta.Name {
			return errors.New("confirmation tool mismatch")
		}
		if cf.RequestUserID != uid && !h.isAdmin(uid) {
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
		t := h.runtime.NewApproval(uid, ApprovalTicket{
			Tool:   meta.Name,
			Params: params,
			Risk:   meta.Risk,
			Mode:   meta.Mode,
			Meta:   meta,
		})
		return &tools.ApprovalRequiredError{
			Token:     t.ID,
			Tool:      t.Tool,
			ExpiresAt: t.ExpiresAt,
			Message:   "approval required",
		}
	}
	t, ok := h.runtime.GetApproval(approvalToken)
	if !ok {
		return errors.New("approval not found")
	}
	if t.Tool != meta.Name {
		return errors.New("approval tool mismatch")
	}
	if t.RequestUID != uid && !h.isAdmin(uid) {
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

func shouldSkipToolApproval(meta tools.ToolMeta, params map[string]any) bool {
	if meta.Name != "service_deploy" {
		return false
	}
	applyRequested := toBool(params["apply"])
	previewRequested := toBool(params["preview"])
	return previewRequested && !applyRequested
}

func (h *AIHandler) newConfirmationRequiredError(ctx context.Context, uid uint64, runtime map[string]any, meta tools.ToolMeta, params map[string]any, confirmationSvc *logic.ConfirmationService) error {
	metas := []tools.ToolMeta{}
	if h.svcCtx != nil && h.svcCtx.AI != nil {
		metas = h.svcCtx.AI.ToolMetas()
	}
	previewBuilder := logic.NewPreviewBuilder(h.svcCtx.DB, metas)
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
	return &tools.ConfirmationRequiredError{
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

func (h *AIHandler) hasPermission(uid uint64, code string) bool {
	if uid == 0 {
		return false
	}
	if h.isAdmin(uid) {
		return true
	}
	if code == "" {
		return true
	}
	perms, err := h.fetchPermissions(uid)
	if err != nil {
		return false
	}
	parts := strings.Split(code, ":")
	resource := code
	if len(parts) > 0 {
		resource = parts[0]
	}
	for _, p := range perms {
		if p == code || p == resource+":*" || p == "*:*" {
			return true
		}
	}
	return false
}

func (h *AIHandler) isAdmin(uid uint64) bool {
	var u model.User
	if err := h.svcCtx.DB.Select("id", "username").Where("id = ?", uid).First(&u).Error; err == nil {
		if strings.EqualFold(strings.TrimSpace(u.Username), "admin") {
			return true
		}
	}
	type roleRow struct {
		Code string `gorm:"column:code"`
	}
	var rows []roleRow
	err := h.svcCtx.DB.Table("roles").
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

func (h *AIHandler) fetchPermissions(uid uint64) ([]string, error) {
	type row struct {
		Code string `gorm:"column:code"`
	}
	var rows []row
	err := h.svcCtx.DB.Table("permissions").
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

func uidFromContext(c *gin.Context) (uint64, bool) {
	v, ok := c.Get("uid")
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case uint:
		return uint64(x), true
	case uint64:
		return x, true
	case int:
		return uint64(x), true
	case int64:
		return uint64(x), true
	case float64:
		return uint64(x), true
	default:
		return 0, false
	}
}
