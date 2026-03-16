package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	cicdv1 "github.com/cy77cc/OpsPilot/api/cicd/v1"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/cicd/repo"
	deploymentlogic "github.com/cy77cc/OpsPilot/internal/service/deployment"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

const timelineTTL = 30 * time.Second

type Logic struct {
	svcCtx      *svc.ServiceContext
	repo        *repo.Repository
	deployLogic *deploymentlogic.Logic
}

func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	return &Logic{
		svcCtx:      svcCtx,
		repo:        repo.New(svcCtx.DB),
		deployLogic: deploymentlogic.NewLogic(svcCtx),
	}
}

func (l *Logic) UpsertServiceCIConfig(ctx context.Context, uid uint, serviceID uint, req UpsertServiceCIConfigReq) (*cicdv1.ServiceCIConfigResp, error) {
	mode := normalizeTriggerMode(req.TriggerMode)
	if mode == "" {
		return nil, fmt.Errorf("trigger_mode must be one of: manual, source-event, both")
	}
	if strings.TrimSpace(req.RepoURL) == "" || strings.TrimSpace(req.ArtifactTarget) == "" {
		return nil, fmt.Errorf("repo_url and artifact_target are required")
	}
	row, err := l.repo.UpsertServiceCIConfig(ctx, model.CICDServiceCIConfig{
		ServiceID:      serviceID,
		RepoURL:        strings.TrimSpace(req.RepoURL),
		Branch:         defaultIfEmpty(strings.TrimSpace(req.Branch), "main"),
		BuildStepsJSON: mustJSON(req.BuildSteps),
		ArtifactTarget: strings.TrimSpace(req.ArtifactTarget),
		TriggerMode:    mode,
		Status:         "active",
		UpdatedBy:      uid,
	})
	if err != nil {
		return nil, err
	}
	_ = l.writeAudit(ctx, serviceID, 0, 0, "ci.config.updated", uid, map[string]any{"ci_config_id": row.ID, "trigger_mode": row.TriggerMode})
	l.invalidateTimelineCache(ctx, serviceID)
	return toServiceCIConfigResp(row), nil
}

