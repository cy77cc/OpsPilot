package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	sshclient "github.com/cy77cc/k8s-manage/internal/client/ssh"
	"github.com/cy77cc/k8s-manage/internal/config"
	"github.com/cy77cc/k8s-manage/internal/model"
	projectlogic "github.com/cy77cc/k8s-manage/internal/service/project/logic"
	"github.com/cy77cc/k8s-manage/internal/utils"
	"gorm.io/gorm"
)

func (l *Logic) DeployPreview(ctx context.Context, id uint, req DeployReq) (DeployPreviewResp, error) {
	var service model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&service, id).Error; err != nil {
		return DeployPreviewResp{}, err
	}
	target, err := l.resolveDeployTarget(ctx, id, req)
	if err != nil {
		return DeployPreviewResp{}, err
	}
	resolved, unresolved, err := l.resolveServiceTemplate(ctx, &service, req.Env, req.Variables)
	if err != nil {
		return DeployPreviewResp{}, err
	}
	checks := []RenderDiagnostic{
		{Level: "info", Code: "cluster_selected", Message: fmt.Sprintf("cluster=%d namespace=%s", target.ClusterID, target.Namespace)},
	}
	warnings := make([]RenderDiagnostic, 0)
	if len(unresolved) > 0 {
		warnings = append(warnings, RenderDiagnostic{Level: "warning", Code: "unresolved_vars", Message: strings.Join(unresolved, ",")})
	}
	return DeployPreviewResp{
		ResolvedYAML: resolved,
		Checks:       checks,
		Warnings:     warnings,
		Target:       target,
	}, nil
}

func (l *Logic) Deploy(ctx context.Context, id uint, operator uint64, req DeployReq) (uint, error) {
	var service model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&service, id).Error; err != nil {
		return 0, err
	}
	targetResp, err := l.resolveDeployTarget(ctx, id, req)
	if err != nil {
		return 0, err
	}
	target := defaultIfEmpty(req.DeployTarget, targetResp.DeployTarget)
	resolved, unresolved, err := l.resolveServiceTemplate(ctx, &service, req.Env, req.Variables)
	if err != nil {
		return 0, err
	}
	if len(unresolved) > 0 {
		return 0, fmt.Errorf("unresolved template vars: %s", strings.Join(unresolved, ","))
	}

	rec := &model.ServiceReleaseRecord{
		ServiceID:         id,
		RevisionID:        service.LastRevisionID,
		ClusterID:         targetResp.ClusterID,
		Namespace:         targetResp.Namespace,
		Env:               defaultIfEmpty(req.Env, service.Env),
		DeployTarget:      target,
		Status:            "running",
		RenderedYAML:      resolved,
		VariablesSnapshot: mustJSON(req.Variables),
		Operator:          uint(operator),
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(rec).Error; err != nil {
		return 0, err
	}

	switch target {
	case "compose":
		out, execErr := l.applyComposeByTarget(ctx, targetResp.ClusterID, rec.ID, resolved)
		if execErr != nil {
			rec.Status = "failed"
			rec.Error = truncateStr(execErr.Error(), 500)
			_ = l.svcCtx.DB.WithContext(ctx).Save(rec).Error
			return rec.ID, fmt.Errorf("compose apply failed: %s; %s", execErr.Error(), truncateStr(out, 300))
		}
		rec.Status = "succeeded"
		_ = l.svcCtx.DB.WithContext(ctx).Save(rec).Error
		return rec.ID, nil
	case "helm":
		if err := l.deployHelm(ctx, id); err != nil {
			rec.Status = "failed"
			rec.Error = err.Error()
			_ = l.svcCtx.DB.WithContext(ctx).Save(rec).Error
			return rec.ID, err
		}
		rec.Status = "succeeded"
		_ = l.svcCtx.DB.WithContext(ctx).Save(rec).Error
		return rec.ID, nil
	default:
		var cluster model.Cluster
		if err := l.svcCtx.DB.WithContext(ctx).First(&cluster, targetResp.ClusterID).Error; err != nil {
			rec.Status = "failed"
			rec.Error = err.Error()
			_ = l.svcCtx.DB.WithContext(ctx).Save(rec).Error
			return rec.ID, err
		}
		if strings.TrimSpace(resolved) == "" {
			return rec.ID, fmt.Errorf("empty rendered yaml")
		}
		if err := projectlogic.DeployToCluster(ctx, &cluster, resolved); err != nil {
			rec.Status = "failed"
			rec.Error = err.Error()
			_ = l.svcCtx.DB.WithContext(ctx).Save(rec).Error
			return rec.ID, err
		}
		rec.Status = "succeeded"
		_ = l.svcCtx.DB.WithContext(ctx).Save(rec).Error
		return rec.ID, nil
	}
}

