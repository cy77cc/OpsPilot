// Package handler 提供 Kubernetes 集群管理的 HTTP Handler 实现。
//
// 本文件实现集群服务相关的 HTTP Handler，处理已部署服务查询，以及
// Service/Ingress 基础写操作的校验、审批与审计。
package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	corev1typed "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingv1typed "k8s.io/client-go/kubernetes/typed/networking/v1"
)

// ClusterServiceInfo 集群服务信息响应结构。
type ClusterServiceInfo struct {
	ID           uint   `json:"id"`             // 服务 ID
	Name         string `json:"name"`           // 服务名称
	ProjectName  string `json:"project_name"`   // 项目名称
	TeamName     string `json:"team_name"`      // 团队名称
	Env          string `json:"env"`            // 环境标识
	LastDeployAt string `json:"last_deploy_at"` // 最后部署时间
	Status       string `json:"status"`         // 部署状态
}

type kubernetesServiceIngressClient interface {
	CoreV1() corev1typed.CoreV1Interface
	NetworkingV1() networkingv1typed.NetworkingV1Interface
}

type serviceIngressMutationTarget struct {
	Namespace       string
	Resource        string
	Name            string
	Action          string
	RequireApproval bool
}

type ServiceMutationReq struct {
	Name          string               `json:"name"`
	Type          string               `json:"type"`
	Selector      map[string]string    `json:"selector"`
	Ports         []ServiceMutationPort `json:"ports"`
	ApprovalToken string               `json:"approval_token"`
}

type ServiceMutationPort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port"`
	Protocol   string `json:"protocol"`
	NodePort   int32  `json:"node_port,omitempty"`
}

type IngressMutationReq struct {
	Name             string                `json:"name"`
	IngressClassName string                `json:"ingress_class_name,omitempty"`
	Rules            []IngressMutationRule `json:"rules"`
	TLS              []IngressMutationTLS  `json:"tls,omitempty"`
	ApprovalToken    string                `json:"approval_token"`
}

type IngressMutationRule struct {
	Host  string                `json:"host"`
	Paths []IngressMutationPath `json:"paths"`
}

type IngressMutationPath struct {
	Path        string `json:"path"`
	PathType    string `json:"path_type,omitempty"`
	ServiceName string `json:"service_name"`
	ServicePort int32  `json:"service_port"`
}

type IngressMutationTLS struct {
	SecretName string   `json:"secret_name,omitempty"`
	Hosts      []string `json:"hosts"`
}

type deleteApprovalReq struct {
	ApprovalToken string `json:"approval_token"`
}

// GetClusterServices 获取集群已部署的服务列表。
//
// @Summary 获取集群服务列表
// @Description 获取部署到指定集群的所有服务信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/services [get]
func (h *Handler) GetClusterServices(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	var targets []model.DeploymentTarget
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Where("cluster_id = ?", id).
		Find(&targets).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]ClusterServiceInfo, 0)
	for _, target := range targets {
		var release model.DeploymentRelease
		if err := h.svcCtx.DB.WithContext(c.Request.Context()).
			Where("target_id = ?", target.ID).
			Order("id DESC").
			First(&release).Error; err == nil {

			var service model.Service
			if err := h.svcCtx.DB.WithContext(c.Request.Context()).
				First(&service, release.ServiceID).Error; err == nil {

				projectName := ""
				if target.ProjectID > 0 {
					var project model.Project
					if err := h.svcCtx.DB.WithContext(c.Request.Context()).
						First(&project, target.ProjectID).Error; err == nil {
						projectName = project.Name
					}
				}

				items = append(items, ClusterServiceInfo{
					ID:           service.ID,
					Name:         service.Name,
					ProjectName:  projectName,
					TeamName:     "",
					Env:          target.Env,
					LastDeployAt: release.CreatedAt.Format("2006-01-02 15:04:05"),
					Status:       release.Status,
				})
			}
		}
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// CreateService 创建 Service。
func (h *Handler) CreateService(c *gin.Context) {
	h.CreateServiceImpl(c)
}

// UpdateService 更新 Service。
func (h *Handler) UpdateService(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	clusterID, namespace, ok := parseNamespacedMutation(c)
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpx.BindErr(c, nil)
		return
	}

	var req ServiceMutationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	req.Name = name
	if err := validateServiceMutationReq(req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), clusterID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if _, err := client.CoreV1().Services(namespace).Get(c.Request.Context(), name, metav1.GetOptions{}); err != nil {
		h.handleServiceIngressMutationError(c, "service", err)
		return
	}

	target := serviceIngressMutationTarget{
		Namespace:       namespace,
		Resource:        "service",
		Name:            name,
		Action:          "service.update",
		RequireApproval: serviceMutationRequiresApproval(req),
	}
	result, err := h.executeServiceIngressMutationWithClient(c, clusterID, target, req.ApprovalToken, client, func(ctx context.Context, kubeClient kubernetesServiceIngressClient) (map[string]any, error) {
		return h.updateService(ctx, kubeClient, namespace, req)
	})
	if err != nil {
		h.handleServiceIngressMutationError(c, "service", err)
		return
	}
	httpx.OK(c, result)
}