func (l *Logic) GetServiceCIConfig(ctx context.Context, serviceID uint) (*cicdv1.ServiceCIConfigResp, error) {
	row, err := l.repo.GetServiceCIConfig(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	return toServiceCIConfigResp(row), nil
}

func (l *Logic) DeleteServiceCIConfig(ctx context.Context, uid uint, serviceID uint) error {
	if err := l.repo.DeleteServiceCIConfig(ctx, serviceID); err != nil {
		return err
	}
	_ = l.writeAudit(ctx, serviceID, 0, 0, "ci.config.deleted", uid, map[string]any{"service_id": serviceID})
	l.invalidateTimelineCache(ctx, serviceID)
	return nil
}

func (l *Logic) TriggerCIRun(ctx context.Context, uid uint, serviceID uint, req TriggerCIRunReq) (*cicdv1.CIRunResp, error) {
	cfg, err := l.repo.GetServiceCIConfig(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("service ci config not found: %w", err)
	}
	triggerType := normalizeTriggerType(req.TriggerType)
	if triggerType == "" {
		return nil, fmt.Errorf("trigger_type must be manual or source-event")
	}
	if !allowTrigger(cfg.TriggerMode, triggerType) {
		_ = l.writeAudit(ctx, serviceID, 0, 0, "ci.run.blocked", uid, map[string]any{"trigger_type": triggerType, "reason": "trigger mode mismatch"})
		return nil, fmt.Errorf("trigger mode %s does not allow trigger type %s", cfg.TriggerMode, triggerType)
	}
	run, err := l.repo.CreateCIRun(ctx, model.CICDServiceCIRun{
		ServiceID:   serviceID,
		CIConfigID:  cfg.ID,
		TriggerType: triggerType,
		Status:      "queued",
		Reason:      strings.TrimSpace(req.Reason),
		TriggeredBy: uid,
		TriggeredAt: time.Now(),
	})
	if err != nil {
		return nil, err
	}
	_ = l.writeAudit(ctx, serviceID, 0, 0, "ci.run.queued", uid, map[string]any{"ci_run_id": run.ID, "trigger_type": run.TriggerType})
	l.invalidateTimelineCache(ctx, serviceID)
	return toCIRunResp(run), nil
}

func (l *Logic) ListCIRuns(ctx context.Context, serviceID uint) ([]cicdv1.CIRunResp, error) {
	rows, err := l.repo.ListCIRuns(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	out := make([]cicdv1.CIRunResp, 0, len(rows))
	for i := range rows {
		out = append(out, *toCIRunResp(&rows[i]))
	}
	return out, nil
}

func (l *Logic) UpsertDeploymentCDConfig(ctx context.Context, uid uint, deploymentID uint, req UpsertDeploymentCDConfigReq) (*cicdv1.DeploymentCDConfigResp, error) {
	runtimeType := normalizeRuntimeType(req.RuntimeType)
	if strings.TrimSpace(req.RuntimeType) != "" && runtimeType == "" {
		return nil, fmt.Errorf("runtime_type must be one of: k8s, compose")
	}
	if runtimeType == "" {
		runtimeType = "k8s"
	}
	strategy := normalizeStrategy(req.Strategy)
	if strategy == "" {
		return nil, fmt.Errorf("strategy must be one of: rolling, blue-green, canary")
	}
	if runtimeType == "compose" && strategy == "canary" {
		return nil, fmt.Errorf("compose runtime does not support canary strategy")
	}
	if strategy == "canary" {
		if _, ok := req.StrategyConfig["traffic_percent"]; !ok {
			return nil, fmt.Errorf("canary strategy requires strategy_config.traffic_percent")
		}
		if _, ok := req.StrategyConfig["steps"]; !ok {
			return nil, fmt.Errorf("canary strategy requires strategy_config.steps")
		}
	}
	row, err := l.repo.UpsertDeploymentCDConfig(ctx, model.CICDDeploymentCDConfig{
		DeploymentID:       deploymentID,
		Env:                defaultIfEmpty(strings.TrimSpace(req.Env), "staging"),
		RuntimeType:        runtimeType,
		Strategy:           strategy,
		StrategyConfigJSON: mustJSON(req.StrategyConfig),
		ApprovalRequired:   req.ApprovalRequired,
		UpdatedBy:          uid,
	})
	if err != nil {
		return nil, err
	}
	_ = l.writeAudit(ctx, 0, deploymentID, 0, "cd.config.updated", uid, map[string]any{"cd_config_id": row.ID, "env": row.Env, "runtime": row.RuntimeType, "strategy": row.Strategy})
	return toCDConfigResp(row), nil
}

func (l *Logic) GetDeploymentCDConfig(ctx context.Context, deploymentID uint, env, runtimeType string) (*cicdv1.DeploymentCDConfigResp, error) {
	row, err := l.repo.GetDeploymentCDConfig(ctx, deploymentID, strings.TrimSpace(env), normalizeRuntimeType(runtimeType))
	if err != nil {
		return nil, err
	}
	return toCDConfigResp(row), nil
}

func (l *Logic) TriggerRelease(ctx context.Context, uid uint, req TriggerReleaseReq) (*cicdv1.ReleaseResp, error) {
	runtimeType := normalizeRuntimeType(req.RuntimeType)
	if strings.TrimSpace(req.RuntimeType) != "" && runtimeType == "" {
		return nil, fmt.Errorf("runtime_type must be one of: k8s, compose")
	}
	targetID, resolvedRuntime, resolutionSource, err := l.resolveReleaseTarget(ctx, req.ServiceID, req.DeploymentID, strings.TrimSpace(req.Env), runtimeType)
	if err != nil {
		return nil, err
	}
	if runtimeType == "" {
		runtimeType = resolvedRuntime
	}
	cfg := &model.CICDDeploymentCDConfig{
		DeploymentID: targetID,
		Env:          defaultIfEmpty(strings.TrimSpace(req.Env), "staging"),
		RuntimeType:  defaultIfEmpty(runtimeType, resolvedRuntime),
		Strategy:     "rolling",
	}
	if loaded, cfgErr := l.repo.GetDeploymentCDConfig(ctx, targetID, strings.TrimSpace(req.Env), runtimeType); cfgErr == nil {
		cfg = loaded
		if runtimeType == "" {
			runtimeType = cfg.RuntimeType
		}
	} else if !errors.Is(cfgErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load cd config: %w", cfgErr)
	}
	if runtimeType == "" {
		runtimeType = normalizeRuntimeType(cfg.RuntimeType)
	}
	if runtimeType == "" {
		runtimeType = "k8s"
	}

	previewReq := deploymentlogic.ReleasePreviewReq{
		ServiceID: req.ServiceID,
		TargetID:  targetID,
		Env:       strings.TrimSpace(req.Env),
		Strategy:  cfg.Strategy,
	}
	preview, err := l.deployLogic.PreviewRelease(ctx, previewReq)
	if err != nil {
		return nil, err
	}
	applyReq := previewReq
	applyReq.PreviewToken = preview.PreviewToken
	applyReq.TriggerSource = defaultIfEmpty(strings.TrimSpace(req.TriggerSource), "ci")
	applyReq.CIRunID = req.CIRunID
	applyReq.TriggerContext = map[string]any{
		"entry":         "cicd.release",
		"version":       strings.TrimSpace(req.Version),
		"deployment_id": targetID,
		"runtime_type":  runtimeType,
		"ci_run_id":     req.CIRunID,
		"target_source": resolutionSource,
	}
	resp, err := l.deployLogic.ApplyRelease(ctx, uint64(uid), applyReq)
	if err != nil {
		return nil, err
	}
	release, err := l.deployLogic.GetRelease(ctx, resp.ReleaseID)
	if err != nil {
		return nil, err
	}
	_ = l.writeAudit(ctx, req.ServiceID, targetID, release.ID, "release.triggered", uid, map[string]any{
		"status":             release.Status,
		"approval_required":  cfg.ApprovalRequired,
		"version":            strings.TrimSpace(req.Version),
		"unified_release_id": release.ID,
		"target_source":      resolutionSource,
	})
	l.invalidateTimelineCache(ctx, req.ServiceID)
	return toReleaseRespFromDeployment(release, targetID, strings.TrimSpace(req.Version)), nil
}

func (l *Logic) resolveReleaseTarget(ctx context.Context, serviceID, deploymentID uint, env, runtimeType string) (uint, string, string, error) {
	if deploymentID > 0 {
		var target model.DeploymentTarget
		if err := l.svcCtx.DB.WithContext(ctx).First(&target, deploymentID).Error; err != nil {
			return 0, "", "", fmt.Errorf("deployment target not found: %w", err)
		}
		resolved := normalizeRuntimeType(target.TargetType)
		if resolved == "" {
			resolved = normalizeRuntimeType(target.RuntimeType)
		}
		return deploymentID, defaultIfEmpty(resolved, runtimeType), "explicit", nil
	}

	var svc model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&svc, serviceID).Error; err != nil {
		return 0, "", "", err
	}
	rt := normalizeRuntimeType(runtimeType)
	if rt == "" {
		rt = normalizeRuntimeType(defaultIfEmpty(svc.RenderTarget, svc.RuntimeType))
	}
	if rt == "" {
		rt = "k8s"
	}
	scopeEnv := strings.TrimSpace(defaultIfEmpty(env, svc.Env))

	var sdt model.ServiceDeployTarget
	if err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ? AND is_default = 1", serviceID).First(&sdt).Error; err == nil {
		targetRuntime := normalizeRuntimeType(defaultIfEmpty(sdt.DeployTarget, rt))
		if targetRuntime == "" {
			targetRuntime = rt
		}
		q := l.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentTarget{}).Where("status = ? AND target_type = ?", "active", targetRuntime)
		if targetRuntime == "compose" {
			q = q.Where("id = ?", sdt.ClusterID)
		} else {
			q = q.Where("cluster_id = ?", sdt.ClusterID)
		}
		var target model.DeploymentTarget
		if err := q.Order("id DESC").First(&target).Error; err == nil {
			return target.ID, targetRuntime, "service_default", nil
		}
	}

	q := l.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentTarget{}).
		Where("target_type = ? AND status = ?", rt, "active")
	if svc.ProjectID > 0 {
		q = q.Where("project_id = ?", svc.ProjectID)
	}
	if svc.TeamID > 0 {
		q = q.Where("team_id = ?", svc.TeamID)
	}
	if scopeEnv != "" {
		q = q.Where("env = ? OR env = ''", scopeEnv)
	}
	var target model.DeploymentTarget
	if err := q.Order("CASE WHEN readiness_status = 'ready' THEN 0 WHEN readiness_status = 'unknown' THEN 1 ELSE 2 END, id DESC").First(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, "", "", fmt.Errorf(
				"deploy target not configured (project_id=%d, team_id=%d, env=%s, target_type=%s): no active deployment target found for service scope; hint: 配置 CD deployment_id、服务默认目标或创建匹配作用域的 active deployment target",
				svc.ProjectID,
				svc.TeamID,
				defaultIfEmpty(scopeEnv, "staging"),
				rt,
			)
		}
		return 0, "", "", err
	}
	return target.ID, rt, "scope_fallback", nil
}