func (l *Logic) HelmImport(ctx context.Context, uid uint64, req HelmImportReq) (*model.ServiceHelmRelease, error) {
	rec := &model.ServiceHelmRelease{
		ServiceID:    req.ServiceID,
		ChartName:    req.ChartName,
		ChartVersion: req.ChartVersion,
		ChartRef:     req.ChartRef,
		ValuesYAML:   req.ValuesYAML,
		RenderedYAML: req.RenderedYAML,
		Status:       "imported",
		CreatedBy:    uint(uid),
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(rec).Error; err != nil {
		return nil, err
	}
	return rec, nil
}

func (l *Logic) HelmRender(ctx context.Context, req HelmRenderReq) (string, []RenderDiagnostic, error) {
	diags := make([]RenderDiagnostic, 0)
	if strings.TrimSpace(req.RenderedYAML) != "" {
		return req.RenderedYAML, diags, nil
	}
	chartRef := strings.TrimSpace(req.ChartRef)
	if chartRef == "" && req.ReleaseID > 0 {
		var release model.ServiceHelmRelease
		if err := l.svcCtx.DB.WithContext(ctx).First(&release, req.ReleaseID).Error; err != nil {
			return "", nil, err
		}
		chartRef = release.ChartRef
		if req.ValuesYAML == "" {
			req.ValuesYAML = release.ValuesYAML
		}
	}
	if chartRef == "" {
		return "", []RenderDiagnostic{{Level: "error", Code: "helm_chart_ref_required", Message: "chart_ref is required"}}, errors.New("chart_ref required")
	}
	_, err := exec.LookPath("helm")
	if err != nil {
		return "", []RenderDiagnostic{{Level: "error", Code: "helm_binary_missing", Message: "helm binary not found in PATH"}}, err
	}
	valuesFile, err := os.CreateTemp("", "helm-values-*.yaml")
	if err != nil {
		return "", nil, err
	}
	defer os.Remove(valuesFile.Name())
	if _, err := valuesFile.WriteString(req.ValuesYAML); err != nil {
		_ = valuesFile.Close()
		return "", nil, err
	}
	_ = valuesFile.Close()

	ctx2, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx2, "helm", "template", defaultIfEmpty(req.ChartName, "release"), chartRef, "-f", valuesFile.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", []RenderDiagnostic{{Level: "error", Code: "helm_template_failed", Message: string(out)}}, err
	}
	return string(out), diags, nil
}

func (l *Logic) deployHelm(ctx context.Context, serviceID uint) error {
	var release model.ServiceHelmRelease
	if err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ?", serviceID).Order("id DESC").First(&release).Error; err != nil {
		return err
	}
	if strings.TrimSpace(release.RenderedYAML) == "" {
		release.RenderedYAML = "# helm release imported but not rendered\n"
	}
	release.Status = "deployed"
	return l.svcCtx.DB.WithContext(ctx).Save(&release).Error
}

