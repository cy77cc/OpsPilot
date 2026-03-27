// Package deployment 提供发布管理的业务逻辑实现。
//
// 本文件包含发布预览、执行、审批和回滚等核心业务逻辑。
package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	projectlogic "github.com/cy77cc/OpsPilot/internal/service/project/logic"
)

// 发布状态常量定义
const (
	releaseStatusPreviewed       = "previewed"        // 已预览
	releaseStatusPendingApproval = "pending_approval" // 待审批
	releaseStatusApproved        = "approved"         // 已批准
	releaseStatusRejected        = "rejected"         // 已拒绝
	releaseStatusApplying        = "applying"         // 应用中
	releaseStatusApplied         = "applied"          // 已应用
	releaseStatusFailed          = "failed"           // 失败
	releaseStatusRollback        = "rollback"         // 回滚中
	releaseStatusRolledBack      = "rolled_back"      // 已回滚 (兼容历史数据)
)

// previewTokenTTL 是预览令牌的有效期。
const previewTokenTTL = 30 * time.Minute

// releaseDiagnostic 是发布诊断信息结构。
type releaseDiagnostic struct {
	Runtime string `json:"runtime"` // 运行时类型
	Stage   string `json:"stage"`   // 阶段
	Code    string `json:"code"`    // 错误码
	Message string `json:"message"` // 错误消息
	Summary string `json:"summary"` // 摘要
}

// PreviewRelease 预览发布。
//
// 参数:
//   - ctx: 上下文
//   - req: 发布预览请求
//
// 返回: 发布预览响应，包含解析后的清单和预览令牌
func (l *Logic) PreviewRelease(ctx context.Context, req ReleasePreviewReq) (ReleasePreviewResp, error) {
	svc, target, manifest, err := l.resolveReleaseContext(ctx, req)
	if err != nil {
		return ReleasePreviewResp{}, err
	}
	env := strings.ToLower(strings.TrimSpace(defaultIfEmpty(req.Env, defaultIfEmpty(target.Env, svc.Env))))
	checks := []map[string]string{
		{"code": "target", "message": fmt.Sprintf("target=%s:%d", target.TargetType, target.ID), "level": "info"},
		{"code": "service", "message": fmt.Sprintf("service=%s", svc.Name), "level": "info"},
	}
	var warnings []map[string]string
	if target.TargetType == "compose" {
		if !strings.Contains(manifest, "services:") {
			warnings = append(warnings, map[string]string{"code": "compose_shape", "message": "manifest may not be valid docker compose schema", "level": "warning"})
		}
	}
	expiresAt := time.Now().Add(previewTokenTTL).UTC()
	previewToken, _ := issuePreviewToken(req, target.TargetType, env, manifest, expiresAt)
	return ReleasePreviewResp{
		ResolvedManifest: manifest,
		Checks:           checks,
		Warnings:         warnings,
		Runtime:          target.TargetType,
		PreviewToken:     previewToken,
		PreviewExpiresAt: &expiresAt,
	}, nil
}

