// Package service 提供服务部署相关的业务逻辑。
//
// 本文件实现服务部署预览、执行、Helm 操作等业务逻辑。
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

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	deploymentlogic "github.com/cy77cc/OpsPilot/internal/service/deployment"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"gorm.io/gorm"
)

// DeployPreview 预览服务部署结果。
//
// 解析部署目标、预览渲染结果、生成检查项和警告。
//
// 参数:
//   - ctx: 上下文
//   - id: 服务 ID
//   - req: 部署请求
//
// 返回: 部署预览响应和错误信息
func (l *Logic) DeployPreview(ctx context.Context, id uint, req DeployReq) (DeployPreviewResp, error) {
	var service model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&service, id).Error; err != nil {
		return DeployPreviewResp{}, err
	}
	target, err := l.resolveDeployTarget(ctx, id, req)
	if err != nil {
		return DeployPreviewResp{}, err
	}
	targetID, err := l.ensureUnifiedTargetID(ctx, &service, target, req.Env)
	if err != nil {
		return DeployPreviewResp{}, err
	}
	deployLogic := deploymentlogic.NewLogic(l.svcCtx)
	preview, err := deployLogic.PreviewRelease(ctx, deploymentlogic.ReleasePreviewReq{
		ServiceID: id,
		TargetID:  targetID,
		Env:       req.Env,
		Strategy:  "rolling",
		Variables: req.Variables,
	})
	if err != nil {
		return DeployPreviewResp{}, err
	}
	checks := make([]RenderDiagnostic, 0, len(preview.Checks))
	for i := range preview.Checks {
		checks = append(checks, RenderDiagnostic{Level: preview.Checks[i]["level"], Code: preview.Checks[i]["code"], Message: preview.Checks[i]["message"]})
	}
	warnings := make([]RenderDiagnostic, 0, len(preview.Warnings))
	for i := range preview.Warnings {
		warnings = append(warnings, RenderDiagnostic{Level: preview.Warnings[i]["level"], Code: preview.Warnings[i]["code"], Message: preview.Warnings[i]["message"]})
	}
	return DeployPreviewResp{
		ResolvedYAML:     preview.ResolvedManifest,
		Checks:           checks,
		Warnings:         warnings,
		Target:           target,
		TargetID:         targetID,
		PreviewToken:     preview.PreviewToken,
		PreviewExpiresAt: preview.PreviewExpiresAt,
	}, nil
}

// Deploy 执行服务部署。
//
// 解析部署目标、预览渲染、执行部署、返回发布记录 ID。
//
// 参数:
//   - ctx: 上下文
//   - id: 服务 ID
//   - operator: 操作者 ID
//   - req: 部署请求
//
// 返回: 发布记录 ID 和错误信息
func (l *Logic) Deploy(ctx context.Context, id uint, operator uint64, req DeployReq) (uint, error) {
	var service model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&service, id).Error; err != nil {
		return 0, err
	}
	targetResp, err := l.resolveDeployTarget(ctx, id, req)
	if err != nil {
		return 0, err
	}
	targetID, err := l.ensureUnifiedTargetID(ctx, &service, targetResp, req.Env)
	if err != nil {
		return 0, err
	}
	deployLogic := deploymentlogic.NewLogic(l.svcCtx)
	preview, err := deployLogic.PreviewRelease(ctx, deploymentlogic.ReleasePreviewReq{
		ServiceID: id,
		TargetID:  targetID,
		Env:       req.Env,
		Strategy:  "rolling",
		Variables: req.Variables,
	})
	if err != nil {
		return 0, err
	}
	apply, err := deployLogic.ApplyRelease(ctx, operator, deploymentlogic.ReleasePreviewReq{
		ServiceID:      id,
		TargetID:       targetID,
		Env:            req.Env,
		Strategy:       "rolling",
		Variables:      req.Variables,
		PreviewToken:   preview.PreviewToken,
		TriggerSource:  "manual",
		TriggerContext: map[string]any{"entry": "service.deploy", "namespace": targetResp.Namespace, "deploy_target": targetResp.DeployTarget},
	})
	if err != nil {
		return apply.ReleaseID, err
	}
	return apply.ReleaseID, nil
}

// HelmImport 导入 Helm Chart。
//
// 创建 Helm 发布记录，保存 Chart 信息和 Values。
//
// 参数:
//   - ctx: 上下文
//   - uid: 操作用户 ID
//   - req: Helm 导入请求
//
// 返回: Helm 发布记录和错误信息
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

// HelmRender 渲染 Helm 模板。
//
// 使用 helm template 命令渲染 Helm Chart。
//
// 参数:
//   - ctx: 上下文
//   - req: Helm 渲染请求
//
// 返回: 渲染后的 YAML、诊断信息和错误
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

// deployHelm 部署 Helm 服务。
//
// 更新 Helm 发布记录状态为 deployed。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//
// 返回: 错误信息
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