// DeleteService 删除 Service。
func (h *Handler) DeleteService(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	clusterID, namespace, ok := parseNamespacedMutation(c)
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpx.BindErr(c, nil)
		return
	}

	var req deleteApprovalReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpx.BindErr(c, err)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), clusterID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	target := serviceIngressMutationTarget{
		Namespace:       namespace,
		Resource:        "service",
		Name:            name,
		Action:          "service.delete",
		RequireApproval: true,
	}
	result, err := h.executeServiceIngressMutationWithClient(c, clusterID, target, req.ApprovalToken, client, func(ctx context.Context, kubeClient kubernetesServiceIngressClient) (map[string]any, error) {
		return h.deleteService(ctx, kubeClient, namespace, name)
	})
	if err != nil {
		h.handleServiceIngressMutationError(c, "service", err)
		return
	}
	httpx.OK(c, result)
}

// CreateIngress 创建 Ingress。
func (h *Handler) CreateIngress(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	clusterID, namespace, ok := parseNamespacedMutation(c)
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	var req IngressMutationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if err := validateIngressMutationReq(req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), clusterID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if _, err := client.NetworkingV1().Ingresses(namespace).Get(c.Request.Context(), req.Name, metav1.GetOptions{}); err == nil {
		httpx.BadRequest(c, "ingress already exists")
		return
	} else if err != nil && !apierrors.IsNotFound(err) {
		h.handleServiceIngressMutationError(c, "ingress", err)
		return
	}

	target := serviceIngressMutationTarget{
		Namespace:       namespace,
		Resource:        "ingress",
		Name:            req.Name,
		Action:          "ingress.create",
		RequireApproval: ingressMutationRequiresApproval(req),
	}
	result, err := h.executeServiceIngressMutationWithClient(c, clusterID, target, req.ApprovalToken, client, func(ctx context.Context, kubeClient kubernetesServiceIngressClient) (map[string]any, error) {
		return h.createIngress(ctx, kubeClient, namespace, req)
	})
	if err != nil {
		h.handleServiceIngressMutationError(c, "ingress", err)
		return
	}
	httpx.OK(c, result)
}

// UpdateIngress 更新 Ingress。
func (h *Handler) UpdateIngress(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	clusterID, namespace, ok := parseNamespacedMutation(c)
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpx.BindErr(c, nil)
		return
	}

	var req IngressMutationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	req.Name = name
	if err := validateIngressMutationReq(req); err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), clusterID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if _, err := client.NetworkingV1().Ingresses(namespace).Get(c.Request.Context(), name, metav1.GetOptions{}); err != nil {
		h.handleServiceIngressMutationError(c, "ingress", err)
		return
	}

	target := serviceIngressMutationTarget{
		Namespace:       namespace,
		Resource:        "ingress",
		Name:            name,
		Action:          "ingress.update",
		RequireApproval: ingressMutationRequiresApproval(req),
	}
	result, err := h.executeServiceIngressMutationWithClient(c, clusterID, target, req.ApprovalToken, client, func(ctx context.Context, kubeClient kubernetesServiceIngressClient) (map[string]any, error) {
		return h.updateIngress(ctx, kubeClient, namespace, req)
	})
	if err != nil {
		h.handleServiceIngressMutationError(c, "ingress", err)
		return
	}
	httpx.OK(c, result)
}

