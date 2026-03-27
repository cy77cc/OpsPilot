// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件实现集群高级操作的 HTTP Handler，包括事件查询、HPA 管理、
// 资源配额、证书管理和集群升级等功能。
package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventInfo 集群事件信息结构。
type EventInfo struct {
	Name      string `json:"name"`       // 事件名称
	Namespace string `json:"namespace"`  // 命名空间
	Type      string `json:"type"`       // 事件类型: Normal/Warning
	Reason    string `json:"reason"`     // 事件原因
	Message   string `json:"message"`    // 事件消息
	Source    string `json:"source"`     // 事件来源
	Count     int32  `json:"count"`      // 发生次数
	Age       string `json:"age"`        // 事件时长
	FirstSeen string `json:"first_seen"` // 首次发生时间
	LastSeen  string `json:"last_seen"`  // 最后发生时间
}

// HPAInfo Horizontal Pod Autoscaler 信息结构。
type HPAInfo struct {
	Name        string          `json:"name"`        // HPA 名称
	Namespace   string          `json:"namespace"`   // 命名空间
	Reference   string          `json:"reference"`   // 目标引用
	MinReplicas int32           `json:"min_replicas"` // 最小副本数
	MaxReplicas int32           `json:"max_replicas"` // 最大副本数
	CurrentCPU  string          `json:"current_cpu"` // 当前 CPU 使用率
	TargetCPU   string          `json:"target_cpu"`  // 目标 CPU 使用率
	CurrentMem  string          `json:"current_mem"` // 当前内存使用率
	TargetMem   string          `json:"target_mem"`  // 目标内存使用率
	Replicas    int32           `json:"replicas"`    // 当前副本数
	Metrics     []HPAMetricInfo `json:"metrics"`     // 指标列表
	Age         string          `json:"age"`         // 创建时长
	CreatedAt   string          `json:"created_at"`  // 创建时间
}

// HPAMetricInfo HPA 指标信息结构。
type HPAMetricInfo struct {
	Name    string `json:"name"`    // 指标名称
	Type    string `json:"type"`    // 指标类型: Resource/Pods/Object/External
	Current string `json:"current"` // 当前值
	Target  string `json:"target"`  // 目标值
}

// ResourceQuotaInfo ResourceQuota 信息结构。
type ResourceQuotaInfo struct {
	Name      string            `json:"name"`      // 配额名称
	Namespace string            `json:"namespace"` // 命名空间
	Hard      map[string]string `json:"hard"`      // 硬限制
	Used      map[string]string `json:"used"`      // 已使用量
	Age       string            `json:"age"`       // 创建时长
	CreatedAt string            `json:"created_at"` // 创建时间
}

// LimitRangeInfo LimitRange 信息结构。
type LimitRangeInfo struct {
	Name      string           `json:"name"`      // 限制范围名称
	Namespace string           `json:"namespace"` // 命名空间
	Type      string           `json:"type"`      // 类型
	Limits    []LimitRangeItem `json:"limits"`    // 限制项列表
	Age       string           `json:"age"`       // 创建时长
	CreatedAt string           `json:"created_at"` // 创建时间
}

// LimitRangeItem 限制范围项结构。
type LimitRangeItem struct {
	Type           string            `json:"type"`            // 类型: Container/Pod/PVC
	Max            map[string]string `json:"max"`             // 最大值
	Min            map[string]string `json:"min"`             // 最小值
	Default        map[string]string `json:"default"`         // 默认值
	DefaultRequest map[string]string `json:"default_request"` // 默认请求值
}

// ClusterVersionInfo 集群版本信息结构。
type ClusterVersionInfo struct {
	KubernetesVersion string `json:"kubernetes_version"` // Kubernetes 版本
	GitVersion        string `json:"git_version"`        // Git 版本
	Platform          string `json:"platform"`           // 平台信息
	GoVersion         string `json:"go_version"`         // Go 版本
}

// ClusterUpgradePlan 集群升级计划结构。
type ClusterUpgradePlan struct {
	CurrentVersion string   `json:"current_version"` // 当前版本
	TargetVersion  string   `json:"target_version"`  // 目标版本
	Upgradable     bool     `json:"upgradable"`      // 是否可升级
	Steps          []string `json:"steps"`           // 升级步骤
	Warnings       []string `json:"warnings"`        // 警告信息
}

