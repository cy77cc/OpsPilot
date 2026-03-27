// Package deployment 提供环境引导的业务逻辑实现。
//
// 本文件包含环境引导安装、凭证管理和连通性测试等业务逻辑。
package deployment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// runtimePackageManifest 是运行时安装包清单结构。
type runtimePackageManifest struct {
	Runtime         string `json:"runtime"`          // 运行时类型
	Version         string `json:"version"`          // 版本
	PackageFile     string `json:"package_file"`     // 安装包文件名
	SHA256          string `json:"sha256"`           // 校验和
	PreflightScript string `json:"preflight_script"` // 预检查脚本
	InstallScript   string `json:"install_script"`   // 安装脚本
	VerifyScript    string `json:"verify_script"`    // 验证脚本
	UninstallScript string `json:"uninstall_script"` // 卸载脚本
	PreflightCmd    string `json:"preflight_command"` // 预检查命令
	InstallCmd      string `json:"install_command"`  // 安装命令
	VerifyCmd       string `json:"verify_command"`   // 验证命令
	UninstallCmd    string `json:"uninstall_command"` // 卸载命令
}

// StartEnvironmentBootstrap 启动环境引导安装任务。
//
// 参数:
//   - ctx: 上下文
//   - uid: 用户 ID
//   - req: 环境引导请求
//
// 返回: 环境引导响应
func (l *Logic) StartEnvironmentBootstrap(ctx context.Context, uid uint64, req EnvironmentBootstrapReq) (EnvironmentBootstrapResp, error) {
	runtimeType := normalizedRuntime(req.RuntimeType, req.RuntimeType)
	if runtimeType != "k8s" && runtimeType != "compose" {
		return EnvironmentBootstrapResp{}, fmt.Errorf("unsupported runtime_type: %s", runtimeType)
	}
	manifest, manifestPath, err := l.resolveRuntimePackage(runtimeType, strings.TrimSpace(req.PackageVersion))
	if err != nil {
		return EnvironmentBootstrapResp{}, err
	}
	job := &model.EnvironmentInstallJob{
		ID:              fmt.Sprintf("envboot-%d", time.Now().UnixNano()),
		Name:            strings.TrimSpace(req.Name),
		RuntimeType:     runtimeType,
		TargetEnv:       defaultIfEmpty(strings.TrimSpace(req.Env), "staging"),
		TargetID:        req.TargetID,
		ClusterID:       req.ClusterID,
		Status:          "queued",
		PackageVersion:  strings.TrimSpace(req.PackageVersion),
		PackagePath:     manifestPath,
		PackageChecksum: manifest.SHA256,
		CreatedBy:       uid,
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(job).Error; err != nil {
		return EnvironmentBootstrapResp{}, err
	}
	now := time.Now().UTC()
	job.StartedAt = &now
	job.Status = "running"
	_ = l.svcCtx.DB.WithContext(ctx).Save(job).Error

	var hosts []model.Node
	switch runtimeType {
	case "k8s":
		control, workers, hostErr := l.loadBootstrapHosts(ctx, req.ControlPlaneID, req.WorkerIDs)
		if hostErr != nil {
			l.finishInstallJob(ctx, job, "failed", hostErr.Error(), nil)
			return EnvironmentBootstrapResp{JobID: job.ID, Status: "failed", RuntimeType: runtimeType, PackageVersion: req.PackageVersion, TargetID: req.TargetID}, hostErr
		}
		hosts = append(hosts, *control)
		hosts = append(hosts, workers...)
	case "compose":
		nodes, hostErr := l.loadComposeHosts(ctx, req.NodeIDs)
		if hostErr != nil {
			l.finishInstallJob(ctx, job, "failed", hostErr.Error(), nil)
			return EnvironmentBootstrapResp{JobID: job.ID, Status: "failed", RuntimeType: runtimeType, PackageVersion: req.PackageVersion, TargetID: req.TargetID}, hostErr
		}
		hosts = nodes
	}

	if err := l.runBootstrapPhase(ctx, job, hosts, "preflight", manifest.PreflightScript, manifest.PreflightCmd, runtimeType); err != nil {
		rollbackErr := l.runBootstrapPhase(ctx, job, hosts, "rollback", manifest.UninstallScript, manifest.UninstallCmd, runtimeType)
		if rollbackErr != nil {
			err = fmt.Errorf("%v; rollback failed: %v", err, rollbackErr)
		}
		l.finishInstallJob(ctx, job, "failed", err.Error(), nil)
		return EnvironmentBootstrapResp{JobID: job.ID, Status: "failed", RuntimeType: runtimeType, PackageVersion: req.PackageVersion, TargetID: req.TargetID}, err
	}
	if err := l.runBootstrapPhase(ctx, job, hosts, "install", manifest.InstallScript, manifest.InstallCmd, runtimeType); err != nil {
		rollbackErr := l.runBootstrapPhase(ctx, job, hosts, "rollback", manifest.UninstallScript, manifest.UninstallCmd, runtimeType)
		if rollbackErr != nil {
			err = fmt.Errorf("%v; rollback failed: %v", err, rollbackErr)
		}
		l.finishInstallJob(ctx, job, "failed", err.Error(), nil)
		return EnvironmentBootstrapResp{JobID: job.ID, Status: "failed", RuntimeType: runtimeType, PackageVersion: req.PackageVersion, TargetID: req.TargetID}, err
	}
	if err := l.runBootstrapPhase(ctx, job, hosts, "verify", manifest.VerifyScript, manifest.VerifyCmd, runtimeType); err != nil {
		l.finishInstallJob(ctx, job, "failed", err.Error(), nil)
		return EnvironmentBootstrapResp{JobID: job.ID, Status: "failed", RuntimeType: runtimeType, PackageVersion: req.PackageVersion, TargetID: req.TargetID}, err
	}

	if req.TargetID > 0 {
		_ = l.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentTarget{}).
			Where("id = ?", req.TargetID).
			Updates(map[string]any{"bootstrap_job_id": job.ID, "readiness_status": "ready"}).Error
	}
	l.finishInstallJob(ctx, job, "succeeded", "", map[string]any{"host_total": len(hosts), "manifest": manifestPath})
	return EnvironmentBootstrapResp{JobID: job.ID, Status: "succeeded", RuntimeType: runtimeType, PackageVersion: req.PackageVersion, TargetID: req.TargetID}, nil
}