// ApplyRelease 执行发布。
//
// 参数:
//   - ctx: 上下文
//   - uid: 用户 ID
//   - req: 发布请求
//
// 返回: 发布响应，生产环境需要审批流程
func (l *Logic) ApplyRelease(ctx context.Context, uid uint64, req ReleasePreviewReq) (ReleaseApplyResp, error) {
	svc, target, manifest, err := l.resolveReleaseContext(ctx, req)
	if err != nil {
		return ReleaseApplyResp{}, err
	}
	triggerSource := strings.TrimSpace(req.TriggerSource)
	if triggerSource == "" {
		triggerSource = "manual"
	}
	triggerContext := req.TriggerContext
	if triggerContext == nil {
		triggerContext = map[string]any{}
	}
	triggerContext["source"] = triggerSource
	if req.CIRunID > 0 {
		triggerContext["ci_run_id"] = req.CIRunID
	}
	env := strings.ToLower(strings.TrimSpace(defaultIfEmpty(req.Env, defaultIfEmpty(target.Env, svc.Env))))
	previewContextHash, previewTokenHash, previewExpiresAt, reasonCode, err := validatePreviewToken(req, target.TargetType, env, manifest)
	if err != nil {
		return ReleaseApplyResp{ReasonCode: reasonCode}, err
	}
	approvalRequired := env == "production"
	release := &model.DeploymentRelease{
		ServiceID:          svc.ID,
		TargetID:           target.ID,
		NamespaceOrProject: env,
		RuntimeType:        target.TargetType,
		Strategy:           defaultIfEmpty(req.Strategy, "rolling"),
		RevisionID:         svc.LastRevisionID,
		TriggerSource:      triggerSource,
		PreviewContextHash: previewContextHash,
		PreviewTokenHash:   previewTokenHash,
		PreviewExpiresAt:   previewExpiresAt,
		Status:             releaseStatusPreviewed,
		ManifestSnapshot:   manifest,
		RuntimeContextJSON: toJSON(map[string]any{
			"runtime":   target.TargetType,
			"target_id": target.ID,
			"env":       env,
			"service":   svc.Name,
		}),
		TriggerContextJSON: toJSON(triggerContext),
		ChecksJSON:         "[]",
		WarningsJSON:       "[]",
		DiagnosticsJSON:    "[]",
		VerificationJSON:   "{}",
		Operator:           uint(uid),
		CIRunID:            req.CIRunID,
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(release).Error; err != nil {
		return ReleaseApplyResp{}, err
	}
	l.writeReleaseAudit(ctx, release.ID, uint(uid), "release.previewed", map[string]any{"runtime": target.TargetType, "env": env})

	if approvalRequired {
		ticket := fmt.Sprintf("dep-appr-%d", time.Now().UnixNano())
		approval := model.DeploymentReleaseApproval{
			ReleaseID:   release.ID,
			Ticket:      ticket,
			Decision:    "pending",
			RequestedBy: uint(uid),
		}
		if err := l.svcCtx.DB.WithContext(ctx).Create(&approval).Error; err != nil {
			return ReleaseApplyResp{}, err
		}
		release.Status = releaseStatusPendingApproval
		_ = l.svcCtx.DB.WithContext(ctx).Save(release).Error
		l.writeReleaseAudit(ctx, release.ID, uint(uid), "release.pending_approval", map[string]any{"ticket": ticket})
		return ReleaseApplyResp{
			ReleaseID:        release.ID,
			UnifiedReleaseID: release.ID,
			Status:           release.Status,
			RuntimeType:      release.RuntimeType,
			TriggerSource:    release.TriggerSource,
			TriggerContext:   triggerContext,
			CIRunID:          release.CIRunID,
			ApprovalRequired: true,
			ApprovalTicket:   ticket,
			LifecycleState:   l.releaseLifecycleState(release.Status),
		}, nil
	}

	release.Status = releaseStatusApproved
	_ = l.svcCtx.DB.WithContext(ctx).Save(release).Error
	l.writeReleaseAudit(ctx, release.ID, uint(uid), "release.approved", map[string]any{"auto": true})
	if execErr := l.executeRelease(ctx, release, target); execErr != nil {
		return ReleaseApplyResp{
			ReleaseID:        release.ID,
			UnifiedReleaseID: release.ID,
			Status:           release.Status,
			RuntimeType:      release.RuntimeType,
			TriggerSource:    release.TriggerSource,
			TriggerContext:   triggerContext,
			CIRunID:          release.CIRunID,
			LifecycleState:   l.releaseLifecycleState(release.Status),
		}, execErr
	}
	return ReleaseApplyResp{
		ReleaseID:        release.ID,
		UnifiedReleaseID: release.ID,
		Status:           release.Status,
		RuntimeType:      release.RuntimeType,
		TriggerSource:    release.TriggerSource,
		TriggerContext:   triggerContext,
		CIRunID:          release.CIRunID,
		LifecycleState:   l.releaseLifecycleState(release.Status),
	}, nil
}

// RollbackRelease 回滚发布。
//
// 参数:
//   - ctx: 上下文
//   - id: 当前发布 ID
//   - uid: 用户 ID
//
// 返回: 回滚发布响应
func (l *Logic) RollbackRelease(ctx context.Context, id uint, uid uint64) (ReleaseApplyResp, error) {
	var current model.DeploymentRelease
	if err := l.svcCtx.DB.WithContext(ctx).First(&current, id).Error; err != nil {
		return ReleaseApplyResp{}, err
	}
	var prev model.DeploymentRelease
	if err := l.svcCtx.DB.WithContext(ctx).
		Where("service_id = ? AND target_id = ? AND id < ?", current.ServiceID, current.TargetID, current.ID).
		Order("id DESC").First(&prev).Error; err != nil {
		return ReleaseApplyResp{}, fmt.Errorf("no previous release to rollback")
	}
	rollback := &model.DeploymentRelease{
		ServiceID:          current.ServiceID,
		TargetID:           current.TargetID,
		NamespaceOrProject: current.NamespaceOrProject,
		RuntimeType:        current.RuntimeType,
		Strategy:           "rollback",
		TriggerSource:      current.TriggerSource,
		RevisionID:         prev.RevisionID,
		SourceReleaseID:    current.ID,
		TargetRevision:     fmt.Sprintf("%d", prev.RevisionID),
		Status:             releaseStatusRollback,
		ManifestSnapshot:   prev.ManifestSnapshot,
		RuntimeContextJSON: toJSON(map[string]any{"runtime": current.RuntimeType, "rollback_from": current.ID}),
		TriggerContextJSON: toJSON(map[string]any{"rollback_from_release_id": current.ID}),
		ChecksJSON:         "[]",
		WarningsJSON:       "[]",
		DiagnosticsJSON:    "[]",
		VerificationJSON:   "{}",
		Operator:           uint(uid),
		CIRunID:            current.CIRunID,
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(rollback).Error; err != nil {
		return ReleaseApplyResp{}, err
	}
	l.writeReleaseAudit(ctx, rollback.ID, uint(uid), "release.rollback_started", map[string]any{"from_release_id": current.ID})
	switch current.RuntimeType {
	case "k8s":
		var target model.DeploymentTarget
		if err := l.svcCtx.DB.WithContext(ctx).First(&target, current.TargetID).Error; err != nil {
			rollback.Status = releaseStatusFailed
			rollback.DiagnosticsJSON = toJSON([]releaseDiagnostic{{Runtime: "k8s", Stage: "rollback", Code: "target_not_found", Message: err.Error(), Summary: "rollback target missing"}})
			_ = l.svcCtx.DB.WithContext(ctx).Save(rollback).Error
			return ReleaseApplyResp{ReleaseID: rollback.ID, Status: rollback.Status, RuntimeType: rollback.RuntimeType}, err
		}
		var cluster model.Cluster
		if err := l.svcCtx.DB.WithContext(ctx).First(&cluster, target.ClusterID).Error; err != nil {
			rollback.Status = releaseStatusFailed
			rollback.DiagnosticsJSON = toJSON([]releaseDiagnostic{{Runtime: "k8s", Stage: "rollback", Code: "cluster_not_found", Message: err.Error(), Summary: "rollback cluster missing"}})
			_ = l.svcCtx.DB.WithContext(ctx).Save(rollback).Error
			return ReleaseApplyResp{ReleaseID: rollback.ID, Status: rollback.Status, RuntimeType: rollback.RuntimeType}, err
		}
		if err := projectlogic.DeployToCluster(ctx, &cluster, prev.ManifestSnapshot); err != nil {
			rollback.Status = releaseStatusFailed
			rollback.DiagnosticsJSON = toJSON([]releaseDiagnostic{{Runtime: "k8s", Stage: "rollback", Code: "rollback_apply_failed", Message: err.Error(), Summary: "k8s rollback apply failed"}})
			_ = l.svcCtx.DB.WithContext(ctx).Save(rollback).Error
			return ReleaseApplyResp{ReleaseID: rollback.ID, Status: rollback.Status, RuntimeType: rollback.RuntimeType}, err
		}
	case "compose":
		var target model.DeploymentTarget
		if err := l.svcCtx.DB.WithContext(ctx).First(&target, current.TargetID).Error; err != nil {
			rollback.Status = releaseStatusFailed
			rollback.DiagnosticsJSON = toJSON([]releaseDiagnostic{{Runtime: "compose", Stage: "rollback", Code: "target_not_found", Message: err.Error(), Summary: "rollback target missing"}})
			_ = l.svcCtx.DB.WithContext(ctx).Save(rollback).Error
			return ReleaseApplyResp{ReleaseID: rollback.ID, Status: rollback.Status, RuntimeType: rollback.RuntimeType}, err
		}
		out, err := l.applyComposeRelease(ctx, &target, rollback.ID, prev.ManifestSnapshot)
		if err != nil {
			rollback.Status = releaseStatusFailed
			rollback.DiagnosticsJSON = toJSON([]releaseDiagnostic{{Runtime: "compose", Stage: "rollback", Code: "rollback_apply_failed", Message: err.Error(), Summary: truncateText(out, 800)}})
			_ = l.svcCtx.DB.WithContext(ctx).Save(rollback).Error
			return ReleaseApplyResp{ReleaseID: rollback.ID, Status: rollback.Status, RuntimeType: rollback.RuntimeType}, err
		}
		rollback.ChecksJSON = toJSON([]map[string]string{{"code": "compose_rollback_ps", "message": truncateText(out, 1200), "level": "info"}})
	default:
		rollback.Status = releaseStatusRejected
		rollback.DiagnosticsJSON = toJSON([]releaseDiagnostic{{Runtime: current.RuntimeType, Stage: "rollback", Code: "runtime_not_supported", Message: "unsupported runtime", Summary: "rollback rejected"}})
		_ = l.svcCtx.DB.WithContext(ctx).Save(rollback).Error
		return ReleaseApplyResp{ReleaseID: rollback.ID, Status: rollback.Status, RuntimeType: rollback.RuntimeType}, fmt.Errorf("unsupported runtime: %s", current.RuntimeType)
	}
	rollback.Status = releaseStatusRollback
	rollback.VerificationJSON = toJSON(map[string]any{"runtime": current.RuntimeType, "rollback_succeeded": true})
	_ = l.svcCtx.DB.WithContext(ctx).Save(rollback).Error
	l.writeReleaseAudit(ctx, rollback.ID, uint(uid), "release.rollback_completed", map[string]any{"from_release_id": current.ID})
	return ReleaseApplyResp{
		ReleaseID:        rollback.ID,
		UnifiedReleaseID: rollback.ID,
		Status:           rollback.Status,
		RuntimeType:      rollback.RuntimeType,
		TriggerSource:    rollback.TriggerSource,
		TriggerContext:   map[string]any{"from_release_id": current.ID},
		CIRunID:          rollback.CIRunID,
		LifecycleState:   l.releaseLifecycleState(rollback.Status),
	}, nil
}

// ApproveRelease 审批通过发布。
//
// 参数:
//   - ctx: 上下文
//   - id: 发布 ID
//   - uid: 用户 ID
//   - comment: 审批意见
//
// 返回: 发布响应
func (l *Logic) ApproveRelease(ctx context.Context, id uint, uid uint64, comment string) (ReleaseApplyResp, error) {
	var release model.DeploymentRelease
	if err := l.svcCtx.DB.WithContext(ctx).First(&release, id).Error; err != nil {
		return ReleaseApplyResp{}, err
	}
	if release.Status != releaseStatusPendingApproval {
		return ReleaseApplyResp{}, fmt.Errorf("release state %s cannot be approved", release.Status)
	}
	var approval model.DeploymentReleaseApproval
	if err := l.svcCtx.DB.WithContext(ctx).
		Where("release_id = ? AND decision = ?", release.ID, "pending").
		Order("id DESC").First(&approval).Error; err != nil {
		return ReleaseApplyResp{}, fmt.Errorf("approval record not found")
	}
	approval.Decision = "approved"
	approval.Comment = strings.TrimSpace(comment)
	approval.ApproverID = uint(uid)
	if err := l.svcCtx.DB.WithContext(ctx).Save(&approval).Error; err != nil {
		return ReleaseApplyResp{}, err
	}
	release.Status = releaseStatusApproved
	if err := l.svcCtx.DB.WithContext(ctx).Save(&release).Error; err != nil {
		return ReleaseApplyResp{}, err
	}
	l.writeReleaseAudit(ctx, release.ID, uint(uid), "release.approved", map[string]any{"ticket": approval.Ticket, "comment": approval.Comment})
	var target model.DeploymentTarget
	if err := l.svcCtx.DB.WithContext(ctx).First(&target, release.TargetID).Error; err != nil {
		return ReleaseApplyResp{}, err
	}
	if execErr := l.executeRelease(ctx, &release, &target); execErr != nil {
		return ReleaseApplyResp{
			ReleaseID:        release.ID,
			UnifiedReleaseID: release.ID,
			Status:           release.Status,
			RuntimeType:      release.RuntimeType,
			TriggerSource:    release.TriggerSource,
			TriggerContext:   map[string]any{"approval_ticket": approval.Ticket},
			CIRunID:          release.CIRunID,
			LifecycleState:   l.releaseLifecycleState(release.Status),
		}, execErr
	}
	return ReleaseApplyResp{
		ReleaseID:        release.ID,
		UnifiedReleaseID: release.ID,
		Status:           release.Status,
		RuntimeType:      release.RuntimeType,
		TriggerSource:    release.TriggerSource,
		TriggerContext:   map[string]any{"approval_ticket": approval.Ticket},
		CIRunID:          release.CIRunID,
		LifecycleState:   l.releaseLifecycleState(release.Status),
	}, nil
}

// RejectRelease 拒绝发布。
//
// 参数:
//   - ctx: 上下文
//   - id: 发布 ID
//   - uid: 用户 ID
//   - comment: 拒绝原因
//
// 返回: 发布响应
func (l *Logic) RejectRelease(ctx context.Context, id uint, uid uint64, comment string) (ReleaseApplyResp, error) {
	var release model.DeploymentRelease
	if err := l.svcCtx.DB.WithContext(ctx).First(&release, id).Error; err != nil {
		return ReleaseApplyResp{}, err
	}
	if release.Status != releaseStatusPendingApproval {
		return ReleaseApplyResp{}, fmt.Errorf("release state %s cannot be rejected", release.Status)
	}
	var approval model.DeploymentReleaseApproval
	if err := l.svcCtx.DB.WithContext(ctx).
		Where("release_id = ? AND decision = ?", release.ID, "pending").
		Order("id DESC").First(&approval).Error; err != nil {
		return ReleaseApplyResp{}, fmt.Errorf("approval record not found")
	}
	approval.Decision = "rejected"
	approval.Comment = strings.TrimSpace(comment)
	approval.ApproverID = uint(uid)
	if err := l.svcCtx.DB.WithContext(ctx).Save(&approval).Error; err != nil {
		return ReleaseApplyResp{}, err
	}
	release.Status = releaseStatusRejected
	if err := l.svcCtx.DB.WithContext(ctx).Save(&release).Error; err != nil {
		return ReleaseApplyResp{}, err
	}
	l.writeReleaseAudit(ctx, release.ID, uint(uid), "release.rejected", map[string]any{"ticket": approval.Ticket, "comment": approval.Comment})
	return ReleaseApplyResp{
		ReleaseID:        release.ID,
		UnifiedReleaseID: release.ID,
		Status:           release.Status,
		RuntimeType:      release.RuntimeType,
		TriggerSource:    release.TriggerSource,
		TriggerContext:   map[string]any{"approval_ticket": approval.Ticket},
		CIRunID:          release.CIRunID,
		LifecycleState:   l.releaseLifecycleState(release.Status),
	}, nil
}

// ListReleases 获取发布列表。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID (可选筛选)
//   - targetID: 目标 ID (可选筛选)
//   - runtimeType: 运行时类型 (可选筛选)
//
// 返回: 发布记录列表
func (l *Logic) ListReleases(ctx context.Context, serviceID, targetID uint, runtimeType string) ([]model.DeploymentRelease, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentRelease{})
	if serviceID > 0 {
		q = q.Where("service_id = ?", serviceID)
	}
	if targetID > 0 {
		q = q.Where("target_id = ?", targetID)
	}
	if runtime := strings.TrimSpace(runtimeType); runtime != "" {
		q = q.Where("runtime_type = ?", runtime)
	}
	var rows []model.DeploymentRelease
	if err := q.Order("id DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetRelease 获取发布详情。
//
// 参数:
//   - ctx: 上下文
//   - id: 发布 ID
//
// 返回: 发布记录
func (l *Logic) GetRelease(ctx context.Context, id uint) (*model.DeploymentRelease, error) {
	var row model.DeploymentRelease
	if err := l.svcCtx.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ListReleaseTimeline 获取发布时间线事件列表。
//
// 参数:
//   - ctx: 上下文
//   - releaseID: 发布 ID
//
// 返回: 时间线事件列表
func (l *Logic) ListReleaseTimeline(ctx context.Context, releaseID uint) ([]ReleaseTimelineEventResp, error) {
	var rows []model.DeploymentReleaseAudit
	if err := l.svcCtx.DB.WithContext(ctx).
		Where("release_id = ?", releaseID).
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ReleaseTimelineEventResp, 0, len(rows))
	for i := range rows {
		var detail any
		if strings.TrimSpace(rows[i].DetailJSON) != "" {
			_ = json.Unmarshal([]byte(rows[i].DetailJSON), &detail)
		}
		out = append(out, ReleaseTimelineEventResp{
			ID:            rows[i].ID,
			ReleaseID:     rows[i].ReleaseID,
			CorrelationID: rows[i].CorrelationID,
			TraceID:       rows[i].TraceID,
			Action:        rows[i].Action,
			Actor:         rows[i].Actor,
			Detail:        detail,
			CreatedAt:     rows[i].CreatedAt,
		})
	}
	return out, nil
}

// resolveReleaseContext 解析发布上下文，获取服务、目标和清单。
//
// 参数:
//   - ctx: 上下文
//   - req: 发布预览请求
//
// 返回: 服务对象、目标对象、解析后的清单
func (l *Logic) resolveReleaseContext(ctx context.Context, req ReleasePreviewReq) (*model.Service, *model.DeploymentTarget, string, error) {
	var svc model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&svc, req.ServiceID).Error; err != nil {
		return nil, nil, "", err
	}
	var target model.DeploymentTarget
	if err := l.svcCtx.DB.WithContext(ctx).First(&target, req.TargetID).Error; err != nil {
		return nil, nil, "", err
	}
	if target.TargetType != "k8s" && target.TargetType != "compose" {
		return nil, nil, "", fmt.Errorf("unsupported runtime target")
	}
	if target.TargetType == "k8s" && target.ClusterID == 0 && target.CredentialID == 0 {
		return nil, nil, "", fmt.Errorf("k8s target missing cluster binding or credential")
	}
	if target.TargetType == "compose" {
		var cnt int64
		if err := l.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentTargetNode{}).
			Where("target_id = ? AND status = ?", target.ID, "active").Count(&cnt).Error; err != nil {
			return nil, nil, "", err
		}
		if cnt == 0 {
			return nil, nil, "", fmt.Errorf("compose target has no active host node")
		}
	}
	if rs := strings.TrimSpace(target.ReadinessStatus); rs != "" && rs != "ready" && rs != "unknown" {
		return nil, nil, "", fmt.Errorf("target is not bootstrap ready: %s", rs)
	}
	manifest := strings.TrimSpace(defaultIfEmpty(svc.CustomYAML, svc.YamlContent))
	if manifest == "" {
		return nil, nil, "", fmt.Errorf("empty service manifest")
	}
	for k, v := range req.Variables {
		manifest = strings.ReplaceAll(manifest, "{{"+k+"}}", v)
	}
	if strings.Contains(manifest, "{{") && strings.Contains(manifest, "}}") {
		return nil, nil, "", fmt.Errorf("manifest contains unresolved template variables")
	}
	return &svc, &target, manifest, nil
}

// executeRelease 执行发布，将清单应用到目标。
//
// 参数:
//   - ctx: 上下文
//   - release: 发布记录
//   - target: 部署目标
//
// 返回: 执行错误
func (l *Logic) executeRelease(ctx context.Context, release *model.DeploymentRelease, target *model.DeploymentTarget) error {
	release.Status = releaseStatusApplying
	_ = l.svcCtx.DB.WithContext(ctx).Save(release).Error
	l.writeReleaseAudit(ctx, release.ID, release.Operator, "release.applying", map[string]any{"runtime": target.TargetType})
	switch target.TargetType {
	case "k8s":
		var cluster model.Cluster
		if err := l.svcCtx.DB.WithContext(ctx).First(&cluster, target.ClusterID).Error; err != nil {
			release.Status = releaseStatusFailed
			release.DiagnosticsJSON = toJSON([]releaseDiagnostic{{
				Runtime: "k8s", Stage: "validate", Code: "cluster_not_found", Message: err.Error(), Summary: "cluster binding not found",
			}})
			_ = l.svcCtx.DB.WithContext(ctx).Save(release).Error
			l.writeReleaseAudit(ctx, release.ID, release.Operator, "release.failed", map[string]any{"reason": "cluster_not_found"})
			return err
		}
		if err := projectlogic.DeployToCluster(ctx, &cluster, release.ManifestSnapshot); err != nil {
			release.Status = releaseStatusFailed
			release.DiagnosticsJSON = toJSON([]releaseDiagnostic{{
				Runtime: "k8s", Stage: "execute", Code: "deploy_failed", Message: err.Error(), Summary: "k8s runtime apply failed",
			}})
			_ = l.svcCtx.DB.WithContext(ctx).Save(release).Error
			l.writeReleaseAudit(ctx, release.ID, release.Operator, "release.failed", map[string]any{"reason": "deploy_failed"})
			return err
		}
		release.VerificationJSON = toJSON(map[string]any{"runtime": "k8s", "checks": []string{"apply_succeeded"}, "passed": true})
		release.Status = releaseStatusApplied
	default:
		out, execErr := l.applyComposeRelease(ctx, target, release.ID, release.ManifestSnapshot)
		if execErr != nil {
			release.Status = releaseStatusFailed
			release.WarningsJSON = toJSON([]map[string]string{{"code": "compose_apply_failed", "message": truncateText(out, 1200), "level": "warning"}})
			release.DiagnosticsJSON = toJSON([]releaseDiagnostic{{
				Runtime: "compose", Stage: "execute", Code: "compose_apply_failed", Message: truncateText(execErr.Error(), 500), Summary: truncateText(out, 800),
			}})
			_ = l.svcCtx.DB.WithContext(ctx).Save(release).Error
			l.writeReleaseAudit(ctx, release.ID, release.Operator, "release.failed", map[string]any{"reason": "compose_apply_failed"})
			return execErr
		}
		release.Status = releaseStatusApplied
		release.VerificationJSON = toJSON(map[string]any{"runtime": "compose", "checks": []string{"docker_compose_ps"}, "passed": true})
		release.ChecksJSON = toJSON([]map[string]string{{"code": "compose_ps", "message": truncateText(out, 1200), "level": "info"}})
	}
	_ = l.svcCtx.DB.WithContext(ctx).Save(release).Error
	l.writeReleaseAudit(ctx, release.ID, release.Operator, "release.applied", map[string]any{"runtime": target.TargetType})
	return nil
}

// writeReleaseAudit 写入发布审计日志。
//
// 参数:
//   - ctx: 上下文
//   - releaseID: 发布 ID
//   - actor: 操作人 ID
//   - action: 操作类型
//   - detail: 详情
func (l *Logic) writeReleaseAudit(ctx context.Context, releaseID, actor uint, action string, detail any) {
	if releaseID == 0 || strings.TrimSpace(action) == "" {
		return
	}
	now := time.Now().UnixNano()
	_ = l.svcCtx.DB.WithContext(ctx).Create(&model.DeploymentReleaseAudit{
		ReleaseID:     releaseID,
		CorrelationID: fmt.Sprintf("release-%d", releaseID),
		TraceID:       fmt.Sprintf("trace-%d-%d", releaseID, now),
		Action:        action,
		Actor:         actor,
		DetailJSON:    toJSON(detail),
	}).Error
}

// releaseLifecycleState 将发布状态转换为生命周期状态。
//
// 参数:
//   - status: 发布状态
//
// 返回: 生命周期状态
func (l *Logic) releaseLifecycleState(status string) string {
	switch strings.TrimSpace(status) {
	case releaseStatusPreviewed:
		return "preview"
	case releaseStatusPendingApproval:
		return "pending_approval"
	case releaseStatusApplying:
		return "applying"
	case releaseStatusApproved:
		return "approved"
	case releaseStatusApplied:
		return "applied"
	case releaseStatusFailed:
		return "failed"
	case releaseStatusRejected:
		return "rejected"
	case releaseStatusRollback, releaseStatusRolledBack:
		return "rollback"
	default:
		return status
	}
}