func (l *Logic) ApproveRelease(ctx context.Context, uid uint, releaseID uint, comment string) (*cicdv1.ReleaseResp, error) {
	resp, err := l.deployLogic.ApproveRelease(ctx, releaseID, uint64(uid), comment)
	if err != nil {
		return nil, err
	}
	row, err := l.deployLogic.GetRelease(ctx, resp.ReleaseID)
	if err != nil {
		return nil, err
	}
	_ = l.writeAudit(ctx, row.ServiceID, row.TargetID, row.ID, "release.approved", uid, map[string]any{"comment": strings.TrimSpace(comment)})
	l.invalidateTimelineCache(ctx, row.ServiceID)
	return toReleaseRespFromDeployment(row, row.TargetID, ""), nil
}

func (l *Logic) RejectRelease(ctx context.Context, uid uint, releaseID uint, comment string) (*cicdv1.ReleaseResp, error) {
	resp, err := l.deployLogic.RejectRelease(ctx, releaseID, uint64(uid), comment)
	if err != nil {
		return nil, err
	}
	row, err := l.deployLogic.GetRelease(ctx, resp.ReleaseID)
	if err != nil {
		return nil, err
	}
	_ = l.writeAudit(ctx, row.ServiceID, row.TargetID, row.ID, "release.rejected", uid, map[string]any{"comment": strings.TrimSpace(comment)})
	l.invalidateTimelineCache(ctx, row.ServiceID)
	return toReleaseRespFromDeployment(row, row.TargetID, ""), nil
}