// GetEnvironmentBootstrapJob 获取环境引导任务详情。
//
// 参数:
//   - ctx: 上下文
//   - jobID: 任务 ID
//
// 返回: 环境安装任务对象
func (l *Logic) GetEnvironmentBootstrapJob(ctx context.Context, jobID string) (*model.EnvironmentInstallJob, error) {
	if strings.TrimSpace(jobID) == "" {
		return nil, fmt.Errorf("job_id is required")
	}
	var job model.EnvironmentInstallJob
	if err := l.svcCtx.DB.WithContext(ctx).Where("id = ?", strings.TrimSpace(jobID)).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// RegisterPlatformCredential 注册平台托管集群凭证。
//
// 参数:
//   - ctx: 上下文
//   - uid: 用户 ID
//   - req: 注册请求
//
// 返回: 凭证响应
func (l *Logic) RegisterPlatformCredential(ctx context.Context, uid uint64, req PlatformCredentialRegisterReq) (ClusterCredentialResp, error) {
	var cluster model.Cluster
	if err := l.svcCtx.DB.WithContext(ctx).First(&cluster, req.ClusterID).Error; err != nil {
		return ClusterCredentialResp{}, fmt.Errorf("cluster not found: %w", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = defaultIfEmpty(strings.TrimSpace(cluster.Name), fmt.Sprintf("cluster-%d", cluster.ID))
	}
	cred := model.ClusterCredential{
		Name:         name,
		RuntimeType:  normalizedRuntime(req.RuntimeType, "k8s"),
		Source:       "platform_managed",
		ClusterID:    cluster.ID,
		Endpoint:     strings.TrimSpace(cluster.Endpoint),
		AuthMethod:   defaultIfEmpty(strings.TrimSpace(cluster.AuthMethod), "kubeconfig"),
		MetadataJSON: toJSON(map[string]any{"cluster_name": cluster.Name, "management_mode": cluster.ManagementMode}),
		Status:       "active",
		CreatedBy:    uid,
	}
	if err := l.fillEncryptedCredentialMaterials(&cred, cluster.KubeConfig, cluster.CACert, "", "", cluster.Token); err != nil {
		return ClusterCredentialResp{}, err
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&cred).Error; err != nil {
		return ClusterCredentialResp{}, err
	}
	return toCredentialResp(cred), nil
}

// ImportExternalCredential 导入外部集群凭证。
//
// 参数:
//   - ctx: 上下文
//   - uid: 用户 ID
//   - req: 导入请求
//
// 返回: 凭证响应
func (l *Logic) ImportExternalCredential(ctx context.Context, uid uint64, req ClusterCredentialImportReq) (ClusterCredentialResp, error) {
	authMethod := strings.TrimSpace(req.AuthMethod)
	if authMethod == "" {
		if strings.TrimSpace(req.Kubeconfig) != "" {
			authMethod = "kubeconfig"
		} else {
			authMethod = "cert"
		}
	}
	if authMethod == "kubeconfig" {
		if strings.TrimSpace(req.Kubeconfig) == "" {
			return ClusterCredentialResp{}, fmt.Errorf("kubeconfig is required")
		}
		if _, err := clientcmd.Load([]byte(req.Kubeconfig)); err != nil {
			return ClusterCredentialResp{}, fmt.Errorf("invalid kubeconfig: %w", err)
		}
	} else {
		if strings.TrimSpace(req.Endpoint) == "" || strings.TrimSpace(req.CACert) == "" || strings.TrimSpace(req.Cert) == "" || strings.TrimSpace(req.Key) == "" {
			return ClusterCredentialResp{}, fmt.Errorf("endpoint/ca_cert/cert/key are required for certificate auth")
		}
	}
	cred := model.ClusterCredential{
		Name:         strings.TrimSpace(req.Name),
		RuntimeType:  normalizedRuntime(req.RuntimeType, "k8s"),
		Source:       "external_managed",
		Endpoint:     strings.TrimSpace(req.Endpoint),
		AuthMethod:   authMethod,
		MetadataJSON: toJSON(map[string]any{"imported": true}),
		Status:       "active",
		CreatedBy:    uid,
	}
	if err := l.fillEncryptedCredentialMaterials(&cred, req.Kubeconfig, req.CACert, req.Cert, req.Key, req.Token); err != nil {
		return ClusterCredentialResp{}, err
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&cred).Error; err != nil {
		return ClusterCredentialResp{}, err
	}
	return toCredentialResp(cred), nil
}

// TestCredentialConnectivity 测试凭证连通性。
//
// 参数:
//   - ctx: 上下文
//   - credentialID: 凭证 ID
//
// 返回: 连通性测试响应
func (l *Logic) TestCredentialConnectivity(ctx context.Context, credentialID uint) (ClusterCredentialTestResp, error) {
	var cred model.ClusterCredential
	if err := l.svcCtx.DB.WithContext(ctx).First(&cred, credentialID).Error; err != nil {
		return ClusterCredentialTestResp{}, err
	}
	start := time.Now()
	connected := false
	message := "ok"

	cfg, err := l.buildRestConfigFromCredential(&cred)
	if err != nil {
		message = err.Error()
	} else {
		cli, cliErr := kubernetes.NewForConfig(cfg)
		if cliErr != nil {
			message = cliErr.Error()
		} else if _, verErr := cli.Discovery().ServerVersion(); verErr != nil {
			message = verErr.Error()
		} else {
			connected = true
		}
	}

	now := time.Now().UTC()
	status := "failed"
	if connected {
		status = "ok"
	}
	_ = l.svcCtx.DB.WithContext(ctx).Model(&model.ClusterCredential{}).Where("id = ?", cred.ID).Updates(map[string]any{
		"last_test_at":      &now,
		"last_test_status":  status,
		"last_test_message": truncateText(message, 500),
	}).Error

	return ClusterCredentialTestResp{
		CredentialID: cred.ID,
		Connected:    connected,
		Message:      truncateText(message, 500),
		LatencyMS:    time.Since(start).Milliseconds(),
	}, nil
}

// ListCredentials 获取凭证列表。
//
// 参数:
//   - ctx: 上下文
//   - runtimeType: 运行时类型 (可选筛选)
//
// 返回: 凭证响应列表
func (l *Logic) ListCredentials(ctx context.Context, runtimeType string) ([]ClusterCredentialResp, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.ClusterCredential{})
	if rt := strings.TrimSpace(runtimeType); rt != "" {
		q = q.Where("runtime_type = ?", rt)
	}
	var rows []model.ClusterCredential
	if err := q.Order("id DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ClusterCredentialResp, 0, len(rows))
	for i := range rows {
		out = append(out, toCredentialResp(rows[i]))
	}
	return out, nil
}

// resolveRuntimePackage 解析运行时安装包清单。
//
// 参数:
//   - runtimeType: 运行时类型
//   - version: 版本号
//
// 返回: 清单对象、清单路径、错误
func (l *Logic) resolveRuntimePackage(runtimeType, version string) (*runtimePackageManifest, string, error) {
	ver := strings.TrimSpace(version)
	if ver == "" {
		return nil, "", fmt.Errorf("package_version is required")
	}
	manifestPath := filepath.Join("script", "runtime", runtimeType, ver, "manifest.json")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", fmt.Errorf("runtime package manifest not found: %w", err)
	}
	var manifest runtimePackageManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return nil, "", fmt.Errorf("invalid runtime manifest: %w", err)
	}
	if strings.TrimSpace(manifest.Runtime) != "" && !strings.EqualFold(strings.TrimSpace(manifest.Runtime), runtimeType) {
		return nil, "", fmt.Errorf("runtime mismatch in manifest")
	}
	if strings.TrimSpace(manifest.PackageFile) != "" && strings.TrimSpace(manifest.SHA256) != "" {
		packagePath := filepath.Join(filepath.Dir(manifestPath), strings.TrimSpace(manifest.PackageFile))
		pkgContent, readErr := os.ReadFile(packagePath)
		if readErr != nil {
			return nil, "", fmt.Errorf("runtime package file not found: %w", readErr)
		}
		sum := sha256.Sum256(pkgContent)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), strings.TrimSpace(manifest.SHA256)) {
			return nil, "", fmt.Errorf("runtime package checksum mismatch")
		}
	}
	if strings.TrimSpace(manifest.InstallScript) == "" && strings.TrimSpace(manifest.InstallCmd) == "" {
		return nil, "", fmt.Errorf("runtime manifest missing install action")
	}
	return &manifest, manifestPath, nil
}