// DeleteIngress 删除 Ingress。
func (h *Handler) DeleteIngress(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	clusterID, namespace, ok := parseNamespacedMutation(c)
	if !ok {
		httpx.BindErr(c, nil)
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpx.BindErr(c, nil)
		return
	}

	var req deleteApprovalReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpx.BindErr(c, err)
		return
	}

	client, err := h.getClusterClient(c.Request.Context(), clusterID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	target := serviceIngressMutationTarget{
		Namespace:       namespace,
		Resource:        "ingress",
		Name:            name,
		Action:          "ingress.delete",
		RequireApproval: true,
	}
	result, err := h.executeServiceIngressMutationWithClient(c, clusterID, target, req.ApprovalToken, client, func(ctx context.Context, kubeClient kubernetesServiceIngressClient) (map[string]any, error) {
		return h.deleteIngress(ctx, kubeClient, namespace, name)
	})
	if err != nil {
		h.handleServiceIngressMutationError(c, "ingress", err)
		return
	}
	httpx.OK(c, result)
}

func parseNamespacedMutation(c *gin.Context) (uint, string, bool) {
	clusterID := httpx.UintFromParam(c, "id")
	namespace := strings.TrimSpace(c.Param("namespace"))
	if clusterID == 0 || namespace == "" {
		return 0, "", false
	}
	return clusterID, namespace, true
}

func validateServiceMutationReq(req ServiceMutationReq) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return fmt.Errorf("name required")
	}
	serviceType := normalizeServiceType(req.Type)
	if serviceType == "" {
		return fmt.Errorf("service type must be one of ClusterIP, NodePort or LoadBalancer")
	}
	if len(req.Ports) == 0 {
		return fmt.Errorf("ports required")
	}
	if len(req.Selector) == 0 {
		return fmt.Errorf("selector required")
	}
	for key, value := range req.Selector {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return fmt.Errorf("selector key/value required")
		}
	}
	for _, port := range req.Ports {
		if port.Port <= 0 || port.Port > 65535 {
			return fmt.Errorf("service port must be between 1 and 65535")
		}
		if strings.TrimSpace(port.TargetPort) == "" {
			return fmt.Errorf("target_port required")
		}
		if port.NodePort != 0 && (port.NodePort < 30000 || port.NodePort > 32767) {
			return fmt.Errorf("node_port must be between 30000 and 32767")
		}
		protocol := normalizeServiceProtocol(port.Protocol)
		if protocol == "" {
			return fmt.Errorf("protocol must be TCP or UDP")
		}
		if serviceType == string(corev1.ServiceTypeClusterIP) && port.NodePort != 0 {
			return fmt.Errorf("node_port is only supported for NodePort or LoadBalancer services")
		}
	}
	return nil
}

func validateIngressMutationReq(req IngressMutationReq) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return fmt.Errorf("name required")
	}
	if len(req.Rules) == 0 {
		return fmt.Errorf("rules required")
	}
	for _, rule := range req.Rules {
		if strings.TrimSpace(rule.Host) == "" {
			return fmt.Errorf("ingress host required")
		}
		if len(rule.Paths) == 0 {
			return fmt.Errorf("ingress paths required")
		}
		for _, path := range rule.Paths {
			if strings.TrimSpace(path.Path) == "" {
				return fmt.Errorf("ingress path required")
			}
			if normalizeIngressPathType(path.PathType) == "" {
				return fmt.Errorf("path_type must be Prefix, Exact or ImplementationSpecific")
			}
			if strings.TrimSpace(path.ServiceName) == "" {
				return fmt.Errorf("backend service_name required")
			}
			if path.ServicePort <= 0 || path.ServicePort > 65535 {
				return fmt.Errorf("backend service_port must be between 1 and 65535")
			}
		}
	}
	for _, tls := range req.TLS {
		if len(tls.Hosts) == 0 {
			return fmt.Errorf("tls hosts required")
		}
		for _, host := range tls.Hosts {
			if strings.TrimSpace(host) == "" {
				return fmt.Errorf("tls hosts required")
			}
		}
	}
	return nil
}

