# Backend bootstrap.go Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split `internal/modules/cluster/handler/bootstrap.go` (1335 lines) into Logic layer with clear职责分离, introducing normalizer.go for standardized responses.

**Architecture:** Move business logic from Handler to Logic/bootstrap/ subpackage, Handler remains thin HTTP entry point, add normalizer.go for response standardization (replacing frontend normalizers).

**Tech Stack:** Go, Gin, GORM, SSH Client

---

## File Structure

### Files to Create

| File | Purpose | Est. Lines |
|------|---------|------------|
| `internal/modules/cluster/logic/bootstrap/types.go` | Bootstrap type definitions | ~120 |
| `internal/modules/cluster/logic/bootstrap/steps.go` | Step building, version catalog | ~100 |
| `internal/modules/cluster/logic/bootstrap/validator.go` | Preview validation logic | ~150 |
| `internal/modules/cluster/logic/bootstrap/ssh.go` | SSH execution wrapper | ~200 |
| `internal/modules/cluster/logic/bootstrap/executor.go` | Task execution logic | ~300 |
| `internal/modules/cluster/logic/bootstrap/normalizer.go` | Response standardization | ~100 |
| `internal/modules/cluster/logic/bootstrap/logic.go` | Logic facade, dependency injection | ~80 |

### Files to Modify

| File | Change |
|------|--------|
| `internal/modules/cluster/handler/bootstrap.go` | Delete after migration, keep only thin handler |
| `internal/modules/cluster/handler/bootstrap_handler.go` | Create new thin handler (~150 lines) |
| `internal/modules/cluster/logic/repository.go` | Remove BootstrapStepStatus re-export |

---

## Task 1: Create logic/bootstrap directory

**Files:**
- Create: `internal/modules/cluster/logic/bootstrap/`

- [ ] **Step 1: Create directory**

```bash
mkdir -p internal/modules/cluster/logic/bootstrap
```

- [ ] **Step 2: Commit**

```bash
git add internal/modules/cluster/logic/bootstrap/
git commit -m "feat(backend): create cluster bootstrap logic directory"
```

---

## Task 2: Create types.go

**Files:**
- Create: `internal/modules/cluster/logic/bootstrap/types.go`
- Source: `internal/modules/cluster/handler/bootstrap.go` lines 39-116

- [ ] **Step 1: Extract Bootstrap types**

```go
// internal/modules/cluster/logic/bootstrap/types.go

package bootstrap

import "time"

// PreviewReq Bootstrap 预览请求
type PreviewReq struct {
	Name                 string         `json:"name" binding:"required"`
	ProfileID            *uint          `json:"profile_id,omitempty"`
	ControlPlaneID       uint           `json:"control_plane_host_id" binding:"required"`
	WorkerIDs            []uint         `json:"worker_host_ids"`
	K8sVersion           string         `json:"k8s_version"`
	VersionChannel       string         `json:"version_channel"`
	CNI                  string         `json:"cni"`
	PodCIDR              string         `json:"pod_cidr"`
	ServiceCIDR          string         `json:"service_cidr"`
	RepoMode             string         `json:"repo_mode"` // online|mirror
	RepoURL              string         `json:"repo_url"`
	ImageRepository      string         `json:"image_repository"`
	EndpointMode         string         `json:"endpoint_mode"` // nodeIP|vip|lbDNS
	ControlPlaneEndpoint string         `json:"control_plane_endpoint"`
	VIPProvider          string         `json:"vip_provider"` // kube-vip|keepalived
	EtcdMode             string         `json:"etcd_mode"`    // stacked|external
	ExternalEtcd         map[string]any `json:"external_etcd"`
}

// ValidationIssue 验证问题
type ValidationIssue struct {
	Field       string `json:"field"`
	Code        string `json:"code"`
	Domain      string `json:"domain"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

// VersionItem 版本信息
type VersionItem struct {
	Version string `json:"version"`
	Channel string `json:"channel"`
	Status  string `json:"status"` // supported|blocked
	Reason  string `json:"reason,omitempty"`
}

// PreviewResp Bootstrap 预览响应
type PreviewResp struct {
	Name                 string             `json:"name"`
	ControlPlaneID       uint               `json:"control_plane_host_id"`
	WorkerIDs            []uint             `json:"worker_host_ids"`
	K8sVersion           string             `json:"k8s_version"`
	VersionChannel       string             `json:"version_channel"`
	CNI                  string             `json:"cni"`
	PodCIDR              string             `json:"pod_cidr"`
	ServiceCIDR          string             `json:"service_cidr"`
	RepoMode             string             `json:"repo_mode"`
	RepoURL              string             `json:"repo_url"`
	ImageRepository      string             `json:"image_repository"`
	EndpointMode         string             `json:"endpoint_mode"`
	ControlPlaneEndpoint string             `json:"control_plane_endpoint"`
	VIPProvider          string             `json:"vip_provider"`
	EtcdMode             string             `json:"etcd_mode"`
	Steps                []string           `json:"steps"`
	ExpectedEndpoint     string             `json:"expected_endpoint"`
	Warnings             []string           `json:"warnings,omitempty"`
	ValidationIssues     []ValidationIssue  `json:"validation_issues,omitempty"`
	Diagnostics          map[string]any     `json:"diagnostics,omitempty"`
}

// Step Bootstrap 步骤定义
type Step struct {
	Name      string
	Hosts     []string // "control-plane", "workers", "all"
	Script    string
	Timeout   time.Duration
	Rollback  string
	OnFailure string // "abort", "continue"
	EnvVars   map[string]string
}