// runBootstrapPhase 执行引导阶段命令。
//
// 参数:
//   - ctx: 上下文
//   - job: 安装任务
//   - hosts: 主机列表
//   - phase: 阶段名称
//   - scriptRel: 脚本相对路径
//   - fallbackCmd: 回退命令
//   - runtimeType: 运行时类型
//
// 返回: 执行错误
func (l *Logic) runBootstrapPhase(ctx context.Context, job *model.EnvironmentInstallJob, hosts []model.Node, phase, scriptRel, fallbackCmd, runtimeType string) error {
	for i := range hosts {
		host := hosts[i]
		step := model.EnvironmentInstallJobStep{JobID: job.ID, StepName: phase, Phase: phase, Status: "running", HostID: uint(host.ID)}
		now := time.Now().UTC()
		step.StartedAt = &now
		if err := l.svcCtx.DB.WithContext(ctx).Create(&step).Error; err != nil {
			return err
		}
		cmd, err := l.resolvePhaseCommand(runtimeType, strings.TrimSpace(job.PackageVersion), scriptRel, fallbackCmd, phase)
		if err != nil {
			l.finishStep(ctx, &step, "failed", "", err.Error())
			return err
		}
		if err := l.execCommandOnNode(ctx, host, cmd, &step); err != nil {
			l.finishStep(ctx, &step, "failed", step.Output, err.Error())
			return fmt.Errorf("%s on host %d failed: %w", phase, host.ID, err)
		}
		l.finishStep(ctx, &step, "succeeded", step.Output, "")
	}
	return nil
}