func (l *Logic) RollbackRelease(ctx context.Context, uid uint, releaseID uint, targetVersion, comment string) (*cicdv1.ReleaseResp, error) {
	rollback, err := l.deployLogic.RollbackRelease(ctx, releaseID, uint64(uid))
	if err != nil {
		return nil, err
	}
	row, err := l.deployLogic.GetRelease(ctx, rollback.ReleaseID)
	if err != nil {
		return nil, err
	}
	_ = l.writeAudit(ctx, row.ServiceID, row.TargetID, row.ID, "release.rolled_back", uid, map[string]any{
		"runtime":        row.RuntimeType,
		"target_version": strings.TrimSpace(targetVersion),
		"comment":        strings.TrimSpace(comment),
	})
	l.invalidateTimelineCache(ctx, row.ServiceID)
	return toReleaseRespFromDeployment(row, row.TargetID, strings.TrimSpace(targetVersion)), nil
}

func (l *Logic) ListReleases(ctx context.Context, serviceID, deploymentID uint, runtimeType string) ([]cicdv1.ReleaseResp, error) {
	rows, err := l.deployLogic.ListReleases(ctx, serviceID, deploymentID, normalizeRuntimeType(runtimeType))
	if err != nil {
		return nil, err
	}
	out := make([]cicdv1.ReleaseResp, 0, len(rows)+8)
	for i := range rows {
		out = append(out, *toReleaseRespFromDeployment(&rows[i], rows[i].TargetID, ""))
	}

	legacyRows, err := l.repo.ListReleases(ctx, serviceID, deploymentID, normalizeRuntimeType(runtimeType))
	if err != nil {
		return out, nil
	}
	for i := range legacyRows {
		out = append(out, *toReleaseResp(&legacyRows[i]))
	}
	return out, nil
}