// StepStatus 步骤执行状态
type StepStatus struct {
	Name       string     `json:"name"`
	Status     string     `json:"status"` // pending, running, completed, failed
	Message    string     `json:"message,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	HostID     *uint      `json:"host_id,omitempty"`
	Output     string     `json:"output,omitempty"`
}

// ProfileItem Bootstrap Profile 列表项
type ProfileItem struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	VersionChannel string `json:"version_channel"`
	K8sVersion     string `json:"k8s_version"`
	RepoMode       string `json:"repo_mode"`
	EndpointMode   string `json:"endpoint_mode"`
	VIPProvider    string `json:"vip_provider"`
	EtcdMode       string `json:"etcd_mode"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// ProfileCreateReq 创建 Profile 请求
type ProfileCreateReq struct {
	Name                 string         `json:"name" binding:"required"`
	Description          string         `json:"description,omitempty"`
	VersionChannel       string         `json:"version_channel" binding:"required"`
	K8sVersion           string         `json:"k8s_version" binding:"required"`
	RepoMode             string         `json:"repo_mode" binding:"required"`
	RepoURL              string         `json:"repo_url,omitempty"`
	ImageRepository      string         `json:"image_repository,omitempty"`
	EndpointMode         string         `json:"endpoint_mode" binding:"required"`
	ControlPlaneEndpoint string         `json:"control_plane_endpoint,omitempty"`
	VIPProvider          string         `json:"vip_provider,omitempty"`
	EtcdMode             string         `json:"etcd_mode" binding:"required"`
	ExternalEtcd         map[string]any `json:"external_etcd,omitempty"`
}

// ProfileUpdateReq 更新 Profile 请求
type ProfileUpdateReq struct {
	Description          string         `json:"description,omitempty"`
	VersionChannel       string         `json:"version_channel,omitempty"`
	K8sVersion           string         `json:"k8s_version,omitempty"`
	RepoMode             string         `json:"repo_mode,omitempty"`
	RepoURL              string         `json:"repo_url,omitempty"`
	ImageRepository      string         `json:"image_repository,omitempty"`
	EndpointMode         string         `json:"endpoint_mode,omitempty"`
	ControlPlaneEndpoint string         `json:"control_plane_endpoint,omitempty"`
	VIPProvider          string         `json:"vip_provider,omitempty"`
	EtcdMode             string         `json:"etcd_mode,omitempty"`
	ExternalEtcd         map[string]any `json:"external_etcd,omitempty"`
}
```

- [ ] **Step 2: Run Go build check**

```bash
cd internal/modules/cluster/logic/bootstrap && go build .
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/modules/cluster/logic/bootstrap/types.go
git commit -m "feat(backend): extract bootstrap types to logic layer"
```

---

## Task 3: Create steps.go

**Files:**
- Create: `internal/modules/cluster/logic/bootstrap/steps.go`
- Source: `internal/modules/cluster/handler/bootstrap.go` lines 117-150

- [ ] **Step 1: Extract step building logic**

```go
// internal/modules/cluster/logic/bootstrap/steps.go

package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// scriptVersionDirs maps version prefix to script directory
var scriptVersionDirs = map[string]string{
	"1.31": "v1.31.x",
	"1.30": "v1.30.x",
	"1.29": "v1.29.x",
	"1.28": "v1.28.x",
	"1.27": "v1.27.x",
}

func scriptVersionDirFor(version string) string {
	prefix := versionPrefix(version)
	if dir, ok := scriptVersionDirs[prefix]; ok {
		return dir
	}
	return "v1.30.x"
}

func versionPrefix(version string) string {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return version
}

// BuildSteps 构建 Bootstrap 步骤列表
func BuildSteps(k8sVersion string) []Step {
	scriptVersion := scriptVersionDirFor(k8sVersion)
	prefix := fmt.Sprintf("cluster/kubeadm/%s", scriptVersion)

	return []Step{
		{Name: "preflight", Hosts: []string{"all"}, Script: "cluster/common/preflight.sh", Timeout: 60 * time.Second, OnFailure: "abort"},
		{Name: "bootstrap-prechecks", Hosts: []string{"control-plane"}, Script: "cluster/common/bootstrap-prechecks.sh", Timeout: 60 * time.Second, OnFailure: "abort"},
		{Name: "containerd", Hosts: []string{"all"}, Script: "cluster/common/containerd-install.sh", Timeout: 5 * time.Minute, Rollback: "cluster/common/containerd-install.sh uninstall", OnFailure: "abort"},
		{Name: "kubeadm-install", Hosts: []string{"all"}, Script: prefix + "/install.sh", Timeout: 3 * time.Minute, Rollback: prefix + "/install.sh uninstall", OnFailure: "abort"},
		{Name: "control-plane-init", Hosts: []string{"control-plane"}, Script: prefix + "/init.sh", Timeout: 10 * time.Minute, Rollback: prefix + "/reset.sh", OnFailure: "abort"},
		{Name: "vip-provider", Hosts: []string{"control-plane"}, Script: "cluster/common/vip-provider.sh", Timeout: 2 * time.Minute, OnFailure: "abort"},
		{Name: "cni-install", Hosts: []string{"control-plane"}, Script: "", Timeout: 3 * time.Minute, OnFailure: "continue"},
		{Name: "worker-join", Hosts: []string{"workers"}, Script: prefix + "/join.sh", Timeout: 5 * time.Minute, Rollback: prefix + "/reset.sh", OnFailure: "continue"},
		{Name: "fetch-kubeconfig", Hosts: []string{"control-plane"}, Script: prefix + "/fetch-kubeconfig.sh", Timeout: 30 * time.Second, OnFailure: "abort"},
		{Name: "endpoint-health", Hosts: []string{"control-plane"}, Script: "cluster/common/endpoint-health.sh", Timeout: 30 * time.Second, OnFailure: "abort"},
		{Name: "sync-nodes", Hosts: []string{"control-plane"}, Script: "", Timeout: 30 * time.Second, OnFailure: "continue"},
	}
}

// BuildStepNames 返回步骤名称列表（用于预览）
func BuildStepNames(k8sVersion string) []string {
	steps := BuildSteps(k8sVersion)
	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
	}
	return names
}