func (h *Handler) executeServiceIngressMutationWithClient(c *gin.Context, clusterID uint, target serviceIngressMutationTarget, approvalToken string, client kubernetesServiceIngressClient, fn func(context.Context, kubernetesServiceIngressClient) (map[string]any, error)) (ClusterOperationResponse, error) {
	ctx := c.Request.Context()

	var cluster model.Cluster
	if err := h.svcCtx.DB.WithContext(ctx).First(&cluster, clusterID).Error; err != nil {
		return ClusterOperationResponse{}, fmt.Errorf("cluster not found: %w", err)
	}
	if cluster.Source != "platform_managed" {
		return ClusterOperationResponse{}, fmt.Errorf("cannot modify resources in externally managed cluster")
	}

	if target.RequireApproval {
		gate := h.requireHighRiskApproval(ctx, clusterID, target.Namespace, target.Action, target.Resource, target.Name, approvalToken, httpx.UIDFromCtx(c))
		if !gate.Allowed {
			return operationResponseFromGate(clusterID, target.Resource, target.Name, gate), nil
		}
	}

	details, opErr := fn(ctx, client)
	operatorID := uint(httpx.UIDFromCtx(c))
	if opErr != nil {
		audit, _ := h.recordClusterOperationAuditWithCode(ctx, clusterID, target.Namespace, target.Action, target.Resource, target.Name, "failed", OperationCodeFailed, opErr.Error(), operatorID)
		var auditID uint
		if audit != nil {
			auditID = audit.ID
		}
		return ClusterOperationResponse{
			State:       OperationStateFailed,
			Code:        OperationCodeFailed,
			Message:     sanitizeOperationText(opErr.Error()),
			ClusterID:   clusterID,
			Resource:    target.Resource,
			ResourceID:  target.Name,
			AuditID:     auditID,
			Diagnostics: sanitizeOperationText(opErr.Error()),
		}, nil
	}

	audit, err := h.RecordClusterOperationAudit(ctx, clusterID, target.Namespace, target.Action, target.Resource, target.Name, "success", target.Action+" executed", operatorID)
	if err != nil {
		return operationSuccessResponse(clusterID, target.Resource, target.Name, target.Action+" executed", 0, details), nil
	}
	return operationSuccessResponse(clusterID, target.Resource, target.Name, target.Action+" executed", audit.ID, details), nil
}

func (h *Handler) handleServiceIngressMutationError(c *gin.Context, resource string, err error) {
	switch {
	case err == nil:
		return
	case apierrors.IsNotFound(err):
		httpx.NotFound(c, resource+" not found")
	case apierrors.IsAlreadyExists(err):
		httpx.BadRequest(c, resource+" already exists")
	case err.Error() == "cannot modify resources in externally managed cluster":
		httpx.BadRequest(c, err.Error())
	case strings.Contains(err.Error(), "cluster not found"):
		httpx.NotFound(c, "cluster not found")
	default:
		httpx.ServerErr(c, err)
	}
}

func (h *Handler) createService(ctx context.Context, client kubernetesServiceIngressClient, namespace string, req ServiceMutationReq) (map[string]any, error) {
	service, err := buildServiceObject(namespace, req)
	if err != nil {
		return nil, err
	}
	if _, err := client.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "service", "name": req.Name, "type": string(service.Spec.Type)}, nil
}

