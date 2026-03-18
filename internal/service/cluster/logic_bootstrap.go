package cluster

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/gin-gonic/gin"
)

// BootstrapStep defines a step in the bootstrap workflow
type BootstrapStep struct {
	Name      string
	Hosts     []string // "control-plane", "workers", "all"
	Script    string
	Timeout   time.Duration
	Rollback  string
	OnFailure string // "abort", "continue"
	EnvVars   map[string]string
}

// BootstrapPreviewReq represents a bootstrap preview request
type BootstrapPreviewReq struct {
	Name                 string                 `json:"name" binding:"required"`
	ProfileID            *uint                  `json:"profile_id,omitempty"`
	ControlPlaneID       uint                   `json:"control_plane_host_id" binding:"required"`
	WorkerIDs            []uint                 `json:"worker_host_ids"`
	K8sVersion           string                 `json:"k8s_version"`
	VersionChannel       string                 `json:"version_channel"`
	CNI                  string                 `json:"cni"`
	PodCIDR              string                 `json:"pod_cidr"`
	ServiceCIDR          string                 `json:"service_cidr"`
	RepoMode             string                 `json:"repo_mode"` // online|mirror
	RepoURL              string                 `json:"repo_url"`
	ImageRepository      string                 `json:"image_repository"`
	EndpointMode         string                 `json:"endpoint_mode"` // nodeIP|vip|lbDNS
	ControlPlaneEndpoint string                 `json:"control_plane_endpoint"`
	VIPProvider          string                 `json:"vip_provider"` // kube-vip|keepalived
	EtcdMode             string                 `json:"etcd_mode"`    // stacked|external
	ExternalEtcd         map[string]interface{} `json:"external_etcd"`
}