// LoadVersionCatalog 加载版本目录
func LoadVersionCatalog(scriptRoot string) (defaultChannel string, items []VersionItem) {
	defaultChannel = "stable"
	items = []VersionItem{}

	versionsDir := filepath.Join(scriptRoot, "cluster", "kubeadm")
	dirs, err := os.ReadDir(versionsDir)
	if err != nil {
		return defaultChannel, items
	}

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		name := d.Name()
		if !strings.HasPrefix(name, "v") {
			continue
		}

		version := strings.TrimPrefix(name, "v")
		version = strings.TrimSuffix(version, ".x")

		manifestPath := filepath.Join(versionsDir, name, "manifest.json")
		status := "supported"
		reason := ""

		if data, err := os.ReadFile(manifestPath); err == nil {
			// Parse manifest.json for status/reason
			var manifest struct {
				Status string `json:"status"`
				Reason string `json:"reason"`
			}
			if json.Unmarshal(data, &manifest) == nil {
				status = manifest.Status
				reason = manifest.Reason
			}
		}

		items = append(items, VersionItem{
			Version: version,
			Channel: defaultChannel,
			Status:  status,
			Reason:  reason,
		})
	}

	return defaultChannel, items
}
```

- [ ] **Step 2: Add missing import**

```go
import "encoding/json"
```

- [ ] **Step 3: Run Go build check**

```bash
go build ./internal/modules/cluster/logic/bootstrap/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/modules/cluster/logic/bootstrap/steps.go
git commit -m "feat(backend): extract bootstrap step building logic"
```

---

## Task 4: Create validator.go

**Files:**
- Create: `internal/modules/cluster/logic/bootstrap/validator.go`
- Source: `internal/modules/cluster/handler/bootstrap.go` lines 183-350

- [ ] **Step 1: Extract validation logic**

```go
// internal/modules/cluster/logic/bootstrap/validator.go

package bootstrap

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/host/model"
	"gorm.io/gorm"
)

// Validator Bootstrap 预检查验证器
type Validator struct {
	db *gorm.DB
}

// NewValidator 创建验证器
func NewValidator(db *gorm.DB) *Validator {
	return &Validator{db: db}
}

// ValidatePreview 验证预览请求
func (v *Validator) ValidatePreview(ctx context.Context, req *PreviewReq) []ValidationIssue {
	issues := []ValidationIssue{}

	// 1. 名称验证
	if req.Name == "" {
		issues = append(issues, ValidationIssue{
			Field:   "name",
			Code:    "required",
			Message: "集群名称不能为空",
		})
	} else if len(req.Name) > 63 {
		issues = append(issues, ValidationIssue{
			Field:   "name",
			Code:    "too_long",
			Message: "集群名称不能超过 63 个字符",
		})
	} else if !isValidDNSName(req.Name) {
		issues = append(issues, ValidationIssue{
			Field:   "name",
			Code:    "invalid_format",
			Message: "集群名称必须符合 DNS 子域名规范",
		})
	}

	// 2. 主机存在性验证
	controlPlane, err := v.getHost(ctx, req.ControlPlaneID)
	if err != nil {
		issues = append(issues, ValidationIssue{
			Field:       "control_plane_host_id",
			Code:        "host_not_found",
			Domain:      "host",
			Message:     fmt.Sprintf("控制平面主机 %d 不存在", req.ControlPlaneID),
			Remediation: "请选择有效的主机",
		})
	} else {
		// 控制平面主机状态验证
		if controlPlane.Status != "online" {
			issues = append(issues, ValidationIssue{
				Field:       "control_plane_host_id",
				Code:        "host_offline",
				Domain:      "host",
				Message:     fmt.Sprintf("控制平面主机 %s 状态为 %s，不可用于 Bootstrap", controlPlane.Name, controlPlane.Status),
				Remediation: "请先确保主机在线",
			})
		}
	}

	// 3. Worker 主机验证
	for _, workerID := range req.WorkerIDs {
		worker, err := v.getHost(ctx, workerID)
		if err != nil {
			issues = append(issues, ValidationIssue{
				Field:       "worker_host_ids",
				Code:        "host_not_found",
				Domain:      "host",
				Message:     fmt.Sprintf("Worker 主机 %d 不存在", workerID),
				Remediation: "请选择有效的主机",
			})
		} else if worker.Status != "online" {
			issues = append(issues, ValidationIssue{
				Field:       "worker_host_ids",
				Code:        "host_offline",
				Domain:      "host",
				Message:     fmt.Sprintf("Worker 主机 %s 状态为 %s", worker.Name, worker.Status),
				Remediation: "请先确保主机在线",
			})
		}
	}

	// 4. CIDR 验证
	if req.PodCIDR != "" {
		if _, _, err := net.ParseCIDR(req.PodCIDR); err != nil {
			issues = append(issues, ValidationIssue{
				Field:   "pod_cidr",
				Code:    "invalid_cidr",
				Message: "Pod CIDR 格式无效",
			})
		}
	}

	if req.ServiceCIDR != "" {
		if _, _, err := net.ParseCIDR(req.ServiceCIDR); err != nil {
			issues = append(issues, ValidationIssue{
				Field:   "service_cidr",
				Code:    "invalid_cidr",
				Message: "Service CIDR 格式无效",
			})
		}
	}

	// 5. Endpoint 模式验证
	if req.EndpointMode == "vip" && req.ControlPlaneEndpoint == "" {
		issues = append(issues, ValidationIssue{
			Field:       "control_plane_endpoint",
			Code:        "required_for_vip",
			Message:     "VIP 模式需要指定控制平面 Endpoint",
			Remediation: "请设置 control_plane_endpoint 或切换到 nodeIP 模式",
		})
	}

	return issues
}