func (h *Handler) updateService(ctx context.Context, client kubernetesServiceIngressClient, namespace string, req ServiceMutationReq) (map[string]any, error) {
	existing, err := client.CoreV1().Services(namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	ports, err := buildServicePorts(req.Ports)
	if err != nil {
		return nil, err
	}
	existing.Spec.Type = corev1.ServiceType(normalizeServiceType(req.Type))
	existing.Spec.Selector = trimSelector(req.Selector)
	existing.Spec.Ports = ports
	if _, err := client.CoreV1().Services(namespace).Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "service", "name": req.Name, "type": string(existing.Spec.Type)}, nil
}

func (h *Handler) upsertService(ctx context.Context, client kubernetesServiceIngressClient, namespace string, req ServiceMutationReq) (map[string]any, error) {
	if _, err := client.CoreV1().Services(namespace).Get(ctx, req.Name, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return h.createService(ctx, client, namespace, req)
		}
		return nil, err
	}
	return h.updateService(ctx, client, namespace, req)
}

func (h *Handler) deleteService(ctx context.Context, client kubernetesServiceIngressClient, namespace, name string) (map[string]any, error) {
	if err := client.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "service", "name": name, "deleted": true}, nil
}

func (h *Handler) createIngress(ctx context.Context, client kubernetesServiceIngressClient, namespace string, req IngressMutationReq) (map[string]any, error) {
	ingress, err := buildIngressObject(namespace, req)
	if err != nil {
		return nil, err
	}
	if _, err := client.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "ingress", "name": req.Name, "rules": len(ingress.Spec.Rules)}, nil
}

func (h *Handler) updateIngress(ctx context.Context, client kubernetesServiceIngressClient, namespace string, req IngressMutationReq) (map[string]any, error) {
	existing, err := client.NetworkingV1().Ingresses(namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	rules, tls, className, err := buildIngressSpec(req)
	if err != nil {
		return nil, err
	}
	existing.Spec.Rules = rules
	existing.Spec.TLS = tls
	existing.Spec.IngressClassName = className
	if _, err := client.NetworkingV1().Ingresses(namespace).Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "ingress", "name": req.Name, "rules": len(existing.Spec.Rules)}, nil
}

func (h *Handler) upsertIngress(ctx context.Context, client kubernetesServiceIngressClient, namespace string, req IngressMutationReq) (map[string]any, error) {
	if _, err := client.NetworkingV1().Ingresses(namespace).Get(ctx, req.Name, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return h.createIngress(ctx, client, namespace, req)
		}
		return nil, err
	}
	return h.updateIngress(ctx, client, namespace, req)
}

func (h *Handler) deleteIngress(ctx context.Context, client kubernetesServiceIngressClient, namespace, name string) (map[string]any, error) {
	if err := client.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return nil, err
	}
	return map[string]any{"namespace": namespace, "resource": "ingress", "name": name, "deleted": true}, nil
}

func serviceMutationRequiresApproval(req ServiceMutationReq) bool {
	serviceType := normalizeServiceType(req.Type)
	if serviceType == string(corev1.ServiceTypeNodePort) || serviceType == string(corev1.ServiceTypeLoadBalancer) {
		return true
	}
	for _, port := range req.Ports {
		if port.NodePort > 0 {
			return true
		}
	}
	return false
}

func ingressMutationRequiresApproval(req IngressMutationReq) bool {
	if len(req.TLS) > 0 {
		return true
	}
	for _, rule := range req.Rules {
		host := strings.TrimSpace(strings.ToLower(rule.Host))
		if strings.Contains(host, "*") {
			return true
		}
	}
	return false
}

func buildServiceObject(namespace string, req ServiceMutationReq) (*corev1.Service, error) {
	ports, err := buildServicePorts(req.Ports)
	if err != nil {
		return nil, err
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.TrimSpace(req.Name),
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(normalizeServiceType(req.Type)),
			Selector: trimSelector(req.Selector),
			Ports:    ports,
		},
	}, nil
}