// applyComposeByTarget 通过 SSH 执行 Docker Compose 部署。
//
// 连接到部署目标节点，上传 docker-compose.yaml 并执行部署。
//
// 参数:
//   - ctx: 上下文
//   - targetID: 部署目标 ID
//   - releaseID: 发布记录 ID
//   - manifest: docker-compose.yaml 内容
//
// 返回: 执行输出和错误信息
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

// toDeployTargetResp 将部署目标模型转换为响应结构。
//
// 参数:
//   - t: 部署目标模型
//
// 返回: 部署目标响应
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

// resolveDeployTarget 解析部署目标。
//
// 优先使用请求中指定的集群，否则查找服务的默认部署目标。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//   - req: 部署请求
//
// 返回: 部署目标响应和错误信息
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
			fallback, ferr := l.resolveFallbackDeployTarget(ctx, serviceID, req)
			if ferr != nil {
				return DeployTargetResp{}, l.newDeployTargetNotConfiguredErr(ctx, serviceID, req, ferr)
			}
			return fallback, nil
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

// newDeployTargetNotConfiguredErr 创建部署目标未配置错误。
//
// 生成包含详细上下文信息的错误消息。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//   - req: 部署请求
//   - cause: 原始错误
//
// 返回: 格式化的错误
func (l *Logic) newDeployTargetNotConfiguredErr(ctx context.Context, serviceID uint, req DeployReq, cause error) error {
	var svc model.Service
	if err := l.svcCtx.DB.WithContext(ctx).Select("id", "project_id", "team_id", "env").First(&svc, serviceID).Error; err != nil {
		return fmt.Errorf("deploy target not configured: %w", cause)
	}
	runtime := strings.TrimSpace(defaultIfEmpty(req.DeployTarget, "k8s"))
	env := strings.TrimSpace(defaultIfEmpty(req.Env, svc.Env))
	return fmt.Errorf(
		"deploy target not configured (project_id=%d, team_id=%d, env=%s, target_type=%s): %w; hint: 配置服务默认部署目标或创建匹配作用域的 active deployment target",
		svc.ProjectID,
		svc.TeamID,
		defaultIfEmpty(env, "staging"),
		runtime,
		cause,
	)
}

// resolveFallbackDeployTarget 解析备用部署目标。
//
// 当服务没有配置默认部署目标时，根据项目、团队、环境查找匹配的部署目标。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//   - req: 部署请求
//
// 返回: 部署目标响应和错误信息
func (l *Logic) resolveFallbackDeployTarget(ctx context.Context, serviceID uint, req DeployReq) (DeployTargetResp, error) {
	var svc model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&svc, serviceID).Error; err != nil {
		return DeployTargetResp{}, err
	}
	runtime := strings.TrimSpace(defaultIfEmpty(req.DeployTarget, "k8s"))
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentTarget{}).
		Where("target_type = ? AND status = ?", runtime, "active")
	if svc.ProjectID > 0 {
		q = q.Where("project_id = ?", svc.ProjectID)
	}
	if svc.TeamID > 0 {
		q = q.Where("team_id = ?", svc.TeamID)
	}
	env := strings.TrimSpace(defaultIfEmpty(req.Env, svc.Env))
	if env != "" {
		q = q.Where("env = ? OR env = ''", env)
	}

	var target model.DeploymentTarget
	if err := q.Order("CASE WHEN readiness_status = 'ready' THEN 0 WHEN readiness_status = 'unknown' THEN 1 ELSE 2 END, id DESC").First(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeployTargetResp{}, fmt.Errorf("no active deployment target found for service scope")
		}
		return DeployTargetResp{}, err
	}

	clusterID := target.ClusterID
	if runtime == "compose" {
		clusterID = target.ID
	}
	resp := DeployTargetResp{
		ServiceID:    serviceID,
		ClusterID:    clusterID,
		Namespace:    defaultIfEmpty(req.Namespace, "default"),
		DeployTarget: runtime,
		IsDefault:    true,
	}

	// 回填默认目标，避免后续重复触发 fallback 查询（失败不影响本次部署）。
	_ = l.cacheFallbackDefaultTarget(ctx, serviceID, resp)
	return resp, nil
}