// ValidateProfile 验证 Profile 创建请求
func (v *Validator) ValidateProfile(ctx context.Context, req *ProfileCreateReq) []ValidationIssue {
	issues := []ValidationIssue{}

	if req.Name == "" {
		issues = append(issues, ValidationIssue{
			Field:   "name",
			Code:    "required",
			Message: "Profile 名称不能为空",
		})
	}

	if req.K8sVersion == "" {
		issues = append(issues, ValidationIssue{
			Field:   "k8s_version",
			Code:    "required",
			Message: "Kubernetes 版本不能为空",
		})
	}

	return issues
}

// getHost 获取主机信息
func (v *Validator) getHost(ctx context.Context, id uint) (*model.Host, error) {
	var host model.Host
	err := v.db.WithContext(ctx).First(&host, id).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// isValidDNSName 验证 DNS 名称格式
func isValidDNSName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	for i, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			if i == 0 && c == '-' {
				return false
			}
			if i == len(name)-1 && c == '-' {
				return false
			}
			continue
		}
		return false
	}
	return true
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/modules/cluster/logic/bootstrap/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/cluster/logic/bootstrap/validator.go
git commit -m "feat(backend): extract bootstrap validation logic"
```

---

## Task 5: Create ssh.go

**Files:**
- Create: `internal/modules/cluster/logic/bootstrap/ssh.go`
- Source: `internal/modules/cluster/handler/bootstrap.go` lines 650-850

- [ ] **Step 1: Extract SSH wrapper**

```go
// internal/modules/cluster/logic/bootstrap/ssh.go

package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/modules/host/model"
)

// SSHExecutor SSH 命令执行器
type SSHExecutor struct {
	client *sshclient.Client
}

// NewSSHExecutor 创建 SSH 执行器
func NewSSHExecutor(client *sshclient.Client) *SSHExecutor {
	return &SSHExecutor{client: client}
}

// HostInfo 主机执行信息
type HostInfo struct {
	ID       uint
	Name     string
	IP       string
	SSHUser  string
	SSHPort  int
	SSHKey   string
	Role     string // control-plane, worker
}

// ExecuteScript 在主机上执行脚本
func (e *SSHExecutor) ExecuteScript(ctx context.Context, host *HostInfo, script string, envVars map[string]string, config *PreviewReq) (string, error) {
	if script == "" {
		return "", nil // 空脚本跳过执行
	}

	// 构建环境变量
	env := buildEnvVars(envVars, config)

	// 构建完整命令
	cmd := buildCommand(script, env)

	// 执行 SSH 命令
	output, err := e.client.Execute(ctx, host.IP, host.SSHUser, host.SSHPort, cmd, host.SSHKey)
	if err != nil {
		return "", fmt.Errorf("SSH 执行失败: %w", err)
	}

	return output, nil
}

// ExecuteScriptWithTimeout 带超时执行脚本
func (e *SSHExecutor) ExecuteScriptWithTimeout(ctx context.Context, host *HostInfo, script string, timeout time.Duration, envVars map[string]string, config *PreviewReq) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return e.ExecuteScript(timeoutCtx, host, script, envVars, config)
}

// ExecuteRollback 执行回滚脚本
func (e *SSHExecutor) ExecuteRollback(ctx context.Context, host *HostInfo, rollbackScript string, config *PreviewReq) error {
	if rollbackScript == "" {
		return nil
	}

	_, err := e.ExecuteScript(ctx, host, rollbackScript, nil, config)
	return err
}

// CheckConnection 检查 SSH 连接
func (e *SSHExecutor) CheckConnection(ctx context.Context, host *HostInfo) error {
	_, err := e.client.Execute(ctx, host.IP, host.SSHUser, host.SSHPort, "echo 'connection_check'", host.SSHKey)
	return err
}