func buildServicePorts(reqPorts []ServiceMutationPort) ([]corev1.ServicePort, error) {
	ports := make([]corev1.ServicePort, 0, len(reqPorts))
	for _, port := range reqPorts {
		targetPort, err := parseTargetPort(port.TargetPort)
		if err != nil {
			return nil, err
		}
		ports = append(ports, corev1.ServicePort{
			Name:       strings.TrimSpace(port.Name),
			Port:       port.Port,
			TargetPort: targetPort,
			Protocol:   corev1.Protocol(normalizeServiceProtocol(port.Protocol)),
			NodePort:   port.NodePort,
		})
	}
	return ports, nil
}

func buildIngressObject(namespace string, req IngressMutationReq) (*networkingv1.Ingress, error) {
	rules, tls, className, err := buildIngressSpec(req)
	if err != nil {
		return nil, err
	}
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.TrimSpace(req.Name),
			Namespace: namespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: className,
			Rules:            rules,
			TLS:              tls,
		},
	}, nil
}

func buildIngressSpec(req IngressMutationReq) ([]networkingv1.IngressRule, []networkingv1.IngressTLS, *string, error) {
	rules := make([]networkingv1.IngressRule, 0, len(req.Rules))
	for _, rule := range req.Rules {
		paths := make([]networkingv1.HTTPIngressPath, 0, len(rule.Paths))
		for _, path := range rule.Paths {
			pathType := networkingv1.PathType(normalizeIngressPathType(path.PathType))
			paths = append(paths, networkingv1.HTTPIngressPath{
				Path:     strings.TrimSpace(path.Path),
				PathType: &pathType,
				Backend: networkingv1.IngressBackend{
					Service: &networkingv1.IngressServiceBackend{
						Name: strings.TrimSpace(path.ServiceName),
						Port: networkingv1.ServiceBackendPort{Number: path.ServicePort},
					},
				},
			})
		}
		rules = append(rules, networkingv1.IngressRule{
			Host: strings.TrimSpace(rule.Host),
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{Paths: paths},
			},
		})
	}

	tls := make([]networkingv1.IngressTLS, 0, len(req.TLS))
	for _, item := range req.TLS {
		hosts := make([]string, 0, len(item.Hosts))
		for _, host := range item.Hosts {
			hosts = append(hosts, strings.TrimSpace(host))
		}
		tls = append(tls, networkingv1.IngressTLS{
			Hosts:      hosts,
			SecretName: strings.TrimSpace(item.SecretName),
		})
	}

	var className *string
	if trimmed := strings.TrimSpace(req.IngressClassName); trimmed != "" {
		className = &trimmed
	}
	return rules, tls, className, nil
}

func trimSelector(selector map[string]string) map[string]string {
	out := make(map[string]string, len(selector))
	for key, value := range selector {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		out[trimmedKey] = trimmedValue
	}
	return out
}

func normalizeServiceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "clusterip":
		return string(corev1.ServiceTypeClusterIP)
	case "nodeport":
		return string(corev1.ServiceTypeNodePort)
	case "loadbalancer":
		return string(corev1.ServiceTypeLoadBalancer)
	default:
		return ""
	}
}

func normalizeServiceProtocol(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "TCP":
		return string(corev1.ProtocolTCP)
	case "UDP":
		return string(corev1.ProtocolUDP)
	default:
		return ""
	}
}

func normalizeIngressPathType(value string) string {
	switch strings.TrimSpace(value) {
	case "", string(networkingv1.PathTypePrefix):
		return string(networkingv1.PathTypePrefix)
	case string(networkingv1.PathTypeExact):
		return string(networkingv1.PathTypeExact)
	case string(networkingv1.PathTypeImplementationSpecific):
		return string(networkingv1.PathTypeImplementationSpecific)
	default:
		return ""
	}
}

func parseTargetPort(value string) (intstr.IntOrString, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return intstr.IntOrString{}, fmt.Errorf("target_port required")
	}
	if number, err := strconv.Atoi(trimmed); err == nil {
		if number <= 0 || number > 65535 {
			return intstr.IntOrString{}, fmt.Errorf("target_port must be between 1 and 65535")
		}
		return intstr.FromInt(number), nil
	}
	return intstr.FromString(trimmed), nil
}