func (l *Logic) ListApprovals(ctx context.Context, releaseID uint) ([]cicdv1.ReleaseApprovalResp, error) {
	var deployRows []model.DeploymentReleaseApproval
	if err := l.svcCtx.DB.WithContext(ctx).Where("release_id = ?", releaseID).Order("id DESC").Find(&deployRows).Error; err == nil && len(deployRows) > 0 {
		out := make([]cicdv1.ReleaseApprovalResp, 0, len(deployRows))
		for i := range deployRows {
			out = append(out, cicdv1.ReleaseApprovalResp{
				ID:         deployRows[i].ID,
				ReleaseID:  deployRows[i].ReleaseID,
				ApproverID: deployRows[i].ApproverID,
				Decision:   deployRows[i].Decision,
				Comment:    deployRows[i].Comment,
				CreatedAt:  deployRows[i].CreatedAt,
			})
		}
		return out, nil
	}

	rows, err := l.repo.ListApprovals(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	out := make([]cicdv1.ReleaseApprovalResp, 0, len(rows))
	for i := range rows {
		out = append(out, cicdv1.ReleaseApprovalResp{ID: rows[i].ID, ReleaseID: rows[i].ReleaseID, ApproverID: rows[i].ApproverID, Decision: rows[i].Decision, Comment: rows[i].Comment, CreatedAt: rows[i].CreatedAt})
	}
	return out, nil
}

func (l *Logic) ServiceTimeline(ctx context.Context, serviceID uint) ([]cicdv1.ReleaseTimelineEventResp, error) {
	cacheKey := fmt.Sprintf("cicd:timeline:service:%d", serviceID)
	if l.svcCtx.Rdb != nil {
		if raw, err := l.svcCtx.Rdb.Get(ctx, cacheKey).Result(); err == nil && strings.TrimSpace(raw) != "" {
			var cached []cicdv1.ReleaseTimelineEventResp
			if jerr := json.Unmarshal([]byte(raw), &cached); jerr == nil {
				return cached, nil
			}
		}
	}
	rows, err := l.repo.ListAuditEventsByService(ctx, serviceID, 200)
	if err != nil {
		return nil, err
	}
	out := make([]cicdv1.ReleaseTimelineEventResp, 0, len(rows))
	for i := range rows {
		out = append(out, cicdv1.ReleaseTimelineEventResp{
			ID:           rows[i].ID,
			ServiceID:    rows[i].ServiceID,
			DeploymentID: rows[i].DeploymentID,
			ReleaseID:    rows[i].ReleaseID,
			EventType:    rows[i].EventType,
			ActorID:      rows[i].ActorID,
			Payload:      parseAnyJSON(rows[i].PayloadJSON),
			CreatedAt:    rows[i].CreatedAt,
		})
	}
	if l.svcCtx.Rdb != nil {
		if raw, err := json.Marshal(out); err == nil {
			_ = l.svcCtx.Rdb.Set(ctx, cacheKey, string(raw), timelineTTL).Err()
		}
	}
	return out, nil
}

func (l *Logic) ListAuditEvents(ctx context.Context, serviceID uint, traceID, commandID string, limit int) ([]cicdv1.ReleaseTimelineEventResp, error) {
	rows, err := l.repo.ListAuditEvents(ctx, serviceID, strings.TrimSpace(traceID), strings.TrimSpace(commandID), limit)
	if err != nil {
		return nil, err
	}
	out := make([]cicdv1.ReleaseTimelineEventResp, 0, len(rows))
	for i := range rows {
		out = append(out, cicdv1.ReleaseTimelineEventResp{
			ID:           rows[i].ID,
			ServiceID:    rows[i].ServiceID,
			DeploymentID: rows[i].DeploymentID,
			ReleaseID:    rows[i].ReleaseID,
			EventType:    rows[i].EventType,
			ActorID:      rows[i].ActorID,
			Payload:      parseAnyJSON(rows[i].PayloadJSON),
			CreatedAt:    rows[i].CreatedAt,
		})
	}
	return out, nil
}

func (l *Logic) invalidateTimelineCache(ctx context.Context, serviceID uint) {
	if serviceID == 0 || l.svcCtx.Rdb == nil {
		return
	}
	_ = l.svcCtx.Rdb.Del(ctx, fmt.Sprintf("cicd:timeline:service:%d", serviceID)).Err()
}

func (l *Logic) writeAudit(ctx context.Context, serviceID, deploymentID, releaseID uint, eventType string, actor uint, payload any) error {
	rec := model.CICDAuditEvent{
		ServiceID:    serviceID,
		DeploymentID: deploymentID,
		ReleaseID:    releaseID,
		EventType:    eventType,
		ActorID:      actor,
		PayloadJSON:  mustJSON(payload),
	}
	if meta, ok := commandAuditContextFromContext(ctx); ok {
		rec.CommandID = meta.CommandID
		rec.Intent = meta.Intent
		rec.PlanHash = meta.PlanHash
		rec.TraceID = meta.TraceID
		rec.ApprovalContext = mustJSONOrEmpty(meta.ApprovalContext)
		rec.ExecutionSummary = strings.TrimSpace(meta.Summary)
	}
	_, err := l.repo.CreateAuditEvent(ctx, rec)
	return err
}

func toServiceCIConfigResp(row *model.CICDServiceCIConfig) *cicdv1.ServiceCIConfigResp {
	return &cicdv1.ServiceCIConfigResp{
		ID:             row.ID,
		ServiceID:      row.ServiceID,
		RepoURL:        row.RepoURL,
		Branch:         row.Branch,
		BuildSteps:     parseStringSliceJSON(row.BuildStepsJSON),
		ArtifactTarget: row.ArtifactTarget,
		TriggerMode:    row.TriggerMode,
		Status:         row.Status,
		UpdatedBy:      row.UpdatedBy,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func toCIRunResp(row *model.CICDServiceCIRun) *cicdv1.CIRunResp {
	return &cicdv1.CIRunResp{ID: row.ID, ServiceID: row.ServiceID, CIConfigID: row.CIConfigID, TriggerType: row.TriggerType, Status: row.Status, Reason: row.Reason, TriggeredBy: row.TriggeredBy, TriggeredAt: row.TriggeredAt, CreatedAt: row.CreatedAt}
}

func toCDConfigResp(row *model.CICDDeploymentCDConfig) *cicdv1.DeploymentCDConfigResp {
	return &cicdv1.DeploymentCDConfigResp{ID: row.ID, DeploymentID: row.DeploymentID, Env: row.Env, RuntimeType: row.RuntimeType, Strategy: row.Strategy, StrategyConfig: parseMapJSON(row.StrategyConfigJSON), ApprovalRequired: row.ApprovalRequired, UpdatedBy: row.UpdatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func toReleaseResp(row *model.CICDRelease) *cicdv1.ReleaseResp {
	return &cicdv1.ReleaseResp{ID: row.ID, ServiceID: row.ServiceID, DeploymentID: row.DeploymentID, Env: row.Env, RuntimeType: row.RuntimeType, Version: row.Version, Strategy: row.Strategy, Status: row.Status, TriggeredBy: row.TriggeredBy, ApprovedBy: row.ApprovedBy, ApprovalComment: row.ApprovalComment, RollbackFromReleaseID: row.RollbackFromReleaseID, Diagnostics: parseAnyJSON(row.DiagnosticsJSON), StartedAt: row.StartedAt, FinishedAt: row.FinishedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func toReleaseRespFromDeployment(row *model.DeploymentRelease, deploymentID uint, version string) *cicdv1.ReleaseResp {
	if row == nil {
		return &cicdv1.ReleaseResp{}
	}
	if strings.TrimSpace(version) == "" {
		version = fmt.Sprintf("rev-%d", row.RevisionID)
	}
	return &cicdv1.ReleaseResp{
		ID:               row.ID,
		UnifiedReleaseID: row.ID,
		ServiceID:        row.ServiceID,
		DeploymentID:     deploymentID,
		Env:              row.NamespaceOrProject,
		RuntimeType:      row.RuntimeType,
		Version:          version,
		Strategy:         row.Strategy,
		Status:           row.Status,
		TriggeredBy:      row.Operator,
		Diagnostics:      parseAnyJSON(row.DiagnosticsJSON),
		StartedAt:        nil,
		FinishedAt:       nil,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		TriggerSource:    row.TriggerSource,
		TriggerContext:   parseAnyJSON(row.TriggerContextJSON),
		CIRunID:          row.CIRunID,
	}
}

func normalizeTriggerMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "manual", "source-event", "both":
		return strings.TrimSpace(mode)
	default:
		return ""
	}
}

func normalizeTriggerType(triggerType string) string {
	switch strings.TrimSpace(triggerType) {
	case "manual", "source-event":
		return strings.TrimSpace(triggerType)
	default:
		return ""
	}
}

func allowTrigger(mode, triggerType string) bool {
	if mode == "both" {
		return true
	}
	return mode == triggerType
}

func normalizeStrategy(strategy string) string {
	s := strings.TrimSpace(strategy)
	switch s {
	case "rolling", "blue-green", "canary":
		return s
	default:
		return ""
	}
}

func normalizeRuntimeType(runtimeType string) string {
	switch strings.TrimSpace(runtimeType) {
	case "", "k8s", "compose":
		return strings.TrimSpace(runtimeType)
	default:
		return ""
	}
}

func (l *Logic) executeRuntimeRollback(ctx context.Context, runtimeType string) error {
	switch runtimeType {
	case "k8s", "compose":
		return nil
	default:
		return fmt.Errorf("unsupported runtime rollback executor: %s", runtimeType)
	}
}

func parseStringSliceJSON(raw string) []string {
	out := make([]string, 0)
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func parseMapJSON(raw string) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func parseAnyJSON(raw string) any {
	var out any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func defaultIfEmpty(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