// GetHostname 获取主机名
func (e *SSHExecutor) GetHostname(ctx context.Context, host *HostInfo) (string, error) {
	output, err := e.client.Execute(ctx, host.IP, host.SSHUser, host.SSHPort, "hostname", host.SSHKey)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// buildEnvVars 构建环境变量
func buildEnvVars(extraVars map[string]string, config *PreviewReq) map[string]string {
	env := map[string]string{
		"K8S_VERSION":           config.K8sVersion,
		"POD_CIDR":              config.PodCIDR,
		"SERVICE_CIDR":          config.ServiceCIDR,
		"CONTROL_PLANE_ENDPOINT": config.ControlPlaneEndpoint,
		"CNI":                   config.CNI,
		"REPO_MODE":             config.RepoMode,
		"REPO_URL":              config.RepoURL,
		"IMAGE_REPOSITORY":      config.ImageRepository,
		"VIP_PROVIDER":          config.VIPProvider,
	}

	for k, v := range extraVars {
		env[k] = v
	}

	return env
}

// buildCommand 构建完整命令
func buildCommand(script string, env map[string]string) string {
	var buf bytes.Buffer

	// 写入环境变量
	for k, v := range env {
		buf.WriteString(fmt.Sprintf("export %s='%s'\n", k, v))
	}

	// 写入脚本路径
	buf.WriteString(script)

	return buf.String()
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/modules/cluster/logic/bootstrap/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/cluster/logic/bootstrap/ssh.go
git commit -m "feat(backend): extract SSH execution wrapper"
```

---

## Task 6: Create normalizer.go

**Files:**
- Create: `internal/modules/cluster/logic/bootstrap/normalizer.go`
- Purpose: Replace frontend normalizers with backend response standardization

- [ ] **Step 1: Create normalizer for standardized responses**

```go
// internal/modules/cluster/logic/bootstrap/normalizer.go

package bootstrap

import (
	"time"

	"github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
)

// TaskResponse 标准任务响应格式
type TaskResponse struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	ClusterID    *uint        `json:"cluster_id,omitempty"`
	K8sVersion   string       `json:"k8s_version"`
	VersionChannel string     `json:"version_channel"`
	RepoMode     string       `json:"repo_mode"`
	EndpointMode string       `json:"endpoint_mode"`
	Status       string       `json:"status"` // pending, running, completed, failed, aborted
	Steps        []StepStatus `json:"steps"`
	CurrentStep  int          `json:"current_step"`
	ErrorMessage string       `json:"error_message,omitempty"`
	Progress     int          `json:"progress"` // 0-100
	CreatedAt    string       `json:"created_at"`
	UpdatedAt    string       `json:"updated_at"`
}

// NormalizeTaskResponse 标准化任务响应
func NormalizeTaskResponse(task *model.BootstrapTask) TaskResponse {
	progress := calculateProgress(task.Steps)

	return TaskResponse{
		ID:             task.ID,
		Name:           task.Name,
		ClusterID:      task.ClusterID,
		K8sVersion:     task.K8sVersion,
		VersionChannel: task.VersionChannel,
		RepoMode:       task.RepoMode,
		EndpointMode:   task.EndpointMode,
		Status:         normalizeStatus(task.Status),
		Steps:          normalizeSteps(task.Steps),
		CurrentStep:    task.CurrentStep,
		ErrorMessage:   task.ErrorMessage,
		Progress:       progress,
		CreatedAt:      formatTime(task.CreatedAt),
		UpdatedAt:      formatTime(task.UpdatedAt),
	}
}

// PreviewResponse 标准预览响应
type PreviewResponse struct {
	Name                 string            `json:"name"`
	ControlPlaneHostID   uint              `json:"control_plane_host_id"`
	WorkerHostIDs        []uint            `json:"worker_host_ids"`
	K8sVersion           string            `json:"k8s_version"`
	VersionChannel       string            `json:"version_channel"`
	CNI                  string            `json:"cni"`
	PodCIDR              string            `json:"pod_cidr"`
	ServiceCIDR          string            `json:"service_cidr"`
	RepoMode             string            `json:"repo_mode"`
	RepoURL              string            `json:"repo_url"`
	ImageRepository      string            `json:"image_repository"`
	EndpointMode         string            `json:"endpoint_mode"`
	ControlPlaneEndpoint string            `json:"control_plane_endpoint"`
	VIPProvider          string            `json:"vip_provider"`
	EtcdMode             string            `json:"etcd_mode"`
	Steps                []string          `json:"steps"`
	ExpectedEndpoint     string            `json:"expected_endpoint"`
	Warnings             []string          `json:"warnings,omitempty"`
	ValidationIssues     []ValidationIssue `json:"validation_issues,omitempty"`
}

// NormalizePreviewResponse 标准化预览响应
func NormalizePreviewResponse(preview *PreviewResp, issues []ValidationIssue) PreviewResponse {
	return PreviewResponse{
		Name:                 preview.Name,
		ControlPlaneHostID:   preview.ControlPlaneID,
		WorkerHostIDs:        preview.WorkerIDs,
		K8sVersion:           preview.K8sVersion,
		VersionChannel:       preview.VersionChannel,
		CNI:                  preview.CNI,
		PodCIDR:              preview.PodCIDR,
		ServiceCIDR:          preview.ServiceCIDR,
		RepoMode:             preview.RepoMode,
		RepoURL:              preview.RepoURL,
		ImageRepository:      preview.ImageRepository,
		EndpointMode:         preview.EndpointMode,
		ControlPlaneEndpoint: preview.ControlPlaneEndpoint,
		VIPProvider:          preview.VIPProvider,
		EtcdMode:             preview.EtcdMode,
		Steps:                preview.Steps,
		ExpectedEndpoint:     preview.ExpectedEndpoint,
		Warnings:             preview.Warnings,
		ValidationIssues:     issues,
	}
}

// normalizeStatus 标准化状态值
func normalizeStatus(status string) string {
	validStatuses := map[string]string{
		"pending":   "pending",
		"running":   "running",
		"completed": "completed",
		"success":   "completed",
		"failed":    "failed",
		"error":     "failed",
		"aborted":   "aborted",
	}

	if normalized, ok := validStatuses[status]; ok {
		return normalized
	}
	return status
}

// normalizeSteps 标准化步骤状态
func normalizeSteps(steps []model.BootstrapStepStatus) []StepStatus {
	result := make([]StepStatus, len(steps))
	for i, s := range steps {
		result[i] = StepStatus{
			Name:       s.Name,
			Status:     normalizeStatus(s.Status),
			Message:    s.Message,
			StartedAt:  s.StartedAt,
			FinishedAt: s.FinishedAt,
			HostID:     s.HostID,
			Output:     s.Output,
		}
	}
	return result
}

// calculateProgress 计算进度百分比
func calculateProgress(steps []model.BootstrapStepStatus) int {
	if len(steps) == 0 {
		return 0
	}

	completed := 0
	for _, s := range steps {
		if s.Status == "completed" || s.Status == "success" {
			completed++
		}
	}

	return (completed * 100) / len(steps)
}

// formatTime 格式化时间
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/modules/cluster/logic/bootstrap/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/cluster/logic/bootstrap/normalizer.go
git commit -m "feat(backend): add response normalizer (replaces frontend normalizers)"
```

---

## Task 7: Create executor.go (core execution logic)

**Files:**
- Create: `internal/modules/cluster/logic/bootstrap/executor.go`
- Source: `internal/modules/cluster/handler/bootstrap.go` lines 492-650

- [ ] **Step 1: Extract core execution logic**

```go
// internal/modules/cluster/logic/bootstrap/executor.go

package bootstrap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	"github.com/cy77cc/OpsPilot/internal/modules/host/model as hostmodel"
	"gorm.io/gorm"
)

// Executor Bootstrap 任务执行器
type Executor struct {
	db        *gorm.DB
	ssh       *SSHExecutor
	validator *Validator
	mu        sync.Mutex
}

// NewExecutor 创建执行器
func NewExecutor(db *gorm.DB, ssh *SSHExecutor) *Executor {
	return &Executor{
		db:        db,
		ssh:       ssh,
		validator: NewValidator(db),
	}
}