type BootstrapValidationIssue struct {
	Field       string `json:"field"`
	Code        string `json:"code"`
	Domain      string `json:"domain"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

type BootstrapVersionItem struct {
	Version string `json:"version"`
	Channel string `json:"channel"`
	Status  string `json:"status"` // supported|blocked
	Reason  string `json:"reason,omitempty"`
}

// BootstrapPreviewResp represents a bootstrap preview response
type BootstrapPreviewResp struct {
	Name                 string                     `json:"name"`
	ControlPlaneID       uint                       `json:"control_plane_host_id"`
	WorkerIDs            []uint                     `json:"worker_host_ids"`
	K8sVersion           string                     `json:"k8s_version"`
	VersionChannel       string                     `json:"version_channel"`
	CNI                  string                     `json:"cni"`
	PodCIDR              string                     `json:"pod_cidr"`
	ServiceCIDR          string                     `json:"service_cidr"`
	RepoMode             string                     `json:"repo_mode"`
	RepoURL              string                     `json:"repo_url"`
	ImageRepository      string                     `json:"image_repository"`
	EndpointMode         string                     `json:"endpoint_mode"`
	ControlPlaneEndpoint string                     `json:"control_plane_endpoint"`
	VIPProvider          string                     `json:"vip_provider"`
	EtcdMode             string                     `json:"etcd_mode"`
	Steps                []string                   `json:"steps"`
	ExpectedEndpoint     string                     `json:"expected_endpoint"`
	Warnings             []string                   `json:"warnings,omitempty"`
	ValidationIssues     []BootstrapValidationIssue `json:"validation_issues,omitempty"`
	Diagnostics          map[string]interface{}     `json:"diagnostics,omitempty"`
}

// BootstrapApplyResp represents a bootstrap apply response
type BootstrapApplyResp struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	ClusterID *uint  `json:"cluster_id,omitempty"`
}

func buildBootstrapSteps(k8sVersion string) []BootstrapStep {
	scriptVersion := scriptVersionDirFor(k8sVersion)
	prefix := fmt.Sprintf("cluster/kubeadm/%s", scriptVersion)
	return []BootstrapStep{
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

// PreviewBootstrap previews a bootstrap operation
func (h *Handler) GetBootstrapVersions(c *gin.Context) {
	channel, items := loadBootstrapVersionCatalog()
	httpx.OK(c, gin.H{
		"default_channel": channel,
		"list":            items,
	})
}

func (h *Handler) ListBootstrapProfiles(c *gin.Context) {
	items := make([]BootstrapProfileItem, 0)
	raw, _, err := h.svcCtx.CacheFacade.GetOrLoad(c.Request.Context(), CacheKeyBootstrapProfiles(), ClusterPhase1CachePolicies["clusters.bootstrap_profiles"].TTL, func(ctx context.Context) (string, error) {
		rows, rerr := h.repo.ListBootstrapProfiles(ctx)
		if rerr != nil {
			return "", rerr
		}
		buf, merr := json.Marshal(rows)
		if merr != nil {
			return "", merr
		}
		return string(buf), nil
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{
		"list":  items,
		"total": len(items),
	})
}

func (h *Handler) CreateBootstrapProfile(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}
	var req BootstrapProfileCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	cfg, issues := resolveAndValidateBootstrapProfile(req)
	if len(issues) > 0 {
		c.JSON(200, gin.H{
			"code": 2000,
			"msg":  "bootstrap profile validation failed",
			"data": gin.H{"validation_issues": issues},
		})
		return
	}

	row := model.ClusterBootstrapProfile{
		Name:                 strings.TrimSpace(req.Name),
		Description:          strings.TrimSpace(req.Description),
		VersionChannel:       cfg.VersionChannel,
		K8sVersion:           cfg.K8sVersion,
		RepoMode:             cfg.RepoMode,
		RepoURL:              cfg.RepoURL,
		ImageRepository:      cfg.ImageRepository,
		EndpointMode:         cfg.EndpointMode,
		ControlPlaneEndpoint: cfg.ControlPlaneEndpoint,
		VIPProvider:          cfg.VIPProvider,
		EtcdMode:             cfg.EtcdMode,
		ExternalEtcdJSON:     toJSON(cfg.ExternalEtcd),
		CreatedBy:            httpx.UIDFromCtx(c),
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Create(&row).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	h.svcCtx.CacheFacade.Delete(c.Request.Context(), CacheKeyBootstrapProfiles())
	httpx.OK(c, toBootstrapProfileItem(row))
}

func (h *Handler) UpdateBootstrapProfile(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid profile id"))
		return
	}
	var req BootstrapProfileUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	var row model.ClusterBootstrapProfile
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).First(&row, id).Error; err != nil {
		httpx.NotFound(c, "bootstrap profile not found")
		return
	}
	cfg, issues := resolveAndValidateBootstrapProfile(BootstrapProfileCreateReq{
		Name:                 row.Name,
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
		ExternalEtcd:         req.ExternalEtcd,
	})
	if len(issues) > 0 {
		c.JSON(200, gin.H{
			"code": 2000,
			"msg":  "bootstrap profile validation failed",
			"data": gin.H{"validation_issues": issues},
		})
		return
	}

	updates := map[string]interface{}{
		"description":            strings.TrimSpace(req.Description),
		"version_channel":        cfg.VersionChannel,
		"k8s_version":            cfg.K8sVersion,
		"repo_mode":              cfg.RepoMode,
		"repo_url":               cfg.RepoURL,
		"image_repository":       cfg.ImageRepository,
		"endpoint_mode":          cfg.EndpointMode,
		"control_plane_endpoint": cfg.ControlPlaneEndpoint,
		"vip_provider":           cfg.VIPProvider,
		"etcd_mode":              cfg.EtcdMode,
		"external_etcd_json":     toJSON(cfg.ExternalEtcd),
		"updated_at":             time.Now().UTC(),
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Model(&row).
		Updates(updates).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).First(&row, id).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	h.svcCtx.CacheFacade.Delete(c.Request.Context(), CacheKeyBootstrapProfiles())
	httpx.OK(c, toBootstrapProfileItem(row))
}

func (h *Handler) DeleteBootstrapProfile(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid profile id"))
		return
	}
	res := h.svcCtx.DB.WithContext(c.Request.Context()).Delete(&model.ClusterBootstrapProfile{}, id)
	if res.Error != nil {
		httpx.ServerErr(c, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		httpx.NotFound(c, "bootstrap profile not found")
		return
	}
	h.svcCtx.CacheFacade.Delete(c.Request.Context(), CacheKeyBootstrapProfiles())
	httpx.OK(c, gin.H{"id": id, "deleted": true})
}

// PreviewBootstrap previews a bootstrap operation
func (h *Handler) PreviewBootstrap(c *gin.Context) {
	var req BootstrapPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	// Load hosts to validate
	control, workers, err := h.loadBootstrapHosts(c.Request.Context(), req.ControlPlaneID, req.WorkerIDs)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	cfg, issues, warnings, diagnostics := h.resolveAndValidateBootstrapReq(c.Request.Context(), req, control.IP)
	if len(issues) > 0 {
		c.JSON(200, gin.H{
			"code": 2000,
			"msg":  "bootstrap validation failed",
			"data": gin.H{
				"validation_issues": issues,
				"warnings":          warnings,
				"diagnostics":       diagnostics,
			},
		})
		return
	}

	// Build steps description
	steps := []string{
		fmt.Sprintf("1. 预检查所有节点 (%d 节点)", 1+len(workers)),
		"2. 安装 containerd 容器运行时",
		fmt.Sprintf("3. 安装 kubeadm/kubelet/kubectl v%s", cfg.K8sVersion),
		fmt.Sprintf("4. 初始化控制平面节点: %s (%s)", control.Name, control.IP),
		fmt.Sprintf("5. 安装 CNI 插件: %s", cfg.CNI),
		fmt.Sprintf("6. 加入 Worker 节点 (%d 节点)", len(workers)),
		"7. 获取 kubeconfig 并存储",
		"8. 同步节点信息到数据库",
	}

	resp := BootstrapPreviewResp{
		Name:                 cfg.Name,
		ControlPlaneID:       req.ControlPlaneID,
		WorkerIDs:            req.WorkerIDs,
		K8sVersion:           cfg.K8sVersion,
		VersionChannel:       cfg.VersionChannel,
		CNI:                  cfg.CNI,
		PodCIDR:              cfg.PodCIDR,
		ServiceCIDR:          cfg.ServiceCIDR,
		RepoMode:             cfg.RepoMode,
		RepoURL:              cfg.RepoURL,
		ImageRepository:      cfg.ImageRepository,
		EndpointMode:         cfg.EndpointMode,
		ControlPlaneEndpoint: cfg.ControlPlaneEndpoint,
		VIPProvider:          cfg.VIPProvider,
		EtcdMode:             cfg.EtcdMode,
		Steps:                steps,
		ExpectedEndpoint:     fmt.Sprintf("https://%s", cfg.ControlPlaneEndpoint),
		Warnings:             warnings,
		ValidationIssues:     issues,
		Diagnostics:          diagnostics,
	}

	httpx.OK(c, resp)
}

// ApplyBootstrap applies a bootstrap operation
func (h *Handler) ApplyBootstrap(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	var req BootstrapPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	uid := httpx.UIDFromCtx(c)
	control, _, err := h.loadBootstrapHosts(c.Request.Context(), req.ControlPlaneID, req.WorkerIDs)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	cfg, issues, warnings, diagnostics := h.resolveAndValidateBootstrapReq(c.Request.Context(), req, control.IP)
	if len(issues) > 0 {
		c.JSON(200, gin.H{
			"code": 2000,
			"msg":  "bootstrap validation failed",
			"data": gin.H{
				"validation_issues": issues,
				"warnings":          warnings,
				"diagnostics":       diagnostics,
			},
		})
		return
	}

	// Create task record
	task := &model.ClusterBootstrapTask{
		ID:                   fmt.Sprintf("boot-%d", time.Now().UnixNano()),
		Name:                 cfg.Name,
		ControlPlaneID:       req.ControlPlaneID,
		WorkerIDsJSON:        toJSON(req.WorkerIDs),
		K8sVersion:           cfg.K8sVersion,
		VersionChannel:       cfg.VersionChannel,
		RepoMode:             cfg.RepoMode,
		RepoURL:              cfg.RepoURL,
		ImageRepository:      cfg.ImageRepository,
		EndpointMode:         cfg.EndpointMode,
		ControlPlaneEndpoint: cfg.ControlPlaneEndpoint,
		VIPProvider:          cfg.VIPProvider,
		EtcdMode:             cfg.EtcdMode,
		ExternalEtcdJSON:     toJSON(cfg.ExternalEtcd),
		CNI:                  cfg.CNI,
		PodCIDR:              cfg.PodCIDR,
		ServiceCIDR:          cfg.ServiceCIDR,
		ResolvedConfigJSON:   toJSON(cfg),
		DiagnosticsJSON:      toJSON(diagnostics),
		Status:               "queued",
		CreatedBy:            uid,
	}

	if err := h.svcCtx.DB.Create(task).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}

	// Execute bootstrap asynchronously
	go h.executeBootstrap(runtimectx.Detach(c.Request.Context()), task)

	httpx.OK(c, BootstrapApplyResp{
		TaskID: task.ID,
		Status: task.Status,
	})
}

// GetBootstrapTask returns bootstrap task status
func (h *Handler) GetBootstrapTask(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		httpx.BindErr(c, nil)
		return
	}

	var task model.ClusterBootstrapTask
	if err := h.svcCtx.DB.Where("id = ?", taskID).First(&task).Error; err != nil {
		httpx.NotFound(c, "task not found")
		return
	}

	// Parse steps JSON
	var steps []BootstrapStepStatus
	if task.StepsJSON != "" {
		json.Unmarshal([]byte(task.StepsJSON), &steps)
	}

	detail := BootstrapTaskDetail{
		ID:                   task.ID,
		Name:                 task.Name,
		ClusterID:            task.ClusterID,
		K8sVersion:           task.K8sVersion,
		VersionChannel:       task.VersionChannel,
		RepoMode:             task.RepoMode,
		RepoURL:              task.RepoURL,
		ImageRepository:      task.ImageRepository,
		EndpointMode:         task.EndpointMode,
		ControlPlaneEndpoint: task.ControlPlaneEndpoint,
		VIPProvider:          task.VIPProvider,
		EtcdMode:             task.EtcdMode,
		CNI:                  task.CNI,
		PodCIDR:              task.PodCIDR,
		ServiceCIDR:          task.ServiceCIDR,
		Status:               task.Status,
		Steps:                steps,
		ErrorMessage:         task.ErrorMessage,
		ResolvedConfigJSON:   task.ResolvedConfigJSON,
		DiagnosticsJSON:      task.DiagnosticsJSON,
		CreatedAt:            task.CreatedAt,
		UpdatedAt:            task.UpdatedAt,
	}

	httpx.OK(c, detail)
}

// executeBootstrap executes the bootstrap workflow
func (h *Handler) executeBootstrap(ctx context.Context, task *model.ClusterBootstrapTask) {
	// Update status to running
	task.Status = "running"
	h.svcCtx.DB.Save(task)

	// Load hosts
	control, workers, err := h.loadBootstrapHosts(ctx, task.ControlPlaneID, parseWorkerIDs(task.WorkerIDsJSON))
	if err != nil {
		h.failTask(ctx, task, err.Error())
		return
	}

	bootstrapSteps := buildBootstrapSteps(task.K8sVersion)
	// Initialize steps tracking
	steps := make([]BootstrapStepStatus, len(bootstrapSteps))
	stepsJSON, _ := json.Marshal(steps)
	task.StepsJSON = string(stepsJSON)
	h.svcCtx.DB.Save(task)

	controlAdvertiseAddress := control.IP
	configYAML, genErr := buildKubeadmInitConfigYAML(task, controlAdvertiseAddress)
	if genErr != nil {
		h.failTask(ctx, task, fmt.Sprintf("failed to generate kubeadm config: %v", genErr))
		return
	}
	// Prepare environment variables
	envVars := map[string]string{
		"K8S_VERSION":            task.K8sVersion,
		"KUBERNETES_VERSION":     task.K8sVersion,
		"POD_CIDR":               task.PodCIDR,
		"SERVICE_CIDR":           task.ServiceCIDR,
		"CNI":                    task.CNI,
		"VERSION_CHANNEL":        task.VersionChannel,
		"REPO_MODE":              task.RepoMode,
		"REPO_URL":               task.RepoURL,
		"IMAGE_REPOSITORY":       task.ImageRepository,
		"ENDPOINT_MODE":          task.EndpointMode,
		"CONTROL_PLANE_ENDPOINT": task.ControlPlaneEndpoint,
		"VIP_PROVIDER":           task.VIPProvider,
		"ETCD_MODE":              task.EtcdMode,
		"EXTERNAL_ETCD_JSON":     task.ExternalEtcdJSON,
		"KUBEADM_CONFIG_B64":     base64.StdEncoding.EncodeToString([]byte(configYAML)),
		"BOOTSTRAP_INIT_MODE":    "config",
		"ADVERTISE_ADDRESS":      controlAdvertiseAddress,
	}

	// Execute each step
	var lastErr error
	for i, step := range bootstrapSteps {
		steps[i] = BootstrapStepStatus{
			Name:   step.Name,
			Status: "running",
		}
		now := time.Now().UTC()
		steps[i].StartedAt = &now
		h.updateSteps(ctx, task, steps)

		// Determine target hosts
		var targetHosts []*model.Node
		for _, h := range step.Hosts {
			switch h {
			case "control-plane":
				targetHosts = append(targetHosts, control)
			case "workers":
				targetHosts = append(targetHosts, workers...)
			case "all":
				targetHosts = append(targetHosts, control)
				targetHosts = append(targetHosts, workers...)
			}
		}

		// Execute step
		if step.Script != "" {
			err := h.executeStepOnHosts(ctx, step, targetHosts, envVars)
			if err != nil {
				lastErr = err
				steps[i].Status = "failed"
				steps[i].Message = err.Error()
				finished := time.Now().UTC()
				steps[i].FinishedAt = &finished
				h.updateSteps(ctx, task, steps)

				// Execute rollback if defined
				if step.Rollback != "" {
					rollbackStep := BootstrapStep{
						Name:    step.Name + "-rollback",
						Script:  step.Rollback,
						Timeout: step.Timeout,
					}
					h.executeStepOnHosts(ctx, rollbackStep, targetHosts, envVars)
				}

				if step.OnFailure == "abort" {
					break
				}
			} else {
				steps[i].Status = "succeeded"
				finished := time.Now().UTC()
				steps[i].FinishedAt = &finished
				h.updateSteps(ctx, task, steps)
			}
		} else {
			// Special handling for CNI and sync steps
			if step.Name == "cni-install" {
				err := h.installCNI(ctx, control, task.CNI, task.PodCIDR)
				if err != nil {
					lastErr = err
					steps[i].Status = "failed"
					steps[i].Message = err.Error()
				} else {
					steps[i].Status = "succeeded"
				}
				finished := time.Now().UTC()
				steps[i].FinishedAt = &finished
				h.updateSteps(ctx, task, steps)
			} else if step.Name == "sync-nodes" {
				// This will be handled after we get kubeconfig
				steps[i].Status = "succeeded"
				finished := time.Now().UTC()
				steps[i].FinishedAt = &finished
				h.updateSteps(ctx, task, steps)
			}
		}
	}

	// Check if all critical steps succeeded
	if lastErr != nil {
		h.failTask(ctx, task, lastErr.Error())
		return
	}

	// Create cluster record
	cluster := &model.Cluster{
		Name:        task.Name,
		Source:      "platform_managed",
		Type:        "kubernetes",
		K8sVersion:  task.K8sVersion,
		Endpoint:    fmt.Sprintf("https://%s", defaultIfEmpty(task.ControlPlaneEndpoint, fmt.Sprintf("%s:6443", control.IP))),
		PodCIDR:     task.PodCIDR,
		ServiceCIDR: task.ServiceCIDR,
		Status:      "active",
		AuthMethod:  "kubeconfig",
		CreatedBy:   fmt.Sprintf("%d", task.CreatedBy),
	}

	if err := h.svcCtx.DB.Create(cluster).Error; err != nil {
		h.failTask(ctx, task, fmt.Sprintf("failed to create cluster: %v", err))
		return
	}

	task.ClusterID = &cluster.ID
	task.Status = "succeeded"
	task.ResultJSON = toJSON(map[string]interface{}{
		"cluster_id": cluster.ID,
		"endpoint":   cluster.Endpoint,
	})
	h.svcCtx.DB.Save(task)
}

// loadBootstrapHosts loads control plane and worker hosts
func (h *Handler) loadBootstrapHosts(ctx context.Context, controlID uint, workerIDs []uint) (*model.Node, []*model.Node, error) {
	var control model.Node
	if err := h.svcCtx.DB.WithContext(ctx).First(&control, controlID).Error; err != nil {
		return nil, nil, fmt.Errorf("control plane host not found: %w", err)
	}
	if strings.TrimSpace(control.IP) == "" {
		return nil, nil, fmt.Errorf("control plane host missing IP")
	}

	workers := make([]*model.Node, 0, len(workerIDs))
	for _, id := range workerIDs {
		if id == 0 || id == controlID {
			continue
		}
		var row model.Node
		if err := h.svcCtx.DB.WithContext(ctx).First(&row, id).Error; err != nil {
			return nil, nil, fmt.Errorf("worker host %d not found", id)
		}
		workers = append(workers, &row)
	}

	return &control, workers, nil
}

// executeStepOnHosts executes a script step on target hosts
func (h *Handler) executeStepOnHosts(ctx context.Context, step BootstrapStep, hosts []*model.Node, envVars map[string]string) error {
	for _, host := range hosts {
		privateKey, passphrase, err := h.loadNodePrivateKey(ctx, host)
		if err != nil {
			return fmt.Errorf("failed to load SSH key for host %s: %w", host.Name, err)
		}

		password := strings.TrimSpace(host.SSHPassword)
		if strings.TrimSpace(privateKey) != "" {
			password = ""
		}

		cli, err := sshclient.NewSSHClient(host.SSHUser, password, host.IP, host.Port, privateKey, passphrase)
		if err != nil {
			return fmt.Errorf("SSH connection failed to %s: %w", host.Name, err)
		}

		// Build command with environment variables
		scriptPath := filepath.Join("script", step.Script)
		cmd := fmt.Sprintf("bash %s", scriptPath)

		// Add environment variables
		for k, v := range envVars {
			cmd = fmt.Sprintf("%s=%s %s", k, v, cmd)
		}
		for k, v := range step.EnvVars {
			cmd = fmt.Sprintf("%s=%s %s", k, v, cmd)
		}

		_, err = sshclient.RunCommand(cli, cmd)
		cli.Close()

		if err != nil {
			return fmt.Errorf("step %s failed on host %s: %w", step.Name, host.Name, err)
		}
	}

	return nil
}

// installCNI installs the CNI plugin
func (h *Handler) installCNI(ctx context.Context, control *model.Node, cni, podCIDR string) error {
	var scriptPath string
	switch strings.ToLower(cni) {
	case "calico":
		scriptPath = "script/cluster/cni/calico/v3.26/install.sh"
	case "flannel":
		scriptPath = "script/cluster/cni/flannel/v0.22/install.sh"
	case "cilium":
		scriptPath = "script/cluster/cni/cilium/v1.14/install.sh"
	default:
		return fmt.Errorf("unsupported CNI: %s", cni)
	}

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("CNI script not found: %s", scriptPath)
	}

	// Execute via SSH
	privateKey, passphrase, err := h.loadNodePrivateKey(ctx, control)
	if err != nil {
		return err
	}

	password := strings.TrimSpace(control.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}

	cli, err := sshclient.NewSSHClient(control.SSHUser, password, control.IP, control.Port, privateKey, passphrase)
	if err != nil {
		return err
	}
	defer cli.Close()

	cmd := fmt.Sprintf("POD_CIDR=%s bash %s", podCIDR, scriptPath)
	_, err = sshclient.RunCommand(cli, cmd)
	return err
}

// loadNodePrivateKey loads the SSH private key for a node
func (h *Handler) loadNodePrivateKey(ctx context.Context, node *model.Node) (string, string, error) {
	if node.SSHKeyID == nil || *node.SSHKeyID == 0 {
		return "", "", nil
	}

	var key model.SSHKey
	if err := h.svcCtx.DB.WithContext(ctx).First(&key, *node.SSHKeyID).Error; err != nil {
		return "", "", err
	}

	enc := strings.TrimSpace(config.CFG.Security.EncryptionKey)
	if enc != "" && key.Encrypted {
		decrypted, err := utils.DecryptText(key.PrivateKey, enc)
		if err != nil {
			return "", "", err
		}
		return decrypted, key.Passphrase, nil
	}

	return key.PrivateKey, key.Passphrase, nil
}

// failTask marks a task as failed
func (h *Handler) failTask(ctx context.Context, task *model.ClusterBootstrapTask, message string) {
	task.Status = "failed"
	task.ErrorMessage = message
	h.svcCtx.DB.WithContext(ctx).Save(task)
}

// updateSteps updates the steps JSON in the task
func (h *Handler) updateSteps(ctx context.Context, task *model.ClusterBootstrapTask, steps []BootstrapStepStatus) {
	stepsJSON, _ := json.Marshal(steps)
	task.StepsJSON = string(stepsJSON)
	h.svcCtx.DB.WithContext(ctx).Save(task)
}

// Helper functions
type resolvedBootstrapConfig struct {
	Name                 string                 `json:"name"`
	ProfileID            *uint                  `json:"profile_id,omitempty"`
	K8sVersion           string                 `json:"k8s_version"`
	VersionChannel       string                 `json:"version_channel"`
	CNI                  string                 `json:"cni"`
	PodCIDR              string                 `json:"pod_cidr"`
	ServiceCIDR          string                 `json:"service_cidr"`
	RepoMode             string                 `json:"repo_mode"`
	RepoURL              string                 `json:"repo_url"`
	ImageRepository      string                 `json:"image_repository"`
	EndpointMode         string                 `json:"endpoint_mode"`
	ControlPlaneEndpoint string                 `json:"control_plane_endpoint"`
	VIPProvider          string                 `json:"vip_provider"`
	EtcdMode             string                 `json:"etcd_mode"`
	ExternalEtcd         map[string]interface{} `json:"external_etcd,omitempty"`
}

func (h *Handler) resolveAndValidateBootstrapReq(ctx context.Context, req BootstrapPreviewReq, controlPlaneIP string) (resolvedBootstrapConfig, []BootstrapValidationIssue, []string, map[string]interface{}) {
	profile := model.ClusterBootstrapProfile{}
	profileFound := false
	if req.ProfileID != nil && *req.ProfileID > 0 {
		if err := h.svcCtx.DB.WithContext(ctx).First(&profile, *req.ProfileID).Error; err != nil {
			issues := []BootstrapValidationIssue{{
				Field:       "profile_id",
				Code:        "not_found",
				Domain:      "profile",
				Message:     fmt.Sprintf("bootstrap profile %d not found", *req.ProfileID),
				Remediation: "Select an existing profile or clear profile selection",
			}}
			return resolvedBootstrapConfig{}, issues, nil, map[string]interface{}{"profile": "not_found"}
		}
		profileFound = true
	}

	profileExternal := map[string]interface{}{}
	if profile.ExternalEtcdJSON != "" {
		_ = json.Unmarshal([]byte(profile.ExternalEtcdJSON), &profileExternal)
	}

	defaultVersionChannel, _ := loadBootstrapVersionCatalog()
	cfg := resolvedBootstrapConfig{
		Name:                 strings.TrimSpace(req.Name),
		ProfileID:            req.ProfileID,
		K8sVersion:           firstNonEmpty(strings.TrimSpace(req.K8sVersion), profile.K8sVersion, "1.28.0"),
		VersionChannel:       firstNonEmpty(strings.TrimSpace(req.VersionChannel), profile.VersionChannel, defaultVersionChannel),
		CNI:                  defaultIfEmpty(req.CNI, "calico"),
		PodCIDR:              defaultIfEmpty(req.PodCIDR, "10.244.0.0/16"),
		ServiceCIDR:          defaultIfEmpty(req.ServiceCIDR, "10.96.0.0/12"),
		RepoMode:             firstNonEmpty(strings.TrimSpace(req.RepoMode), profile.RepoMode, "online"),
		RepoURL:              firstNonEmpty(strings.TrimSpace(req.RepoURL), strings.TrimSpace(profile.RepoURL)),
		ImageRepository:      firstNonEmpty(strings.TrimSpace(req.ImageRepository), strings.TrimSpace(profile.ImageRepository)),
		EndpointMode:         firstNonEmpty(strings.TrimSpace(req.EndpointMode), profile.EndpointMode, "nodeIP"),
		ControlPlaneEndpoint: firstNonEmpty(strings.TrimSpace(req.ControlPlaneEndpoint), strings.TrimSpace(profile.ControlPlaneEndpoint)),
		VIPProvider:          firstNonEmpty(strings.TrimSpace(req.VIPProvider), strings.TrimSpace(profile.VIPProvider)),
		EtcdMode:             firstNonEmpty(strings.TrimSpace(req.EtcdMode), profile.EtcdMode, "stacked"),
		ExternalEtcd:         firstMap(req.ExternalEtcd, profileExternal),
	}
	if cfg.EndpointMode == "nodeIP" || strings.TrimSpace(cfg.ControlPlaneEndpoint) == "" {
		cfg.ControlPlaneEndpoint = fmt.Sprintf("%s:6443", controlPlaneIP)
	}

	issues := make([]BootstrapValidationIssue, 0)
	warnings := make([]string, 0)
	addIssue := func(field, code, domain, msg, remediation string) {
		issues = append(issues, BootstrapValidationIssue{
			Field:       field,
			Code:        code,
			Domain:      domain,
			Message:     msg,
			Remediation: remediation,
		})
	}
	diagnostics := map[string]interface{}{
		"profile_used": profileFound,
		"domains":      map[string]interface{}{},
	}
	setDomainResult := func(domain string, ok bool, detail string) {
		diagnostics["domains"].(map[string]interface{})[domain] = map[string]interface{}{
			"ok":     ok,
			"detail": detail,
		}
	}

	if cfg.Name == "" {
		addIssue("name", "required", "request", "name is required", "Provide a non-empty cluster name")
	}
	if cfg.RepoMode != "online" && cfg.RepoMode != "mirror" {
		addIssue("repo_mode", "invalid", "repo", "repo_mode must be online or mirror", "Use repo_mode=online or repo_mode=mirror")
	}
	if cfg.RepoMode == "mirror" && cfg.RepoURL == "" {
		addIssue("repo_url", "required", "repo", "repo_url is required when repo_mode=mirror", "Set repo_url to internal apt/yum mirror")
	}
	if cfg.EndpointMode != "nodeIP" && cfg.EndpointMode != "vip" && cfg.EndpointMode != "lbDNS" {
		addIssue("endpoint_mode", "invalid", "endpoint", "endpoint_mode must be nodeIP, vip, or lbDNS", "Use one of nodeIP/vip/lbDNS")
	}
	if cfg.EndpointMode != "nodeIP" && cfg.ControlPlaneEndpoint == "" {
		addIssue("control_plane_endpoint", "required", "endpoint", "control_plane_endpoint is required when endpoint_mode is vip or lbDNS", "Set VIP/LB address like 10.0.0.10:6443")
	}
	if cfg.EndpointMode == "vip" {
		if cfg.VIPProvider == "" {
			cfg.VIPProvider = "kube-vip"
		}
		if cfg.VIPProvider != "kube-vip" && cfg.VIPProvider != "keepalived" {
			addIssue("vip_provider", "invalid", "vip", "vip_provider must be kube-vip or keepalived", "Choose kube-vip or keepalived")
		}
	}
	if cfg.EtcdMode != "stacked" && cfg.EtcdMode != "external" {
		addIssue("etcd_mode", "invalid", "etcd", "etcd_mode must be stacked or external", "Use etcd_mode=stacked or etcd_mode=external")
	}
	if cfg.EtcdMode == "external" {
		endpoints, _ := cfg.ExternalEtcd["endpoints"].([]interface{})
		ca, _ := cfg.ExternalEtcd["ca_cert"].(string)
		cert, _ := cfg.ExternalEtcd["cert"].(string)
		key, _ := cfg.ExternalEtcd["key"].(string)
		if len(endpoints) == 0 || strings.TrimSpace(ca) == "" || strings.TrimSpace(cert) == "" || strings.TrimSpace(key) == "" {
			addIssue("external_etcd", "required", "etcd", "external_etcd endpoints/ca_cert/cert/key are required when etcd_mode=external", "Provide endpoints and PEM cert materials")
		}
	}

	_, catalog := loadBootstrapVersionCatalog()
	if !isBootstrapVersionSupported(cfg.K8sVersion, catalog) {
		alternatives := suggestSupportedVersions(catalog, 3)
		addIssue("k8s_version", "blocked_version", "version", fmt.Sprintf("k8s_version %s is blocked for bootstrap", cfg.K8sVersion), fmt.Sprintf("Use supported versions: %s", strings.Join(alternatives, ", ")))
	}
	if cfg.RepoMode == "online" && cfg.ImageRepository == "" {
		warnings = append(warnings, "image_repository not set; default registry behavior will be used")
	}
	diagIssues, diagWarnings := performBootstrapPreflightDiagnostics(cfg)
	warnings = append(warnings, diagWarnings...)
	for _, di := range diagIssues {
		issues = append(issues, di)
	}
	if cfg.RepoMode == "mirror" {
		setDomainResult("repo", len(filterIssuesByDomain(issues, "repo")) == 0, "mirror repository checks completed")
	}
	setDomainResult("endpoint", len(filterIssuesByDomain(issues, "endpoint")) == 0, "endpoint preflight checks completed")
	setDomainResult("etcd", len(filterIssuesByDomain(issues, "etcd")) == 0, "external etcd checks completed")
	setDomainResult("version", len(filterIssuesByDomain(issues, "version")) == 0, "version matrix checks completed")

	return cfg, issues, warnings, diagnostics
}

func isBootstrapVersionSupported(version string, items []BootstrapVersionItem) bool {
	for _, it := range items {
		if strings.TrimSpace(it.Version) == strings.TrimSpace(version) && it.Status == "supported" {
			return true
		}
	}
	return false
}

func loadBootstrapVersionCatalog() (string, []BootstrapVersionItem) {
	channels := []string{"stable-1", "stable"}
	seen := map[string]struct{}{}
	items := make([]BootstrapVersionItem, 0, 4)
	for _, ch := range channels {
		v, err := fetchK8sReleaseChannel(ch)
		if err != nil || v == "" {
			continue
		}
		v = strings.TrimPrefix(v, "v")
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		items = append(items, BootstrapVersionItem{Version: v, Channel: ch, Status: "blocked", Reason: "not in local supported matrix"})
	}

	supported := getSupportedK8sVersionsFromScripts()
	for _, s := range supported {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		items = append(items, BootstrapVersionItem{Version: s, Channel: "local-supported", Status: "supported"})
	}
	for i := range items {
		for _, s := range supported {
			if items[i].Version == s {
				items[i].Status = "supported"
				items[i].Reason = ""
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Version > items[j].Version })
	return "stable-1", items
}

func fetchK8sReleaseChannel(channel string) (string, error) {
	url := fmt.Sprintf("https://dl.k8s.io/release/%s.txt", channel)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func getSupportedK8sVersionsFromScripts() []string {
	entries, err := os.ReadDir(filepath.Join("script", "cluster", "kubeadm"))
	if err != nil {
		return []string{"1.28.0"}
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "v1.") {
			continue
		}
		trimmed := strings.TrimPrefix(name, "v")
		parts := strings.Split(trimmed, ".")
		if len(parts) < 2 {
			continue
		}
		major, err1 := strconv.Atoi(parts[0])
		minor, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || major != 1 {
			continue
		}
		out = append(out, fmt.Sprintf("%d.%d.0", major, minor))
	}
	if len(out) == 0 {
		return []string{"1.28.0"}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] > out[j] })
	return out
}

func scriptVersionDirFor(k8sVersion string) string {
	parts := strings.Split(strings.TrimSpace(k8sVersion), ".")
	if len(parts) >= 2 {
		return fmt.Sprintf("v%s.%s", parts[0], parts[1])
	}
	return "v1.28"
}

func toBootstrapProfileItem(row model.ClusterBootstrapProfile) BootstrapProfileItem {
	var external interface{}
	if strings.TrimSpace(row.ExternalEtcdJSON) != "" {
		tmp := map[string]interface{}{}
		if err := json.Unmarshal([]byte(row.ExternalEtcdJSON), &tmp); err == nil {
			external = tmp
		}
	}
	return BootstrapProfileItem{
		ID:                   row.ID,
		Name:                 row.Name,
		Description:          row.Description,
		VersionChannel:       row.VersionChannel,
		K8sVersion:           row.K8sVersion,
		RepoMode:             row.RepoMode,
		RepoURL:              row.RepoURL,
		ImageRepository:      row.ImageRepository,
		EndpointMode:         row.EndpointMode,
		ControlPlaneEndpoint: row.ControlPlaneEndpoint,
		VIPProvider:          row.VIPProvider,
		EtcdMode:             row.EtcdMode,
		ExternalEtcd:         external,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func resolveAndValidateBootstrapProfile(req BootstrapProfileCreateReq) (resolvedBootstrapConfig, []BootstrapValidationIssue) {
	cfg := resolvedBootstrapConfig{
		Name:                 strings.TrimSpace(req.Name),
		K8sVersion:           strings.TrimSpace(req.K8sVersion),
		VersionChannel:       firstNonEmpty(strings.TrimSpace(req.VersionChannel), "stable-1"),
		RepoMode:             firstNonEmpty(strings.TrimSpace(req.RepoMode), "online"),
		RepoURL:              strings.TrimSpace(req.RepoURL),
		ImageRepository:      strings.TrimSpace(req.ImageRepository),
		EndpointMode:         firstNonEmpty(strings.TrimSpace(req.EndpointMode), "nodeIP"),
		ControlPlaneEndpoint: strings.TrimSpace(req.ControlPlaneEndpoint),
		VIPProvider:          strings.TrimSpace(req.VIPProvider),
		EtcdMode:             firstNonEmpty(strings.TrimSpace(req.EtcdMode), "stacked"),
		ExternalEtcd:         map[string]interface{}{},
	}
	if req.ExternalEtcd != nil {
		cfg.ExternalEtcd["endpoints"] = req.ExternalEtcd.Endpoints
		cfg.ExternalEtcd["ca_cert"] = req.ExternalEtcd.CACert
		cfg.ExternalEtcd["cert"] = req.ExternalEtcd.Cert
		cfg.ExternalEtcd["key"] = req.ExternalEtcd.Key
	}

	issues := make([]BootstrapValidationIssue, 0)
	addIssue := func(field, code, domain, msg, remediation string) {
		issues = append(issues, BootstrapValidationIssue{
			Field:       field,
			Code:        code,
			Domain:      domain,
			Message:     msg,
			Remediation: remediation,
		})
	}

	if cfg.Name == "" {
		addIssue("name", "required", "profile", "name is required", "Set a unique profile name")
	}
	if cfg.RepoMode != "online" && cfg.RepoMode != "mirror" {
		addIssue("repo_mode", "invalid", "repo", "repo_mode must be online or mirror", "Use online or mirror")
	}
	if cfg.RepoMode == "mirror" && cfg.RepoURL == "" {
		addIssue("repo_url", "required", "repo", "repo_url is required when repo_mode=mirror", "Set mirror repo URL")
	}
	if cfg.EndpointMode != "nodeIP" && cfg.EndpointMode != "vip" && cfg.EndpointMode != "lbDNS" {
		addIssue("endpoint_mode", "invalid", "endpoint", "endpoint_mode must be nodeIP/vip/lbDNS", "Use one of nodeIP/vip/lbDNS")
	}
	if cfg.EndpointMode != "nodeIP" && cfg.ControlPlaneEndpoint == "" {
		addIssue("control_plane_endpoint", "required", "endpoint", "control_plane_endpoint is required for vip/lbDNS", "Set endpoint like 10.0.0.10:6443")
	}
	if cfg.EndpointMode == "vip" && cfg.VIPProvider != "" && cfg.VIPProvider != "kube-vip" && cfg.VIPProvider != "keepalived" {
		addIssue("vip_provider", "invalid", "vip", "vip_provider must be kube-vip or keepalived", "Choose kube-vip or keepalived")
	}
	if cfg.EtcdMode != "stacked" && cfg.EtcdMode != "external" {
		addIssue("etcd_mode", "invalid", "etcd", "etcd_mode must be stacked or external", "Use stacked or external")
	}
	if cfg.EtcdMode == "external" {
		endpoints, _ := cfg.ExternalEtcd["endpoints"].([]string)
		ca, _ := cfg.ExternalEtcd["ca_cert"].(string)
		cert, _ := cfg.ExternalEtcd["cert"].(string)
		key, _ := cfg.ExternalEtcd["key"].(string)
		if len(endpoints) == 0 || strings.TrimSpace(ca) == "" || strings.TrimSpace(cert) == "" || strings.TrimSpace(key) == "" {
			addIssue("external_etcd", "required", "etcd", "external_etcd endpoints/ca_cert/cert/key are required", "Provide endpoint list and PEM cert materials")
		}
	}
	return cfg, issues
}

func buildKubeadmInitConfigYAML(task *model.ClusterBootstrapTask, advertiseAddress string) (string, error) {
	clusterName := strings.TrimSpace(task.Name)
	if clusterName == "" {
		return "", fmt.Errorf("task name is empty")
	}
	networking := fmt.Sprintf("  podSubnet: %s\n  serviceSubnet: %s", task.PodCIDR, task.ServiceCIDR)
	imageRepoLine := ""
	if strings.TrimSpace(task.ImageRepository) != "" {
		imageRepoLine = fmt.Sprintf("imageRepository: %s\n", task.ImageRepository)
	}
	controlEndpointLine := ""
	if strings.TrimSpace(task.ControlPlaneEndpoint) != "" {
		controlEndpointLine = fmt.Sprintf("controlPlaneEndpoint: %s\n", task.ControlPlaneEndpoint)
	}

	etcdBlock := "etcd:\n  local: {}"
	if strings.EqualFold(task.EtcdMode, "external") && strings.TrimSpace(task.ExternalEtcdJSON) != "" {
		external := map[string]interface{}{}
		if err := json.Unmarshal([]byte(task.ExternalEtcdJSON), &external); err == nil {
			endpoints := make([]string, 0)
			if raw, ok := external["endpoints"].([]interface{}); ok {
				for _, ep := range raw {
					if s, ok := ep.(string); ok && strings.TrimSpace(s) != "" {
						endpoints = append(endpoints, strings.TrimSpace(s))
					}
				}
			}
			ca, _ := external["ca_cert"].(string)
			cert, _ := external["cert"].(string)
			key, _ := external["key"].(string)
			if len(endpoints) > 0 {
				quoted := make([]string, 0, len(endpoints))
				for _, ep := range endpoints {
					quoted = append(quoted, fmt.Sprintf("\"%s\"", ep))
				}
				etcdBlock = fmt.Sprintf("etcd:\n  external:\n    endpoints: [%s]\n    caFile: /etc/kubernetes/pki/external-etcd/ca.crt\n    certFile: /etc/kubernetes/pki/external-etcd/client.crt\n    keyFile: /etc/kubernetes/pki/external-etcd/client.key", strings.Join(quoted, ", "))
				if strings.TrimSpace(ca) == "" || strings.TrimSpace(cert) == "" || strings.TrimSpace(key) == "" {
					return "", fmt.Errorf("external etcd cert material missing")
				}
			}
		}
	}

	cfg := fmt.Sprintf(`apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
nodeRegistration:
  criSocket: unix:///run/containerd/containerd.sock
localAPIEndpoint:
  advertiseAddress: %s
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v%s
clusterName: %s
%s%snetworking:
%s
%s
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
cgroupDriver: systemd
`, advertiseAddress, task.K8sVersion, clusterName, controlEndpointLine, imageRepoLine, networking, etcdBlock)
	return cfg, nil
}

func performBootstrapPreflightDiagnostics(cfg resolvedBootstrapConfig) ([]BootstrapValidationIssue, []string) {
	issues := make([]BootstrapValidationIssue, 0)
	warnings := make([]string, 0)
	addIssue := func(field, code, domain, msg, remediation string) {
		issues = append(issues, BootstrapValidationIssue{
			Field:       field,
			Code:        code,
			Domain:      domain,
			Message:     msg,
			Remediation: remediation,
		})
	}

	if cfg.RepoMode == "mirror" && strings.TrimSpace(cfg.RepoURL) != "" {
		client := &http.Client{Timeout: 2 * time.Second}
		req, _ := http.NewRequest(http.MethodHead, cfg.RepoURL, nil)
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode >= 400 {
			addIssue("repo_url", "unreachable", "repo", "mirror repository is unreachable", "Verify repo_url connectivity from bootstrap network")
		}
	}
	if strings.TrimSpace(cfg.ImageRepository) != "" {
		registryURL := cfg.ImageRepository
		if !strings.HasPrefix(registryURL, "http") {
			registryURL = "https://" + strings.TrimSuffix(registryURL, "/") + "/v2/"
		}
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(registryURL)
		if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 401) {
			warnings = append(warnings, "image_repository reachability check failed; bootstrap may fail to pull images")
		}
	}
	if cfg.EndpointMode != "nodeIP" && strings.TrimSpace(cfg.ControlPlaneEndpoint) != "" {
		conn, err := net.DialTimeout("tcp", cfg.ControlPlaneEndpoint, 2*time.Second)
		if err != nil {
			warnings = append(warnings, "control_plane_endpoint is not currently reachable; verify VIP/LB network path")
		} else {
			_ = conn.Close()
		}
	}
	if strings.EqualFold(cfg.EtcdMode, "external") {
		endpoints, _ := cfg.ExternalEtcd["endpoints"].([]interface{})
		for _, ep := range endpoints {
			target, _ := ep.(string)
			if strings.TrimSpace(target) == "" {
				continue
			}
			hostPort := strings.TrimPrefix(strings.TrimPrefix(target, "https://"), "http://")
			dialer := &net.Dialer{Timeout: 2 * time.Second}
			conn, err := tls.DialWithDialer(dialer, "tcp", hostPort, &tls.Config{InsecureSkipVerify: true})
			if err != nil {
				addIssue("external_etcd", "tls_handshake_failed", "etcd", fmt.Sprintf("external etcd TLS handshake failed for %s", target), "Check endpoint reachability and certificate chain")
				continue
			}
			_ = conn.Close()
		}
	}

	return issues, warnings
}

func filterIssuesByDomain(issues []BootstrapValidationIssue, domain string) []BootstrapValidationIssue {
	out := make([]BootstrapValidationIssue, 0)
	for _, it := range issues {
		if it.Domain == domain {
			out = append(out, it)
		}
	}
	return out
}

func suggestSupportedVersions(items []BootstrapVersionItem, limit int) []string {
	out := make([]string, 0, limit)
	for _, it := range items {
		if it.Status != "supported" {
			continue
		}
		out = append(out, it.Version)
		if len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return []string{"1.28.0"}
	}
	return out
}

func formatValidationIssues(issues []BootstrapValidationIssue) string {
	if len(issues) == 0 {
		return ""
	}
	msgs := make([]string, 0, len(issues))
	for _, it := range issues {
		msgs = append(msgs, fmt.Sprintf("%s: %s", it.Field, it.Message))
	}
	return strings.Join(msgs, "; ")
}

func defaultIfEmpty(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstMap(values ...map[string]interface{}) map[string]interface{} {
	for _, v := range values {
		if len(v) > 0 {
			return v
		}
	}
	return map[string]interface{}{}
}

func parseWorkerIDs(jsonStr string) []uint {
	var ids []uint
	json.Unmarshal([]byte(jsonStr), &ids)
	return ids
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