// cacheFallbackDefaultTarget 缓存备用默认部署目标。
//
// 将查找到的备用部署目标回填为服务的默认目标，避免后续重复查询。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//   - target: 部署目标响应
//
// 返回: 错误信息
func (l *Logic) cacheFallbackDefaultTarget(ctx context.Context, serviceID uint, target DeployTargetResp) error {
	if serviceID == 0 || target.ClusterID == 0 {
		return nil
	}
	deployTarget := strings.TrimSpace(defaultIfEmpty(target.DeployTarget, "k8s"))
	if deployTarget != "k8s" && deployTarget != "compose" {
		return nil
	}
	var existing model.ServiceDeployTarget
	if err := l.svcCtx.DB.WithContext(ctx).
		Select("id", "cluster_id", "namespace", "deploy_target").
		Where("service_id = ? AND is_default = 1", serviceID).
		First(&existing).Error; err == nil {
		// 竞争场景：其他请求已回填默认目标，避免覆盖。
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	_, err := l.UpsertDeployTarget(ctx, serviceID, 0, DeployTargetUpsertReq{
		ClusterID:    target.ClusterID,
		Namespace:    defaultIfEmpty(target.Namespace, "default"),
		DeployTarget: deployTarget,
		Policy:       map[string]any{},
	})
	return err
}

// loadNodeSSHPrivateKey 加载节点 SSH 私钥。
//
// 从数据库读取 SSH 密钥，处理加密密钥的解密。
//
// 参数:
//   - ctx: 上下文
//   - node: 节点模型
//
// 返回: 私钥、密码和错误信息
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

// resolveServiceTemplate 解析服务模板变量。
//
// 合并请求变量和环境变量，替换模板中的变量占位符。
//
// 参数:
//   - ctx: 上下文
//   - service: 服务模型
//   - env: 环境名称
//   - reqValues: 请求变量映射
//
// 返回: 解析后的 YAML、未解析变量列表和错误信息
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

// UpsertDeployTarget 创建或更新服务部署目标。
//
// 设置服务的默认部署目标配置。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//   - uid: 操作用户 ID
//   - req: 部署目标更新请求
//
// 返回: 部署目标响应和错误信息
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

// ListReleaseRecords 获取服务发布记录列表。
//
// 查询统一发布记录表和服务发布记录表，返回合并后的记录列表。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//
// 返回: 发布记录列表和错误信息
func (l *Logic) ListReleaseRecords(ctx context.Context, serviceID uint) ([]ReleaseRecordItem, error) {
	var releases []model.DeploymentRelease
	if err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ?", serviceID).Order("id DESC").Limit(50).Find(&releases).Error; err != nil {
		return nil, err
	}
	out := make([]ReleaseRecordItem, 0, len(releases))
	for i := range releases {
		item := ReleaseRecordItem{
			ID:               releases[i].ID,
			UnifiedReleaseID: releases[i].ID,
			ServiceID:        releases[i].ServiceID,
			RevisionID:       releases[i].RevisionID,
			Env:              releases[i].NamespaceOrProject,
			DeployTarget:     releases[i].RuntimeType,
			Status:           releases[i].Status,
			TriggerSource:    releases[i].TriggerSource,
			CIRunID:          releases[i].CIRunID,
			CreatedAt:        releases[i].CreatedAt,
		}
		var target model.DeploymentTarget
		if err := l.svcCtx.DB.WithContext(ctx).First(&target, releases[i].TargetID).Error; err == nil {
			item.ClusterID = target.ClusterID
			item.Namespace = defaultIfEmpty(target.Env, releases[i].NamespaceOrProject)
		}
		out = append(out, item)
	}
	if len(out) > 0 {
		return out, nil
	}

	var legacyRows []model.ServiceReleaseRecord
	if err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ?", serviceID).Order("id DESC").Limit(50).Find(&legacyRows).Error; err != nil {
		return nil, err
	}
	for i := range legacyRows {
		out = append(out, ReleaseRecordItem{
			ID:           legacyRows[i].ID,
			ServiceID:    legacyRows[i].ServiceID,
			RevisionID:   legacyRows[i].RevisionID,
			ClusterID:    legacyRows[i].ClusterID,
			Namespace:    legacyRows[i].Namespace,
			Env:          legacyRows[i].Env,
			DeployTarget: legacyRows[i].DeployTarget,
			Status:       legacyRows[i].Status,
			Error:        legacyRows[i].Error,
			CreatedAt:    legacyRows[i].CreatedAt,
		})
	}
	return out, nil
}

// ensureUnifiedTargetID 确保统一部署目标 ID 存在。
//
// 对于 K8s 目标，查找或创建对应的 DeploymentTarget 记录；
// 对于 Compose 目标，直接返回目标 ID。
//
// 参数:
//   - ctx: 上下文
//   - service: 服务模型
//   - target: 部署目标响应
//   - env: 环境名称
//
// 返回: 统一目标 ID 和错误信息
func (l *Logic) ensureUnifiedTargetID(ctx context.Context, service *model.Service, target DeployTargetResp, env string) (uint, error) {
	runtime := strings.TrimSpace(defaultIfEmpty(target.DeployTarget, "k8s"))
	if runtime == "compose" {
		return target.ClusterID, nil
	}
	var row model.DeploymentTarget
	err := l.svcCtx.DB.WithContext(ctx).
		Where("target_type = ? AND cluster_id = ?", "k8s", target.ClusterID).
		Order("id DESC").
		First(&row).Error
	if err == nil {
		return row.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	row = model.DeploymentTarget{
		Name:            fmt.Sprintf("svc-%d-cluster-%d", service.ID, target.ClusterID),
		TargetType:      "k8s",
		RuntimeType:     "k8s",
		ClusterID:       target.ClusterID,
		ClusterSource:   "platform_managed",
		ProjectID:       service.ProjectID,
		TeamID:          service.TeamID,
		Env:             defaultIfEmpty(env, service.Env),
		Status:          "active",
		ReadinessStatus: "unknown",
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}