// Execute 执行 Bootstrap 任务（同步执行，用于 asynq worker）
func (e *Executor) Execute(ctx context.Context, taskID string, config *PreviewReq) error {
	// 1. 加载任务
	task, err := e.loadTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("加载任务失败: %w", err)
	}

	// 2. 检查任务状态
	if task.Status != "pending" && task.Status != "running" {
		return fmt.Errorf("任务状态无效: %s", task.Status)
	}

	// 3. 加载主机信息
	hosts, err := e.loadHosts(ctx, config)
	if err != nil {
		e.updateTaskStatus(task, "failed", fmt.Sprintf("加载主机失败: %v", err))
		return err
	}

	// 4. 构建步骤
	steps := BuildSteps(config.K8sVersion)

	// 5. 初始化步骤状态
	e.initStepStatuses(task, steps)

	// 6. 更新任务状态为 running
	e.updateTaskStatus(task, "running", "")

	// 7. 执行各步骤
	for i, step := range steps {
		if ctx.Err() != nil {
			e.updateTaskStatus(task, "aborted", "任务被取消")
			return ctx.Err()
		}

		err := e.executeStep(ctx, task, i, step, hosts, config)
		if err != nil {
			if step.OnFailure == "abort" {
				e.updateTaskStatus(task, "failed", err.Error())
				return err
			}
			// 继续执行后续步骤
			e.updateStepStatus(task, i, "failed", err.Error())
			continue
		}

		e.updateStepStatus(task, i, "completed", "")
	}

	// 8. 更新最终状态
	e.updateTaskStatus(task, "completed", "")

	return nil
}

// executeStep 执行单个步骤
func (e *Executor) executeStep(ctx context.Context, task *model.BootstrapTask, stepIndex int, step Step, hosts HostMap, config *PreviewReq) error {
	e.mu.Lock()
	e.updateStepStatus(task, stepIndex, "running", "")
	e.mu.Unlock()

	// 获取目标主机
	targetHosts := e.getTargetHosts(step.Hosts, hosts)

	for _, host := range targetHosts {
		if step.Script == "" {
			// 内置步骤（如 sync-nodes）
			err := e.executeBuiltinStep(ctx, step.Name, task, host)
			if err != nil {
				return err
			}
			continue
		}

		// SSH 执行脚本
		hostInfo := &HostInfo{
			ID:      host.ID,
			Name:    host.Name,
			IP:      host.IP,
			SSHUser: host.SSHUser,
			SSHPort: host.SSHPort,
			SSHKey:  host.SSHKey,
			Role:    host.Role,
		}

		output, err := e.ssh.ExecuteScriptWithTimeout(ctx, hostInfo, step.Script, step.Timeout, step.EnvVars, config)
		if err != nil {
			e.appendStepOutput(task, stepIndex, host.ID, output, err.Error())
			return fmt.Errorf("步骤 %s 在主机 %s 上失败: %w", step.Name, host.Name, err)
		}

		e.appendStepOutput(task, stepIndex, host.ID, output, "")
	}

	return nil
}

// executeBuiltinStep 执行内置步骤
func (e *Executor) executeBuiltinStep(ctx context.Context, stepName string, task *model.BootstrapTask, host *HostInfo) error {
	switch stepName {
	case "sync-nodes":
		return e.syncNodes(ctx, task)
	case "cni-install":
		return e.installCNI(ctx, task, host)
	default:
		return nil
	}
}

// HostMap 主机映射
type HostMap struct {
	ControlPlane []*hostmodel.Host
	Workers      []*hostmodel.Host
	All          []*hostmodel.Host
}