func (l *Logic) applyComposeByTarget(ctx context.Context, targetID uint, releaseID uint, manifest string) (string, error) {
	if targetID == 0 {
		return "", fmt.Errorf("compose target id is required")
	}
	var links []model.DeploymentTargetNode
	if err := l.svcCtx.DB.WithContext(ctx).
		Where("target_id = ? AND status = ?", targetID, "active").
		Order("CASE WHEN role = 'manager' THEN 0 ELSE 1 END, id ASC").
		Find(&links).Error; err != nil {
		return "", err
	}
	if len(links) == 0 {
		return "", fmt.Errorf("compose target has no active nodes")
	}
	var node model.Node
	if err := l.svcCtx.DB.WithContext(ctx).First(&node, links[0].HostID).Error; err != nil {
		return "", err
	}
	privateKey, passphrase, err := l.loadNodeSSHPrivateKey(ctx, &node)
	if err != nil {
		return "", err
	}
	password := strings.TrimSpace(node.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	workDir := fmt.Sprintf("/tmp/opspilot/service-releases/%d", releaseID)
	composeFile := fmt.Sprintf("%s/docker-compose.yaml", workDir)
	encoded := base64.StdEncoding.EncodeToString([]byte(manifest))
	cmd := fmt.Sprintf("mkdir -p %s && echo '%s' | base64 -d > %s && docker compose -f %s pull && docker compose -f %s up -d && docker compose -f %s ps", workDir, encoded, composeFile, composeFile, composeFile, composeFile)
	return sshclient.RunCommand(cli, cmd)
}

func toDeployTargetResp(t *model.ServiceDeployTarget) DeployTargetResp {
	resp := DeployTargetResp{
		ID:           t.ID,
		ServiceID:    t.ServiceID,
		ClusterID:    t.ClusterID,
		Namespace:    t.Namespace,
		DeployTarget: t.DeployTarget,
		IsDefault:    t.IsDefault,
		UpdatedAt:    t.UpdatedAt,
	}
	if strings.TrimSpace(t.PolicyJSON) != "" {
		_ = json.Unmarshal([]byte(t.PolicyJSON), &resp.Policy)
	}
	return resp
}

func (l *Logic) resolveDeployTarget(ctx context.Context, serviceID uint, req DeployReq) (DeployTargetResp, error) {
	if req.ClusterID > 0 {
		if strings.EqualFold(defaultIfEmpty(req.DeployTarget, "k8s"), "compose") {
			var target model.DeploymentTarget
			if err := l.svcCtx.DB.WithContext(ctx).Where("id = ? AND target_type = ?", req.ClusterID, "compose").First(&target).Error; err != nil {
				return DeployTargetResp{}, fmt.Errorf("compose deployment target not found: %w", err)
			}
		}
		return DeployTargetResp{
			ServiceID:    serviceID,
			ClusterID:    req.ClusterID,
			Namespace:    defaultIfEmpty(req.Namespace, "default"),
			DeployTarget: defaultIfEmpty(req.DeployTarget, "k8s"),
			IsDefault:    false,
		}, nil
	}
	var row model.ServiceDeployTarget
	if err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ? AND is_default = 1", serviceID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeployTargetResp{}, fmt.Errorf("deploy target not configured")
		}
		return DeployTargetResp{}, err
	}
	resp := toDeployTargetResp(&row)
	if strings.TrimSpace(req.Namespace) != "" {
		resp.Namespace = req.Namespace
	}
	if strings.TrimSpace(req.DeployTarget) != "" {
		resp.DeployTarget = req.DeployTarget
	}
	if strings.EqualFold(resp.DeployTarget, "compose") {
		var target model.DeploymentTarget
		if err := l.svcCtx.DB.WithContext(ctx).Where("id = ? AND target_type = ?", resp.ClusterID, "compose").First(&target).Error; err != nil {
			return DeployTargetResp{}, fmt.Errorf("compose deployment target not found: %w", err)
		}
	}
	return resp, nil
}

func (l *Logic) loadNodeSSHPrivateKey(ctx context.Context, node *model.Node) (string, string, error) {
	if node == nil || node.SSHKeyID == nil {
		return "", "", nil
	}
	var key model.SSHKey
	if err := l.svcCtx.DB.WithContext(ctx).
		Select("id", "private_key", "passphrase", "encrypted").
		Where("id = ?", uint64(*node.SSHKeyID)).
		First(&key).Error; err != nil {
		return "", "", err
	}
	passphrase := strings.TrimSpace(key.Passphrase)
	if !key.Encrypted {
		return strings.TrimSpace(key.PrivateKey), passphrase, nil
	}
	if strings.TrimSpace(config.CFG.Security.EncryptionKey) == "" {
		return "", "", fmt.Errorf("security.encryption_key is required")
	}
	privateKey, err := utils.DecryptText(strings.TrimSpace(key.PrivateKey), config.CFG.Security.EncryptionKey)
	if err != nil {
		return "", "", err
	}
	return privateKey, passphrase, nil
}