// CertificateInfo 证书信息结构。
type CertificateInfo struct {
	Name           string   `json:"name"`            // 证书名称
	ExpiresAt      string   `json:"expires_at"`      // 过期时间
	DaysLeft       int      `json:"days_left"`       // 剩余天数
	CA             bool     `json:"ca"`              // 是否为 CA 证书
	AlternateNames []string `json:"alternate_names"` // 备用名称
}

// GetEvents 获取集群事件列表。
//
// @Summary 获取集群事件
// @Description 获取指定集群的事件列表，支持按命名空间筛选
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace query string false "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/events [get]
func (h *Handler) GetEvents(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Query("namespace")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	var list *corev1.EventList
	if ns != "" {
		list, err = client.CoreV1().Events(ns).List(c.Request.Context(), metav1.ListOptions{})
	} else {
		list, err = client.CoreV1().Events("").List(c.Request.Context(), metav1.ListOptions{})
	}
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]EventInfo, 0, len(list.Items))
	for _, event := range list.Items {
		source := ""
		if event.Source.Component != "" {
			source = event.Source.Component
		} else if event.ReportingController != "" {
			source = event.ReportingController
		}

		items = append(items, EventInfo{
			Name:      event.Name,
			Namespace: event.Namespace,
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Source:    source,
			Count:     event.Count,
			Age:       getAge(event.CreationTimestamp),
			FirstSeen: event.FirstTimestamp.Format("2006-01-02 15:04:05"),
			LastSeen:  event.LastTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetHPAs 获取命名空间的 HPA 列表。
//
// @Summary 获取 HPA 列表
// @Description 获取指定命名空间的 Horizontal Pod Autoscaler 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/hpas [get]
func (h *Handler) GetHPAs(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.AutoscalingV2().HorizontalPodAutoscalers(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]HPAInfo, 0, len(list.Items))
	for _, hpa := range list.Items {
		info := HPAInfo{
			Name:        hpa.Name,
			Namespace:   hpa.Namespace,
			Reference:   fmt.Sprintf("%s/%s", hpa.Spec.ScaleTargetRef.Kind, hpa.Spec.ScaleTargetRef.Name),
			MinReplicas: *hpa.Spec.MinReplicas,
			MaxReplicas: hpa.Spec.MaxReplicas,
			Replicas:    hpa.Status.CurrentReplicas,
			Age:         getAge(hpa.CreationTimestamp),
			CreatedAt:   hpa.CreationTimestamp.Format("2006-01-02 15:04:05"),
			Metrics:     make([]HPAMetricInfo, 0),
		}

		// Parse metrics
		for _, metric := range hpa.Spec.Metrics {
			metricInfo := HPAMetricInfo{}
			switch metric.Type {
			case autoscalingv2.ResourceMetricSourceType:
				metricInfo.Name = string(metric.Resource.Name)
				metricInfo.Type = "Resource"
				if metric.Resource.Target.AverageUtilization != nil {
					metricInfo.Target = fmt.Sprintf("%d%%", *metric.Resource.Target.AverageUtilization)
				} else if metric.Resource.Target.AverageValue != nil {
					metricInfo.Target = metric.Resource.Target.AverageValue.String()
				}
			case autoscalingv2.PodsMetricSourceType:
				metricInfo.Name = metric.Pods.Metric.Name
				metricInfo.Type = "Pods"
			case autoscalingv2.ObjectMetricSourceType:
				metricInfo.Name = metric.Object.Metric.Name
				metricInfo.Type = "Object"
			case autoscalingv2.ExternalMetricSourceType:
				metricInfo.Name = metric.External.Metric.Name
				metricInfo.Type = "External"
			}

			// Get current value from status
			for _, current := range hpa.Status.CurrentMetrics {
				if current.Type == metric.Type {
					if current.Resource != nil && current.Resource.Current.AverageUtilization != nil {
						metricInfo.Current = fmt.Sprintf("%d%%", *current.Resource.Current.AverageUtilization)
					}
					break
				}
			}

			info.Metrics = append(info.Metrics, metricInfo)
		}

		// Set target CPU if exists (for backward compatibility display)
		for _, metric := range hpa.Spec.Metrics {
			if metric.Type == autoscalingv2.ResourceMetricSourceType && metric.Resource.Name == corev1.ResourceCPU {
				if metric.Resource.Target.AverageUtilization != nil {
					info.TargetCPU = fmt.Sprintf("%d%%", *metric.Resource.Target.AverageUtilization)
				}
			}
			if metric.Type == autoscalingv2.ResourceMetricSourceType && metric.Resource.Name == corev1.ResourceMemory {
				if metric.Resource.Target.AverageUtilization != nil {
					info.TargetMem = fmt.Sprintf("%d%%", *metric.Resource.Target.AverageUtilization)
				}
			}
		}

		// Set current CPU/Memory from status
		for _, current := range hpa.Status.CurrentMetrics {
			if current.Type == autoscalingv2.ResourceMetricSourceType {
				if current.Resource.Name == corev1.ResourceCPU && current.Resource.Current.AverageUtilization != nil {
					info.CurrentCPU = fmt.Sprintf("%d%%", *current.Resource.Current.AverageUtilization)
				}
				if current.Resource.Name == corev1.ResourceMemory && current.Resource.Current.AverageUtilization != nil {
					info.CurrentMem = fmt.Sprintf("%d%%", *current.Resource.Current.AverageUtilization)
				}
			}
		}

		items = append(items, info)
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetResourceQuotas 获取命名空间的 ResourceQuota 列表。
//
// @Summary 获取 ResourceQuota 列表
// @Description 获取指定命名空间的 ResourceQuota 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/resourcequotas [get]
func (h *Handler) GetResourceQuotas(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.CoreV1().ResourceQuotas(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]ResourceQuotaInfo, 0, len(list.Items))
	for _, quota := range list.Items {
		hard := make(map[string]string)
		used := make(map[string]string)

		for k, v := range quota.Status.Hard {
			hard[string(k)] = v.String()
		}
		for k, v := range quota.Status.Used {
			used[string(k)] = v.String()
		}

		items = append(items, ResourceQuotaInfo{
			Name:      quota.Name,
			Namespace: quota.Namespace,
			Hard:      hard,
			Used:      used,
			Age:       getAge(quota.CreationTimestamp),
			CreatedAt: quota.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetLimitRanges 获取命名空间的 LimitRange 列表。
//
// @Summary 获取 LimitRange 列表
// @Description 获取指定命名空间的 LimitRange 列表
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param namespace path string true "命名空间"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/namespaces/{namespace}/limitranges [get]
func (h *Handler) GetLimitRanges(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	ns := c.Param("namespace")
	if id == 0 || ns == "" {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	list, err := client.CoreV1().LimitRanges(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]LimitRangeInfo, 0, len(list.Items))
	for _, lr := range list.Items {
		limits := make([]LimitRangeItem, 0, len(lr.Spec.Limits))
		for _, item := range lr.Spec.Limits {
			limitItem := LimitRangeItem{
				Type:           string(item.Type),
				Max:            make(map[string]string),
				Min:            make(map[string]string),
				Default:        make(map[string]string),
				DefaultRequest: make(map[string]string),
			}
			for k, v := range item.Max {
				limitItem.Max[string(k)] = v.String()
			}
			for k, v := range item.Min {
				limitItem.Min[string(k)] = v.String()
			}
			for k, v := range item.Default {
				limitItem.Default[string(k)] = v.String()
			}
			for k, v := range item.DefaultRequest {
				limitItem.DefaultRequest[string(k)] = v.String()
			}
			limits = append(limits, limitItem)
		}

		items = append(items, LimitRangeInfo{
			Name:      lr.Name,
			Namespace: lr.Namespace,
			Type:      string(lr.Spec.Limits[0].Type),
			Limits:    limits,
			Age:       getAge(lr.CreationTimestamp),
			CreatedAt: lr.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetClusterVersion 获取集群版本信息。
//
// @Summary 获取集群版本
// @Description 获取指定集群的 Kubernetes 版本信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=ClusterVersionInfo}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/version [get]
func (h *Handler) GetClusterVersion(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	version, err := client.Discovery().ServerVersion()
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	info := ClusterVersionInfo{
		KubernetesVersion: version.GitVersion,
		GitVersion:        version.GitVersion,
		Platform:          version.Platform,
		GoVersion:         version.GoVersion,
	}

	httpx.OK(c, info)
}

// GetCertificates 获取集群证书信息。
//
// @Summary 获取集群证书
// @Description 获取指定集群的证书信息，包括过期时间
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/certificates [get]
func (h *Handler) GetCertificates(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	// Try to get certificates from kube-system namespace
	// This is typically available on kubeadm-managed clusters
	secrets, err := client.CoreV1().Secrets("kube-system").List(c.Request.Context(), metav1.ListOptions{
		LabelSelector: "kubeadm.io/component",
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]CertificateInfo, 0)
	for _, secret := range secrets.Items {
		if secret.Type != corev1.SecretTypeTLS {
			continue
		}

		// Parse certificate expiry from annotation
		expiresAt := ""
		daysLeft := 0
		if exp, ok := secret.Annotations["kubeadm.io/expiration"]; ok {
			expiresAt = exp
			if t, err := time.Parse(time.RFC3339, exp); err == nil {
				daysLeft = int(time.Until(t).Hours() / 24)
			}
		}

		altNames := []string{}
		if names, ok := secret.Annotations["kubeadm.io/alt-names"]; ok {
			altNames = splitAltNames(names)
		}

		items = append(items, CertificateInfo{
			Name:           secret.Name,
			ExpiresAt:      expiresAt,
			DaysLeft:       daysLeft,
			CA:             secret.Name == "ca" || secret.Name == "etcd-ca",
			AlternateNames: altNames,
		})
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// GetUpgradePlan 获取集群升级计划。
//
// @Summary 获取升级计划
// @Description 获取指定集群的升级计划和步骤
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=ClusterUpgradePlan}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/upgrade-plan [get]
func (h *Handler) GetUpgradePlan(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	version, err := client.Discovery().ServerVersion()
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	currentVersion := version.GitVersion

	// Get available versions from cluster model
	var cluster model.Cluster
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).First(&cluster, id).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}

	// Build upgrade plan
	plan := ClusterUpgradePlan{
		CurrentVersion: currentVersion,
		TargetVersion:  "",
		Upgradable:     false,
		Steps:          []string{},
		Warnings:       []string{},
	}

	// Check if this is a self-hosted cluster
	if cluster.Source == "platform_managed" {
		plan.Upgradable = true
		plan.Steps = []string{
			"1. Backup etcd data and cluster state",
			"2. Upgrade control plane nodes one by one",
			"3. Upgrade worker nodes",
			"4. Verify cluster health after upgrade",
		}
		plan.Warnings = []string{
			"Ensure you have a valid backup before proceeding",
			"Upgrade should be done during maintenance window",
			"Check compatibility of workloads with new version",
		}
	} else {
		plan.Warnings = []string{
			"Only platform-managed clusters support managed upgrades",
			"Imported clusters should be upgraded using their native tools",
		}
	}

	httpx.OK(c, plan)
}

// UpgradeClusterReq 集群升级请求结构。
type UpgradeClusterReq struct {
	TargetVersion string `json:"target_version" binding:"required"` // 目标版本 (必填)
}

// UpgradeClusterResult 集群升级结果结构。
type UpgradeClusterResult struct {
	ClusterID    uint     `json:"cluster_id"`    // 集群 ID
	FromVersion  string   `json:"from_version"`  // 原版本
	ToVersion    string   `json:"to_version"`    // 目标版本
	Status       string   `json:"status"`        // 升级状态
	Message      string   `json:"message"`       // 升级消息
	UpgradeSteps []string `json:"upgrade_steps"` // 升级步骤
}

// UpgradeCluster 升级平台管理的集群。
//
// @Summary 升级集群
// @Description 升级平台管理的 Kubernetes 集群到指定版本
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body UpgradeClusterReq true "升级请求"
// @Success 200 {object} httpx.Response{data=UpgradeClusterResult}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/upgrade [post]
func (h *Handler) UpgradeCluster(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	var req UpgradeClusterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	// Get cluster
	var cluster model.Cluster
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).First(&cluster, id).Error; err != nil {
		httpx.NotFound(c, "cluster not found")
		return
	}

	// Check if platform managed
	if cluster.Source != "platform_managed" {
		httpx.BadRequest(c, "only platform-managed clusters can be upgraded through this API")
		return
	}

	// Get current version
	client, err := h.getClusterClient(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	version, err := client.Discovery().ServerVersion()
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	// For now, return a preview response - actual upgrade would require
	// SSH access to nodes and careful orchestration
	result := UpgradeClusterResult{
		ClusterID:   id,
		FromVersion: version.GitVersion,
		ToVersion:   req.TargetVersion,
		Status:      "preview",
		Message:     "Cluster upgrade would require SSH access to all nodes. This is a preview.",
		UpgradeSteps: []string{
			fmt.Sprintf("1. Drain and cordon control plane nodes"),
			fmt.Sprintf("2. Upgrade kubeadm to v%s on control plane", req.TargetVersion),
			fmt.Sprintf("3. Run 'kubeadm upgrade apply v%s' on control plane", req.TargetVersion),
			fmt.Sprintf("4. Upgrade kubelet and kubectl on control plane"),
			fmt.Sprintf("5. Uncordon control plane nodes"),
			fmt.Sprintf("6. Repeat steps 1-5 for worker nodes"),
			"7. Verify cluster health",
		},
	}

	httpx.OK(c, result)
}

// RenewCertificates 续期集群证书。
//
// @Summary 续期集群证书
// @Description 续期平台管理的集群证书
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/certificates/renew [post]
func (h *Handler) RenewCertificates(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	// Get cluster
	var cluster model.Cluster
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).First(&cluster, id).Error; err != nil {
		httpx.NotFound(c, "cluster not found")
		return
	}

	// Check if platform managed
	if cluster.Source != "platform_managed" {
		httpx.BadRequest(c, "only platform-managed clusters can renew certificates through this API")
		return
	}

	// Get control plane nodes
	var controlPlaneNodes []model.ClusterNode
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Where("cluster_id = ? AND role = ?", id, "control-plane").
		Find(&controlPlaneNodes).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}

	if len(controlPlaneNodes) == 0 {
		httpx.BadRequest(c, "no control plane nodes found")
		return
	}

	// Execute certificate renewal on each control plane node via SSH
	results := make([]map[string]interface{}, 0, len(controlPlaneNodes))
	for _, node := range controlPlaneNodes {
		if node.HostID == nil {
			results = append(results, map[string]interface{}{
				"node_name": node.Name,
				"success":   false,
				"message":   "no associated host for SSH access",
			})
			continue
		}

		var host model.Node
		if err := h.svcCtx.DB.WithContext(c.Request.Context()).First(&host, *node.HostID).Error; err != nil {
			results = append(results, map[string]interface{}{
				"node_name": node.Name,
				"success":   false,
				"message":   "host not found",
			})
			continue
		}

		// Execute kubeadm certs renew all via SSH
		err := h.executeCertRenewal(c.Request.Context(), &host)
		if err != nil {
			results = append(results, map[string]interface{}{
				"node_name": node.Name,
				"host_name": host.Name,
				"success":   false,
				"message":   err.Error(),
			})
		} else {
			results = append(results, map[string]interface{}{
				"node_name": node.Name,
				"host_name": host.Name,
				"success":   true,
				"message":   "certificates renewed successfully",
			})
		}
	}

	httpx.OK(c, gin.H{
		"cluster_id": id,
		"results":    results,
		"message":    fmt.Sprintf("Processed %d control plane nodes", len(controlPlaneNodes)),
	})
}

// executeCertRenewal 在主机上执行证书续期。
//
// 参数:
//   - ctx: 上下文
//   - host: 主机模型
//
// 返回: 失败返回错误
func (h *Handler) executeCertRenewal(ctx context.Context, host *model.Node) error {
	privateKey, passphrase, err := h.loadNodePrivateKey(ctx, host)
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

	// Execute kubeadm certs renew all
	_, err = sshclient.RunCommand(cli, "sudo kubeadm certs renew all")
	return err
}

// splitAltNames 解析备用名称字符串。
//
// 参数:
//   - names: 逗号分隔的名称字符串
//
// 返回: 名称数组
func splitAltNames(names string) []string {
	result := []string{}
	for _, name := range splitString(names, ",") {
		if name != "" {
			result = append(result, name)
		}
	}
	return result
}

// splitString 分割字符串。
//
// 参数:
//   - s: 待分割字符串
//   - sep: 分隔符
//
// 返回: 分割后的字符串数组
func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}