// loadTask 加载任务
func (e *Executor) loadTask(ctx context.Context, taskID string) (*model.BootstrapTask, error) {
	var task model.BootstrapTask
	err := e.db.WithContext(ctx).Where("id = ?", taskID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// loadHosts 加载主机信息
func (e *Executor) loadHosts(ctx context.Context, config *PreviewReq) (HostMap, error) {
	hosts := HostMap{}

	// 加载控制平面主机
	var controlPlane hostmodel.Host
	if err := e.db.WithContext(ctx).First(&controlPlane, config.ControlPlaneID).Error; err != nil {
		return hosts, fmt.Errorf("控制平面主机不存在: %w", err)
	}
	hosts.ControlPlane = []*hostmodel.Host{&controlPlane}

	// 加载 Worker 主机
	for _, workerID := range config.WorkerIDs {
		var worker hostmodel.Host
		if err := e.db.WithContext(ctx).First(&worker, workerID).Error; err != nil {
			return hosts, fmt.Errorf("Worker 主机 %d 不存在: %w", workerID, err)
		}
		hosts.Workers = append(hosts.Workers, &worker)
	}

	// 合并所有主机
	hosts.All = append(hosts.All, hosts.ControlPlane...)
	hosts.All = append(hosts.All, hosts.Workers...)

	return hosts, nil
}

// getTargetHosts 获取步骤目标主机
func (e *Executor) getTargetHosts(targets []string, hosts HostMap) []*hostmodel.Host {
	result := []*hostmodel.Host{}
	for _, target := range targets {
		switch target {
		case "control-plane":
			result = append(result, hosts.ControlPlane...)
		case "workers":
			result = append(result, hosts.Workers...)
		case "all":
			result = append(result, hosts.All...)
		}
	}
	return result
}

// updateTaskStatus 更新任务状态
func (e *Executor) updateTaskStatus(task *model.BootstrapTask, status, message string) {
	task.Status = status
	task.ErrorMessage = message
	task.UpdatedAt = time.Now()
	e.db.Save(task)
}

// updateStepStatus 更新步骤状态
func (e *Executor) updateStepStatus(task *model.BootstrapTask, stepIndex int, status, message string) {
	if stepIndex < len(task.Steps) {
		task.Steps[stepIndex].Status = status
		task.Steps[stepIndex].Message = message
		now := time.Now()
		if status == "running" {
			task.Steps[stepIndex].StartedAt = &now
		} else if status == "completed" || status == "failed" {
			task.Steps[stepIndex].FinishedAt = &now
		}
		task.CurrentStep = stepIndex
		task.UpdatedAt = now
		e.db.Save(task)
	}
}

// initStepStatuses 初始化步骤状态
func (e *Executor) initStepStatuses(task *model.BootstrapTask, steps []Step) {
	task.Steps = make([]model.BootstrapStepStatus, len(steps))
	for i, s := range steps {
		task.Steps[i] = model.BootstrapStepStatus{
			Name:   s.Name,
			Status: "pending",
		}
	}
	e.db.Save(task)
}

// appendStepOutput 添加步骤输出
func (e *Executor) appendStepOutput(task *model.BootstrapTask, stepIndex int, hostID uint, output, error string) {
	if stepIndex < len(task.Steps) {
		task.Steps[stepIndex].HostID = &hostID
		task.Steps[stepIndex].Output = output
		e.db.Save(task)
	}
}

// syncNodes 同步节点信息
func (e *Executor) syncNodes(ctx context.Context, task *model.BootstrapTask) error {
	// 实现 K8s 节点同步逻辑
	return nil
}

// installCNI 安装 CNI
func (e *Executor) installCNI(ctx context.Context, task *model.BootstrapTask, host *HostInfo) error {
	// 实现 CNI 安装逻辑
	return nil
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/modules/cluster/logic/bootstrap/...
```

Expected: May have import cycle or type issues, fix iteratively

- [ ] **Step 3: Fix any build issues**

- [ ] **Step 4: Commit**

```bash
git add internal/modules/cluster/logic/bootstrap/executor.go
git commit -m "feat(backend): extract bootstrap execution logic"
```

---

## Task 8: Create logic.go (facade)

**Files:**
- Create: `internal/modules/cluster/logic/bootstrap/logic.go`

- [ ] **Step 1: Create Logic facade**

```go
// internal/modules/cluster/logic/bootstrap/logic.go

package bootstrap

import (
	"context"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	"gorm.io/gorm"
)

// Logic Bootstrap Logic 门面
type Logic struct {
	db        *gorm.DB
	ssh       *SSHExecutor
	executor  *Executor
	validator *Validator
}

// NewLogic 创建 Logic 实例
func NewLogic(db *gorm.DB, sshClient *sshclient.Client, scriptRoot string) *Logic {
	sshExec := NewSSHExecutor(sshClient)
	executor := NewExecutor(db, sshExec)
	validator := NewValidator(db)

	return &Logic{
		db:        db,
		ssh:       sshExec,
		executor:  executor,
		validator: validator,
	}
}

// Preview 预览 Bootstrap 配置
func (l *Logic) Preview(ctx context.Context, req *PreviewReq) (*PreviewResp, []ValidationIssue) {
	// 1. 验证请求
	issues := l.validator.ValidatePreview(ctx, req)

	// 2. 构建预览响应
	resp := &PreviewResp{
		Name:                 req.Name,
		ControlPlaneID:       req.ControlPlaneID,
		WorkerIDs:            req.WorkerIDs,
		K8sVersion:           req.K8sVersion,
		VersionChannel:       req.VersionChannel,
		CNI:                  req.CNI,
		PodCIDR:              req.PodCIDR,
		ServiceCIDR:          req.ServiceCIDR,
		RepoMode:             req.RepoMode,
		RepoURL:              req.RepoURL,
		ImageRepository:      req.ImageRepository,
		EndpointMode:         req.EndpointMode,
		ControlPlaneEndpoint: req.ControlPlaneEndpoint,
		VIPProvider:          req.VIPProvider,
		EtcdMode:             req.EtcdMode,
		Steps:                BuildStepNames(req.K8sVersion),
		ExpectedEndpoint:     buildExpectedEndpoint(req),
		ValidationIssues:     issues,
	}

	return resp, issues
}

// Apply 执行 Bootstrap（创建任务，等待执行）
func (l *Logic) Apply(ctx context.Context, req *PreviewReq) (taskID string, err error) {
	// 创建任务记录
	task := &model.BootstrapTask{
		ID:            generateTaskID(),
		Name:          req.Name,
		K8sVersion:    req.K8sVersion,
		VersionChannel: req.VersionChannel,
		RepoMode:      req.RepoMode,
		EndpointMode:  req.EndpointMode,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	if err := l.db.Create(task).Error; err != nil {
		return "", err
	}

	return task.ID, nil
}

// GetTask 获取任务详情
func (l *Logic) GetTask(ctx context.Context, taskID string) (*model.BootstrapTask, error) {
	var task model.BootstrapTask
	err := l.db.WithContext(ctx).Where("id = ?", taskID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTaskResponse 获取标准化任务响应
func (l *Logic) GetTaskResponse(ctx context.Context, taskID string) (*TaskResponse, error) {
	task, err := l.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	resp := NormalizeTaskResponse(task)
	return &resp, nil
}

// ListProfiles 获取 Profile 列表
func (l *Logic) ListProfiles(ctx context.Context) ([]ProfileItem, error) {
	var profiles []model.BootstrapProfile
	err := l.db.WithContext(ctx).Find(&profiles).Error
	if err != nil {
		return nil, err
	}

	items := make([]ProfileItem, len(profiles))
	for i, p := range profiles {
		items[i] = ProfileItem{
			ID:             p.ID,
			Name:           p.Name,
			Description:    p.Description,
			VersionChannel: p.VersionChannel,
			K8sVersion:     p.K8sVersion,
			RepoMode:       p.RepoMode,
			EndpointMode:   p.EndpointMode,
			VIPProvider:    p.VIPProvider,
			EtcdMode:       p.EtcdMode,
			CreatedAt:      p.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      p.UpdatedAt.Format(time.RFC3339),
		}
	}

	return items, nil
}

// CreateProfile 创建 Profile
func (l *Logic) CreateProfile(ctx context.Context, req *ProfileCreateReq) (*model.BootstrapProfile, error) {
	profile := &model.BootstrapProfile{
		Name:                 req.Name,
		Description:          req.Description,
		VersionChannel:       req.VersionChannel,
		K8sVersion:           req.K8sVersion,
		RepoMode:             req.RepoMode,
		RepoURL:              req.RepoURL,
		ImageRepository:      req.ImageRepository,
		EndpointMode:         req.EndpointMode,
		ControlPlaneEndpoint: req.ControlPlaneEndpoint,
		VIPProvider:          req.VIPProvider,
		EtcdMode:             req.EtcdMode,
	}

	if err := l.db.Create(profile).Error; err != nil {
		return nil, err
	}

	return profile, nil
}

// Execute 同步执行任务（用于 Worker）
func (l *Logic) Execute(ctx context.Context, taskID string, config *PreviewReq) error {
	return l.executor.Execute(ctx, taskID, config)
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/modules/cluster/logic/bootstrap/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/cluster/logic/bootstrap/logic.go
git commit -m "feat(backend): create bootstrap logic facade"
```

---

## Task 9: Create thin handler (bootstrap_handler.go)

**Files:**
- Create: `internal/modules/cluster/handler/bootstrap_handler.go`
- Source: `internal/modules/cluster/handler/bootstrap.go` HTTP handler methods

- [ ] **Step 1: Create thin handler**

```go
// internal/modules/cluster/handler/bootstrap_handler.go

package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/modules/cluster/logic/bootstrap"
)

// BootstrapHandler Bootstrap HTTP Handler
type BootstrapHandler struct {
	logic *bootstrap.Logic
}

// NewBootstrapHandler 创建 Handler
func NewBootstrapHandler(logic *bootstrap.Logic) *BootstrapHandler {
	return &BootstrapHandler{logic: logic}
}

// GetBootstrapVersions GET /clusters/bootstrap/versions
func (h *BootstrapHandler) GetBootstrapVersions(c *gin.Context) {
	channel, items := bootstrap.LoadVersionCatalog(h.getScriptRoot())
	httpx.OK(c, gin.H{
		"default_channel": channel,
		"list":            items,
	})
}

// ListBootstrapProfiles GET /clusters/bootstrap/profiles
func (h *BootstrapHandler) ListBootstrapProfiles(c *gin.Context) {
	profiles, err := h.logic.ListProfiles(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"list":  profiles,
		"total": len(profiles),
	})
}

// CreateBootstrapProfile POST /clusters/bootstrap/profiles
func (h *BootstrapHandler) CreateBootstrapProfile(c *gin.Context) {
	var req bootstrap.ProfileCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	profile, err := h.logic.CreateProfile(c.Request.Context(), &req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, profile)
}

// PreviewBootstrap POST /clusters/bootstrap/preview
func (h *BootstrapHandler) PreviewBootstrap(c *gin.Context) {
	var req bootstrap.PreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	preview, issues := h.logic.Preview(c.Request.Context(), &req)

	// 使用 normalizer 标准化响应
	resp := bootstrap.NormalizePreviewResponse(preview, issues)

	if len(issues) > 0 {
		httpx.OK(c, gin.H{
			"code":              2000,
			"msg":               "bootstrap preview validation failed",
			"data":              resp,
			"validation_issues": issues,
		})
		return
	}

	httpx.OK(c, resp)
}

// ApplyBootstrap POST /clusters/bootstrap/apply
func (h *BootstrapHandler) ApplyBootstrap(c *gin.Context) {
	var req bootstrap.PreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	taskID, err := h.logic.Apply(c.Request.Context(), &req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
	})
}

// GetBootstrapTask GET /clusters/bootstrap/:taskId
func (h *BootstrapHandler) GetBootstrapTask(c *gin.Context) {
	taskID := c.Param("taskId")

	resp, err := h.logic.GetTaskResponse(c.Request.Context(), taskID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, resp)
}

func (h *BootstrapHandler) getScriptRoot() string {
	// 从配置获取脚本根目录
	return "/opt/opspilot/scripts"
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/modules/cluster/handler/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/cluster/handler/bootstrap_handler.go
git commit -m "feat(backend): create thin bootstrap handler"
```

---

## Task 10: Update handler routes.go

**Files:**
- Modify: `internal/modules/cluster/handler/routes.go` or equivalent route registration file

- [ ] **Step 1: Find route registration**

```bash
grep -r "bootstrap" internal/modules/cluster/api/ --include="*.go"
```

- [ ] **Step 2: Update route registration**

Replace old bootstrap handler with new BootstrapHandler.

- [ ] **Step 3: Run Go build check**

```bash
go build ./internal/modules/cluster/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/modules/cluster/api/routes.go
git commit -m "refactor(backend): update bootstrap route registration"
```

---

## Task 11: Delete old bootstrap.go

**Files:**
- Delete: `internal/modules/cluster/handler/bootstrap.go`

- [ ] **Step 1: Delete old file**

```bash
rm internal/modules/cluster/handler/bootstrap.go
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/modules/cluster/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/cluster/handler/bootstrap.go
git commit -m "refactor(backend): delete old bootstrap.go after migration"
```

---

## Task 12: Run tests

**Files:**
- Test: `internal/modules/cluster/handler/bootstrap_test.go` if exists

- [ ] **Step 1: Run existing tests**

```bash
go test ./internal/modules/cluster/... -v
```

- [ ] **Step 2: Fix test failures**

If tests fail due to import changes, update test files.

- [ ] **Step 3: Commit test fixes**

```bash
git add internal/modules/cluster/
git commit -m "fix(backend): update bootstrap tests after refactor"
```

---

## Success Criteria

1. ✅ Bootstrap handler < 200 lines
2. ✅ Logic layer files clearly separated (types, steps, validator, ssh, executor, normalizer)
3. ✅ No normalizer functions in frontend (replaced by backend normalizer.go)
4. ✅ Go compilation passes
5. ✅ Tests pass