func (l *Logic) resolveServiceTemplate(ctx context.Context, service *model.Service, env string, reqValues map[string]string) (string, []string, error) {
	content := defaultIfEmpty(service.CustomYAML, service.YamlContent)
	if strings.TrimSpace(content) == "" {
		return "", nil, fmt.Errorf("empty service template")
	}
	envValues := map[string]string{}
	var set model.ServiceVariableSet
	err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ? AND env = ?", service.ID, defaultIfEmpty(env, service.Env)).First(&set).Error
	if err == nil && strings.TrimSpace(set.ValuesJSON) != "" {
		_ = json.Unmarshal([]byte(set.ValuesJSON), &envValues)
	}
	resolved, unresolved := resolveTemplateVars(content, normalizeStringMap(reqValues), normalizeStringMap(envValues))
	return resolved, unresolved, nil
}

func (l *Logic) UpsertDeployTarget(ctx context.Context, serviceID uint, uid uint64, req DeployTargetUpsertReq) (DeployTargetResp, error) {
	if req.ClusterID == 0 {
		return DeployTargetResp{}, fmt.Errorf("cluster_id is required")
	}
	ns := defaultIfEmpty(req.Namespace, "default")
	deployTarget := defaultIfEmpty(req.DeployTarget, "k8s")
	if deployTarget == "compose" {
		var target model.DeploymentTarget
		if err := l.svcCtx.DB.WithContext(ctx).Where("id = ? AND target_type = ?", req.ClusterID, "compose").First(&target).Error; err != nil {
			return DeployTargetResp{}, fmt.Errorf("compose deployment target not found: %w", err)
		}
	}
	policyJSON := mustJSON(req.Policy)
	var row model.ServiceDeployTarget
	err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ? AND is_default = 1", serviceID).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return DeployTargetResp{}, err
		}
		row = model.ServiceDeployTarget{
			ServiceID:    serviceID,
			ClusterID:    req.ClusterID,
			Namespace:    ns,
			DeployTarget: deployTarget,
			PolicyJSON:   policyJSON,
			IsDefault:    true,
			UpdatedBy:    uint(uid),
		}
		if err := l.svcCtx.DB.WithContext(ctx).Create(&row).Error; err != nil {
			return DeployTargetResp{}, err
		}
	} else {
		row.ClusterID = req.ClusterID
		row.Namespace = ns
		row.DeployTarget = deployTarget
		row.PolicyJSON = policyJSON
		row.UpdatedBy = uint(uid)
		if err := l.svcCtx.DB.WithContext(ctx).Save(&row).Error; err != nil {
			return DeployTargetResp{}, err
		}
	}
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.Service{}).Where("id = ?", serviceID).Update("default_target_id", row.ID).Error; err != nil {
		return DeployTargetResp{}, err
	}
	return toDeployTargetResp(&row), nil
}

func (l *Logic) ListReleaseRecords(ctx context.Context, serviceID uint) ([]ReleaseRecordItem, error) {
	var rows []model.ServiceReleaseRecord
	if err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ?", serviceID).Order("id DESC").Limit(50).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ReleaseRecordItem, 0, len(rows))
	for i := range rows {
		out = append(out, ReleaseRecordItem{
			ID:           rows[i].ID,
			ServiceID:    rows[i].ServiceID,
			RevisionID:   rows[i].RevisionID,
			ClusterID:    rows[i].ClusterID,
			Namespace:    rows[i].Namespace,
			Env:          rows[i].Env,
			DeployTarget: rows[i].DeployTarget,
			Status:       rows[i].Status,
			Error:        rows[i].Error,
			CreatedAt:    rows[i].CreatedAt,
		})
	}
	return out, nil
}