// resolvePhaseCommand 解析阶段执行命令。
//
// 参数:
//   - runtimeType: 运行时类型
//   - version: 版本号
//   - scriptRel: 脚本相对路径
//   - fallbackCmd: 回退命令
//   - phase: 阶段名称
//
// 返回: 执行命令
func (l *Logic) resolvePhaseCommand(runtimeType, version, scriptRel, fallbackCmd, phase string) (string, error) {
	if script := strings.TrimSpace(scriptRel); script != "" {
		scriptPath := filepath.Join("script", "runtime", runtimeType, version, script)
		if _, err := os.Stat(scriptPath); err != nil {
			return "", fmt.Errorf("%s script missing: %s", phase, scriptPath)
		}
		return fmt.Sprintf("bash %s", scriptPath), nil
	}
	if strings.TrimSpace(fallbackCmd) != "" {
		return strings.TrimSpace(fallbackCmd), nil
	}
	if phase == "preflight" {
		if runtimeType == "compose" {
			return "command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1", nil
		}
		return "command -v kubeadm >/dev/null 2>&1 && command -v kubectl >/dev/null 2>&1", nil
	}
	return "", fmt.Errorf("no executable action configured for phase %s", phase)
}

// execCommandOnNode 在节点上执行命令。
//
// 参数:
//   - ctx: 上下文
//   - host: 主机节点
//   - cmd: 执行命令
//   - step: 步骤记录
//
// 返回: 执行错误
func (l *Logic) execCommandOnNode(ctx context.Context, host model.Node, cmd string, step *model.EnvironmentInstallJobStep) error {
	privateKey, passphrase, err := l.loadNodePrivateKey(ctx, &host)
	if err != nil {
		return err
	}
	password := strings.TrimSpace(host.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(host.SSHUser, password, host.IP, host.Port, privateKey, passphrase)
	if err != nil {
		return err
	}
	defer cli.Close()
	out, runErr := sshclient.RunCommand(cli, cmd)
	step.Output = truncateText(out, 2000)
	return runErr
}

// finishStep 完成步骤记录。
//
// 参数:
//   - ctx: 上下文
//   - step: 步骤记录
//   - status: 状态
//   - output: 输出
//   - errMsg: 错误信息
func (l *Logic) finishStep(ctx context.Context, step *model.EnvironmentInstallJobStep, status, output, errMsg string) {
	now := time.Now().UTC()
	step.Status = status
	step.Output = truncateText(output, 2000)
	step.ErrorMessage = truncateText(errMsg, 1000)
	step.FinishedAt = &now
	_ = l.svcCtx.DB.WithContext(ctx).Save(step).Error
}

// finishInstallJob 完成安装任务。
//
// 参数:
//   - ctx: 上下文
//   - job: 安装任务
//   - status: 状态
//   - errMsg: 错误信息
//   - result: 结果数据
func (l *Logic) finishInstallJob(ctx context.Context, job *model.EnvironmentInstallJob, status, errMsg string, result any) {
	now := time.Now().UTC()
	job.Status = status
	job.ErrorMessage = truncateText(errMsg, 1000)
	job.FinishedAt = &now
	if result != nil {
		job.ResultJSON = toJSON(result)
	}
	_ = l.svcCtx.DB.WithContext(ctx).Save(job).Error
}

// loadComposeHosts 加载 Docker Compose 主机列表。
//
// 参数:
//   - ctx: 上下文
//   - nodeIDs: 节点 ID 列表
//
// 返回: 主机列表
func (l *Logic) loadComposeHosts(ctx context.Context, nodeIDs []uint) ([]model.Node, error) {
	if len(nodeIDs) == 0 {
		return nil, fmt.Errorf("node_ids is required for compose runtime")
	}
	hosts := make([]model.Node, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		var host model.Node
		if err := l.svcCtx.DB.WithContext(ctx).First(&host, id).Error; err != nil {
			return nil, fmt.Errorf("host node %d not found", id)
		}
		if ok, reason := hostlogic.EvaluateOperationalEligibility(&host); !ok {
			return nil, fmt.Errorf("host node %d unavailable: %s", id, reason)
		}
		hosts = append(hosts, host)
	}
	return hosts, nil
}

// fillEncryptedCredentialMaterials 填充加密的凭证材料。
//
// 参数:
//   - cred: 凭证对象
//   - kubeconfig: kubeconfig 内容
//   - caCert: CA 证书
//   - cert: 客户端证书
//   - key: 客户端私钥
//   - token: Bearer Token
//
// 返回: 错误信息
func (l *Logic) fillEncryptedCredentialMaterials(cred *model.ClusterCredential, kubeconfig, caCert, cert, key, token string) error {
	if cred == nil {
		return fmt.Errorf("credential is nil")
	}
	enc := strings.TrimSpace(config.CFG.Security.EncryptionKey)
	if enc == "" {
		return fmt.Errorf("security.encryption_key is required")
	}
	var err error
	if strings.TrimSpace(kubeconfig) != "" {
		cred.KubeconfigEnc, err = utils.EncryptText(strings.TrimSpace(kubeconfig), enc)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(caCert) != "" {
		cred.CACertEnc, err = utils.EncryptText(strings.TrimSpace(caCert), enc)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(cert) != "" {
		cred.CertEnc, err = utils.EncryptText(strings.TrimSpace(cert), enc)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(key) != "" {
		cred.KeyEnc, err = utils.EncryptText(strings.TrimSpace(key), enc)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(token) != "" {
		cred.TokenEnc, err = utils.EncryptText(strings.TrimSpace(token), enc)
		if err != nil {
			return err
		}
	}
	return nil
}

// buildRestConfigFromCredential 从凭证构建 REST 配置。
//
// 参数:
//   - cred: 凭证对象
//
// 返回: Kubernetes REST 配置
func (l *Logic) buildRestConfigFromCredential(cred *model.ClusterCredential) (*rest.Config, error) {
	enc := strings.TrimSpace(config.CFG.Security.EncryptionKey)
	if enc == "" {
		return nil, fmt.Errorf("security.encryption_key is required")
	}
	if strings.TrimSpace(cred.KubeconfigEnc) != "" {
		kubeconfig, err := utils.DecryptText(cred.KubeconfigEnc, enc)
		if err != nil {
			return nil, err
		}
		return clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	}
	ca, err := utils.DecryptText(cred.CACertEnc, enc)
	if err != nil {
		return nil, err
	}
	cert, err := utils.DecryptText(cred.CertEnc, enc)
	if err != nil {
		return nil, err
	}
	key, err := utils.DecryptText(cred.KeyEnc, enc)
	if err != nil {
		return nil, err
	}
	token := ""
	if strings.TrimSpace(cred.TokenEnc) != "" {
		token, _ = utils.DecryptText(cred.TokenEnc, enc)
	}
	if strings.TrimSpace(cred.Endpoint) == "" {
		return nil, fmt.Errorf("credential endpoint is empty")
	}
	return &rest.Config{
		Host: strings.TrimSpace(cred.Endpoint),
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   []byte(ca),
			CertData: []byte(cert),
			KeyData:  []byte(key),
		},
		BearerToken: token,
	}, nil
}

// toCredentialResp 将凭证模型转换为响应结构。
//
// 参数:
//   - row: 凭证模型
//
// 返回: 凭证响应
func toCredentialResp(row model.ClusterCredential) ClusterCredentialResp {
	return ClusterCredentialResp{
		ID:              row.ID,
		Name:            row.Name,
		RuntimeType:     row.RuntimeType,
		Source:          row.Source,
		ClusterID:       row.ClusterID,
		Endpoint:        row.Endpoint,
		AuthMethod:      row.AuthMethod,
		Status:          row.Status,
		LastTestAt:      row.LastTestAt,
		LastTestStatus:  row.LastTestStatus,
		LastTestMessage: row.LastTestMessage,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
